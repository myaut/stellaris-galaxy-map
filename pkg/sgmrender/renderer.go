package sgmrender

import (
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/beevik/etree"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

const (
	canvasPadding = 40

	starSize     = 2.0
	outpostSize  = 3.0
	starbaseSize = 4.2
	fontSize     = 3.2

	maxCellSize      = 40.0
	countryBorderGap = 1.2

	iconSizeSm  = 4.8
	iconStepSm  = 3.6
	iconSizeMd  = 6
	iconStepMd  = 4.8
	svgIconSize = 16
)

const (
	traceFlagStarIndex = 1 << iota
	traceFlagShowGraphEdges

	traceFlags = 0
)

//go:embed icons/*.svg
var iconsFS embed.FS

type Renderer struct {
	state  *sgm.GameState
	doc    *etree.Document
	canvas *etree.Element

	bounds       sgmmath.BoundingRect
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
	r.canvas.CreateAttr("width", "1080")
	r.canvas.CreateAttr("height", "1080")
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
	r.renderCountries()
	r.renderHyperlanes()
	r.renderStars()
}

func (r *Renderer) renderCountries() {
	cr := r.createCountryRenderer()

	for _, cell := range cr.diagram.Cells {
		star := cr.starMap[cell.Site]
		if star.Starbase == nil {
			continue
		}

		owner := star.Starbase.Owner
		country := r.state.Countries[owner]

		// TODO: handle "use_as_border_color", cases when colors are not found
		borderColor := sgm.ColorMap.Colors[country.Flag.Colors[1]]
		countryColor := sgm.ColorMap.Colors[country.Flag.Colors[0]]
		countryStyle := baseCountryStyle.With(
			StyleOption{"stroke", borderColor.Map.Color().ToHexCode().String()},
			StyleOption{"fill", countryColor.Map.Color().ToHexCode().String()},
		)

		path := cr.buildPath(star, cell, nil, owner, NewPath())
		r.createPath(r.canvas, countryStyle, path.Complete())
	}

	if traceFlags&traceFlagShowGraphEdges != 0 {
		for _, edge := range cr.diagram.Edges {
			r.createPath(r.canvas, hyperlaneStyle,
				NewPath().MoveTo(edge.Va.X, edge.Va.Y).LineTo(edge.Vb.X, edge.Vb.Y))
		}
	}
}

type starRenderContext struct {
	starId sgm.StarId
	star   *sgm.Star

	g            *etree.Element
	iconOffset   float64
	colonyOffset float64
	quadrant     int

	hadRingWorld bool
}

func (r *Renderer) renderStars() {
	var stars []*starRenderContext

	for starId, star := range r.state.Stars {
		ctx := &starRenderContext{starId: starId, star: star}
		ctx.quadrant = r.pickTextQuadrant(ctx.starId)

		ctx.g = r.canvas.CreateElement("g")
		p := star.Point()
		ctx.g.CreateAttr("transform", fmt.Sprintf("translate(%f, %f)", p.X, p.Y))

		// Render starbase if one exists, otherwise render
		if star.Starbase == nil {
			r.createPath(ctx.g, defaultStarStyle, defaultStarPath)
			ctx.iconOffset = starSize
		} else if star.Starbase.Level == sgm.StarbaseOutpost {
			r.createPath(ctx.g, baseStarbaseStyle, outpostPath)
			ctx.iconOffset = outpostSize
		} else {
			r.createPath(ctx.g, starbaseStyles[star.Starbase.Level], starbasePath)
			ctx.iconOffset = starbaseSize

			if star.Starbase.Level == sgm.StarbaseCitadel {
				r.createPath(ctx.g, baseStarbaseStyle, citadelInnerPath)
			}

			role := star.Starbase.Role()
			if role != sgm.StarbaseRoleMax {
				rolePoint := sgmmath.Point{X: -ctx.iconOffset / 2, Y: -ctx.iconOffset / 3}
				r.createIcon(ctx.g, rolePoint, "starbase-"+role.String(), iconSizeSm)
			}
		}
		ctx.colonyOffset = ctx.iconOffset / 2

		// Render system features: megastructures, planets, etc.
		r.renderFeatures(ctx)

		if star.HasPops() || star.HasUpgradedStarbase() ||
			star.HasSignificatMegastructures() {

			stars = append(stars, ctx)
		}
	}

	for _, ctx := range stars {
		r.renderStarName(ctx)
	}
}

func (r *Renderer) renderFeatures(ctx *starRenderContext) {
	bypasses := ctx.star.Bypasses()
	transportPoint := sgmmath.Point{
		X: -ctx.iconOffset/2 - float64(len(bypasses))*iconSizeSm,
		Y: -ctx.iconOffset/2 - iconSizeSm/2,
	}
	if ctx.quadrant >= 0 {
		transportPoint.Y = 0.0
	}

	for _, bypass := range bypasses {
		r.createIcon(ctx.g, transportPoint, "bypass-"+bypass, iconSizeSm)
		transportPoint.X += iconSizeSm
	}

	colonies := ctx.star.Colonies(false)
	habitats := ctx.star.Colonies(true)
	megastructures := ctx.star.MegastructuresBySize(sgm.MegastructureStar)
	sort.Slice(colonies, func(i, j int) bool {
		return colonies[i].EmployablePops > colonies[j].EmployablePops
	})

	planetPoint := sgmmath.Point{X: ctx.colonyOffset, Y: -ctx.iconOffset}
	for _, ms := range megastructures {
		r.renderMegastructure(ctx, planetPoint, "star", ms, iconSizeMd)
		planetPoint.X += iconStepMd
	}
	for _, planet := range colonies {
		r.renderPlanet(ctx, planetPoint, planet)
		planetPoint.X += iconStepMd
	}
	ctx.colonyOffset = planetPoint.X

	planetStations := append(
		ctx.star.MegastructuresBySize(sgm.MegastructurePlanet),
		make([]*sgm.Megastructure, len(habitats))...)
	planetStationsStep := 2
	if len(planetStations) > 4 {
		planetStationsStep = 3
	}
	for i, ms := range planetStations {
		if ms != nil {
			r.renderMegastructure(ctx, planetPoint, "planet", ms, iconSizeSm)
		} else {
			r.createIcon(ctx.g, planetPoint, "habitat", iconSizeSm)
		}

		if (i+1)%planetStationsStep == 0 {
			planetPoint.X = ctx.colonyOffset
			planetPoint.Y += iconStepSm
		} else {
			planetPoint.X += iconStepSm
		}
	}
}

func (r *Renderer) renderMegastructure(
	ctx *starRenderContext, point sgmmath.Point, iconSuffix string,
	ms *sgm.Megastructure, iconSize float64,
) {
	msType, stage := ms.TypeStage()
	if msType == sgm.MegastructureRingWorld {
		if ctx.hadRingWorld {
			return
		}
		ctx.hadRingWorld = true
	}

	icons := []string{"megastructure-" + iconSuffix}
	if stage < 0 {
		icons = []string{"megastructure-ruined"}
	}
	icons = append(icons, "megastructure-"+strings.ReplaceAll(msType, "_", "-"))

	for _, icon := range icons {
		r.createIcon(ctx.g, point, icon, iconSize)
	}
}

func (r *Renderer) renderPlanet(ctx *starRenderContext, point sgmmath.Point, planet *sgm.Planet) {
	radius := float64(iconSizeMd) / 2
	style := basePlanetStyle
	switch {
	case planet.EmployablePops > 75:
		style = style.With(StyleOption{"stroke-width", "0.8pt"})
		radius -= 0.4
	case planet.EmployablePops > 50:
		style = style.With(StyleOption{"stroke-width", "0.5pt"})
		radius -= 0.25
	case planet.EmployablePops > 25:
		style = style.With(StyleOption{"stroke-width", "0.33pt"})
		radius -= 0.17
	}

	circle := ctx.g.CreateElement("circle")
	circle.CreateAttr("cx", fmt.Sprintf("%f", point.X+radius))
	circle.CreateAttr("cy", fmt.Sprintf("%f", point.Y+radius))
	circle.CreateAttr("r", fmt.Sprintf("%f", radius))
	circle.CreateAttr("style", style.String())

	if planet.Designation == sgm.PlanetDesignationCapital {
		r.createIcon(ctx.g, point, "colony-capital", iconSizeMd)
	} else if planet.Class == sgm.PlanetClassEcumenopolis {
		r.createIcon(ctx.g, point, "colony-ecumenopolis", iconSizeMd)
	}
}

func (r *Renderer) renderStarName(ctx *starRenderContext) {
	name := ctx.star.Name
	if strings.HasPrefix(name, "NAME_") {
		name = strings.ReplaceAll(name[5:], "_", " ")
	}

	p := ctx.star.Point()

	var textAnchor string
	switch ctx.quadrant {
	case 0, -1:
		textAnchor = "start"
		p.X -= starbaseSize / 2
	case 1, -2:
		textAnchor = "end"
		p.X += ctx.iconOffset + ctx.colonyOffset
	}
	if ctx.quadrant >= 0 {
		p.Y -= ctx.iconOffset + 2*fontSize/3
	} else {
		p.Y += ctx.iconOffset + 2*fontSize/3
	}

	text := r.canvas.CreateElement("text")
	text.CreateAttr("x", fmt.Sprintf("%f", p.X-ctx.iconOffset))
	text.CreateAttr("y", fmt.Sprintf("%f", p.Y+ctx.iconOffset/2))
	text.CreateAttr("style", starTextStyle.String())
	text.CreateAttr("text-anchor", textAnchor)
	text.CreateText(name)
}

func (r *Renderer) renderHyperlanes() {
	renderedStars := make(map[sgm.StarId]struct{})

	for starId, star := range r.state.Stars {
		for _, hyperlane := range star.Hyperlanes {
			if _, isRendered := renderedStars[hyperlane.ToId]; isRendered {
				continue
			}

			pv := sgmmath.NewVector(star, hyperlane.To).ToPolar()
			r.createPath(r.canvas, hyperlaneStyle,
				NewPath().
					MoveToPoint(pv.PointAtLength(2*starSize)).
					LineToPoint(pv.PointAtLength(pv.Length-2*starSize)))
		}

		renderedStars[starId] = struct{}{}
	}
}

func (r *Renderer) createPath(el *etree.Element, style Style, path Path) {
	pathEl := el.CreateElement("path")
	pathEl.CreateAttr("style", style.String())
	pathEl.CreateAttr("d", path.String())
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

func (r *Renderer) Write(outPath string) error {
	return r.doc.WriteToFile(outPath)
}
