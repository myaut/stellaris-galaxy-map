package sgmrender

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/beevik/etree"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-map/pkg/sgmmath"
)

type starRenderContext struct {
	starId sgm.StarId
	star   *sgm.Star

	g            *etree.Element
	iconOffset   float64
	colonyOffset float64
	fleetOffset  float64
	quadrant     int
}

func (r *Renderer) renderStars() (stars []*starRenderContext) {
	for starId, star := range r.state.Stars {
		ctx := &starRenderContext{starId: starId, star: star}
		ctx.quadrant = r.pickTextQuadrant(ctx.starId)

		p := star.Point()
		ctx.g = r.canvas.CreateElement("g")
		ctx.g.CreateAttr("transform", fmt.Sprintf("translate(%f, %f)", p.X, p.Y))

		// Render starbase if one exists, otherwise render
		if star.PrimaryStarbase() == nil || !star.IsSignificant() {
			r.createPath(ctx.g, defaultStarStyle, defaultStarPath)
			continue
		}
		stars = append(stars, ctx)
	}

	return
}

func (r *Renderer) renderStarbase(ctx *starRenderContext) {
	starbase := ctx.star.PrimaryStarbase()
	if starbase.Level == sgm.StarbaseOutpost {
		r.createPath(ctx.g, outpostStyle, outpostPath)
		ctx.iconOffset = outpostHalfSize
	} else {
		style := baseStarbaseStyle
		starbaseStroke := starbaseStrokes[starbase.Level]
		if starbaseStroke > 0.0 {
			style = style.With(StyleOption{"stroke-width", fmt.Sprintf("%fpt", starbaseStroke)})
		}
		r.createPath(ctx.g, style, starbasePath)
		ctx.iconOffset = starbaseHalfSize + starbaseStroke/2

		if starbase.Level == sgm.StarbaseCitadel {
			r.createPath(ctx.g, baseStarbaseStyle, citadelInnerPath)
		}

		role := starbase.Role()
		if role != sgm.StarbaseRoleMax {
			rolePoint := sgmmath.Point{X: -ctx.iconOffset / 2, Y: -ctx.iconOffset / 3}
			r.createIcon(ctx.g, rolePoint, "starbase-"+role.String(), iconSizeSm)
		}
	}
	ctx.colonyOffset = ctx.iconOffset / 2
}

