package sgmrender

import (
	"embed"
	"fmt"
	"log"

	"github.com/beevik/etree"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-map/pkg/sgmmath"
)

const (
	mapTitle   = "Galaxy Map"
	footerText = "Made with Stellaris Galaxy Map"

	canvasPadding = 40
	canvasWidth   = 2160
	canvasHeight  = 2160

	fontSize = 3.2

	maxCellSize      = 48.0
	countryBorderGap = 1.2

	iconSizeSm  = 4.8
	iconStepSm  = 3.6
	iconSizeMd  = 6
	iconStepMd  = 4.8
	svgIconSize = 16

	gridSplit = 16
)

const (
	traceFlagStarIndex = 1 << iota
	traceFlagCountrySegments
	traceFlagShowGraphEdges

	traceFlags = traceFlagCountrySegments
)

//go:embed icons/*.svg
var iconsFS embed.FS

type Renderer struct {
	state  *sgm.GameState
	doc    *etree.Document
	canvas *etree.Element

	bounds       sgmmath.BoundingRect
	innerBounds  sgmmath.BoundingRect
	starGeoIndex StarGeoIndex

	iconCache map[string]*etree.Document
}

func NewRenderer(state *sgm.GameState) *Renderer {
	r := &Renderer{state: state}
	r.computeBounds()
	r.buildStarIndex()

	r.doc = etree.NewDocument()
	r.doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)

	r.canvas = r.doc.CreateElement("svg")
	r.canvas.CreateAttr("width", fmt.Sprint(canvasWidth))
	r.canvas.CreateAttr("height", fmt.Sprint(canvasHeight))
	r.canvas.CreateAttr("viewBox",
		fmt.Sprintf("%f %f %f %f",
			r.bounds.Min.X-canvasPadding,
			r.bounds.Min.Y-canvasPadding,
			r.bounds.Max.X-r.bounds.Min.X+2*canvasPadding,
			r.bounds.Max.Y-r.bounds.Min.Y+2*canvasPadding))
	r.canvas.CreateAttr("xmlns", "http://www.w3.org/2000/svg")

	r.iconCache = make(map[string]*etree.Document)

	return r
}

func (r *Renderer) getIcon(iconFile string) *etree.Document {
	icon, ok := r.iconCache[iconFile]
	if ok {
		return icon
	}
	defer func() { r.iconCache[iconFile] = icon }()

	buf, err := iconsFS.ReadFile("icons/" + iconFile + ".svg")
	if err != nil {
		log.Printf("error reading icon %s: %s", iconFile, err.Error())
		return nil
	}

	icon = etree.NewDocument()
	err = icon.ReadFromBytes(buf)
	if err != nil {
		log.Printf("error parsing icon %s: %s", iconFile, err.Error())
		return nil
	}

	// Remove some inkscape junk
	svg := icon.Root()
	svg.RemoveChild(svg.FindElement("defs"))
	svg.RemoveChild(svg.FindElement("metadata"))
	return icon
}

func (r *Renderer) computeBounds() {
	for _, star := range r.state.Stars {
		r.bounds.Add(star.Point())
	}
}

func (r *Renderer) Render() {
	countries := r.renderCountries()
	r.renderGrid()
	r.renderHyperlanes()

	significantStars := r.renderStars()

	r.renderCountryNames(countries)
	for _, ctx := range significantStars {
		r.renderStarbase(ctx)
		r.renderStarFeatures(ctx)
	}
	for _, ctx := range significantStars {
		r.renderStarName(ctx)
	}

	titlePoint := r.bounds.Min
	r.createText(r.canvas, countryTextStyle, titlePoint, mapTitle)
	titlePoint.Y += countryFontSize
	r.createText(r.canvas, countryLegendStyle, titlePoint, r.state.Name)
	titlePoint.Y += 0.6 * countryFontSize
	r.createText(r.canvas, countryLegendStyle, titlePoint, r.state.Date)

	footerStyle := starTextStyle.With(StyleOption{"text-anchor", "end"})
	footerPoint := r.bounds.Max.Add(sgmmath.Point{0.0, -2 * fontSize})
	r.createText(r.canvas, footerStyle, footerPoint, footerText)
}

