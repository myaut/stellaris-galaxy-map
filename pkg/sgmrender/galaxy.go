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

	g                               *etree.Element
	iconOffset                      float64
	nameOffsetTop, nameOffsetBottom float64
	quadrant                        int
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
	lostControl := ctx.star.Occupier() != sgm.DefaultCountryId
	if starbase.Level == sgm.StarbaseOutpost {
		style := outpostStyle
		if lostControl {
			style = outpostLostStyle
		}

		r.createPath(ctx.g, style, outpostPath)
		ctx.iconOffset = outpostHalfSize
	} else {
		baseStyle := baseStarbaseStyle
		if lostControl {
			baseStyle = starbaseLostStyle
		}

		style := baseStyle
		starbaseStroke := starbaseStrokes[starbase.Level]
		if starbaseStroke > 0.0 {
			style = style.With(StyleOption{"stroke-width", fmt.Sprintf("%fpt", starbaseStroke)})
		}
		r.createPath(ctx.g, style, starbasePath)
		ctx.iconOffset = starbaseHalfSize + starbaseStroke/2

		if starbase.Level == sgm.StarbaseCitadel {
			r.createPath(ctx.g, baseStyle, citadelInnerPath)
		}

		if !lostControl {
			role := starbase.Role()
			if role != sgm.StarbaseRoleMax {
				rolePoint := sgmmath.Point{X: -ctx.iconOffset / 2, Y: -ctx.iconOffset / 3}
				r.createIcon(ctx.g, rolePoint, "starbase-"+role.String(), iconSizeSm)
			}
		}
	}
}

func (r *Renderer) renderStarFeatures(ctx *starRenderContext) {
	// Fleets
	r.renderAllFleets(ctx)

	// Other features
	megastructures := ctx.star.MegastructuresBySize(sgm.MegastructureSizeStar)
	ringWorlds := ctx.star.MegastructuresBySize(sgm.MegastructureSizeRingWorld)
	if len(ringWorlds) > 0 {
		megastructures = append(megastructures, ringWorlds[0])
	}

	colonies := ctx.star.Colonies(false)
	sort.Slice(colonies, func(i, j int) bool {
		return colonies[i].EmployablePops > colonies[j].EmployablePops
	})

	planetPoint := sgmmath.Point{X: -2 * ctx.iconOffset / 3, Y: -ctx.iconOffset}
	for _, ms := range megastructures {
		planetPoint.X -= iconStepMd
		r.renderMegastructure(ctx, planetPoint, sgm.MegastructureSizeStar, ms, iconSizeMd)
	}
	for _, planet := range colonies {
		planetPoint.X -= iconStepMd
		r.renderPlanet(ctx, planetPoint, planet)
	}

	stationPoint := sgmmath.Point{X: 2 * ctx.iconOffset / 3, Y: -ctx.iconOffset}
	habitats := ctx.star.Colonies(true)
	planetStations := append(
		ctx.star.MegastructuresBySize(sgm.MegastructureSizePlanet),
		make([]*sgm.Megastructure, len(habitats))...)
	planetStationsStep := 2
	if len(planetStations) > 4 {
		planetStationsStep = 3
	}
	for i, ms := range planetStations {
		if ms != nil {
			r.renderMegastructure(ctx, stationPoint, sgm.MegastructureSizePlanet, ms, iconSizeSm)
		} else {
			r.createIcon(ctx.g, stationPoint, "habitat", iconSizeSm)
		}

		if (i+1)%planetStationsStep == 0 {
			stationPoint.X = 2 * ctx.iconOffset / 3
			stationPoint.Y += iconStepSm
		} else {
			stationPoint.X += iconStepSm
		}
	}

	// Bypasses
	bypasses := ctx.star.Bypasses()
	transportPoint := sgmmath.Point{X: -ctx.iconOffset / 2, Y: ctx.iconOffset - iconStepMd}
	if ctx.iconOffset == outpostHalfSize && (len(colonies)+len(megastructures)) > 0 {
		transportPoint.Y += iconStepMd / 2
		ctx.nameOffsetBottom += iconStepMd / 2
	}
	for _, bypass := range bypasses {
		transportPoint.X -= iconStepSm
		r.createIcon(ctx.g, transportPoint, "bypass-"+bypass, iconSizeSm)
	}
}

