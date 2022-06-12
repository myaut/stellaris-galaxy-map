package sgmrender

import (
	"fmt"
	"log"
	"math"
	"strings"
	"unicode"

	"github.com/pzsz/voronoi"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-map/pkg/sgmmath"
)

type countrySegFlag uint

var (
	DefaultCountryBorderColor = "#202020"
	DefaultCountryFillColor   = "#c0c0c0"
)

const (
	countrySegHasCapital countrySegFlag = 1 << iota
	countrySegHasOuterEdge
	countrySegHasInnerEdge
)

func (flags countrySegFlag) String() string {
	var flagStr []string
	switch {
	case flags&countrySegHasCapital != 0:
		flagStr = append(flagStr, "CAPITAL")
	case flags&countrySegHasOuterEdge != 0:
		flagStr = append(flagStr, "EOUTER")
	case flags&countrySegHasInnerEdge != 0:
		flagStr = append(flagStr, "EINNER")
	}
	return strings.Join(flagStr, "|")
}

const (
	borderEpsilon        = 1.0
	borderAngleThreshold = 5 * math.Pi / 8
)

type countryBorderKey struct {
	X, Y int
}

func borderEdgeKey(vertex voronoi.Vertex) countryBorderKey {
	return countryBorderKey{
		X: int(math.Floor(vertex.X / borderEpsilon)),
		Y: int(math.Floor(vertex.Y / borderEpsilon)),
	}
}

type countryBorder struct {
	edges []*voronoi.Halfedge
	rect  sgmmath.BoundingRect
}

type countrySegment struct {
	countryId sgm.CountryId

	flags  countrySegFlag
	bounds sgmmath.BoundingRect

	stars map[sgm.StarId]struct{}

	borderMap map[countryBorderKey]*countryBorder
	borders   []*countryBorder

	cells []*voronoi.Cell
}

type countryGetter func(s *sgm.Star) sgm.CountryId

type countryRenderer struct {
	r *Renderer

	diagram *voronoi.Diagram
	starMap map[voronoi.Vertex]sgm.StarId
}

type countryRenderContext struct {
	country *sgm.Country
	seg     *countrySegment
	style   Style
}

func (r *Renderer) createCountryRenderer() *countryRenderer {
	cr := &countryRenderer{
		r: r,

		starMap: make(map[voronoi.Vertex]sgm.StarId),
	}

	sites := make([]voronoi.Vertex, 0, len(r.state.Stars))
	for starId, star := range r.state.Stars {
		vertex := voronoi.Vertex(star.Point())
		cr.starMap[vertex] = starId

		sites = append(sites, vertex)
	}

	bbox := voronoi.NewBBox(
		r.bounds.Min.X-canvasPadding/2,
		r.bounds.Max.X+canvasPadding/2,
		r.bounds.Min.Y-canvasPadding/2,
		r.bounds.Max.Y+canvasPadding/2)
	cr.diagram = voronoi.ComputeDiagram(sites, bbox, true)

	return cr
}

func (cr *countryRenderer) starByCell(cell *voronoi.Cell) (sgm.StarId, *sgm.Star) {
	starId := cr.starMap[cell.Site]
	return starId, cr.r.state.Stars[starId]
}

func (cr *countryRenderer) buildSegments(getter countryGetter) []*countrySegment {
	segments := make([]*countrySegment, 0, 16)
	cellMap := make(map[*voronoi.Cell]*countrySegment)

	// Walk all cells and build country segments
	for _, cell := range cr.diagram.Cells {
		if _, walkedCell := cellMap[cell]; walkedCell {
			continue
		}

		starId, star := cr.starByCell(cell)
		countryId := getter(star)
		if countryId == sgm.DefaultCountryId {
			continue
		}

		seg := &countrySegment{
			countryId: countryId,
			bounds:    sgmmath.NewBoundingRect(),
			stars: map[sgm.StarId]struct{}{
				starId: struct{}{},
			},
			cells:     []*voronoi.Cell{cell},
			borderMap: make(map[countryBorderKey]*countryBorder),
		}

		cr.buildSegment(cellMap, getter, seg, starId, star, cell, nil)
		segments = append(segments, seg)
	}

	if traceFlags&traceFlagCountrySegments != 0 {
		for _, seg := range segments {
			log.Printf("InitSeg: %p (%s) - stars: %d, borders: %d, flags: %s, bounds: %v\n",
				seg, sgm.CountryName(seg.countryId, cr.r.state.Countries[seg.countryId]),
				len(seg.stars), len(seg.borders), seg.flags.String(), seg.bounds)
		}
	}

	return segments
}