func (r *Renderer) renderStarFeatures(ctx *starRenderContext) {
	// Bypasses
	bypasses := ctx.star.Bypasses()
	transportPoint := sgmmath.Point{
		X: -ctx.iconOffset - float64(len(bypasses))*iconStepSm,
		Y: -iconSizeSm / 2,
	}
	for _, bypass := range bypasses {
		r.createIcon(ctx.g, transportPoint, "bypass-"+bypass, iconSizeSm)
		transportPoint.X += iconStepSm
	}

	// Fleets
	fleets := ctx.star.MobileMilitaryFleets()
	fleetPoint := sgmmath.Point{
		X: -ctx.iconOffset / 3,
		Y: ctx.iconOffset + fleetHalfSize/3,
	}
	var extraFleets int
	if len(fleets) > 4 {
		extraFleets = len(fleets) - 4
		fleets = fleets[:4]
	}
	for _, fleet := range fleets {
		r.renderFleet(ctx, fleetPoint, fleet)
		fleetPoint.X += fleetStep
	}
	if extraFleets > 0 {
		r.createText(ctx.g, fleetTextStyle,
			fleetPoint.Add(sgmmath.Point{-fleetStep, fleetHalfSize}),
			fmt.Sprintf("+%d", extraFleets))
	}
	if len(fleets) > 0 {
		ctx.fleetOffset = fleetStep
	}

	// Other features
	colonies := ctx.star.Colonies(false)
	habitats := ctx.star.Colonies(true)
	megastructures := ctx.star.MegastructuresBySize(sgm.MegastructureSizeStar)
	ringWorlds := ctx.star.MegastructuresBySize(sgm.MegastructureSizeRingWorld)
	sort.Slice(colonies, func(i, j int) bool {
		return colonies[i].EmployablePops > colonies[j].EmployablePops
	})

	planetPoint := sgmmath.Point{X: ctx.colonyOffset, Y: -ctx.iconOffset}
	for _, ms := range megastructures {
		r.renderMegastructure(ctx, planetPoint, sgm.MegastructureSizeStar, ms, iconSizeMd)
		planetPoint.X += iconStepMd
	}
	if len(ringWorlds) > 0 {
		r.renderMegastructure(ctx, planetPoint, sgm.MegastructureSizeRingWorld, ringWorlds[0], iconSizeMd)
		planetPoint.X += iconStepMd
	}
	for _, planet := range colonies {
		r.renderPlanet(ctx, planetPoint, planet)
		planetPoint.X += iconStepMd
	}
	ctx.colonyOffset = planetPoint.X

	planetStations := append(
		ctx.star.MegastructuresBySize(sgm.MegastructureSizePlanet),
		make([]*sgm.Megastructure, len(habitats))...)
	planetStationsStep := 2
	if len(planetStations) > 4 {
		planetStationsStep = 3
	}
	for i, ms := range planetStations {
		if ms != nil {
			r.renderMegastructure(ctx, planetPoint, sgm.MegastructureSizePlanet, ms, iconSizeSm)
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

func (r *Renderer) renderFleet(ctx *starRenderContext, point sgmmath.Point, fleet *sgm.Fleet) {
	fleetStrength := math.Floor(math.Log10(fleet.MilitaryPower)) / 10.0
	if fleetStrength <= 0.33 {
		fleetStrength = 0.33
	}
	style := fleetStyle.With(StyleOption{"stroke-width", fmt.Sprintf("%fpt", fleetStrength)})

	fleetPath := newFleetPath(fleetHalfSize+fleetStrength, fleetStrength/2)
	fleetPath.Translate(point)
	pathEl := r.createPath(ctx.g, style, fleetPath)
	r.createTitle(pathEl, fmt.Sprintf("%s (%s)", fleet.Name(), fleet.MilitaryPowerString()))

	if fleet.Owner != nil {
		bgColor := sgm.ColorMap.Colors[fleet.Owner.Flag.Colors[0]]
		fgColor := sgm.ColorMap.Colors[fleet.Owner.Flag.Colors[1]]
		if bgColor != nil && fgColor != nil {
			fleetIdentStyle := fleetIdentStyle.With(
				StyleOption{"stroke", fgColor.Ship.Color().ToHexCode().String()},
				StyleOption{"fill", bgColor.Ship.Color().ToHexCode().String()},
			)

			fleetIdentPath := newDiamondPath(fleetHalfSize / 2)
			fleetIdentPath.Translate(point)
			r.createPath(ctx.g, fleetIdentStyle, fleetIdentPath)
		}
	}
}

func (r *Renderer) renderMegastructure(
	ctx *starRenderContext, point sgmmath.Point, msSize int,
	ms *sgm.Megastructure, iconSize float64,
) {
	msType, stage := ms.TypeStage()

	var icons []string
	if stage < 0 {
		icons = []string{"megastructure-ruined"}
	} else {
		switch msSize {
		case sgm.MegastructureSizePlanet:
			icons = []string{"megastructure-planet"}
			fallthrough
		default:
			icons = append(icons, "megastructure-"+strings.ReplaceAll(msType, "_", "-"))
		}
	}

	for _, icon := range icons {
		r.createIcon(ctx.g, point, icon, iconSize)
	}
}

func (r *Renderer) renderPlanet(ctx *starRenderContext, point sgmmath.Point, planet *sgm.Planet) {
	radius := float64(iconSizeMd) / 2
	center := point.Add(sgmmath.Point{radius, radius})
	style := basePlanetStyle
	switch {
	case planet.EmployablePops > 50:
		style = style.With(StyleOption{"stroke-width", "0.8pt"})
		radius -= 0.4
	case planet.EmployablePops > 25:
		style = style.With(StyleOption{"stroke-width", "0.6pt"})
		radius -= 0.2
	default:
		radius -= 0.1
	}

	title := fmt.Sprintf("%s (%d pops)", planet.Name(), planet.EmployablePops)
	r.createTitle(r.createCircle(ctx.g, style, center, radius), title)
	if planet.EmployablePops > 75 {
		r.createTitle(r.createCircle(ctx.g, basePlanetStyle, center, radius/2), title)
	}

	if planet.Designation == sgm.PlanetDesignationCapital {
		r.createIcon(ctx.g, point, "colony-capital", iconSizeMd)
	} else if planet.Class == sgm.PlanetClassEcumenopolis {
		r.createIcon(ctx.g, point, "colony-ecumenopolis", iconSizeMd)
	}
}

func (r *Renderer) renderStarName(ctx *starRenderContext) {
	var textAnchor string
	point := ctx.star.Point()
	name := ctx.star.Name()
	if strings.HasPrefix(name, "NAME_") {
		name = strings.ReplaceAll(name[5:], "_", " ")
	}

	switch ctx.quadrant {
	case 0, -1:
		textAnchor = "start"
		point.X -= ctx.iconOffset
	case 1, -2:
		textAnchor = "end"
		point.X += ctx.colonyOffset
	}
	if ctx.quadrant >= 0 {
		point.Y -= ctx.iconOffset/2 + 2*fontSize/3
	} else {
		point.Y += ctx.iconOffset + fontSize + ctx.fleetOffset
	}

	textEl := r.createText(r.canvas, starTextStyle, point, name)
	textEl.CreateAttr("text-anchor", textAnchor)
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
					MoveToPoint(pv.PointAtLength(2*starHalfSize)).
					LineToPoint(pv.PointAtLength(pv.Length-2*starHalfSize)))
		}

		renderedStars[starId] = struct{}{}
	}
}
