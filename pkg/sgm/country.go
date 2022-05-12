package sgm

import (
	"math"
)

const (
	StarbaseOutpost  = "starbase_level_outpost"
	StarbaseStarport = "starbase_level_starport"
	StarbaseFortress = "starbase_level_starfortress"
	StarbaseCitadel  = "starbase_level_citadel"

	StarbaseMarauder   = "starbase_level_marauder"
	StarbaseCaravaneer = "starbase_level_caravaneer"
)

var DefaultStarbaseId = StarbaseId(math.MaxUint32)

type StarbaseId uint32
type Starbase struct {
	Level   string         `sgm:"level"`
	Modules map[int]string `sgm:"modules"`
	Owner   CountryId      `sgm:"owner"`

	Star *Star
}

var DefaultCountryId = CountryId(math.MaxUint32)

type CountryId uint32
type Country struct {
	Name string `sgm:"name"`

	Flag CountryFlag `sgm:"flag"`
}

type MegastructureId uint32
type Megastructure struct {
	Type     string   `sgm:"type"`
	Owner    int      `sgm:"owner"`
	PlanetId PlanetId `sgm:"planet"`

	Star   *Star
	Planet *Planet
}

type CountryFlag struct {
	Colors []string `sgm:"colors"`
}