func (cr *countryRenderer) buildSegment(
	cellMap map[*voronoi.Cell]*countrySegment, getter countryGetter,
	seg *countrySegment, starId sgm.StarId, star *sgm.Star,
	cell, prevCell *voronoi.Cell,
) {
	if _, walkedCell := cellMap[cell]; walkedCell {
		return
	}
	cellMap[cell] = seg

	seg.bounds.Add(star.Point())
	if star.HasCapital() {
		seg.flags |= countrySegHasCapital
	}

	// Continue walking halfedges from the side opposite to the last walked cell
	halfedges := cell.Halfedges
	if prevCell != nil {
		for i, he := range halfedges {
			otherCell := he.Edge.GetOtherCell(cell)
			if otherCell == prevCell {
				halfedges = append(halfedges[i:], halfedges[:i]...)
				break
			}
		}
	}

	for _, he := range halfedges {
		neighCell := he.Edge.GetOtherCell(cell)
		if neighCell != nil {
			neighStarId, neighStar := cr.starByCell(neighCell)

			if getter(neighStar) == seg.countryId && star.IsDistant() == neighStar.IsDistant() {
				// Recursively walk edges counter-clockwise
				seg.cells = append(seg.cells, neighCell)
				seg.stars[neighStarId] = struct{}{}

				cr.buildSegment(cellMap, getter, seg, neighStarId, neighStar, neighCell, cell)
				continue
			}

			if math.Signbit(he.Cell.Site.X) != math.Signbit(neighCell.Site.X) &&
				math.Signbit(he.Cell.Site.Y) != math.Signbit(neighCell.Site.Y) {
				seg.flags |= countrySegHasInnerEdge
			}
		}

		startVertex, endVertex := he.Edge.Va.Vertex, he.Edge.Vb.Vertex
		if he.Edge.RightCell == cell {
			startVertex, endVertex = endVertex, startVertex
		}
		startKey, endKey := borderEdgeKey(startVertex), borderEdgeKey(endVertex)

		if startBorder, knownStart := seg.borderMap[startKey]; knownStart {
			seg.borderMap[endKey] = startBorder
			startBorder.edges = append(startBorder.edges, he)
			startBorder.rect.Add(sgmmath.Point(endVertex))
		} else if endBorder, knownEnd := seg.borderMap[endKey]; knownEnd {
			seg.borderMap[startKey] = endBorder
			endBorder.edges = append(endBorder.edges, he)
			endBorder.rect.Add(sgmmath.Point(startVertex))
		} else {
			if traceFlags&traceFlagCountrySegments != 0 {
				fmt.Printf("NewBorder: %p -> [%v, %v], %v\n", seg,
					startKey, endKey, seg.borderMap)
			}

			border := &countryBorder{
				edges: []*voronoi.Halfedge{he},
				rect:  sgmmath.NewPointBoundingRect(sgmmath.Point(startVertex)),
			}
			border.rect.Add(sgmmath.Point(endVertex))

			seg.borderMap[startKey] = border
			seg.borderMap[endKey] = border
			seg.borders = append(seg.borders, border)
		}
	}
}

func (cr *countryRenderer) buildPoint(
	cellPoint sgmmath.Point, ev voronoi.EdgeVertex, insetSign float64,
) sgmmath.Point {
	point := sgmmath.Point{ev.X, ev.Y}
	pv := sgmmath.NewVector(cellPoint, point).ToPolar()

	if pv.Length > maxCellSize {
		return pv.PointAtLength(maxCellSize)
	}
	return pv.PointAtLength(pv.Length - insetSign*countryBorderGap)
}

func (cr *countryRenderer) buildEdge(he *voronoi.Halfedge, insetSign float64) (sgmmath.Point, sgmmath.Point) {
	cellPoint := sgmmath.Point(he.Cell.Site)
	startPoint := cr.buildPoint(cellPoint, he.Edge.Va, insetSign)
	endPoint := cr.buildPoint(cellPoint, he.Edge.Vb, insetSign)
	if he.Edge.RightCell == he.Cell {
		return endPoint, startPoint
	}
	return startPoint, endPoint
}

