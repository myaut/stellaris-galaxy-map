package sgmrender

import (
	"github.com/pzsz/voronoi"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

type countryRenderer struct {
	r *Renderer

	starMap map[voronoi.Vertex]*sgm.Star
	diagram *voronoi.Diagram

	renderedCells map[*voronoi.Cell]struct{}
}

func (r *Renderer) createCountryRenderer() *countryRenderer {
	cr := &countryRenderer{
		starMap:       make(map[voronoi.Vertex]*sgm.Star),
		renderedCells: make(map[*voronoi.Cell]struct{}),
	}

	sites := make([]voronoi.Vertex, 0, len(r.state.Stars))
	for _, star := range r.state.Stars {
		point := star.Point()
		vertex := voronoi.Vertex{X: point.X, Y: point.Y}

		sites = append(sites, vertex)
		cr.starMap[vertex] = star
	}

	bbox := voronoi.NewBBox(
		r.bounds.Min.X-canvasPadding/2,
		r.bounds.Max.X+canvasPadding/2,
		r.bounds.Min.Y-canvasPadding/2,
		r.bounds.Max.Y+canvasPadding/2)
	cr.diagram = voronoi.ComputeDiagram(sites, bbox, true)
	return cr
}

func (cr *countryRenderer) buildPoint(cellPoint sgmmath.Point, ev voronoi.EdgeVertex) sgmmath.Point {
	point := sgmmath.Point{ev.X, ev.Y}
	pv := sgmmath.NewVector(cellPoint, point).ToPolar()

	if pv.Length > maxCellSize {
		return pv.PointAtLength(maxCellSize)
	}
	return pv.PointAtLength(pv.Length - countryBorderGap)
}

func (cr *countryRenderer) buildPath(
	star *sgm.Star, cell, prevCell *voronoi.Cell, owner sgm.CountryId, path Path,
) Path {
	if _, isRendered := cr.renderedCells[cell]; isRendered {
		return path
	}
	cr.renderedCells[cell] = struct{}{}

	halfEdges := cell.Halfedges
	if prevCell != nil {
		for i, he := range halfEdges {
			otherCell := he.Edge.GetOtherCell(cell)
			if otherCell == prevCell {
				halfEdges = append(halfEdges[i:], halfEdges[:i]...)
				break
			}
		}
	}

	cellPoint := sgmmath.Point{cell.Site.X, cell.Site.Y}
	for _, he := range halfEdges {
		otherCell := he.Edge.GetOtherCell(cell)
		if otherCell != nil {
			otherStar := cr.starMap[otherCell.Site]
			if otherStar.IsOwnedBy(owner) {
				path = cr.buildPath(otherStar, otherCell, cell, owner, path)
				continue
			}

			// Enclaves break logic by making crazy moves throughout entire canvas
			// for now completely ignore them
			// TODO: Compute set difference for them
			var nonEnclaveEdges int
			for _, otherEdge := range otherCell.Halfedges {
				if neighCell := otherEdge.Edge.GetOtherCell(otherCell); neighCell != nil {
					neighStar := cr.starMap[neighCell.Site]
					if neighStar.Starbase == nil || neighStar.IsOwnedBy(otherStar.Owner()) {
						nonEnclaveEdges++
					}
				}
			}
			if nonEnclaveEdges == 0 {
				continue
			}
		}

		startPoint := cr.buildPoint(cellPoint, he.Edge.Va)
		endPoint := cr.buildPoint(cellPoint, he.Edge.Vb)
		if he.Edge.LeftCell == otherCell {
			startPoint, endPoint = endPoint, startPoint
		}

		if len(path.path) == 0 {
			path = path.MoveToPoint(startPoint)
		}
		path = path.LineToPoint(endPoint)
	}
	return path
}
