package main

import (
	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
)

type countryRow struct {
	CountryId sgm.CountryId
	Name      string
}

var countriesCommand = &cobra.Command{
	Use:   "countries SAVEGAME",
	Short: "shows countries",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		gs, err := sgm.LoadGameState(args[0])
		exitOnError(err)

		countries := make([]countryRow, 0, len(gs.Countries))
		for countryId, country := range gs.Countries {
			if country == nil {
				countries = append(countries, countryRow{
					CountryId: countryId,
				})
				continue
			}

			row := countryRow{
				CountryId: countryId,
				Name:      country.Name(),
			}

			countries = append(countries, row)
		}

		tbl := table.New("ID", "Country")
		tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
		for _, row := range countries {
			tbl.AddRow(row.CountryId, row.Name)
		}
		tbl.Print()
	},
}

func init() {
	rootCmd.AddCommand(countriesCommand)
}
