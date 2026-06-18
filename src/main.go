package main

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mieuxvoter/majority-judgment-library-go/judgment"
	"github.com/mieuxvoter/merit-profile-library-go/merit"
	"github.com/tyler-sommer/stick"
	"github.com/tyler-sommer/stick/twig"
	"log/slog"
	"main/src/templates"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

var placeholderNames = []string{
	"Pizza",
	"Pasta",
	"Rice",
	"Soup",
	"Bread",
	"Salad",
	"Chips",
	"Maki",
	"Burger",
	"Lentils",
	"Tacos",
	"Lasagna",
	"Beans",
	"Gnocchi",
	"Spinach",
	"Cake",
}

func main() {

	serverPort := "8033" // TODO: get from env

	logger := slog.Default()
	logger.Info("Starting web server…")

	templateEngine := twig.New(
		&templates.EmbedFSLoader{
			FS: templates.TemplatesFS,
		},
	)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err := templateEngine.Execute(
			"index.html.twig",
			w,
			map[string]stick.Value{
				"placeholderNames": placeholderNames,
			},
		)
		if err != nil {
			handleServerError(err, w)
		}
	})

	router.Get("/merit.svg", func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		queryProposals := query["name"]
		queryTalliesAsStrings := query["tally"]
		queryH2l := query["h2l"]

		// debug
		//w.Write([]byte(fmt.Sprintf("%d queryH2l: %v\n", len(queryH2l), queryH2l)))
		//w.Write([]byte(fmt.Sprintf("query: %v\n", query)))
		//w.Write([]byte(fmt.Sprintf("%d proposals: %v\n", len(queryProposals), queryProposals)))
		//w.Write([]byte(fmt.Sprintf("%d tallies: %v\n", len(queryTalliesAsStrings), queryTalliesAsStrings)))

		bestOnTheLeft := false
		if len(queryH2l) > 0 {
			if queryH2l[0] == "on" {
				bestOnTheLeft = true
			}
		}
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

			queryTally, err := deserializeTally(queryTallyString)
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

		meritProposals := make([]merit.Proposal, amountOfProposals)
		for i := range amountOfProposals {
			meritProposal := merit.Proposal{
				Name:  proposalsNames[i],
				Tally: proposalsTallies[i].Tally,
			}
			meritProposals[i] = meritProposal
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

		_, _ = w.Write([]byte(svg))
	})

	logger.Info("Visit http://localhost:" + serverPort)
	_ = http.ListenAndServe(":"+serverPort, router)
}

func handleServerError(err error, writer http.ResponseWriter) {
	_, _ = writer.Write([]byte(err.Error()))
}

func handleUserError(err error, writer http.ResponseWriter) {
	_, _ = writer.Write([]byte(err.Error()))
}

func deserializeTally(tallyAsString string) ([]uint64, error) {
	spliceOfStrings := strings.Split(tallyAsString, ",")
	out := make([]uint64, len(spliceOfStrings))
	for i, s := range spliceOfStrings {
		t, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return out, err
		}
		out[i] = t
	}
	return out, nil
}
