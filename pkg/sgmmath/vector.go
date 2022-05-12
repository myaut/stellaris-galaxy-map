package sgmmath

import (
	"math"
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
