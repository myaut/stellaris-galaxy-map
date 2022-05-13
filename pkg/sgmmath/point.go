package sgmmath

import (
	"math"
)

type Point struct {
	X, Y float64
}

func (p Point) Point() Point {
	return p
}

func (p Point) Add(diff Point) Point {
	return Point{X: p.X + diff.X, Y: p.Y + diff.Y}
}

func (p Point) Distance(other Point) float64 {
	v := Vector{p, other}
	return v.ToPolar().Length
}

func MeanPoint(p1 Point, points ...Point) Point {
	for _, p := range points {
		p1 = p1.Add(p)
	}
	p1.X /= float64(len(points) + 1)
	p1.Y /= float64(len(points) + 1)
	return p1
}

type BoundingRect struct {
	Min Point
	Max Point
}

func NewBoundingRect(point Point) BoundingRect {
	return BoundingRect{point, point}
}

func (rect *BoundingRect) Add(point Point) {
	if rect.Min.X > point.X {
		rect.Min.X = point.X
	}
	if rect.Min.Y > point.Y {
		rect.Min.Y = point.Y
	}
	if rect.Max.X < point.X {
		rect.Max.X = point.X
	}
	if rect.Max.Y < point.Y {
		rect.Max.Y = point.Y
	}
}

func (rect *BoundingRect) Sub(point Point) {
	center := rect.Center()
	if point.X < center.X && rect.Min.X < point.X {
		rect.Min.X = point.X
	}
	if point.Y < center.Y && rect.Min.Y < point.Y {
		rect.Min.Y = point.Y
	}
	if point.X > center.X && rect.Max.X > point.X {
		rect.Max.X = point.X
	}
	if point.Y > center.Y && rect.Max.Y > point.Y {
		rect.Max.Y = point.Y
	}
}

func (rect *BoundingRect) Center() Point {
	return Point{
		X: (rect.Min.X + rect.Max.X) / 2,
		Y: (rect.Min.Y + rect.Max.Y) / 2,
	}
}

func (rect *BoundingRect) Includes(point Point) bool {
	return (rect.Min.X <= point.X && rect.Min.Y <= point.Y &&
		rect.Max.X >= point.X && rect.Max.Y >= point.Y)
}

func (rect *BoundingRect) Contains(other BoundingRect) bool {
	return (rect.Min.X <= other.Min.X && rect.Min.Y <= other.Min.Y &&
		rect.Max.X >= other.Max.X && rect.Max.Y >= other.Max.Y)
}

func (rect *BoundingRect) Size() (float64, float64) {
	return math.Abs(rect.Max.X - rect.Min.X), math.Abs(rect.Max.Y - rect.Min.Y)
}
