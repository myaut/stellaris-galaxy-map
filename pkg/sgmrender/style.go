package sgmrender

import (
	"bytes"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
)

var (
	hyperlaneStyle = NewStyle(
		StyleOption{"stroke-width", "0.8pt"},
		StyleOption{"stroke", "#e5e8e8"},
	)

	defaultStarStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", "#2e4053"},
		StyleOption{"fill", "#aeb6bf"},
	)

	baseStarbaseStyle = NewStyle(
		StyleOption{"stroke-width", "0.33pt"},
		StyleOption{"stroke", "#2e86c1"},
		StyleOption{"fill", "#aed6f1"},
	)

	starbaseStyles = map[string]Style{
		sgm.StarbaseStarport: baseStarbaseStyle.With(StyleOption{"stroke-width", "0.5pt"}),
		sgm.StarbaseFortress: baseStarbaseStyle.With(StyleOption{"stroke-width", "0.8pt"}),
		sgm.StarbaseStarhold: baseStarbaseStyle.With(StyleOption{"stroke-width", "1.0pt"}),
		sgm.StarbaseCitadel:  baseStarbaseStyle.With(StyleOption{"stroke-width", "1.0pt"}),

		sgm.StarbaseMarauder:   defaultStarStyle,
		sgm.StarbaseCaravaneer: defaultStarStyle,
	}

	basePlanetStyle = NewStyle(
		StyleOption{"stroke-width", "0.5pt"},
		StyleOption{"stroke", "#27ae60"},
		StyleOption{"fill", "#abebc6"},
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
		StyleOption{"stroke-width", "0.5pt"},
		StyleOption{"stroke", "black"},
		StyleOption{"fill", "white"},
	)

	baseCountryStyle = NewStyle(
		StyleOption{"stroke-width", "2pt"},
		StyleOption{"stroke-linejoin", "miter"},
		StyleOption{"fill-opacity", ".4"},
		StyleOption{"fill-rule", "evenodd"},
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
