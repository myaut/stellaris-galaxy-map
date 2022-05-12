package sgm

import (
	"math"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

type StarId int
type Star struct {
	Type string `sgm:"type"`
	Name string `sgm:"name"`

	Coordinate Coordinate  `sgm:"coordinate"`
	Hyperlanes []Hyperlane `sgm:"hyperlane"`

	StarbaseId StarbaseId `sgm:"starbase"`
	Starbase   *Starbase

	PlanetIds []PlanetId `sgm:"planet"`
	Planets   []*Planet

	MegastructureIds []MegastructureId `sgm:"megastructures"`
	Megastructures   []*Megastructure

	WormholeIds []WormholeId `sgm:"natural_wormholes"`
	Wormholes   []*Wormhole
}

func (s *Star) Point() sgmmath.Point {
	return sgmmath.Point{X: -s.Coordinate.X, Y: s.Coordinate.Y}
}

func (s *Star) IsOwnedBy(countryId CountryId) bool {
	if s.Starbase == nil {
		return false
	}
	return s.Starbase.Owner == countryId
}

func (s *Star) HasHyperlane(dst *Star) bool {
	for _, hyperlane := range s.Hyperlanes {
		if hyperlane.To == dst {
			return true
		}
	}
	return false
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

	Name           string `sgm:"name"`
	EmployablePops int    `sgm:"employable_pops"`
}
