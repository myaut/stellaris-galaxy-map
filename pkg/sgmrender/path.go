package sgmrender

import (
	"bytes"
	"fmt"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmmath"
)

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
