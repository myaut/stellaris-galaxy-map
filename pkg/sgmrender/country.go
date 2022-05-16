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

const (
	countrySegHasCapital countrySegFlag = 1 << iota
	countrySegHasOuterEdge
	countrySegHasInnerEdge
	countrySegIsEnclave
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
	case flags&countrySegIsEnclave != 0:
		flagStr = append(flagStr, "ENCLAVE")
	}
	return strings.Join(flagStr, "|")
}

type countrySegmentAdjancency struct {
	isEnclave bool
}

type countrySegment struct {
	countryId sgm.CountryId

	flags  countrySegFlag
	bounds sgmmath.BoundingRect

	stars map[sgm.StarId]struct{}

	borders   []*voronoi.Halfedge
	cells     []*voronoi.Cell
	neighbors map[*countrySegment]countrySegmentAdjancency
}

type countryRenderer struct {
	r *Renderer

	diagram *voronoi.Diagram

	starMap  map[voronoi.Vertex]sgm.StarId
	cellMap  map[*voronoi.Cell]*countrySegment
	segments []*countrySegment
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
		cellMap: make(map[*voronoi.Cell]*countrySegment),
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

	cr.initSegments()
	return cr
}

func (cr *countryRenderer) starByCell(cell *voronoi.Cell) (sgm.StarId, *sgm.Star) {
	starId := cr.starMap[cell.Site]
	return starId, cr.r.state.Stars[starId]
}

func (cr *countryRenderer) getBorderNeighbor(border *voronoi.Halfedge) (*voronoi.Cell, *countrySegment) {
	neighCell := border.Edge.GetOtherCell(border.Cell)
	return neighCell, cr.cellMap[neighCell]
}

func (cr *countryRenderer) initSegments() {
	// Walk all cells and build country segments
	for _, cell := range cr.diagram.Cells {
		if _, walkedCell := cr.cellMap[cell]; walkedCell {
			continue
		}

		starId, star := cr.starByCell(cell)
		seg := &countrySegment{
			countryId: star.Owner(),
			bounds:    sgmmath.NewBoundingRect(),
			stars: map[sgm.StarId]struct{}{
				starId: struct{}{},
			},
			cells:     []*voronoi.Cell{cell},
			neighbors: make(map[*countrySegment]countrySegmentAdjancency),
		}
		cr.buildSegment(seg, starId, star, cell, nil)
		cr.segments = append(cr.segments, seg)
	}

	// After we identified all borders, find neighbors and enclaves
	for _, seg := range cr.segments {
		for _, border := range seg.borders {
			_, neighSeg := cr.getBorderNeighbor(border)
			if neighSeg == nil {
				continue
			}

			if _, knownNeighbor := seg.neighbors[neighSeg]; !knownNeighbor {
				var adjNode countrySegmentAdjancency
				if cr.isEnclave(seg, neighSeg) {
					seg.flags |= countrySegIsEnclave
					adjNode.isEnclave = true
				}

				seg.neighbors[neighSeg] = adjNode
			}
		}
	}

	if traceFlags&traceFlagCountrySegments != 0 {
		for _, seg := range cr.segments {
			log.Printf("InitSeg: %p (%s) - stars: %d, neigh: %d, flags: %s, bounds: %v\n",
				seg, sgm.CountryName(seg.countryId, cr.r.state.Countries[seg.countryId]),
				len(seg.stars), len(seg.neighbors), seg.flags.String(), seg.bounds)
		}
	}
}

func (cr *countryRenderer) isEnclave(seg, outerSeg *countrySegment) bool {
	if seg.flags&(countrySegHasOuterEdge|countrySegHasInnerEdge) != 0 {
		return false
	}
	if !outerSeg.bounds.Contains(seg.bounds) {
		return false
	}

	// Check simplest enclave case - empire on all sides of the our segment
	var hasAnotherEmpire bool
	for _, border := range seg.borders {
		otherCell := border.Edge.GetOtherCell(border.Cell)
		if cr.cellMap[otherCell] != outerSeg {
			hasAnotherEmpire = true
			break
		}
	}
	if !hasAnotherEmpire {
		return true
	}

	// Segments might be large, making false positives when checking simple
	// rectangles, so we use adjacent stars this time
	// Note that we don't use direct adjacencies (like voronoi graph provides)
	// due to scenario of multiple conjoined enclaves
	const sectorCount = 6
	sectorAdjancency := make([]float64, sectorCount)
	center := seg.bounds.Center()
	checkedAdjStar := make(map[sgm.StarId]struct{})
	for _, cell := range seg.cells {
		starId := cr.starMap[cell.Site]
		for _, adjNode := range cr.r.starGeoIndex[starId] {
			if _, checked := checkedAdjStar[adjNode.AdjStarId]; checked {
				continue
			}
			checkedAdjStar[adjNode.AdjStarId] = struct{}{}

			pv := sgmmath.NewVector(center, adjNode.AdjStar.Point()).ToPolar()
			weight := 1.0
			if _, inOuterSeg := outerSeg.stars[adjNode.AdjStarId]; !inOuterSeg {
				weight = -weight
			}

			sectorAdjancency[pv.Sector(sectorCount)] += weight
		}
	}

	if traceFlags&traceFlagCountrySegments != 0 {
		log.Printf("IsEnclave: %p -> %p, adj: %v\n", outerSeg, seg, sectorAdjancency)
	}

	// Check that at list for one of six directions we do not have enough adjacent nodes
	for _, adjNodeWeight := range sectorAdjancency {
		if adjNodeWeight < 0 {
			return false
		}
	}
	return true
}

