package sgm

import (
	"fmt"
	"math"

	"strings"
)

const (
	SectorCore   = "core_sector"
	SectorNormal = "normal_sector"

	StarbaseOutpost  = "starbase_level_outpost"
	StarbaseStarport = "starbase_level_starport"
	StarbaseStarhold = "starbase_level_starhold"
	StarbaseFortress = "starbase_level_starfortress"
	StarbaseCitadel  = "starbase_level_citadel"

	StarbaseMarauder   = "starbase_level_marauder"
	StarbaseCaravaneer = "starbase_level_caravaneer"

	StarbaseModuleShipyard       = "shipyard"
	StarbaseModuleTradingHub     = "trading_hub"
	StarbaseModuleAnchorage      = "anchorage"
	StarbaseModuleGunBattery     = "gun_battery"
	StarbaseModuleMissileBattery = "missile_battery"
	StarbaseModuleHangarBay      = "hangar_bay"

	StarbaseBuildingFleetAcademy    = "fleet_academy"
	StarbaseBuildingTitanYards      = "titan_yards"
	StarbaseBuildingColossusYards   = "colossus_yards"
	StarbaseBuildingNavalOffice     = "naval_logistics_office"
	StarbaseBuildingTradingCompany  = "offworld_trading_company"
	StarbaseBuildingCrewQuarters    = "crew_quarters"
	StarbaseBuildingTargetComputer  = "target_uplink_computer"
	StarbaseBuildingCommJammer      = "communications_jammer"
	StarbaseBuildingDefenceGrid     = "defense_grid"
	StarbaseBuildingDisruptionField = "disruption_field"
	StarbaseBuildingWarpFluctuator  = "warp_fluctuator"

	MegastructureRingWorld   = "ring_world"
	MegastructureDysonSphere = "dyson_sphere"
	MegastructureMatterUnzip = "matter_decompressor"

	MegastructureScienceNexus         = "think_tank"
	MegastructureSentryArray          = "spy_orb"
	MegastructureArtInstallation      = "mega_art_installation"
	MegastructureInterstellarAssembly = "interstellar_assembly"
	MegastructureShipyard             = "mega_shipyard"
	MegastructureStrategicCenter      = "strategic_coordination_center"

	MegastructureGateway = "gateway"
	MegastructureLGate   = "lgate_base"

	MegastructureSizeRingWorld = 3
	MegastructureSizeStar      = 2
	MegastructureSizePlanet    = 1
)

var MegastructureSize = map[string]int{
	MegastructureRingWorld: MegastructureSizeRingWorld,

	MegastructureDysonSphere: MegastructureSizeStar,
	MegastructureMatterUnzip: MegastructureSizeStar,

	MegastructureScienceNexus:         MegastructureSizePlanet,
	MegastructureSentryArray:          MegastructureSizePlanet,
	MegastructureArtInstallation:      MegastructureSizePlanet,
	MegastructureInterstellarAssembly: MegastructureSizePlanet,
	MegastructureShipyard:             MegastructureSizePlanet,
	MegastructureStrategicCenter:      MegastructureSizePlanet,
}

type StarbaseRole int

const (
	StarbaseRoleShipyard StarbaseRole = iota
	StarbaseRoleBastion
	StarbaseRoleAnchorage
	StarbaseRoleTradingHub

	StarbaseRoleMax
)

func (role StarbaseRole) String() string {
	return [...]string{
		"shipyard",
		"bastion",
		"anchorage",
		"trading-hub",
		"undefined",
	}[role]
}

var MegastructureStages = map[string]int{
	"ruined":   -1,
	"0":        0,
	"1":        1,
	"2":        2,
	"3":        3,
	"4":        4,
	"5":        5,
	"restored": 10,
	"final":    20,
}

var DefaultStarbaseId = StarbaseId(math.MaxUint32)

type StarbaseId uint32
type Starbase struct {
	Level     string         `sgm:"level"`
	Modules   map[int]string `sgm:"modules"`
	Buildings map[int]string `sgm:"buildings"`
	Owner     CountryId      `sgm:"owner"`

	Star *Star
}

