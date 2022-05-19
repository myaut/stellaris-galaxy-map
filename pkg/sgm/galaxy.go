package sgm

import (
	"math"
	"sort"
	"strings"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgmmath"
)

const (
	DistantStarInitializer       = "distantstars_init"
	DistantStarInitializerLGate0 = "distantstars_init_00"
	DistantStarInitializerLGate6 = "distantstars_init_06"

	PlanetClassEcumenopolis = "pc_city"
	PlanetClassHabitat      = "pc_habitat"

	PlanetDesignationCapital = "col_capital"

	BypassWormhole      = "wormhole"
	BypassLGate         = "lgate"
	BypassGateway       = "gateway"
	BypassGatewayRuined = "gateway-ruined"
)

var DefaultStarId = StarId(math.MaxUint32)

type StarId int
type Star struct {
	Type        string `sgm:"type"`
	NameString  string `sgm:"name"`
	NameStruct  Name   `sgm:"name,struct"`
	Initializer string `sgm:"initializer"`

	Coordinate Coordinate  `sgm:"coordinate"`
	Hyperlanes []Hyperlane `sgm:"hyperlane"`

	SectorId SectorId `sgm:"sector,id"`
	Sector   *Sector

	StarbaseId  StarbaseId   `sgm:"starbase,id"`
	StarbaseIds []StarbaseId `sgm:"starbases"`
	Starbases   []*Starbase

	PlanetIds []PlanetId `sgm:"planet"`
	Planets   []*Planet

	MegastructureIds []MegastructureId `sgm:"megastructures"`
	Megastructures   []*Megastructure

	WormholeIds []WormholeId `sgm:"natural_wormholes"`
	Wormholes   []*Wormhole

	FleetIds []FleetId `sgm:"fleet_presence"`
	Fleets   []*Fleet
}

func (s *Star) Name() string {
	if s.NameString == "" {
		s.NameString = strings.ReplaceAll(s.NameStruct.Resolve(), "_", " ")
	}
	return s.NameString
}

func (s *Star) Point() sgmmath.Point {
	return sgmmath.Point{X: -s.Coordinate.X, Y: s.Coordinate.Y}
}

func (s *Star) PrimaryStarbase() *Starbase {
	if len(s.Starbases) == 0 {
		return nil
	}
	return s.Starbases[0]
}

func (s *Star) Owner() CountryId {
	if s.Sector != nil {
		return s.Sector.Owner
	}

	starbase := s.PrimaryStarbase()
	if starbase != nil {
		// 3.3
		if starbase.Owner != DefaultCountryId {
			return starbase.Owner
		}

		// 3.4
		if starbase.Station != nil && starbase.Station.Fleet != nil {
			return starbase.Station.Fleet.OwnerId
		}
	}
	return DefaultCountryId
}

func (s *Star) IsOwnedBy(countryId CountryId) bool {
	return s.Owner() == countryId
}

func (s *Star) Bypasses() (bypasses []string) {
	if len(s.WormholeIds) > 0 {
		bypasses = append(bypasses, BypassWormhole)
	}

	for _, ms := range s.Megastructures {
		msType, stage := ms.TypeStage()
		switch msType {
		case MegastructureLGate:
			bypasses = append(bypasses, BypassLGate)
		case MegastructureGateway:
			if stage >= 0 {
				bypasses = append(bypasses, BypassGateway)
			} else {
				bypasses = append(bypasses, BypassGatewayRuined)
			}
		}
	}
	return bypasses
}

func (s *Star) HasSignificantMegastructures() bool {
	for _, ms := range s.Megastructures {
		msType, _ := ms.TypeStage()
		msSize, _ := MegastructureSize[msType]
		if msSize >= MegastructureSizePlanet {
			return true
		}
	}
	return false
}

func (s *Star) MegastructuresBySize(size int) (megastructures []*Megastructure) {
	for _, ms := range s.Megastructures {
		msType, _ := ms.TypeStage()
		msSize, _ := MegastructureSize[msType]
		if size == msSize {
			megastructures = append(megastructures, ms)
		}
	}
	return
}

func (s *Star) Colonies(isHabitat bool) (colonies []*Planet) {
	for _, planet := range s.Planets {
		if planet.EmployablePops > 0 && planet.IsHabitat() == isHabitat {
			colonies = append(colonies, planet)
		}
	}

	return
}

func (s *Star) HasCapital() bool {
	if s.Sector == nil || s.Sector.Type != SectorCore {
		return false
	}

	for _, planet := range s.Planets {
		if planet.Designation == PlanetDesignationCapital {
			return true
		}
	}
	return false
}

func (s *Star) HasHyperlane(dst *Star) bool {
	for _, hyperlane := range s.Hyperlanes {
		if hyperlane.To == dst {
			return true
		}
	}
	return false
}

func (s *Star) IsSignificant() bool {
	return s.HasPops() ||
		s.HasUpgradedStarbase() ||
		s.HasSignificantMegastructures() ||
		len(s.Bypasses()) > 0
}

func (s *Star) HasUpgradedStarbase() bool {
	return len(s.Starbases) > 0 && s.Starbases[0].Level != StarbaseOutpost
}

func (s *Star) HasPops() bool {
	for _, planet := range s.Planets {
		if planet.EmployablePops > 0 {
			return true
		}
	}
	return false
}

// IsDistant returns true if star is from distant star pack (L-Galaxy)
func (s *Star) IsDistant() bool {
	if strings.HasPrefix(s.Initializer, DistantStarInitializer) {
		switch s.Initializer {
		case DistantStarInitializerLGate0, DistantStarInitializerLGate6:
			return false
		}
		return true
	}
	return false
}

func (s *Star) MobileMilitaryFleets() []*Fleet {
	var fleets []*Fleet
	for _, fleet := range s.Fleets {
		if !fleet.Civilian && fleet.Mobile && !fleet.IsTransport() {
			fleets = append(fleets, fleet)
		}
	}

	sort.Slice(fleets, func(i, j int) bool {
		return fleets[i].MilitaryPower > fleets[j].MilitaryPower
	})
	return fleets
}

type Coordinate struct {
	X float64 `sgm:"x"`
	Y float64 `sgm:"y"`
}

type Hyperlane struct {
	ToId StarId `sgm:"to"`
	To   *Star
}

type WormholeId uint32
type Wormhole struct {
	Bypass BypassId `sgm:"bypass"`
}

type BypassId uint32
type Bypass struct {
	Type     string   `sgm:"type"`
	LinkedTo BypassId `sgm:"linked_to"`

	Owner struct {
		Type int    `sgm:"type"`
		Id   uint32 `sgm:"id"`
	} `sgm:"owner"`
}

var DefaultPlanetId = PlanetId(math.MaxUint32)

type PlanetId uint32
type Planet struct {
	Star *Star

	NameString string `sgm:"name"`
	NameStruct struct {
		Key string `sgm:"key"`
	} `sgm:"name,struct"`
	Class       string `sgm:"planet_class"`
	Designation string `sgm:"final_designation"`

	Moons  []PlanetId `sgm:"moons"`
	MoonOf PlanetId   `sgm:"moon_of"`

	EmployablePops int `sgm:"employable_pops"`
}

func (p *Planet) IsHabitat() bool {
	return p.Class == PlanetClassHabitat
}

func (p *Planet) Name() string {
	if p.NameString != "" {
		return p.NameString
	}
	return p.NameStruct.Key
}
