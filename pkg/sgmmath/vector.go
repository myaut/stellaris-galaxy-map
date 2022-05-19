package sgmmath

import (
	"fmt"
	"math"
)

type SectorDirection int

const (
	DirectionTop SectorDirection = iota
	DirectionBottom
	DirectionLeft
	DirectionRight

	DirectionMax
)

type Pointable interface {
	Point() Point
}

type Vector struct {
	Begin, End Point
}

type PolarVector struct {
	Begin  Point
	Angle  float64
	Length float64
}

func NewVector(begin, end Pointable) Vector {
	return Vector{
		Begin: begin.Point(),
		End:   end.Point(),
	}
}

func (v Vector) Size() (float64, float64) {
	return math.Abs(v.End.X - v.Begin.X), math.Abs(v.End.Y - v.Begin.Y)
}

func (v Vector) ToPolar() PolarVector {
	// In trigonometry Y axis is directed up. In svg it is directed down
	// Hence the inversion of h.
	w, h := v.End.X-v.Begin.X, v.Begin.Y-v.End.Y
	return PolarVector{
		Begin:  v.Begin,
		Angle:  math.Atan2(h, w),
		Length: math.Sqrt(w*w + h*h),
	}
}

func (pv PolarVector) PointAtOffset(offset float64) Point {
	return pv.PointAtLength(pv.Length * offset)
}

func (pv PolarVector) PointAtLength(l float64) Point {
	return Point{
		X: pv.Begin.X + math.Cos(pv.Angle)*l,
		Y: pv.Begin.Y - math.Sin(pv.Angle)*l,
	}
}

func (pv PolarVector) ToVector() Vector {
	return Vector{
		Begin: pv.Begin,
		End:   pv.PointAtLength(pv.Length),
	}
}

func (pv PolarVector) AngleDiff(pv2 PolarVector) float64 {
	diff := math.Abs(pv.Angle - pv2.Angle)
	if diff > math.Pi {
		diff = 2*math.Pi - diff
	}
	return diff
}

func (pv PolarVector) Sector(sectorCount int) int {
	sector := math.Floor(float64(sectorCount/2) * pv.Angle / math.Pi)
	return (sectorCount/2 + int(sector)) % sectorCount
}

func (pv PolarVector) Direction() SectorDirection {
	sector := pv.Sector(8)
	switch sector {
	case 1, 2:
		return DirectionBottom
	case 3, 4:
		return DirectionRight
	case 5, 6:
		return DirectionTop
	case 7, 0:
		return DirectionLeft
	}

	panic(fmt.Sprintf("unexpected sector value %d", sector))
}
