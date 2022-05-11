package sgmparser

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

type simpleState struct {
	Level   string         `sgm:"level"`
	Modules map[int]string `sgm:"modules"`
}

const simpleObject = `
level="starbase_level_starport"
modules={
	0=shipyard				1=trading_hub			}
`

func TestTokenizer(t *testing.T) {
	tokenizer := NewTokenizer(bytes.NewBufferString(simpleObject))
	err := tokenizer.Run()
	assert.NoError(t, err)

	var tokens []Token
	for token := range tokenizer.C {
		tokens = append(tokens, token)
	}
	assert.ElementsMatch(t, tokens, []Token{
		{TokenIdentifier, "level", 2},
		{TokenEqualSign, "=", 2},
		{TokenString, "starbase_level_starport", 2},
		{TokenIdentifier, "modules", 3},
		{TokenEqualSign, "=", 3},
		{TokenCollectionStart, "{", 3},
		{TokenNumber, "0", 4},
		{TokenEqualSign, "=", 4},
		{TokenIdentifier, "shipyard", 4},
		{TokenNumber, "1", 4},
		{TokenEqualSign, "=", 4},
		{TokenIdentifier, "trading_hub", 4},
		{TokenCollectionEnd, "}", 4},
	})
}

func TestParser(t *testing.T) {
	var tokenizerErr error
	tokenizer := NewTokenizer(bytes.NewBufferString(simpleObject))
	go func() {
		tokenizerErr = tokenizer.Run()
	}()

	var value simpleState
	parser := NewParser(tokenizer.C)
	parserErr := parser.Parse(&value)

	assert.NoError(t, tokenizerErr)
	assert.NoError(t, parserErr)

	assert.Equal(t, simpleState{
		Level: "starbase_level_starport",
		Modules: map[int]string{
			0: "shipyard",
			1: "trading_hub",
		},
	}, value)
}
