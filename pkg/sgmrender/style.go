package sgmrender

import (
	"bytes"
	"fmt"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
)

const (
	countryFontSize = 8.0

	colorStationStroke = "#2e4053"
	colorStationFill   = "#aeb6bf"

	colorStarStroke = "#ac9d93"
	colorStarFill   = "#e9c6af"

	colorMilitaryStroke = "#2e86c1"
	colorMilitaryFill   = "#aed6f1"

	colorPlanetStroke = "#27ae60"
	colorPlanetFill   = "#abebc6"
)

var (
	hyperlaneStyle = NewStyle(
		StyleOption{"stroke-width", "0.8pt"},
		StyleOption{"stroke", "#e5e8e8"},
	)

	defaultStarStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", colorStarStroke},
		StyleOption{"fill", colorStarFill},
	)

	baseStarbaseStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", colorMilitaryStroke},
		StyleOption{"fill", colorMilitaryFill},
	)

	starbaseStyles = map[string]Style{
		sgm.StarbaseStarport: baseStarbaseStyle.With(StyleOption{"stroke-width", "0.5pt"}),
		sgm.StarbaseFortress: baseStarbaseStyle.With(StyleOption{"stroke-width", "0.8pt"}),
		sgm.StarbaseStarhold: baseStarbaseStyle.With(StyleOption{"stroke-width", "1.0pt"}),
		sgm.StarbaseCitadel:  baseStarbaseStyle.With(StyleOption{"stroke-width", "1.0pt"}),

		sgm.StarbaseMarauder:   defaultStarStyle,
		sgm.StarbaseCaravaneer: defaultStarStyle,
	}

	fleetStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", colorMilitaryStroke},
		StyleOption{"fill", colorMilitaryFill},
	)

	outpostStyle = NewStyle(
		StyleOption{"stroke-width", "0.2pt"},
		StyleOption{"stroke", colorStationStroke},
		StyleOption{"fill", colorStationFill},
	)

	basePlanetStyle = NewStyle(
		StyleOption{"stroke-width", "0.4pt"},
		StyleOption{"stroke", colorPlanetStroke},
		StyleOption{"fill", colorPlanetFill},
	)

	starTextStyle = NewStyle(
		StyleOption{"font-family", "sans-serif"},
		StyleOption{"font-size", "3.6pt"},
		StyleOption{"stroke-width", "0.08pt"},
		StyleOption{"stroke", "white"},
		StyleOption{"fill", "black"},
	)

	countryTextStyle = NewStyle(
		StyleOption{"font-family", "sans-serif"},
		StyleOption{"stroke-width", "0.12pt"},
		StyleOption{"font-size", fmt.Sprintf("%fpt", countryFontSize)},
		StyleOption{"font-variant", "petite-caps"},
		StyleOption{"stroke", "white"},
		StyleOption{"fill", "black"},
	)

	baseCountryStyle = NewStyle(
		StyleOption{"stroke-width", "2pt"},
		StyleOption{"stroke-linejoin", "miter"},
		StyleOption{"fill-opacity", ".4"},
		StyleOption{"fill-rule", "evenodd"},
	)

	fleetTextStyle = NewStyle(
		StyleOption{"font-family", "sans-serif"},
		StyleOption{"font-size", "3.2pt"},
		StyleOption{"stroke-width", "0.08pt"},
		StyleOption{"stroke", colorMilitaryFill},
		StyleOption{"fill", colorMilitaryStroke},
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
