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
	"github.com/strukturag/goacceptlanguageparser"
	"github.com/tyler-sommer/stick"
	"github.com/tyler-sommer/stick/twig"
	"log/slog"
	"main/src/input"
	"main/src/locales"
	"main/src/templates"
	"main/src/version"
	"net/http"
	"os"
	"slices"
	"strings"
)

var placeholderNames = []string{
	"Arancini",
	"Burger",
	"Chips",
	"Dal",
	"Empanadas",
	"Fries",
	"Gnocchis",
	"Kale",
	"Lasagna",
	"Makis",
	"Noodles",
	"Oatmeal",
	"Pizza",
	"Rice",
	"Soup",
	"Tacos",
	"Veggies",
}

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
	localization.Init()

	templateEngine := twig.New(
		&templates.EmbedFSLoader{
			FS: templates.TemplatesFS,
		},
	)
	err := templateEngine.Register(locales.LocalizationExtension{
		Localization: localization,
	})
	if err != nil {
		panic(err)
	}

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {

		userAcceptedLanguages := goacceptlanguageparser.ParseAcceptLanguage(
			r.Header.Get("Accept-Language"),
			[]string{"en", "fr"}, // TODO: handle the list of supported languages
		)
		language := "en"
		if len(userAcceptedLanguages) > 0 {
			language = userAcceptedLanguages[0]
		}

		err := templateEngine.Execute(
			"index.html.twig",
			w,
			map[string]stick.Value{
				"placeholderNames": placeholderNames,
				"version":          version.GetVersion(),
				"language":         language,
			},
		)
		if err != nil {
			handleServerError(err, w)
		}
	})

	router.Get("/merit.svg", func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		queryProposals := query["n"]
		queryTalliesAsStrings := query["t"]
		queryHighToLow := query["h2l"]
		querySortWithMj := query["mj"]

		// debug
		//w.Write([]byte(fmt.Sprintf("%d queryHighToLow: %v\n", len(queryHighToLow), queryHighToLow)))
		//w.Write([]byte(fmt.Sprintf("query: %v\n", query)))
		//w.Write([]byte(fmt.Sprintf("%d proposals: %v\n", len(queryProposals), queryProposals)))
		//w.Write([]byte(fmt.Sprintf("%d tallies: %v\n", len(queryTalliesAsStrings), queryTalliesAsStrings)))

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

		amountOfProposals := len(proposalsNames)

		// Consistency and balance check, since input comes straight from userland
		amountOfJudges := uint64(0)
		amountOfGrades := 0
		for i := range amountOfProposals {
			currentAmountOfGrades := len(proposalsTallies[i].Tally)
			currentAmountOfJudges := uint64(0)
			for _, t := range proposalsTallies[i].Tally {
				currentAmountOfJudges += t
			}
			if i == 0 {
				amountOfGrades = currentAmountOfGrades
				amountOfJudges = currentAmountOfJudges
				continue
			}

			if currentAmountOfGrades != amountOfGrades {
				err := errors.New(fmt.Sprintf(
					"The amount of grades for proposal #%d (%s) is %d which is different from %d, the expected amount of grades.  Please make sure your tallies are consistent.",
					i,
					proposalsNames[i],
					currentAmountOfGrades,
					amountOfGrades,
				))
				handleUserError(err, w)
				return
			}

			if currentAmountOfJudges != amountOfJudges {
				err := errors.New(fmt.Sprintf(
					"The total amount of judgments for proposal #%d (%s) is %d which is different from %d, the expected amount of judgments.  Please make sure your tallies are balanced.",
					i,
					proposalsNames[i],
					currentAmountOfJudges,
					amountOfJudges,
				))
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
			return
		}

		// Rule: proposals may be ranked in the merit profile
		// We compute the rank even if we do not use it.  I'm okay with this, it's cheap.
		deliberator := &judgment.MajorityJudgment{}
		pollResult, err := deliberator.Deliberate(pollTally)
		if err != nil {
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

		renderOptions := []merit.RenderOptions{
			merit.WithBestGradeOnLeft(bestOnTheLeft),
			merit.WithWidth(980),
		}
		svg, err := merit.RenderLinearProfileSVG(
			meritProposals,
			renderOptions...,
		)
		if err != nil {
			handleServerError(err, w)
		}

		w.Header().Add("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte(svg))
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
		//fmt.Println("No .env.local file found.  You may create one by copying .env.")
	}
	err = godotenv.Load() // .env
	if err != nil {
		fmt.Println("No .env file found.  Odd.")
	}
}
