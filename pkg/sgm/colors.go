package sgm

import (
	"bytes"
	_ "embed"

	"github.com/myaut/stellaris-galaxy-mod/pkg/sgmparser"

	"github.com/YashdalfTheGray/colorcode"
)

type Color struct {
	HSV []float64 `sgm:"hsv"`
	RGB []uint8   `sgm:"rgb"`
}

type ColorDef struct {
	Flag Color `sgm:"flag"`
	Map  Color `sgm:"map"`
	Ship Color `sgm:"ship"`
}

func (c *Color) Color() colorcode.RGB {
	if c.RGB != nil {
		return colorcode.NewRGB(c.RGB[0], c.RGB[1], c.RGB[2])
	}

	hsv, _ := colorcode.NewHSV(c.HSV[0]*360.0, c.HSV[1]*100.0, c.HSV[2]*100.0)
	return hsv.ToRGB()
}

//go:embed colors.txt
var colorText []byte

var ColorMap struct {
	Colors map[string]*ColorDef `sgm:"colors"`
}

func init() {
	parser := sgmparser.NewParser(sgmparser.NewTokenizer(bytes.NewBuffer(colorText)))
	err := parser.Parse(&ColorMap)
	if err != nil {
		panic(err.Error())
	}
}
