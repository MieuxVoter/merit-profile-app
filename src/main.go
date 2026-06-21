// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/mieuxvoter/majority-judgment-library-go/judgment"
	"github.com/mieuxvoter/merit-profile-library-go/merit"
	"github.com/tyler-sommer/stick"
	"github.com/tyler-sommer/stick/twig"
	"golang.org/x/text/language"
	"io"
	"log/slog"
	"main/src/input"
	"main/src/locales"
	"main/src/templates"
	"main/src/version"
	"math"
	"net/http"
	"os"
	"slices"
	"strings"
)

// polyglotKey MUST be defined in all available language files.
// We use it as a bit of a workaround to detect a usable language for the user.
var polyglotKey = "AppTitle"

func main() {

	loadDotEnv()
	serverPort, foundServerPort := os.LookupEnv("WEB_PORT")
	if !foundServerPort {
		panic("Environment variable WEB_PORT is required.")
	}

	logger := slog.Default()
	logger.Info("Starting web server…")

	localization := &locales.Localization{
		Logger: logger,
	}
	localization.Init(language.English)

	templateEngine := twig.New(
		&templates.EmbedFSLoader{
			FS: templates.TemplatesFS,
		},
	)
	twigErr := templateEngine.Register(locales.LocalizationExtension{
		Localization: localization,
		Localizers:   make(map[string]*locales.Localizer),
	})
	if twigErr != nil {
		panic(twigErr)
	}

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {

		localizer, userLanguage := localization.NewLocalizerAndLanguage(
			polyglotKey,
			r.Header.Get("Accept-Language"),
		)

		placeholderNames := getPlaceholderNames(localizer)

		query := r.URL.Query()
		useCsv := len(query["csv"]) > 0

		err := templateEngine.Execute(
			"form.html.twig",
			w,
			map[string]stick.Value{
				"placeholderNames": placeholderNames,
				"version":          version.GetVersion(),
				"language":         userLanguage.String(),
				"useCsv":           useCsv,
			},
		)
		if err != nil {
			handleServerError(err, w)
			return
		}
	})

	meritHandler := func(
		w http.ResponseWriter,
		r *http.Request,
		proposalsNames []string,
		proposalsTallies []*judgment.ProposalTally,
		bestOnTheLeft bool,
		doSortWithMj bool,
	) {
		localizer, _ := localization.NewLocalizerAndLanguage(
			polyglotKey,
			r.Header.Get("Accept-Language"),
		)

		//placeholderNames := getPlaceholderNames(localizer)
		amountOfProposals := len(proposalsNames)

		// Consistency and balance check, since input comes straight from userland
		amountOfJudges := uint64(0)
		amountOfGrades := 0
		amountsOfJudges := make([]uint64, amountOfProposals)
		amountsOfGrades := make([]int, amountOfProposals)

		for i := range amountOfProposals {
			currentAmountOfGrades := len(proposalsTallies[i].Tally)
			currentAmountOfJudges := uint64(0)
			for _, t := range proposalsTallies[i].Tally {
				currentAmountOfJudges += t
			}
			amountsOfJudges[i] = currentAmountOfJudges
			amountsOfGrades[i] = currentAmountOfGrades
			if i == 0 {
				amountOfGrades = currentAmountOfGrades
				amountOfJudges = currentAmountOfJudges
			}
		}

		for i := range amountOfProposals {
			if amountsOfGrades[i] != amountOfGrades {
				msg := localizer.T("ErrorTallyInconsistent")
				for j := range amountOfProposals {
					msg += fmt.Sprintf(
						"\n%s: %d %s",
						proposalsNames[j],
						amountsOfGrades[j],
						localizer.Tp("Grade", amountsOfGrades[j]),
					)
				}
				err := errors.New(msg)
				handleUserError(err, w)
				return
			}

			if amountsOfJudges[i] != amountOfJudges {
				msg := localizer.T("ErrorTallyImbalanced")
				for j := range amountOfProposals {
					msg += fmt.Sprintf(
						"\n%s: %d %s",
						proposalsNames[j],
						amountsOfJudges[j],
						localizer.Tp("Judgment", int(amountsOfJudges[j])),
					)
				}
				err := errors.New(msg)
				handleUserError(err, w)
				return
			}
		}

		pollTally := &judgment.PollTally{
			Proposals:      proposalsTallies,
			AmountOfJudges: amountOfJudges,
		}

		// Rule: the "worst" grade is the default grade
		// We will never have to balance, since we do a balance check above, but safe > sorry.
		balanceErr := pollTally.BalanceWithStaticDefault(0)
		if balanceErr != nil {
			handleServerError(balanceErr, w)
			return
		}

		// Rule: proposals may be ranked in the merit profile
		// We compute the rank even if we do not use it.  I'm okay with this, it's cheap.
		deliberator := &judgment.MajorityJudgment{}
		pollResult, deliberationErr := deliberator.Deliberate(pollTally)
		if deliberationErr != nil {
			handleServerError(deliberationErr, w)
			return
		}

		meritProposals := make([]merit.Proposal, amountOfProposals)
		for i := range amountOfProposals {
			actualIndex := i
			if doSortWithMj {
				actualIndex = pollResult.ProposalsSorted[i].Index
			}
			actualName := proposalsNames[actualIndex]
			if doSortWithMj {
				actualName = fmt.Sprintf(
					"#%d ⋅ %s",
					pollResult.ProposalsSorted[i].Rank,
					actualName,
				)
			}
			meritProposals[i] = merit.Proposal{
				Name:  actualName,
				Tally: proposalsTallies[actualIndex].Tally,
			}
		}

		gradesOutlines := make([][]int, amountOfProposals)
		for i, proposal := range pollResult.ProposalsSorted {
			gradesOutlines[i] = []int{int(proposal.Analysis.MedianGrade)}
		}

		renderOptions := []merit.RenderOptions{
			merit.WithBestGradeOnLeft(bestOnTheLeft),
			merit.WithWidth(980),
		}
		if doSortWithMj {
			renderOptions = append(renderOptions, merit.WithGradesOutlines(gradesOutlines))
		}

		svg, renderErr := merit.RenderLinearProfileSVG(
			meritProposals,
			renderOptions...,
		)
		if renderErr != nil {
			handleServerError(renderErr, w)
			return
		}

		w.Header().Add("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte(svg))
	}

	router.Get("/merit.svg", func(w http.ResponseWriter, r *http.Request) {
		localizer, _ := localization.NewLocalizerAndLanguage(
			polyglotKey,
			r.Header.Get("Accept-Language"),
		)

		placeholderNames := getPlaceholderNames(localizer)

		query := r.URL.Query()
		queryProposals := query["n"]
		queryTalliesAsStrings := query["t"]
		queryHighToLow := query["h2l"]
		querySortWithMj := query["mj"]

		bestOnTheLeft := input.CheckboxQueryToBool(queryHighToLow)
		doSortWithMj := input.CheckboxQueryToBool(querySortWithMj)

		amountOfPossibleProposals := len(queryTalliesAsStrings)

		proposalsNames := make([]string, 0)
		proposalsTallies := make([]*judgment.ProposalTally, 0)
		for i := range amountOfPossibleProposals {
			proposalName := placeholderNames[i%len(placeholderNames)]
			if i < len(queryProposals) {
				if queryProposals[i] != "" {
					proposalName = queryProposals[i]
				}
			}

			queryTallyString := strings.TrimSpace(queryTalliesAsStrings[i])
			if queryTallyString == "" {
				continue
			}

			queryTally, err := input.DeserializeTally(queryTallyString)
			if err != nil {
				handleUserError(err, w)
				return
			}
			if bestOnTheLeft {
				slices.Reverse(queryTally)
			}

			if 0 < len(queryTally) {
				tally := &judgment.ProposalTally{
					Tally: queryTally,
				}
				proposalsNames = append(proposalsNames, proposalName)
				proposalsTallies = append(proposalsTallies, tally)
			}
		}

		meritHandler(
			w,
			r,
			proposalsNames,
			proposalsTallies,
			bestOnTheLeft,
			doSortWithMj,
		)
	})

	router.Post("/merit.svg", func(w http.ResponseWriter, r *http.Request) {
		localizer, _ := localization.NewLocalizerAndLanguage(
			polyglotKey,
			r.Header.Get("Accept-Language"),
		)

		parseErr := r.ParseMultipartForm(2048)
		if parseErr != nil {
			handleUserError(parseErr, w)
			return
		}

		queryHighToLow := r.MultipartForm.Value["h2l"]
		querySortWithMj := r.MultipartForm.Value["mj"]

		bestOnTheLeft := input.CheckboxQueryToBool(queryHighToLow)
		doSortWithMj := input.CheckboxQueryToBool(querySortWithMj)

		//w.Write([]byte(fmt.Sprintf("MultipartForm.Value: %v\n", r.MultipartForm.Value)))
		//w.Write([]byte(fmt.Sprintf("MultipartForm.File: %v\n", r.MultipartForm.File)))
		//w.Write([]byte(fmt.Sprintf("Form: %v\n", r.Form)))
		//w.Write([]byte(fmt.Sprintf("PostForm: %v\n", r.PostForm)))
		//w.Write([]byte(fmt.Sprintf("URL.Query(): %v\n", r.URL.Query().Encode())))

		f, foundFile := r.MultipartForm.File["csv"]
		if !foundFile {
			err := errors.New(localizer.T("ErrorNoCsvFile"))
			handleUserError(err, w)
			return
		}
		if len(f) == 0 {
			err := errors.New("multipart file not found")
			handleUserError(err, w)
			return
		}

		handle, openErr := f[0].Open()
		if openErr != nil {
			handleServerError(openErr, w)
			return
		}

		fileHandle, fileHandleMarshallOk := handle.(io.Reader)
		if !fileHandleMarshallOk {
			handleServerError(errors.New("file handle marshall error"), w)
			return
		}

		pr := input.ProfilesCsvReader{}
		tallies, proposals, _, csvErr := pr.Read(&fileHandle, !bestOnTheLeft)
		if csvErr != nil {
			handleUserError(csvErr, w)
			return
		}

		proposalsNames := make([]string, 0)
		proposalsTallies := make([]*judgment.ProposalTally, 0)

		// To handle float values in the CSV file (eg: from normalized tallies), we multiply them.
		coefficient := 1.0
		for i := range tallies {
			for j := range tallies[i] {
				coefficient = math.Max(coefficient, detectCoefficientToInt(tallies[i][j]))
			}
		}

		for i := range tallies {
			proposalsNames = append(proposalsNames, proposals[i])
			uint64Tally := make([]uint64, len(tallies[i]))
			for j, t := range tallies[i] {
				uint64Tally[j] = uint64(coefficient * t)
			}
			tally := &judgment.ProposalTally{
				Tally: uint64Tally,
			}
			proposalsTallies = append(proposalsTallies, tally)
		}

		meritHandler(
			w,
			r,
			proposalsNames,
			proposalsTallies,
			bestOnTheLeft,
			doSortWithMj,
		)
	})

	// We also want to serve some static files, like CSS and the favicon
	staticFiles := http.FileServer(http.Dir("public"))
	router.Handle("/*", http.StripPrefix("/", staticFiles))

	// Finally, let's start the webserver and wait for an interrupting signal
	logger.Info("Visit http://localhost:" + serverPort)
	_ = http.ListenAndServe(":"+serverPort, router)
}

