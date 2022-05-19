package sgmparser

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

type ParserError struct {
	elemStack []string
	innerErr  error
}

func (pe *ParserError) Error() string {
	return fmt.Sprintf("error parsing value %s: %s",
		strings.Join(pe.elemStack, " -> "), pe.innerErr)
}

func WrapParserError(elem string, innerErr error) error {
	if pe, ok := innerErr.(*ParserError); ok {
		pe.elemStack = append([]string{elem}, pe.elemStack...)
		return pe
	}

	return &ParserError{[]string{elem}, innerErr}
}

type Parser struct {
	tokenizer *Tokenizer
}

func NewParser(tokenizer *Tokenizer) *Parser {
	return &Parser{
		tokenizer: tokenizer,
	}
}

func (p *Parser) Parse(v interface{}) error {
	var tokenizerErr error
	go func() {
		tokenizerErr = p.tokenizer.Run()
	}()

	err := p.parseStruct(TokenTypeUndefined, reflect.ValueOf(v).Elem())
	if tokenizerErr != nil {
		return tokenizerErr
	}
	return err
}

func (p *Parser) ignoreComplexValue() {
	for token := range p.tokenizer.C {
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
	for token := range p.tokenizer.C {
		if token.Type == TokenCollectionEnd {
			return nil
		}

		key := reflect.New(keyType).Elem()
		if err := p.parseValue(token, key); err != nil {
			return err
		}

		eqToken := <-p.tokenizer.C
		if err := p.ensureToken(eqToken, TokenEqualSign); err != nil {
			return err
		}

		valueToken := <-p.tokenizer.C

		var (
			value    reflect.Value
			valueErr error
		)
		if valueType.Kind() == reflect.Ptr {
			if valueToken.Type == TokenIdentifier && valueToken.Value == TokenNone {
				v.SetMapIndex(key, reflect.Zero(valueType))
				continue
			}

			value = reflect.New(valueType.Elem())
			valueErr = p.parseValue(valueToken, value.Elem())
		} else {
			value = reflect.New(valueType).Elem()
			valueErr = p.parseValue(valueToken, value)
		}

		if valueErr != nil {
			return WrapParserError(token.Value, valueErr)
		}
		v.SetMapIndex(key, value)
	}

	return nil
}

func (p *Parser) prepareFields(v reflect.Value) map[string]reflect.Value {
	fields := make(map[string]reflect.Value)
	for fieldIdx := 0; fieldIdx < v.NumField(); fieldIdx++ {
		fieldDef := v.Type().Field(fieldIdx)
		if tag, ok := fieldDef.Tag.Lookup("sgm"); ok {
			field := v.Field(fieldIdx)
			if strings.HasSuffix(tag, ",id") {
				tag = tag[:len(tag)-3]
				field.SetUint(math.MaxUint32)
			}
			if tag == "-" {
				continue
			}

			fields[tag] = field
		}
	}
	return fields
}

func (p *Parser) parseStruct(terminalTokenType TokenType, v reflect.Value) error {
	fields := p.prepareFields(v)
	for token := range p.tokenizer.C {
		if token.Type == terminalTokenType {
			return nil
		}
		if err := p.ensureToken(token, TokenIdentifier); err != nil {
			return err
		}

		eqToken := <-p.tokenizer.C
		if err := p.ensureToken(eqToken, TokenEqualSign); err != nil {
			return err
		}

		valueToken := <-p.tokenizer.C
		field, hasField := fields[token.Value]
		if !hasField {
			if valueToken.Type == TokenCollectionStart {
				p.ignoreComplexValue()
			}
			continue
		}

		if valueToken.Type == TokenCollectionStart && field.Kind() != reflect.Struct {
			if structField, hasStruct := fields[token.Value+",struct"]; hasStruct {
				field = structField
			}
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
		slice, v = v, reflect.New(v.Type().Elem()).Elem()
		isSlice = true
	}

	switch v.Kind() {
	case reflect.Bool:
		if err := p.ensureToken(valueToken, TokenIdentifier); err != nil {
			return err
		}
		switch valueToken.Value {
		case "yes":
			v.SetBool(true)
		case "no":
			v.SetBool(false)
		default:
			return fmt.Errorf("unexpected boolean '%s' at line %d", valueToken.Value, valueToken.LineNo)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if err := p.ensureToken(valueToken, TokenNumber); err != nil {
			return err
		}

		i, err := strconv.ParseInt(valueToken.Value, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("error parsing number at line %d: %w", valueToken.LineNo, err)
		}

		v.SetInt(int64(i))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if err := p.ensureToken(valueToken, TokenNumber); err != nil {
			return err
		}

		u, err := strconv.ParseUint(valueToken.Value, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("error parsing number at line %d: %w", valueToken.LineNo, err)
		}

		v.SetUint(uint64(u))
	case reflect.Float64:
		if err := p.ensureToken(valueToken, TokenNumber); err != nil {
			return err
		}

		f, err := strconv.ParseFloat(valueToken.Value, 64)
		if err != nil {
			return fmt.Errorf("error parsing number at line %d: %w", valueToken.LineNo, err)
		}

		v.SetFloat(f)
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
		for elToken := range p.tokenizer.C {
			if elToken.Type == TokenCollectionEnd {
				break
			}

			el := reflect.New(v.Type().Elem()).Elem()
			err := p.parseValue(elToken, el)
			if err != nil {
				return err
			}
			v.Set(reflect.Append(v, el))
		}
	case reflect.Struct:
		if valueToken.Type == TokenIdentifier {
			fields := p.prepareFields(v)
			if field, fieldOk := fields[valueToken.Value]; fieldOk {
				nextToken := <-p.tokenizer.C
				return p.parseValue(nextToken, field)
			}
		}

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
		return fmt.Errorf("unexpected token: expected %s, got %s ('%s') at line %d",
			expectedType, token.Type, token.Value, token.LineNo)
	}
	return nil
}
