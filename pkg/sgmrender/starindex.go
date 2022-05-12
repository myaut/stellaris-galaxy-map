package sgmrender

import (
	"sort"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

const (
	quadrantCount    = 16
	maxQuadrantIndex = 15
)

type StarAdjacency struct {
	AdjStarId sgm.StarId
	AdjStar   *sgm.Star
	Vector    sgmmath.PolarVector
}

type StarAdjacencyList []StarAdjacency

func (l StarAdjacencyList) Len() int { return len(l) }
func (l StarAdjacencyList) Less(i, j int) bool {
	return l[i].Vector.Angle < l[j].Vector.Angle
}
func (l StarAdjacencyList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

type StarGeoIndex map[sgm.StarId]StarAdjacencyList

func (r *Renderer) buildStarIndex() {
	type starQuadrantSet map[uint8]struct{}

	w, h := r.bounds.Size()
	starIndex := make([][]sgm.StarId, 256)
	starQuadrants := make(map[sgm.StarId]starQuadrantSet)
	getStarQuadrant := func(point sgmmath.Point) uint8 {
		return (uint8(maxQuadrantIndex*(point.X-r.bounds.Min.X)/w)<<4 |
			uint8(maxQuadrantIndex*(point.Y-r.bounds.Min.Y)/h))
	}

	hqw, hqh := w/quadrantCount/2, h/quadrantCount/2
	for starId, star := range r.state.Stars {
		quadrants := make(starQuadrantSet)

		// If star is offset, try it in different quadrants
		for _, d := range []sgmmath.Point{
			{0.0, 0.0},
			{-hqw, hqh},
			{hqw, hqh},
			{hqw, -hqh},
			{-hqw, -hqh},
		} {
			quadrantIdx := getStarQuadrant(star.Point().Add(d))
			if _, touchedQuadrant := quadrants[quadrantIdx]; touchedQuadrant {
				continue
			}

			starIndex[quadrantIdx] = append(starIndex[quadrantIdx], starId)
			quadrants[quadrantIdx] = struct{}{}
		}
		starQuadrants[starId] = quadrants
	}

	// Pass 2 - collect all adjacent neighbors
	r.starGeoIndex = make(StarGeoIndex)
	for starId, star := range r.state.Stars {
		adjList := make(StarAdjacencyList, 0, 10)
		adjStarSet := make(map[sgm.StarId]struct{})
		for quadrantIdx := range starQuadrants[starId] {
			for _, adjStarId := range starIndex[quadrantIdx] {
				if adjStarId == starId {
					continue
				}
				if _, addedAdj := adjStarSet[adjStarId]; addedAdj {
					continue
				}
				adjStarSet[adjStarId] = struct{}{}

				adjStar := r.state.Stars[adjStarId]
				pv := sgmmath.NewVector(star, adjStar).ToPolar()
				if pv.Length > (hqw + hqh) {
					// The star is too far away, ignore it
					continue
				}

				adjList = append(adjList, StarAdjacency{
					AdjStarId: adjStarId,
					AdjStar:   adjStar,
					Vector:    pv,
				})
			}
		}
		sort.Sort(adjList)
		r.starGeoIndex[starId] = adjList
	}
}
