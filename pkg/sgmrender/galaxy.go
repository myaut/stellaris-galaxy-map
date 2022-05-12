package sgmrender

import (
	"fmt"
	"strings"

	"github.com/pzsz/voronoi"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

const (
	maxCellSize      = 40.0
	countryBorderGap = 0.8
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
			var enclaveEdges int
			for _, otherEdge := range otherCell.Halfedges {
				enclaveCell := otherEdge.Edge.GetOtherCell(otherCell)
				if enclaveCell != nil {
					enclaveStar := cr.starMap[enclaveCell.Site]
					if enclaveStar.IsOwnedBy(owner) {
						enclaveEdges++
					}
				}
			}
			if enclaveEdges >= len(otherCell.Halfedges)-1 {
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

func (r *Renderer) renderCountries() {
	cr := r.createCountryRenderer()

	for _, cell := range cr.diagram.Cells {
		star := cr.starMap[cell.Site]
		if star.Starbase == nil {
			continue
		}

		owner := star.Starbase.Owner

		path := cr.buildPath(star, cell, nil, owner, NewPath())
		r.createPath(r.canvas, defaultStarStyle, path.Complete())
	}

	/*
		for _, edge := range cr.diagram.Edges {
			r.createPath(r.canvas, hyperlaneStyle,
				NewPath().MoveTo(edge.Va.X, edge.Va.Y).LineTo(edge.Vb.X, edge.Vb.Y))
		}
	*/
}

func (r *Renderer) renderStars() {
	var stars []*sgm.Star

	for _, star := range r.state.Stars {
		g := r.canvas.CreateElement("g")
		p := star.Point()
		g.CreateAttr("transform", fmt.Sprintf("translate(%f, %f)", p.X, p.Y))

		if star.Starbase == nil {
			r.createPath(g, defaultStarStyle, defaultStarPath)
			continue
		}

		if star.Starbase.Level == sgm.StarbaseOutpost {
			r.createPath(g, baseStarbaseStyle, outpostPath)
			continue
		}

		r.createPath(g, starbaseStyles[star.Starbase.Level], starbasePath)
		stars = append(stars, star)
	}

	for _, star := range stars {
		name := star.Name
		if strings.HasPrefix(name, "NAME_") {
			name = strings.ReplaceAll(name[5:], "_", " ")
		}

		p := star.Point()
		p = p.Add(sgmmath.Point{X: 1.2 * starbaseSize, Y: starbaseSize})
		style := starTextStyle

		text := r.canvas.CreateElement("text")
		text.CreateAttr("x", fmt.Sprintf("%f", p.X))
		text.CreateAttr("y", fmt.Sprintf("%f", p.Y))
		text.CreateAttr("style", style.String())
		text.CreateText(name)
	}
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
