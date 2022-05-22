package sgm

import (
	"archive/zip"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgmparser"
)

type Date string

func (d Date) Year() int {
	s := string(d)
	yearStr := s[:strings.Index(s, ".")]
	year, _ := strconv.Atoi(yearStr)
	return year
}

type StarbaseMgr struct {
	Starbases map[StarbaseId]*Starbase `sgm:"starbases"`
}

type PlanetState struct {
	Planets map[PlanetId]*Planet `sgm:"planet"`
}

type GameState struct {
	Name string `sgm:"name"`
	Date Date   `sgm:"date"`

	Stars    map[StarId]*Star     `sgm:"galactic_object"`
	Planets  PlanetState          `sgm:"planets"`
	Bypasses map[BypassId]*Bypass `sgm:"bypasses"`

	Countries      map[CountryId]*Country             `sgm:"country"`
	Sectors        map[SectorId]*Sector               `sgm:"sectors"`
	StarbaseMgr    StarbaseMgr                        `sgm:"starbase_mgr"`
	Megastructures map[MegastructureId]*Megastructure `sgm:"megastructures"`
	Fleets         map[FleetId]*Fleet                 `sgm:"fleet"`
	Ships          map[ShipId]*Ship                   `sgm:"ships"`
	Wars           map[WarId]*War                     `sgm:"war"`
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
		state.linkStarRefs(starId, star)
	}
	for _, planet := range state.Planets.Planets {
		if planet.OrbitalFleetId != DefaultFleetId {
			planet.OrbitalFleet = state.Fleets[planet.OrbitalFleetId]
		}
	}
	for countryId, country := range state.Countries {
		state.linkCountryRefs(countryId, country)
	}
	for _, fleet := range state.Fleets {
		state.linkFleetRefs(fleet)
	}
	for shipId, ship := range state.Ships {
		if ship != nil {
			ship.Fleet = state.Fleets[ship.FleetId]
		} else {
			log.Printf("warn: ship #%d is nil", shipId)
		}
	}
	for starbaseId, starbase := range state.StarbaseMgr.Starbases {
		if starbase != nil {
			starbase.Station = state.Ships[starbase.StationId]
			if starbase.Station != nil && starbase.Station.Fleet != nil {
				starbase.Station.Fleet.Starbase = starbase
			}
		} else {
			log.Printf("warn: starbase #%d is nil", starbaseId)
		}
	}
	for warId, war := range state.Wars {
		state.linkWarRefs(warId, war)
	}

	return state, nil
}

func (state *GameState) linkStarRefs(starId StarId, star *Star) {
	if star.SectorId != DefaultSectorId {
		star.Sector = state.Sectors[star.SectorId]
		if star.Sector == nil {
			log.Printf("error: sector #%d is not found", star.SectorId)
		}
	}

	if len(star.StarbaseIds) == 0 {
		star.StarbaseIds = []StarbaseId{star.StarbaseId}
	}
	for _, starbaseId := range star.StarbaseIds {
		if starbaseId == DefaultStarbaseId {
			continue
		}
		starbase := state.StarbaseMgr.Starbases[starbaseId]
		if starbase != nil {
			star.Starbases = append(star.Starbases, starbase)
			starbase.Star = star
		} else {
			log.Printf("error: starbase #%d is not found", starbaseId)
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

	for _, fleetId := range star.FleetIds {
		if fleet := state.Fleets[fleetId]; fleet != nil {
			star.Fleets = append(star.Fleets, fleet)
		} else {
			log.Printf("error: fleet #%d is not found", fleetId)
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

func (state *GameState) linkCountryRefs(countryId CountryId, country *Country) {
	if country == nil {
		return
	}

	for _, ownedFleet := range country.FleetMgr.OwnedFleets {
		if fleet := state.Fleets[ownedFleet.FleetId]; fleet != nil {
			ownedFleet.Fleet = fleet
			fleet.OwnerId = countryId
			fleet.Owner = country
			fleet.OwnershipStatus = ownedFleet.OwnershipStatus
			fleet.DebtorId = ownedFleet.DebtorId
		} else {
			log.Printf("error: fleet #%d is not found owned by #%d",
				ownedFleet.FleetId, countryId)
		}
	}
}

func (state *GameState) linkFleetRefs(fleet *Fleet) {
	if fleet == nil {
		return
	}

	if fleet.Owner == nil {
		fleet.Owner = state.Countries[fleet.OwnerId]
	}
	for _, shipId := range fleet.ShipIds {
		if ship := state.Ships[shipId]; ship != nil {
			fleet.Ships = append(fleet.Ships, ship)
		}
	}
}

func (state *GameState) linkWarRefs(warId WarId, war *War) {
	if war == nil {
		return
	}

	for battleIndex, battle := range war.Battles {
		if battle.Type != BattleTypeShips {
			continue
		}

		if star := state.Stars[battle.StarId]; star != nil {
			star.Battles = append(star.Battles, BattleRef{warId, battleIndex})
		}
	}

	for _, attacker := range war.Attackers {
		if country := state.Countries[attacker.CountryId]; country != nil {
			country.Wars = append(country.Wars, WarRef{warId, war, true})
		}
	}
	for _, defender := range war.Defenders {
		if country := state.Countries[defender.CountryId]; country != nil {
			country.Wars = append(country.Wars, WarRef{warId, war, false})
		}
	}
}
