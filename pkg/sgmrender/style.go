package sgmrender

import (
	"bytes"
	"fmt"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
)

const (
	countryFontSize = 8.0

	colorPrimaryStroke = "#212f3c"
	colorPrimaryFill   = "#f2f2f2"

	colorFleetStroke = "#2e4053"
	colorFleetFill   = "#aeb6bf"

	colorStarStroke = "#ac9d93"
	colorStarFill   = "#e9c6af"

	colorStarbaseStroke = "#2e86c1"
	colorStarbaseFill   = "#aed6f1"

	colorHostileStroke = "#a93226"
	colorHostileFill   = "#f2d7d5"

	colorFriendlyStroke = "#217844"
	colorFriendlyFill   = "#afe9c6"

	colorPlanetStroke = "#27ae60"
	colorPlanetFill   = "#abebc6"
)

var (
	gridStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", "#e5e8e8"},
		StyleOption{"stroke-opacity", "0.2"},
	)

	hyperlaneStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", colorPrimaryStroke},
	)

	defaultStarStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", colorStarStroke},
		StyleOption{"fill", colorStarFill},
	)

	outpostStyle = NewStyle(
		StyleOption{"stroke-width", "0.2pt"},
		StyleOption{"stroke", colorFleetStroke},
		StyleOption{"fill", colorFleetFill},
	)

	outpostLostStyle = outpostStyle.With(
		StyleOption{"stroke", colorHostileStroke},
		StyleOption{"fill", colorHostileFill},
	)

	baseStarbaseStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", colorStarbaseStroke},
		StyleOption{"fill", colorStarbaseFill},
	)

	starbaseLostStyle = baseStarbaseStyle.With(
		StyleOption{"stroke", colorHostileStroke},
		StyleOption{"fill", colorHostileFill},
	)

	starbaseStrokes = map[string]float64{
		sgm.StarbaseStarport: 0.5,
		sgm.StarbaseStarhold: 0.75,
		sgm.StarbaseFortress: 1.0,
		sgm.StarbaseCitadel:  1.0,
	}

	fleetStyles = map[sgm.WarRole]Style{
		sgm.WarRoleStarNeutral: NewStyle(
			StyleOption{"stroke", colorFleetStroke},
			StyleOption{"fill", colorFleetFill},
		),
		sgm.WarRoleStarAttacker: NewStyle(
			StyleOption{"stroke", colorHostileStroke},
			StyleOption{"fill", colorHostileFill},
		),
		sgm.WarRoleStarDefender: NewStyle(
			StyleOption{"stroke", colorFriendlyStroke},
			StyleOption{"fill", colorFriendlyFill},
		),
	}

	fleetIdentStyle = NewStyle(
		StyleOption{"stroke-width", "0.2pt"},
		StyleOption{"fill-opacity", "0.8"},
	)

	basePlanetStyle = NewStyle(
		StyleOption{"stroke-width", "0.4pt"},
		StyleOption{"stroke", colorPlanetStroke},
		StyleOption{"fill", colorPlanetFill},
	)

	planetRingStyle = NewStyle(
		StyleOption{"stroke-width", "0.4pt"},
		StyleOption{"stroke", colorFleetStroke},
		StyleOption{"fill", "none"},
	)

	starTextStyle = NewStyle(
		StyleOption{"font-family", "sans-serif"},
		StyleOption{"font-size", "3.6pt"},
		StyleOption{"stroke-width", "0.08pt"},
		StyleOption{"stroke", colorPrimaryStroke},
		StyleOption{"fill", colorPrimaryFill},
	)

	countryTextStyle = NewStyle(
		StyleOption{"font-family", "sans-serif"},
		StyleOption{"stroke-width", "0.12pt"},
		StyleOption{"font-size", fmt.Sprintf("%fpt", countryFontSize)},
		StyleOption{"font-variant", "petite-caps"},
		StyleOption{"stroke", colorPrimaryStroke},
		StyleOption{"fill", colorPrimaryFill},
	)

	countryLegendStyle = countryTextStyle.With(
		StyleOption{"font-size", fmt.Sprintf("%fpt", countryFontSize/2)},
	)

	baseCountryStyle = NewStyle(
		StyleOption{"stroke-width", "2pt"},
		StyleOption{"stroke-linejoin", "miter"},
		StyleOption{"fill-opacity", "0.6"},
		StyleOption{"fill-rule", "evenodd"},
	)

	occupationPatternStyle = NewStyle(
		StyleOption{"stroke-width", "0.5pt"},
		StyleOption{"stroke-opacity", "0.6"},
	)

	fleetTextStyle = NewStyle(
		StyleOption{"font-family", "sans-serif"},
		StyleOption{"font-size", "3.2pt"},
		StyleOption{"stroke-width", "0.08pt"},
		StyleOption{"stroke", colorFleetFill},
		StyleOption{"fill", colorFleetStroke},
	)
)

type Style struct {
	m map[string]string
	s string
}

type StyleOption struct {
	Property string
	Value    string
}

func newStyle() Style {
	return Style{m: make(map[string]string)}
}

func NewStyle(opts ...StyleOption) Style {
	s := newStyle()
	for _, opt := range opts {
		s.m[opt.Property] = opt.Value
	}
	return s
}

func (s Style) With(opts ...StyleOption) Style {
	s2 := newStyle()
	for prop, value := range s.m {
		s2.m[prop] = value
	}
	for _, opt := range opts {
		s2.m[opt.Property] = opt.Value
	}
	return s2
}

func (s Style) String() string {
	if s.s == "" {
		buf := bytes.NewBuffer(nil)
		for prop, value := range s.m {
			buf.WriteString(prop)
			buf.WriteByte(':')
			buf.WriteString(value)
			buf.WriteByte(';')
		}
		s.s = buf.String()
	}

	return s.s
}
