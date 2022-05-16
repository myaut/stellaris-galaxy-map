package sgmrender

import (
	"log"
	"math"
	"sort"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-map/pkg/sgmmath"
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

				if traceFlags&traceFlagStarIndex != 0 {
					log.Printf("AdjNode: %s -> %s (angle: %f)\n", star.Name, adjStar.Name, pv.Angle)
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

//
// Returns quadrants: I - 0, II - 1, III - -2, IV - -1
func (r *Renderer) pickTextQuadrant(starId sgm.StarId) int {
	quadrantWeight := make([]float64, 4)
	for _, adjNode := range r.starGeoIndex[starId] {
		quadrant := adjNode.Vector.Sector(4)
		weight := 0.1

		// Nodes with pops/starbases produce large icon sets with texts
		// that we're trying to avoid
		if adjNode.AdjStar.HasPops() {
			weight += 1.0
		}
		if adjNode.AdjStar.HasSignificantMegastructures() {
			weight += 1.0
		}
		if adjNode.AdjStar.HasUpgradedStarbase() {
			weight += 2.0
		}

		points := weight / (adjNode.Vector.Length * adjNode.Vector.Length)
		quadrantWeight[quadrant] += points

		// If too close to the quadrant's edge, account to neighboring qudrant too
		if math.Abs(math.Sin(adjNode.Vector.Angle)) < 0.2 {
			quadrantWeight[3-quadrant] += points
		}

	}

	minQuadrant, minWeight := -1, 0.0
	for quadrant, weight := range quadrantWeight {
		if minQuadrant == -1 || weight < minWeight {
			minQuadrant, minWeight = quadrant, weight
		}
	}

	if traceFlags&traceFlagStarIndex != 0 {
		log.Printf("TextQuadrant: %s - %v -> %v\n", r.state.Stars[starId].Name,
			quadrantWeight, minQuadrant-2)
	}
	return minQuadrant - 2
}
