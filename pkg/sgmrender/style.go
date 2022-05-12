package sgmrender

import (
	"bytes"
)

type Style struct {
	m map[string]string
	s string
}

type StyleOption struct {
	Property string
	Value    string
}

func newStyle() Style {
	return Style{m: make(map[string]string)}
}

func NewStyle(opts ...StyleOption) Style {
	s := newStyle()
	for _, opt := range opts {
		s.m[opt.Property] = opt.Value
	}
	return s
}

func (s Style) With(opts ...StyleOption) Style {
	s2 := newStyle()
	for prop, value := range s.m {
		s2.m[prop] = value
	}
	for _, opt := range opts {
		s2.m[opt.Property] = opt.Value
	}
	return s2
}

func (s Style) String() string {
	if s.s == "" {
		buf := bytes.NewBuffer(nil)
		for prop, value := range s.m {
			buf.WriteString(prop)
			buf.WriteByte(':')
			buf.WriteString(value)
			buf.WriteByte(';')
		}
		s.s = buf.String()
	}

	return s.s
}
