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

	MegastructureGateway         = "gateway"
	MegastructureLGate           = "lgate_base"
	MegastructureHyperRelay      = "hyper_relay"
	MegastructureQuantumCatapult = "quantum_catapult"

	MegastructureSizeRingWorld = 3
	MegastructureSizeStar      = 2
	MegastructureSizePlanet    = 1

	FleetOwnershipNormal      = "normal"
	FleetOwnershipLostControl = "lost_control"

	BattleTypeShips  = "ships"
	BattleTypeArmies = "armies"
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

type WarRole int

const (
	WarRoleStarNeutral WarRole = iota
	WarRoleStarDefender
	WarRoleStarAttacker

	WarRoleMax
)

type StationId uint32
type Station struct {
	FleetId FleetId `sgm:"fleet"`
	Fleet   *Fleet
}

var DefaultStarbaseId = StarbaseId(math.MaxUint32)

type StarbaseId uint32
type Starbase struct {
	Level     string         `sgm:"level"`
	Modules   map[int]string `sgm:"modules"`
	Buildings map[int]string `sgm:"buildings"`
	Owner     CountryId      `sgm:"owner,id"`

	StationId ShipId `sgm:"station,id"`
	Station   *Ship

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

type OwnedFleet struct {
	FleetId FleetId `sgm:"fleet"`
	Fleet   *Fleet

	OwnershipStatus string `sgm:"ownership_status"`

	DebtorId CountryId `sgm:"debtor,id"`
}

var DefaultCountryId = CountryId(math.MaxUint32)

type CountryId uint32
type Country struct {
	NameString string `sgm:"name"`
	NameStruct Name   `sgm:"name,struct"`

	Flag CountryFlag `sgm:"flag"`

	CapitalId PlanetId `sgm:"capital"`
	Capital   *Planet

	FleetMgr struct {
		OwnedFleets []OwnedFleet `sgm:"owned_fleets"`
	} `sgm:"fleets_manager"`

	Wars []WarRef
}

type CountryFlag struct {
	Colors []string `sgm:"colors"`
}

func (c *Country) Name() string {
	if c.NameString == "" {
		c.NameString = c.NameStruct.Format(CountryNameFormats)
	}
	return c.NameString
}

func CountryName(countryId CountryId, country *Country) string {
	if country != nil {
		return country.Name()
	}
	if countryId == DefaultCountryId {
		return "_default"
	}
	return fmt.Sprint(countryId)
}

var DefaultSectorId = SectorId(math.MaxUint32)

type SectorId uint32
type Sector struct {
	NameString string `sgm:"name"`
	NameStruct struct {
		Key string `sgm:"key"`
	} `sgm:"name,struct"`

	Type  string    `sgm:"type"`
	Owner CountryId `sgm:"owner,id"`
}

func (s *Sector) Name() string {
	if s.NameString != "" {
		return s.NameString
	}
	return s.NameStruct.Key
}

type MegastructureId uint32
type Megastructure struct {
	Type     string   `sgm:"type"`
	Owner    int      `sgm:"owner"`
	PlanetId PlanetId `sgm:"planet,id"`

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

var DefaultFleetId = FleetId(math.MaxUint32)

type FleetId uint32
type Fleet struct {
	NameString string `sgm:"name"`
	NameStruct struct {
		Key string `sgm:"key"`
	} `sgm:"name,struct"`

	Station  bool `sgm:"station"`
	Mobile   bool `sgm:"mobile"`
	Civilian bool `sgm:"civilian"`

	MilitaryPower float64 `sgm:"military_power"`

	OwnerId         CountryId `sgm:"owner,id"`
	Owner           *Country
	OwnershipStatus string
	DebtorId        CountryId `sgm:"-,id"`

	ShipIds []ShipId `sgm:"ships"`
	Ships   []*Ship

	Starbase *Starbase
}

func (f *Fleet) Name() string {
	if f.NameString != "" {
		return f.NameString
	}
	return f.NameStruct.Key
}

func (fleet *Fleet) IsTransport() bool {
	for _, ship := range fleet.Ships {
		if ship.ArmyId == DefaultArmyId {
			return false
		}
	}
	return true
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

type ShipId uint32
type Ship struct {
	FleetId FleetId `sgm:"fleet,id"`
	Fleet   *Fleet

	ArmyId ArmyId `sgm:"army,id"`
}

var DefaultArmyId = ArmyId(math.MaxUint32)

type ArmyId uint32

type WarId uint32
type War struct {
	StartDate Date `sgm:"start_date"`

	Defenders []WarCountry `sgm:"defenders"`
	Attackers []WarCountry `sgm:"attackers"`

	Battles []Battle `sgm:"battles"`
}

type WarCountry struct {
	CountryId CountryId `sgm:"country,id"`
}

type Battle struct {
	DefenderIds []CountryId `sgm:"defenders"`
	AttackerIds []CountryId `sgm:"attackers"`

	AttackerVictory bool   `sgm:"attacker_victory"`
	Date            Date   `sgm:"date"`
	Type            string `sgm:"type"`

	StarId StarId `sgm:"system,id"`

	AttackerLosses int `sgm:"attacker_losses"`
	DefenderLosses int `sgm:"defender_losses"`
}

type WarRef struct {
	WarId      WarId
	War        *War
	IsAttacker bool
}

type BattleRef struct {
	WarId       WarId
	BattleIndex int
}

func ComputeWarRole(countryId CountryId, country *Country, s *Star) WarRole {
	if len(country.Wars) == 0 {
		return WarRoleStarNeutral
	}

	starCountryId := s.Owner()
	if starCountryId == countryId {
		return WarRoleStarDefender
	}

	for _, warRef := range country.Wars {
		allyList, enemyList := warRef.War.Defenders, warRef.War.Attackers
		if warRef.IsAttacker {
			allyList, enemyList = enemyList, allyList
		}

		for _, country := range enemyList {
			if country.CountryId == starCountryId {
				return WarRoleStarAttacker
			}
		}
		for _, country := range allyList {
			if country.CountryId == starCountryId {
				return WarRoleStarDefender
			}
		}
	}

	// These fleets are merely passing by
	return WarRoleStarNeutral
}
