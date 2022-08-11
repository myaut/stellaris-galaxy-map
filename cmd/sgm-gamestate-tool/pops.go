package main

import (
	"fmt"
	"sort"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
)

var (
	popCountryId uint32
)

type popRow struct {
	PopId     sgm.PopId
	CountryId sgm.CountryId
	Planet    string
	Category  string
	SpeciesId sgm.SpeciesId
}

var popsCommand = &cobra.Command{
	Use:   "pops SAVEGAME",
	Short: "shows pops",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		gs, err := sgm.LoadGameState(args[0])
		exitOnError(err)

		pops := make([]popRow, 0, len(gs.Pops))
		for popId, pop := range gs.Pops {
			row := popRow{
				PopId:     popId,
				Category:  pop.Category,
				SpeciesId: pop.SpeciesId,
			}

			if planet, ok := gs.Planets.Planets[pop.PlanetId]; ok {
				row.Planet = planet.Name()
				row.CountryId = planet.OwnerId
			} else {
				row.Planet = fmt.Sprint(pop.PlanetId)
				row.CountryId = sgm.DefaultCountryId
			}

			if popCountryId != uint32(sgm.DefaultCountryId) && popCountryId != uint32(row.CountryId) {
				continue
			}
			pops = append(pops, row)
		}

		sort.Slice(pops, func(i, j int) bool {
			l, r := pops[i], pops[j]
			if l.CountryId != r.CountryId {
				return l.CountryId < r.CountryId
			}
			if l.Planet != r.Planet {
				return l.Planet < r.Planet
			}
			if l.Category != r.Category {
				return l.Category < r.Category
			}
			if l.SpeciesId != r.SpeciesId {
				return l.SpeciesId < r.SpeciesId
			}
			return l.PopId < r.PopId
		})

		tbl := table.New("ID", "Country", "Planet", "Category", "Species")
		tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
		for _, row := range pops {
			tbl.AddRow(row.PopId, row.CountryId, row.Planet, row.Category, row.SpeciesId)
		}
		tbl.Print()
	},
}

func init() {
	rootCmd.AddCommand(popsCommand)

	popsCommand.Flags().Uint32Var(&popCountryId, "country", uint32(sgm.DefaultCountryId),
		"filter by country")
}
