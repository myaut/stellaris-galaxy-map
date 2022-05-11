package sgmparser

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type ParserError struct {
	elemStack []string
	innerErr  error
}

func (pe *ParserError) Error() string {
	sort.Reverse(sort.StringSlice(pe.elemStack))
	return fmt.Sprintf("error parsing value %s: %s",
		strings.Join(pe.elemStack, " -> "), pe.innerErr)
}

func WrapParserError(elem string, innerErr error) error {
	if pe, ok := innerErr.(*ParserError); ok {
		pe.elemStack = append(pe.elemStack, elem)
		return pe
	}

	return &ParserError{[]string{elem}, innerErr}
}

type Parser struct {
	ch chan Token
}

func NewParser(ch chan Token) *Parser {
	return &Parser{
		ch: ch,
	}
}

func (p *Parser) Parse(v interface{}) error {
	return p.parseStruct(TokenTypeUndefined, reflect.ValueOf(v).Elem())
}

func (p *Parser) ignoreComplexValue() {
	for token := range p.ch {
		switch token.Type {
		case TokenCollectionStart:
			p.ignoreComplexValue()
		case TokenCollectionEnd:
			return
		}
	}
}

func (p *Parser) parseMap(v reflect.Value) error {
	v.Set(reflect.MakeMap(v.Type()))

	keyType, valueType := v.Type().Key(), v.Type().Elem()
	for token := range p.ch {
		if token.Type == TokenCollectionEnd {
			return nil
		}

		key := reflect.New(keyType).Elem()
		if err := p.parseValue(token, key); err != nil {
			return err
		}

		eqToken := <-p.ch
		if err := p.ensureToken(eqToken, TokenEqualSign); err != nil {
			return err
		}

		valueToken := <-p.ch
		value := reflect.New(valueType).Elem()
		if err := p.parseValue(valueToken, value); err != nil {
			return WrapParserError(token.Value, err)
		}

		v.SetMapIndex(key, value)
	}

	return nil
}

func (p *Parser) parseStruct(terminalTokenType TokenType, v reflect.Value) error {
	fields := make(map[string]reflect.Value)
	for fieldIdx := 0; fieldIdx < v.NumField(); fieldIdx++ {
		fieldDef := v.Type().Field(fieldIdx)
		if tag, ok := fieldDef.Tag.Lookup("sgm"); ok {
			fields[tag] = v.Field(fieldIdx)
		}
	}

	for token := range p.ch {
		if token.Type == terminalTokenType {
			return nil
		}
		if err := p.ensureToken(token, TokenIdentifier); err != nil {
			return err
		}

		eqToken := <-p.ch
		if err := p.ensureToken(eqToken, TokenEqualSign); err != nil {
			return err
		}

		valueToken := <-p.ch
		field, hasField := fields[token.Value]
		if !hasField {
			if valueToken.Type == TokenCollectionStart {
				p.ignoreComplexValue()
			}
			continue
		}

		err := p.parseValue(valueToken, field)
		if err != nil {
			return WrapParserError(token.Value, err)
		}
	}
	return nil
}

func (p *Parser) parseValue(valueToken Token, v reflect.Value) error {
	// Some slices are composed by specifying the same key multiple times
	var (
		slice   reflect.Value
		isSlice bool
	)
	if v.Kind() == reflect.Slice && valueToken.Type != TokenCollectionStart {
		slice, v = v, reflect.New(v.Type().Elem())
		isSlice = true
	}

	switch v.Kind() {
	case reflect.Int:
		if err := p.ensureToken(valueToken, TokenNumber); err != nil {
			return err
		}

		i, err := strconv.Atoi(valueToken.Value)
		if err != nil {
			return fmt.Errorf("error parsing number at line %d: %w",
				valueToken.LineNo, err)
		}

		v.SetInt(int64(i))
	case reflect.String:
		switch valueToken.Type {
		case TokenString, TokenIdentifier:
			v.SetString(valueToken.Value)
		default:
			return p.ensureToken(valueToken, TokenString)
		}
	case reflect.Map:
		return p.parseMap(v)
	case reflect.Slice:
		for elToken := range p.ch {
			if elToken.Type == TokenCollectionEnd {
				break
			}

			el := reflect.New(v.Type().Elem())
			err := p.parseValue(elToken, el)
			if err != nil {
				return err
			}
			v.Set(reflect.Append(v, el))
		}
	case reflect.Struct:
		if err := p.ensureToken(valueToken, TokenCollectionStart); err != nil {
			return err
		}

		return p.parseStruct(TokenCollectionEnd, v)
	default:
		return fmt.Errorf("unsupported value of kind %s", v.Kind())
	}

	if isSlice {
		slice.Set(reflect.Append(slice, v))
	}
	return nil
}

func (p *Parser) ensureToken(token Token, expectedType TokenType) error {
	if token.Type != expectedType {
		return fmt.Errorf("unexpected token: expected %s, got %s at line %d",
			expectedType, token.Type, token.LineNo)
	}
	return nil
}
