package sgmparser

import (
	"bufio"
	"fmt"
	"io"
	"unicode"
)

type TokenType int

const TokenNone = "none"

const (
	TokenTypeUndefined TokenType = iota
	TokenIdentifier
	TokenNumber
	TokenString
	TokenCollectionStart
	TokenCollectionEnd
	TokenEqualSign
)

var TokenTypeStr = []string{
	"UNDEF",
	"IDENT",
	"NUMBER",
	"STRING",
	"CSTART",
	"CEND",
	"EQUAL",
}

func (tokenType TokenType) String() string {
	return TokenTypeStr[tokenType]
}

type Token struct {
	Type   TokenType
	Value  string
	LineNo int
}

type Tokenizer struct {
	in     *bufio.Reader
	lineNo int

	curType TokenType
	curBuf  []byte
	C       chan Token
}

func NewTokenizer(in io.Reader) *Tokenizer {
	return &Tokenizer{
		in:     bufio.NewReader(in),
		lineNo: 1,

		C: make(chan Token, 100),
	}
}

func (t *Tokenizer) Run() error {
	defer close(t.C)

	for {
		b, err := t.in.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		// println(string([]byte{b}), t.curType, string(t.curBuf))

		switch {
		case b == '"':
			err = t.readString()
		case b == '=':
			t.flush()
			t.writeToken(TokenEqualSign, "=")
		case b == '{':
			t.flush()
			t.writeToken(TokenCollectionStart, "{")
		case b == '}':
			t.flush()
			t.writeToken(TokenCollectionEnd, "}")
		case unicode.IsSpace(rune(b)):
			if b == '\n' {
				t.lineNo++
			}
			t.flush()
		case unicode.IsDigit(rune(b)) || b == '.':
			if t.curType != TokenIdentifier {
				err = t.setType(TokenNumber)
			}
			t.curBuf = append(t.curBuf, b)
		case b == '-':
			err = t.setType(TokenNumber)
			t.curBuf = append(t.curBuf, b)
		default:
			if t.curType == TokenNumber {
				t.curType = TokenIdentifier
			} else {
				err = t.setType(TokenIdentifier)
			}
			t.curBuf = append(t.curBuf, b)
		}
		if err != nil {
			return fmt.Errorf("error parsing game state: %w at line %d", err, t.lineNo)
		}
	}

	t.flush()
	return nil
}

func (t *Tokenizer) readString() (err error) {
	// TODO: process escaped quotes
	err = t.setType(TokenString)
	if err != nil {
		return err
	}

	t.curBuf, err = t.in.ReadBytes('"')
	if err != nil {
		return err
	}

	if l := len(t.curBuf); l > 0 {
		t.curBuf = t.curBuf[:l-1]
	}
	t.flush()
	return nil
}

func (t *Tokenizer) setType(tokenType TokenType) error {
	if t.curType == tokenType {
		return nil
	}
	if t.curType == TokenTypeUndefined {
		t.curType = tokenType
		return nil
	}

	return fmt.Errorf("unexpected character: cannot transition from state %s to %s",
		t.curType, tokenType)
}

func (t *Tokenizer) writeToken(tokenType TokenType, value string) {
	t.C <- Token{
		Type:   tokenType,
		Value:  value,
		LineNo: t.lineNo,
	}
}

func (t *Tokenizer) flush() {
	if t.curType == TokenTypeUndefined {
		return
	}

	t.writeToken(t.curType, string(t.curBuf))

	t.curType = TokenTypeUndefined
	t.curBuf = make([]byte, 0, 32)
}