func handleServerError(err error, writer http.ResponseWriter) {
	writer.WriteHeader(500)
	_, _ = writer.Write([]byte(err.Error()))
}

func handleUserError(err error, writer http.ResponseWriter) {
	writer.WriteHeader(401)
	_, _ = writer.Write([]byte(err.Error()))
}

// loadDotEnv loads Environment variables from files, for convenience.
func loadDotEnv() {
	err := godotenv.Load(".env.local")
	if err != nil {
		// It's okay, we do not actually *require* a local env, since we have no secrets.
		//fmt.Println("No .env.local file found.  You may create one by copying .env.")
	}
	err = godotenv.Load() // .env
	if err != nil {
		fmt.Println("No .env file found.  Odd.")
	}
}

// getPlaceholderNames grabs the slice of placeholder (proposal) names from the l10n files.
func getPlaceholderNames(localizer *locales.Localizer) []string {
	return readAsCsvSlice(localizer.T("ProposalNamePlaceholders"))
}

// readAsCsvSlice reads a CSV line into a slice, with some whitespace cleaning, but without any type casting.
func readAsCsvSlice(csvLine string) []string {
	csvSlice := strings.Split(csvLine, ",")
	for i := range csvSlice {
		csvSlice[i] = strings.TrimSpace(csvSlice[i])
	}
	return csvSlice
}

// detectCoefficientToInt returns by how much we must multiply the input to get an integer
// without losing precision.  We stop at one million max (gotta stop *somewhere*).
func detectCoefficientToInt(value float64) float64 {
	p := 1.0
	for p < 1000000.0 {
		if math.Floor(value*p) == (value * p) {
			break
		}
		p *= 10.0
	}
	return p
}