func (cr *countryRenderer) buildSegment(
	seg *countrySegment, starId sgm.StarId, star *sgm.Star,
	cell, prevCell *voronoi.Cell,
) {
	if _, walkedCell := cr.cellMap[cell]; walkedCell {
		return
	}
	cr.cellMap[cell] = seg

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
		if neighCell == nil {
			seg.flags |= countrySegHasOuterEdge
			seg.borders = append(seg.borders, he)
			continue
		}

		neighStarId, neighStar := cr.starByCell(neighCell)
		if neighStar.Owner() == seg.countryId && star.IsDistant() == neighStar.IsDistant() {
			// Recursively walk edges counter-clockwise
			seg.cells = append(seg.cells, neighCell)
			seg.stars[neighStarId] = struct{}{}

			cr.buildSegment(seg, neighStarId, neighStar, neighCell, cell)
		} else {
			if math.Signbit(he.Cell.Site.X) != math.Signbit(neighCell.Site.X) &&
				math.Signbit(he.Cell.Site.Y) != math.Signbit(neighCell.Site.Y) {
				seg.flags |= countrySegHasInnerEdge
			}

			seg.borders = append(seg.borders, he)
		}

		if neighStar.HasCapital() {
			seg.flags |= countrySegHasCapital
		}
	}

	seg.bounds.Add(star.Point())
}

func (cr *countryRenderer) hasForeignNeighbors(seg *countrySegment, starId sgm.StarId) bool {
	for _, adjNode := range cr.r.starGeoIndex[starId] {
		if !adjNode.AdjStar.IsOwnedBy(seg.countryId) {
			return true
		}
	}
	return false
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

func (cr *countryRenderer) buildPath(
	cutSegs map[*countrySegment]struct{}, seg, outerSeg, prevSeg *countrySegment,
	path Path, insetSign float64, start bool,
) Path {
	if cutSegs != nil {
		// Already touched seg in recurse call below
		if _, alreadyCut := cutSegs[seg]; alreadyCut {
			return path
		}
		cutSegs[seg] = struct{}{}
	}

	borders := seg.borders
	if prevSeg != nil {
		for i, border := range borders {
			_, neighSeg := cr.getBorderNeighbor(border)
			if neighSeg == prevSeg {
				borders = append(borders[i:], borders[:i]...)
				break
			}
		}
	}

	for i, border := range borders {
		_, neighSeg := cr.getBorderNeighbor(border)
		if neighSeg != nil && neighSeg.flags&countrySegIsEnclave != 0 {
			if outerSeg != nil && neighSeg != outerSeg && neighSeg.neighbors[outerSeg].isEnclave {
				// Complex case of multiple-neighbor enclaves: recurse
				path = cr.buildPath(cutSegs, neighSeg, outerSeg, seg, path, insetSign, false)
			}

			// Do not render borders with enclave during first pass
			continue
		}

		cellPoint := sgmmath.Point(border.Cell.Site)
		startPoint := cr.buildPoint(cellPoint, border.Edge.Va, insetSign)
		endPoint := cr.buildPoint(cellPoint, border.Edge.Vb, insetSign)
		if border.Edge.RightCell == border.Cell {
			startPoint, endPoint = endPoint, startPoint
		}

		if start && i == 0 {
			path = path.MoveToPoint(startPoint)
		}
		path = path.LineToPoint(endPoint)
	}

	return path
}

func (r *Renderer) renderCountries() []countryRenderContext {
	cr := r.createCountryRenderer()

	countries := make([]countryRenderContext, 0, len(cr.segments))
	for _, seg := range cr.segments {
		if seg.countryId == sgm.DefaultCountryId {
			continue
		}
		country := r.state.Countries[seg.countryId]

		// TODO: handle "use_as_border_color", cases when colors are not found
		borderColor := sgm.ColorMap.Colors[country.Flag.Colors[1]]
		countryColor := sgm.ColorMap.Colors[country.Flag.Colors[0]]
		style := baseCountryStyle.With(
			StyleOption{"stroke", borderColor.Map.Color().ToHexCode().String()},
			StyleOption{"fill", countryColor.Map.Color().ToHexCode().String()},
		)

		path := cr.buildPath(nil, seg, nil, nil, NewPath(), 1.0, true).Complete()
		cutEnclaves := make(map[*countrySegment]struct{})
		for neighSeg := range seg.neighbors {
			if neighSeg.flags&countrySegIsEnclave != 0 && neighSeg.neighbors[seg].isEnclave {
				path = cr.buildPath(cutEnclaves, neighSeg, seg, nil,
					path, -1.0, true).Complete()
			}
		}
		r.createPath(r.canvas, style, path)

		countries = append(countries, countryRenderContext{
			country: country,
			seg:     seg,
			style:   style,
		})
	}

	if traceFlags&traceFlagShowGraphEdges != 0 {
		for _, edge := range cr.diagram.Edges {
			r.createPath(r.canvas, hyperlaneStyle,
				NewPath().MoveTo(edge.Va.X, edge.Va.Y).LineTo(edge.Vb.X, edge.Vb.Y))
		}
	}

	return countries
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
		X: r.bounds.Min.X,
		Y: r.bounds.Max.Y - float64(len(smallCountries))*0.6*countryFontSize,
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
		if star.IsSignificant() {
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
			if rect.Includes(sgmmath.Point(border.Edge.Va.Vertex)) ||
				rect.Includes(sgmmath.Point(border.Edge.Vb.Vertex)) {
				continue starLoop
			}
		}
		for _, adjNode := range r.starGeoIndex[starId] {
			var probeOffset float64
			if !adjNode.AdjStar.IsOwnedBy(seg.countryId) {
				probeOffset = 0.5
			} else if adjNode.AdjStar.IsSignificant() {
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
