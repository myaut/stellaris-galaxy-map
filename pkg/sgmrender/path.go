package sgmrender

import (
	"bytes"
	"fmt"

	"github.com/myaut/stellaris-galaxy-map/pkg/sgmmath"
)

const (
	fleetHalfSize = 1.8
	fleetStep     = 3.0

	starHalfSize     = 2.0
	outpostHalfSize  = 3.0
	starbaseHalfSize = 4.0

	countryPatternSize = 8.0
	countryPatternStep = 2.0
)

var (
	defaultStarPath = newDiamondPath(starHalfSize)

	outpostPath      = newStarbasePath(outpostHalfSize)
	starbasePath     = newStarbasePath(starbaseHalfSize)
	citadelInnerPath = newStarbasePath(2 * starbaseHalfSize / 3)
)

func newDiamondPath(size float64) Path {
	return NewPath().
		MoveTo(-size, 0.0).LineTo(0.0, size).
		LineTo(size, 0.0).LineTo(0.0, -size).
		Complete()
}

func newStarbasePath(size float64) Path {
	return NewPath().
		MoveTo(-size, 0.0).LineTo(-size/2, 5*size/6).HorLine(size/2).
		LineTo(size, 0.0).LineTo(size/2, -5*size/6).HorLine(-size / 2).
		Complete()
}

func newFleetPath(size, off float64) Path {
	return NewPath().MoveTo(0.0, -size+off).
		LineTo(-size+off, -size/3.0+off).
		LineTo(-size/2.0+off, size-off).HorLine(size/2.0-off).
		LineTo(size-off, -size/3.0+off).Complete()
}

type PathElement struct {
	Command rune
	X, Y    float64
}

type Path struct {
	path []PathElement
	s    string
}

func NewPath() Path {
	return Path{path: make([]PathElement, 0, 8)}
}

func NewVectorPath(vec sgmmath.Vector) Path {
	return NewPath().MoveToPoint(vec.Begin).LineToPoint(vec.End)
}

func (p Path) MoveTo(x, y float64) Path {
	return Path{path: append(p.path, PathElement{Command: 'M', X: x, Y: y})}
}

func (p Path) MoveToPoint(point sgmmath.Point) Path {
	return p.MoveTo(point.X, point.Y)
}

func (p Path) LineTo(x, y float64) Path {
	return Path{path: append(p.path, PathElement{Command: 'L', X: x, Y: y})}
}

func (p Path) LineToPoint(point sgmmath.Point) Path {
	return p.LineTo(point.X, point.Y)
}

func (p Path) HorLine(x float64) Path {
	return Path{path: append(p.path, PathElement{Command: 'H', X: x})}
}

func (p Path) VertLine(y float64) Path {
	return Path{path: append(p.path, PathElement{Command: 'V', Y: y})}
}

func (p Path) Complete() Path {
	return Path{path: append(p.path, PathElement{Command: 'Z'})}
}

func (p *Path) Translate(point sgmmath.Point) {
	for i := range p.path {
		p.path[i].X += point.X
		p.path[i].Y += point.Y
	}
}

func (p *Path) String() string {
	if p.s == "" {
		buf := bytes.NewBuffer(nil)
		for i, el := range p.path {
			if i > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteRune(el.Command)
			buf.WriteByte(' ')

			switch el.Command {
			case 'M', 'L':
				fmt.Fprintf(buf, "%f %f", el.X, el.Y)
			case 'H':
				fmt.Fprintf(buf, "%f", el.X)
			case 'V':
				fmt.Fprintf(buf, "%f", el.Y)
			case 'Z':
				// Z has no options
			}
		}
		p.s = buf.String()
	}
	return p.s
}
