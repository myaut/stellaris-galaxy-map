package sgmmath

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

func (rect *BoundingRect) Size() (float64, float64) {
	return rect.Max.X - rect.Min.X, rect.Max.Y - rect.Min.Y
}
