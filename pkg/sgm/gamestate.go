package sgm

import (
	"archive/zip"
	"fmt"
	"log"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmparser"
)

type StarbaseMgr struct {
	Starbases map[StarbaseId]*Starbase `sgm:"starbases"`
}

type PlanetState struct {
	Planets map[PlanetId]*Planet `sgm:"planet"`
}

type GameState struct {
	Stars    map[StarId]*Star     `sgm:"galactic_object"`
	Planets  PlanetState          `sgm:"planets"`
	Bypasses map[BypassId]*Bypass `sgm:"bypasses"`

	Countries      map[CountryId]*Country             `sgm:"country"`
	StarbaseMgr    StarbaseMgr                        `sgm:"starbase_mgr"`
	Megastructures map[MegastructureId]*Megastructure `sgm:"megastructures"`
}

func LoadGameState(path string) (*GameState, error) {
	savReader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("error loading sav file: %w", err)
	}
	defer savReader.Close()

	stateFile, err := savReader.Open("gamestate")
	if err != nil {
		return nil, fmt.Errorf("error loading gamestate from sav file: %w", err)
	}

	state := &GameState{}
	parser := sgmparser.NewParser(sgmparser.NewTokenizer(stateFile))
	err = parser.Parse(state)
	if err != nil {
		return nil, fmt.Errorf("error parsing gamestate from sav file: %w", err)
	}

	// Build references between objects in the universe
	for starId, star := range state.Stars {
		if star.StarbaseId != DefaultStarbaseId {
			star.Starbase = state.StarbaseMgr.Starbases[star.StarbaseId]
			if star.Starbase != nil {
				star.Starbase.Star = star
			} else {
				log.Printf("error: starbase #%d is not found", star.StarbaseId)
			}
		}

		for _, planetId := range star.PlanetIds {
			if planet := state.Planets.Planets[planetId]; planet != nil {
				star.Planets = append(star.Planets, planet)
				planet.Star = star
			} else {
				log.Printf("error: planet #%d is not found", planetId)
			}
		}

		for _, megastructureId := range star.MegastructureIds {
			if megastructure := state.Megastructures[megastructureId]; megastructure != nil {
				star.Megastructures = append(star.Megastructures, megastructure)
				megastructure.Star = star
				if megastructure.PlanetId != DefaultPlanetId {
					megastructure.Planet = state.Planets.Planets[megastructure.PlanetId]
				}
			} else {
				log.Printf("error: megastructure #%d is not found", megastructureId)
			}
		}

		for i, hyperlane := range star.Hyperlanes {
			if toStar := state.Stars[hyperlane.ToId]; toStar != nil {
				star.Hyperlanes[i].To = toStar
			} else {
				log.Printf("error: star #%d is not found in hyperlane from star #%d",
					hyperlane.ToId, starId)
			}
		}
	}

	return state, nil
}