func (r *Renderer) renderAllFleets(ctx *starRenderContext) {
	allFleets := ctx.star.MobileMilitaryFleets()
	fleetGroups := make([][]*sgm.Fleet, sgm.WarRoleMax)
	for _, fleet := range allFleets {
		role := fleet.Role(ctx.star)
		fleetGroups[role] = append(fleetGroups[role], fleet)
	}

	for role, fleets := range fleetGroups {
		r.renderFleets(ctx, sgm.WarRole(role), fleets)
	}

}

func (r *Renderer) renderFleets(ctx *starRenderContext, role sgm.WarRole, fleets []*sgm.Fleet) {
	if len(fleets) == 0 {
		return
	}

	fleetPoint := sgmmath.Point{
		Y: -ctx.iconOffset - fleetHalfSize/2,
	}
	step := fleetStep

	switch role {
	case sgm.WarRoleStarNeutral:
		ctx.nameOffsetBottom += fleetStep
		fleetPoint = sgmmath.Point{
			X: -ctx.iconOffset / 3,
			Y: ctx.iconOffset + fleetHalfSize/3,
		}
	case sgm.WarRoleStarDefender:
		ctx.nameOffsetTop += fleetHalfSize + fleetStep
		step = -step
		fleetPoint.X = -fleetHalfSize
	case sgm.WarRoleStarAttacker:
		ctx.nameOffsetTop += fleetHalfSize + fleetStep
		fleetPoint.X = fleetHalfSize
	}

	var extraFleets int
	if len(fleets) > 4 {
		extraFleets = len(fleets) - 4
		fleets = fleets[:4]
	}
	for _, fleet := range fleets {
		r.renderFleet(ctx, fleetPoint, role, fleet)
		fleetPoint.X += step
	}
	if extraFleets > 0 {
		r.createText(ctx.g, fleetTextStyle,
			fleetPoint.Add(sgmmath.Point{-fleetStep, fleetHalfSize}),
			fmt.Sprintf("+%d", extraFleets))
	}
}

func (r *Renderer) renderFleet(
	ctx *starRenderContext, point sgmmath.Point, role sgm.WarRole, fleet *sgm.Fleet,
) {
	fleetStrength := math.Floor(math.Log10(fleet.MilitaryPower)) / 8.0
	if fleetStrength <= 0.2 {
		fleetStrength = 0.2
	}
	style := fleetStyles[role].With(StyleOption{"stroke-width", fmt.Sprintf("%fpt", fleetStrength)})

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

			fleetIdentPath := newDiamondPath(2 * fleetHalfSize / 3)
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

	strokeWidth := 0.4
	iconPoint := point
	if planet.OrbitalStarbase() != nil {
		r.createCircle(ctx.g, planetRingStyle, center, radius)

		strokeWidth -= 0.2
		radius -= 0.6
		iconPoint = iconPoint.Add(sgmmath.Point{0.6, 0.6})
	}

	switch {
	case planet.EmployablePops > 50:
		strokeWidth += 0.4
		radius -= 0.4
	case planet.EmployablePops > 25:
		strokeWidth += 0.2
		radius -= 0.2
	default:
		radius -= 0.1
	}

	style := basePlanetStyle.With(
		StyleOption{"stroke-width", fmt.Sprintf("%.1fpt", strokeWidth)},
	)
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
		point.X -= 2 * ctx.iconOffset / 3
	case 1, -2:
		textAnchor = "end"
		point.X += 2 * ctx.iconOffset / 3
	}
	if ctx.quadrant >= 0 {
		point.Y -= ctx.iconOffset/2 + 2*fontSize/3 + ctx.nameOffsetTop
	} else {
		point.Y += ctx.iconOffset + fontSize + ctx.nameOffsetBottom
	}

	textEl := r.createText(r.canvas, starTextStyle, point, name)
	textEl.CreateAttr("text-anchor", textAnchor)
}

func (r *Renderer) renderHyperlanes() {
	renderedStars := make(map[sgm.StarId]struct{})

	for starId, star := range r.state.Stars {
		hasRelay := star.HasHyperRelay()

		for _, hyperlane := range star.Hyperlanes {
			if _, isRendered := renderedStars[hyperlane.ToId]; isRendered {
				continue
			}

			style := hyperlaneStyle
			if hasRelay && hyperlane.To.HasHyperRelay() {
				style = style.With(StyleOption{"stroke-width", "0.8pt"})
			}

			pv := sgmmath.NewVector(star, hyperlane.To).ToPolar()
			r.createPath(r.canvas, style,
				NewPath().
					MoveToPoint(pv.PointAtLength(2*starHalfSize)).
					LineToPoint(pv.PointAtLength(pv.Length-2*starHalfSize)))
		}

		renderedStars[starId] = struct{}{}
	}
}