func (cr *countryRenderer) buildBorderPath(border *countryBorder, path Path, insetSign float64) Path {
	for i, he := range border.edges {
		startPoint, endPoint := cr.buildEdge(he, insetSign)

		if i == 0 {
			prevIndex := i - 1
			if prevIndex < 0 {
				prevIndex = len(border.edges) - 1
			}
			prevPoint, _ := cr.buildEdge(border.edges[prevIndex], insetSign)

			prevVector := sgmmath.NewVector(startPoint, prevPoint).ToPolar()
			nextVector := sgmmath.NewVector(startPoint, endPoint).ToPolar()
			if prevVector.AngleDiff(nextVector) < borderAngleThreshold {
				startPoint = nextVector.PointAtOffset(0.4)
			}

			path = path.MoveToPoint(startPoint)
		}

		nextIndex := i + 1
		if nextIndex == len(border.edges) {
			nextIndex = 0
		}
		_, nextPoint := cr.buildEdge(border.edges[nextIndex], insetSign)

		points := []sgmmath.Point{endPoint}
		prevVector := sgmmath.NewVector(endPoint, startPoint).ToPolar()
		nextVector := sgmmath.NewVector(endPoint, nextPoint).ToPolar()
		if prevVector.AngleDiff(nextVector) < borderAngleThreshold {
			points = []sgmmath.Point{
				prevVector.PointAtOffset(0.4),
				nextVector.PointAtOffset(0.4),
			}
		}

		for _, point := range points {
			path = path.LineToPoint(point)
		}
	}

	return path.Complete()
}

func (cr *countryRenderer) buildPath(seg *countrySegment) Path {
	path := NewPath()
	for _, border := range seg.borders {
		path = cr.buildBorderPath(border, path, 1.0)
	}
	return path
}

func getCountryMapColor(key string, defaultColor string) string {
	// TODO: handle "use_as_border_color", cases when colors are not found
	color := sgm.ColorMap.Colors[key]
	if color == nil {
		return defaultColor
	}
	return color.Map.Color().ToHexCode().String()
}

func (r *Renderer) renderCountries() []countryRenderContext {
	cr := r.createCountryRenderer()
	segments := cr.buildSegments(func(s *sgm.Star) sgm.CountryId {
		return s.Owner()
	})
	countries := make([]countryRenderContext, 0, len(segments))
	for _, seg := range segments {
		country := r.state.Countries[seg.countryId]
		style := baseCountryStyle.With(
			StyleOption{"stroke", getCountryMapColor(country.Flag.Colors[1], DefaultCountryBorderColor)},
			StyleOption{"fill", getCountryMapColor(country.Flag.Colors[0], DefaultCountryFillColor)},
		)

		r.createPath(r.canvas, style, cr.buildPath(seg))

		countries = append(countries, countryRenderContext{
			country: country,
			seg:     seg,
			style:   style,
		})
	}

	occupiedSegs := cr.buildSegments(func(s *sgm.Star) sgm.CountryId {
		return s.Occupier()
	})
	occupierPatters := make(map[sgm.CountryId]struct{})
	for _, seg := range occupiedSegs {
		country := r.state.Countries[seg.countryId]
		patternId := fmt.Sprintf("occupied-by-%d", seg.countryId)
		if _, hasPattern := occupierPatters[seg.countryId]; !hasPattern {
			r.createCountryPattern(seg.countryId, patternId)
			occupierPatters[seg.countryId] = struct{}{}
		}

		style := occupationCountryStyle.With(
			StyleOption{"stroke", getCountryMapColor(country.Flag.Colors[1], DefaultCountryBorderColor)},
			StyleOption{"fill", fmt.Sprintf("url(#%s)", patternId)},
		)
		r.createPath(r.canvas, style, cr.buildPath(seg))
	}

	if traceFlags&traceFlagShowGraphEdges != 0 {
		for _, edge := range cr.diagram.Edges {
			r.createPath(r.canvas, hyperlaneStyle,
				NewPath().MoveTo(edge.Va.X, edge.Va.Y).LineTo(edge.Vb.X, edge.Vb.Y))
		}
	}

	return countries
}

func (r *Renderer) createCountryPattern(countryId sgm.CountryId, id string) {
	pattern := r.defs.CreateElement("pattern")
	pattern.CreateAttr("id", id)
	pattern.CreateAttr("x", "0")
	pattern.CreateAttr("y", "0")
	pattern.CreateAttr("width", fmt.Sprint(countryPatternSize))
	pattern.CreateAttr("height", fmt.Sprint(countryPatternSize))
	pattern.CreateAttr("patternUnits", "userSpaceOnUse")

	g := pattern.CreateElement("g")
	country := r.state.Countries[countryId]
	style := occupationPatternStyle.With(
		StyleOption{"stroke", getCountryMapColor(country.Flag.Colors[0], DefaultCountryFillColor)},
	)
	for x := 0.0; x < countryPatternSize; x += countryPatternStep {
		r.createPath(g, style,
			NewPath().MoveTo(x+countryPatternStep, 0.0).LineTo(x, countryPatternSize))
	}
}

