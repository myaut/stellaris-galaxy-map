package sgmparser

type StarId int
type Star struct {
	Type string `sgm:"type"`
	Name string `sgm:"name"`

	Coordinate Coordinate  `sgm:"coordinate"`
	Hyperlanes []Hyperlane `sgm:"hyperlane"`

	Starbase       StarbaseId        `sgm:"starbase"`
	Planets        []PlanetId        `sgm:"planets"`
	Megastructures []MegastructureId `sgm:"megastructures"`
	Wormholes      []WormholeId      `sgm:"natural_wormholes"`
}

type Coordinate struct {
	X float64 `sgm:"x"`
	Y float64 `sgm:"y"`
}

type Hyperlane struct {
	To StarId `sgm:"to"`
}

type WormholeId int
type Wormhole struct {
	Bypass BypassId `sgm:"bypass"`
}

type BypassId int
type Bypass struct {
	Type     string   `sgm:"type"`
	LinkedTo BypassId `sgm:"linked_to"`

	Owner struct {
		Type int `sgm:"type"`
		Id   int `sgm:"id"`
	} `sgm:"owner"`
}

type PlanetId int
type Planet struct {
	Name           string `sgm:"name"`
	EmployablePops int    `sgm:"employable_pops"`
}

type StarbaseId int
type Starbase struct {
	Level   string         `sgm:"level"`
	Modules map[int]string `sgm:"modules"`
}

type CountryId int
type Country struct {
	Name string `sgm:"name"`

	Flag CountryFlag `sgm:"flag"`
}

type MegastructureId int
type Megastructure struct {
}

type CountryFlag struct {
	Colors []string `sgm:"colors"`
}

type PlanetState struct {
	Planets map[PlanetId]*Planet `sgm:"planet"`
}

type StarbaseMgr struct {
	Starbases map[StarbaseId]*Starbase `sgm:"starbases"`
}

type GameState struct {
	Stars       map[StarId]*Star       `sgm:"galactic_object"`
	StarbaseMgr StarbaseMgr            `sgm:"starbase_mgr"`
	Planets     PlanetState            `sgm:"planets"`
	Countries   map[CountryId]*Country `sgm:"country"`
	Bypasses    map[BypassId]*Bypass   `sgm:"bypasses"`
}
