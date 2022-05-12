package sgmrender

import (
	"fmt"

	"github.com/beevik/etree"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

const (
	canvasPadding = 20

	starSize     = 2.0
	starbaseSize = 3.0
)

var (
	hyperlaneStyle = NewStyle(
		StyleOption{"stroke-width", "1pt"},
		StyleOption{"stroke", "#e5e8e8"},
	)

	defaultStarStyle = NewStyle(
		StyleOption{"stroke-width", "0.5pt"},
		StyleOption{"stroke", "#2e4053"},
		StyleOption{"fill", "#aeb6bf"},
	)

	baseStarbaseStyle = NewStyle(
		StyleOption{"stroke-width", "0.5pt"},
		StyleOption{"stroke", "#2e86c1"},
		StyleOption{"fill", "#aed6f1"},
	)

	starbaseStyles = map[string]Style{
		sgm.StarbaseStarport: baseStarbaseStyle.With(StyleOption{"stroke-width", "0.8pt"}),
		sgm.StarbaseFortress: baseStarbaseStyle.With(StyleOption{"stroke-width", "1.0pt"}),
		sgm.StarbaseCitadel:  baseStarbaseStyle.With(StyleOption{"stroke-width", "1.2pt"}),

		sgm.StarbaseMarauder:   defaultStarStyle,
		sgm.StarbaseCaravaneer: defaultStarStyle,
	}

	starTextStyle = NewStyle(
		StyleOption{"font-family", "sans-serif"},
		StyleOption{"font-size", "4pt"},
		StyleOption{"stroke-width", "0.08pt"},
		StyleOption{"stroke", "white"},
		StyleOption{"fill", "black"},
	)
)

var (
	defaultStarPath = NewPath().
			MoveTo(-starSize, 0.0).LineTo(0.0, starSize).
			LineTo(starSize, 0.0).LineTo(0.0, -starSize).
			Complete()

	outpostPath  = newStarbasePath(starSize)
	starbasePath = newStarbasePath(starbaseSize)
)

func newStarbasePath(size float64) Path {
	return NewPath().
		MoveTo(-size, 0.0).LineTo(-size/2, size).HorLine(size/2).
		LineTo(size, 0.0).LineTo(size/2, -size).HorLine(-size / 2).
		Complete()
}

type Renderer struct {
	state  *sgm.GameState
	doc    *etree.Document
	canvas *etree.Element

	bounds       sgmmath.BoundingRect
	starGeoIndex StarGeoIndex
}

func NewRenderer(state *sgm.GameState) *Renderer {
	r := &Renderer{state: state}
	r.computeBounds()
	r.buildStarIndex()

	r.doc = etree.NewDocument()
	r.doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)

	r.canvas = r.doc.CreateElement("svg")
	r.canvas.CreateAttr("width", "1080")
	r.canvas.CreateAttr("height", "1080")
	r.canvas.CreateAttr("viewBox",
		fmt.Sprintf("%f %f %f %f",
			r.bounds.Min.X-canvasPadding,
			r.bounds.Min.Y-canvasPadding,
			r.bounds.Max.X-r.bounds.Min.X+2*canvasPadding,
			r.bounds.Max.Y-r.bounds.Min.Y+2*canvasPadding))
	r.canvas.CreateAttr("xmlns", "http://www.w3.org/2000/svg")

	return r
}

func (r *Renderer) computeBounds() {
	for _, star := range r.state.Stars {
		r.bounds.Add(star.Point())
	}
}

func (r *Renderer) Render() {
	r.renderCountries()
	r.renderHyperlanes()
	r.renderStars()
}

func (r *Renderer) createPath(el *etree.Element, style Style, path Path) {
	pathEl := el.CreateElement("path")
	pathEl.CreateAttr("style", style.String())
	pathEl.CreateAttr("d", path.String())
}

func (r *Renderer) Write(outPath string) error {
	return r.doc.WriteToFile(outPath)
}