func (sb *Starbase) Role() StarbaseRole {
	rolePoints := make([]int, StarbaseRoleMax)
	for _, mod := range sb.Modules {
		role := StarbaseRoleMax
		points := 1
		switch mod {
		case StarbaseModuleShipyard:
			role = StarbaseRoleShipyard
			points = 2
		case StarbaseModuleAnchorage:
			role = StarbaseRoleAnchorage
		case StarbaseModuleTradingHub:
			role = StarbaseRoleTradingHub
			points = 2
		case StarbaseModuleGunBattery, StarbaseModuleMissileBattery, StarbaseModuleHangarBay:
			role = StarbaseRoleBastion
		}
		if role != StarbaseRoleMax {
			rolePoints[role] += points
		}
	}

	for _, bld := range sb.Buildings {
		role := StarbaseRoleMax
		points := 2
		switch bld {
		case StarbaseBuildingFleetAcademy:
			role = StarbaseRoleShipyard
		case StarbaseBuildingTitanYards, StarbaseBuildingColossusYards:
			role = StarbaseRoleShipyard
			points = 3
		case StarbaseBuildingNavalOffice, StarbaseBuildingCrewQuarters:
			role = StarbaseRoleAnchorage
		case StarbaseBuildingTradingCompany:
			role = StarbaseRoleTradingHub
			points = 3
		case StarbaseBuildingTargetComputer, StarbaseBuildingCommJammer,
			StarbaseBuildingDisruptionField, StarbaseBuildingDefenceGrid,
			StarbaseBuildingWarpFluctuator:
			role = StarbaseRoleBastion
		}
		if role != StarbaseRoleMax {
			rolePoints[role] += points
		}
	}

	maxRole, maxPoints := StarbaseRoleMax, 0
	for role, points := range rolePoints {
		if points > maxPoints {
			maxRole, maxPoints = StarbaseRole(role), points
		}
	}
	return maxRole
}

var DefaultCountryId = CountryId(math.MaxUint32)

type CountryId uint32
type Country struct {
	Name string `sgm:"name"`

	Flag CountryFlag `sgm:"flag"`
}

var DefaultSectorId = SectorId(math.MaxUint32)

type SectorId uint32
type Sector struct {
	Name string `sgm:"name"`
	Type string `sgm:"type"`
}

type MegastructureId uint32
type Megastructure struct {
	Type     string   `sgm:"type"`
	Owner    int      `sgm:"owner"`
	PlanetId PlanetId `sgm:"planet"`

	Star   *Star
	Planet *Planet
}

func (m Megastructure) TypeStage() (string, int) {
	lastDelimPos := strings.LastIndexByte(m.Type, '_')
	if lastDelimPos == -1 {
		return m.Type, 0
	}

	mtype, stage := m.Type[:lastDelimPos], m.Type[lastDelimPos+1:]
	if stageNum, ok := MegastructureStages[stage]; ok {
		return mtype, stageNum
	}
	return m.Type, 0
}

type CountryFlag struct {
	Colors []string `sgm:"colors"`
}

func CountryName(countryId CountryId, country *Country) string {
	if country != nil {
		return country.Name
	}
	if countryId == DefaultCountryId {
		return "_default"
	}
	return fmt.Sprint(countryId)
}

type FleetId uint32
type Fleet struct {
	Name     string `sgm:"name"`
	Station  bool   `sgm:"station"`
	Mobile   bool   `sgm:"mobile"`
	Civilian bool   `sgm:"civilian"`

	MilitaryPower float64 `sgm:"military_power"`
}

func (fleet *Fleet) MilitaryPowerString() string {
	if fleet.MilitaryPower > 5000.0 {
		kiloPower := math.Floor(fleet.MilitaryPower / 1000.0)
		switch {
		case fleet.MilitaryPower > 100000.0:
			return fmt.Sprintf("%.fK", kiloPower)
		}
		return fmt.Sprintf("%.1fK", kiloPower)
	}
	return fmt.Sprintf("%.f", fleet.MilitaryPower)
}