func (r *Renderer) renderCountryNames(countries []countryRenderContext) {
	var smallCountryNames []string
	smallCountries := make(map[sgm.CountryId]int)

	for _, ctx := range countries {
		lines, maxLineLength := []string{ctx.country.Name()}, len(ctx.country.Name())
		rectW, _ := ctx.seg.bounds.Size()
		if float64(maxLineLength)*countryFontSize > 0.8*rectW {
			lines, maxLineLength = r.countryNameLines(ctx.country.Name())
		}

		point, foundPoint := r.findCountryNamePoint(ctx.seg, maxLineLength, len(lines))
		style := countryTextStyle
		if !foundPoint {
			if index, hasIndex := smallCountries[ctx.seg.countryId]; hasIndex {
				lines = []string{fmt.Sprint(index)}
			} else {
				index := len(smallCountryNames) + 1
				smallCountries[ctx.seg.countryId] = index
				smallCountryNames = append(smallCountryNames, ctx.country.Name())
				lines = []string{fmt.Sprint(index)}
			}

			point, foundPoint = r.findCountryNamePoint(ctx.seg, 2, 1)
			if !foundPoint {
				point = ctx.seg.bounds.Center()
			}
			style = style.With(
				StyleOption{"font-size", fmt.Sprintf("%fpt", countryFontSize*0.75)},
			)
		}

		point.Y += 1.2 * countryFontSize
		for _, line := range lines {
			textEl := r.createText(r.canvas, style, point, line)
			textEl.CreateAttr("text-anchor", "middle")

			point.Y += 1.2 * countryFontSize
		}
	}

	legendPoint := sgmmath.Point{
		X: r.innerBounds.Min.X,
		Y: r.innerBounds.Max.Y - float64(len(smallCountries))*0.6*countryFontSize,
	}
	for idx, name := range smallCountryNames {
		r.createText(r.canvas, countryLegendStyle, legendPoint,
			fmt.Sprintf("%d - %s", idx+1, name))
		legendPoint.Y += 0.6 * countryFontSize
	}
}

func (r *Renderer) countryNameLines(name string) (lines []string, maxLineLength int) {
	lines = strings.Split(name, " ")
	for i, line := range lines {
		if i > 0 && !unicode.IsUpper(rune(line[0])) {
			lines = append(
				append(lines[:i-1], fmt.Sprintf("%s %s", lines[i-1], line)),
				lines[i+1:]...,
			)
			break
		}
	}
	for _, line := range lines {
		if len(line) > maxLineLength {
			maxLineLength = len(line)
		}
	}
	return
}

func (r *Renderer) findCountryNamePoint(seg *countrySegment, maxLineLength, lineCount int) (point sgmmath.Point, found bool) {
	center := seg.bounds.Center()
	distance := math.Inf(+1)

	requiredWidth := float64(maxLineLength) * countryFontSize
	requiredHeight := float64(lineCount) * countryFontSize
starLoop:
	for starId := range seg.stars {
		star := r.state.Stars[starId]
		starPoint := star.Point()
		if starPoint.X-seg.bounds.Min.X < requiredWidth/2 {
			starPoint.X += requiredWidth / 4
		} else if seg.bounds.Max.X-starPoint.X < requiredWidth/2 {
			starPoint.X -= requiredWidth / 4
		}
		if star.IsSignificant() || len(star.Battles) > 0 {
			starPoint.Y += fontSize + countryFontSize/2
		}

		rect := sgmmath.BoundingRect{
			Min: sgmmath.Point{
				X: starPoint.X - requiredWidth/2,
				Y: starPoint.Y - countryFontSize/2,
			},
			Max: sgmmath.Point{
				X: starPoint.X + requiredWidth/2,
				Y: starPoint.Y + requiredHeight + countryFontSize/2,
			},
		}

		for _, border := range seg.borders {
			for _, he := range border.edges {
				if rect.Includes(sgmmath.Point(he.Edge.Va.Vertex)) ||
					rect.Includes(sgmmath.Point(he.Edge.Vb.Vertex)) {
					continue starLoop
				}
			}
		}
		for _, adjNode := range r.starGeoIndex[starId] {
			var probeOffset float64
			if !adjNode.AdjStar.IsOwnedBy(seg.countryId) {
				probeOffset = 0.5
			} else if adjNode.AdjStar.IsSignificant() || len(adjNode.AdjStar.Battles) > 0 {
				probeOffset = 0.8
			} else {
				continue
			}

			probe := adjNode.Vector.PointAtOffset(probeOffset)
			if rect.Includes(probe) {
				continue starLoop
			}
		}

		found = true
		starDistance := center.Distance(starPoint)
		if starDistance < distance {
			point, distance = starPoint, starDistance
		}
	}

	return
}
