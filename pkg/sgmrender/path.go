package sgmrender

import (
	"bytes"
	"fmt"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

const (
	fleetHalfSize = 2.0
	fleetStep     = 3.2

	starHalfSize     = 2.0
	outpostHalfSize  = 3.0
	starbaseHalfSize = 4.2
)

var (
	defaultStarPath = NewPath().
			MoveTo(-starHalfSize, 0.0).LineTo(0.0, starHalfSize).
			LineTo(starHalfSize, 0.0).LineTo(0.0, -starHalfSize).
			Complete()

	outpostPath      = newStarbasePath(outpostHalfSize)
	starbasePath     = newStarbasePath(starbaseHalfSize)
	citadelInnerPath = newStarbasePath(starbaseHalfSize / 2)

	fleetPath = NewPath().MoveTo(0.0, -fleetHalfSize).
			LineTo(-fleetHalfSize, -fleetHalfSize/3.0).
			LineTo(-fleetHalfSize/2.0, fleetHalfSize).HorLine(fleetHalfSize/2.0).
			LineTo(fleetHalfSize, -fleetHalfSize/3.0).Complete()
)

func newStarbasePath(size float64) Path {
	return NewPath().
		MoveTo(-size, 0.0).LineTo(-size/2, 5*size/6).HorLine(size/2).
		LineTo(size, 0.0).LineTo(size/2, -5*size/6).HorLine(-size / 2).
		Complete()
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
