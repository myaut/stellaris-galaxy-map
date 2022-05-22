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

func NewBoundingRect() BoundingRect {
	return BoundingRect{
		Min: Point{math.NaN(), math.NaN()},
		Max: Point{math.NaN(), math.NaN()},
	}
}

func NewPointBoundingRect(point Point) BoundingRect {
	return BoundingRect{Min: point, Max: point}
}

func (rect *BoundingRect) IsZero() bool {
	return math.IsNaN(rect.Min.X)
}

func (rect *BoundingRect) Add(point Point) {
	if rect.IsZero() {
		rect.Min = point
		rect.Max = point
		return
	}

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

func (rect *BoundingRect) Expand(padding float64) {
	rect.Min.X -= padding
	rect.Min.Y -= padding
	rect.Max.X += padding
	rect.Max.Y += padding
}
