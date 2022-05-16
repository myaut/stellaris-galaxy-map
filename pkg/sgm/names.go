package sgm

import (
	_ "embed"
	"regexp"
	"strings"
)

// Generated using sed '/\*/"\1": \2,/'

var ymlLineRe = regexp.MustCompile(`([A-Za-z0-9_.\-]*):[0-9] "([^"]*)"`)

var DefaultNames = map[string]string{}

var (
	CountryNameFormats      map[string]string
	EmpireNames             map[string]string
	SpeciesNames            map[string]string
	PrescriptedCountryNames map[string]string
)

//go:embed names/species.yml
var speciesNamesText string

//go:embed names/prescripted_countries.yml
var prescriptedCountryNamesText string

//go:embed names/prescripted.yml
var empireNamesText string

//go:embed names/empire_formats.yml
var countryNameFormatsText string

type NameVariable struct {
	Key   string `sgm:"key"`
	Value Name   `sgm:"value"`
}

type Name struct {
	Key       string         `sgm:"key"`
	Variables []NameVariable `sgm:"variables"`
}

func (n *Name) Format(formatNames map[string]string) string {
	format, ok := formatNames[n.Key]
	if !ok {
		return n.Resolve()
	}

	for _, v := range n.Variables {
		// TODO: deduce name map
		sub := v.Value.Format(DefaultNames)
		format = strings.ReplaceAll(format, "<"+v.Key+">", sub)
		format = strings.ReplaceAll(format, "["+v.Key+"]", sub)
	}

	return format
}

func (n *Name) Resolve() string {
	var names map[string]string
	switch {
	case strings.HasPrefix(n.Key, "EMPIRE_DESIGN_"):
		names = EmpireNames
	case strings.HasPrefix(n.Key, "SPEC_"):
		names = SpeciesNames
	case strings.HasPrefix(n.Key, "PRESCRIPTED_"):
		names = PrescriptedCountryNames
	case strings.HasPrefix(n.Key, "NAME_"):
		// NAME is too big and ambigous, simply use identifier
		return strings.ReplaceAll(n.Key[5:], "_", " ")
	}

	if names != nil {
		if name, ok := names[n.Key]; ok {
			return name
		}
	}
	return n.Key
}

func loadStellarisYml(text string) map[string]string {
	lines := ymlLineRe.FindAllStringSubmatch(text, -1)

	m := make(map[string]string)
	for _, line := range lines {
		m[line[1]] = line[2]
	}
	return m
}

func init() {
	CountryNameFormats = loadStellarisYml(countryNameFormatsText)
	EmpireNames = loadStellarisYml(empireNamesText)
	SpeciesNames = loadStellarisYml(speciesNamesText)
	PrescriptedCountryNames = loadStellarisYml(prescriptedCountryNamesText)
}
