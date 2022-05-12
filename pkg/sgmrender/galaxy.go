package sgmrender

import (
	"fmt"
	"math"
	"strings"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

const (
	countryAngleStep = math.Pi / 6.0
	defaultCellSize  = 20.0
)

func (r *Renderer) renderCountries() {
	renderedStars := make(map[sgm.StarId]struct{})
	for starId, star := range r.state.Stars {
		if star.Starbase == nil {
			continue
		}

		path := r.renderCountryChunk(starId, star, nil, NewPath(), renderedStars)
		r.createPath(r.canvas, defaultStarStyle, path.Complete())
	}
}

func (r *Renderer) renderCountryChunk(
	starId sgm.StarId, star *sgm.Star, prevStar *sgm.Star, path Path,
	renderedStars map[sgm.StarId]struct{},
) Path {
	if _, isRendered := renderedStars[starId]; isRendered {
		return path
	}
	renderedStars[starId] = struct{}{}

	// Build list of adjancencies except list of recently visited nodes
	adjList := r.starGeoIndex[starId]
	if prevStar != nil {
		for i, adjNode := range adjList {
			if adjNode.AdjStar == prevStar {
				adjList = append(adjList[i+1:], adjList[:i]...)
				break
			}
		}
	}

	addPoint := func(point sgmmath.Point) {
		if len(path.path) == 0 {
			path = path.MoveToPoint(point)
		} else {
			path = path.LineToPoint(point)
		}
	}

	for i, nextNode := range adjList {
		var prevNode StarAdjacency
		if i == 0 {
			prevNode = adjList[len(adjList)-1]
		} else {
			prevNode = adjList[i-1]
		}

		prevStarbase := prevNode.AdjStar.Starbase
		nextStarbase := nextNode.AdjStar.Starbase
		if prevStarbase == nil || prevStarbase.Owner != star.Starbase.Owner {
			prevMiddle := prevNode.Vector.PointAtOffset(0.48)
			nextMiddle := nextNode.Vector.PointAtOffset(0.48)
			v := sgmmath.Vector{prevMiddle, nextMiddle}

			addPoint(v.ToPolar().PointAtOffset(0.5))

			if nextStarbase == nil || nextStarbase.Owner != star.Starbase.Owner {
				addPoint(nextMiddle)
			}
		} else {
			path = r.renderCountryChunk(nextNode.AdjStarId, nextNode.AdjStar,
				star, path, renderedStars)
		}
	}

	return path
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
		for _, hl := range star.Hyperlanes {
			if _, isRendered := renderedStars[hl.To]; isRendered {
				continue
			}

			pv := sgmmath.NewVector(star, r.state.Stars[hl.To]).ToPolar()
			r.createPath(r.canvas, hyperlaneStyle,
				NewPath().
					MoveToPoint(pv.PointAtLength(2*starSize)).
					LineToPoint(pv.PointAtLength(pv.Length-2*starSize)))
		}

		renderedStars[starId] = struct{}{}
	}
}