func (r *Renderer) renderGrid() {
	w, h := r.bounds.Size()
	gridStepX, gridStepY := w/gridSplit, h/gridSplit

	for x := r.bounds.Min.X + gridStepX; x < r.bounds.Max.X-gridStepX/2; x += gridStepX {
		startPoint := sgmmath.Point{x, r.bounds.Min.Y - canvasPadding/2}
		path := NewPath().MoveToPoint(startPoint).VertLine(r.bounds.Max.Y + canvasPadding/2)
		r.createPath(r.canvas, gridStyle, path)
	}
	for y := r.bounds.Min.Y + gridStepY; y < r.bounds.Max.Y-gridStepY/2; y += gridStepY {
		startPoint := sgmmath.Point{r.bounds.Min.X - canvasPadding/2, y}
		path := NewPath().MoveToPoint(startPoint).HorLine(r.bounds.Max.X + canvasPadding/2)
		r.createPath(r.canvas, gridStyle, path)
	}
}

func (r *Renderer) createPath(el *etree.Element, style Style, path Path) *etree.Element {
	pathEl := el.CreateElement("path")
	pathEl.CreateAttr("style", style.String())
	pathEl.CreateAttr("d", path.String())
	return pathEl
}

func (r *Renderer) createRect(el *etree.Element, style Style, rect sgmmath.BoundingRect) *etree.Element {
	w, h := rect.Size()
	rectEl := el.CreateElement("rect")
	rectEl.CreateAttr("style", style.String())
	rectEl.CreateAttr("x", fmt.Sprintf("%f", rect.Min.X))
	rectEl.CreateAttr("y", fmt.Sprintf("%f", rect.Min.Y))
	rectEl.CreateAttr("width", fmt.Sprintf("%f", w))
	rectEl.CreateAttr("height", fmt.Sprintf("%f", h))
	return rectEl
}

func (r *Renderer) createCircle(
	el *etree.Element, style Style, center sgmmath.Point, radius float64,
) *etree.Element {
	circleEl := el.CreateElement("circle")
	circleEl.CreateAttr("cx", fmt.Sprintf("%f", center.X))
	circleEl.CreateAttr("cy", fmt.Sprintf("%f", center.Y))
	circleEl.CreateAttr("r", fmt.Sprintf("%f", radius))
	circleEl.CreateAttr("style", style.String())
	return circleEl
}

func (r *Renderer) createText(
	el *etree.Element, style Style, point sgmmath.Point, text string,
) *etree.Element {
	textEl := el.CreateElement("text")
	textEl.CreateAttr("x", fmt.Sprintf("%f", point.X))
	textEl.CreateAttr("y", fmt.Sprintf("%f", point.Y))
	textEl.CreateAttr("style", style.String())
	textEl.CreateText(text)
	return textEl
}

func (r *Renderer) createIcon(el *etree.Element, p sgmmath.Point, iconName string, size float64) {
	g := el.CreateElement("g")
	g.CreateAttr("transform",
		fmt.Sprintf("translate(%f, %f) scale(%f)", p.X, p.Y, size/float64(svgIconSize)))

	icon := r.getIcon(iconName)
	if icon == nil {
		// TODO: render some placeholder
		return
	}

	g.AddChild(icon.Root().Copy())
}

func (r *Renderer) createTitle(el *etree.Element, title string) {
	titleEl := el.CreateElement("title")
	titleEl.CreateText(title)
}

func (r *Renderer) Write(outPath string) error {
	return r.doc.WriteToFile(outPath)
}

func (r *Renderer) WriteToBytes() ([]byte, error) {
	return r.doc.WriteToBytes()
}
