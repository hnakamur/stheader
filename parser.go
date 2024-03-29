package stheader

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"regexp"
	"strconv"
)

type ParseError struct {
	msg string
	pos int
}

func (e *ParseError) Error() string {
	return e.msg
}

func (e *ParseError) Pos() int {
	return e.pos
}

type Parser struct {
	input []byte
	pos   int
	debug bool
}

func NewParser(input string) *Parser {
	p := &Parser{input: []byte(input)}
	p.skipOWS()
	return p
}

func (p *Parser) ParseDictionary() (Dictionary, error) {
	dict, err := p.parseDictionary()
	if err != nil {
		return nil, err
	}
	if err := p.end(); err != nil {
		return nil, err
	}
	return dict, nil
}

func (p *Parser) ParseList() (List, error) {
	dict, err := p.parseList()
	if err != nil {
		return nil, err
	}
	if err := p.end(); err != nil {
		return nil, err
	}
	return dict, nil
}

func (p *Parser) ParseItem() (Item, error) {
	dict, err := p.parseItem()
	if err != nil {
		return nil, err
	}
	if err := p.end(); err != nil {
		return nil, err
	}
	return dict, nil
}

func (p *Parser) parseDictionary() (Dictionary, error) {
	output := &dictionary{}
	for !p.eol() {
		// Dictionary key
		key, err := p.parseKey()
		if i := output.index(key); i != -1 {
			return nil, &ParseError{
				msg: fmt.Sprintf("Duplicate key in dictionary: %s", key),
				pos: p.pos,
			}
		}

		// Equals sign
		err = p.matchByte('=')
		if err != nil {
			return nil, err
		}

		value, err := p.parseMember()
		if err != nil {
			return nil, err
		}
		output.Store(key, value)

		// Optional whitespace
		p.skipOWS()

		// Exit if at end of string
		if p.eol() {
			return output, nil
		}

		// Comma for separating values
		err = p.matchByte(',')
		if err != nil {
			return nil, err
		}
		// Optional whitespace
		p.skipOWS()

		if p.eol() {
			return nil, &ParseError{
				msg: "Unexpected end of string",
				pos: p.pos,
			}
		}
	}
	return output, nil
}

func (p *Parser) parseList() (List, error) {
	var output []Member
	for !p.eol() {
		member, err := p.parseMember()
		if err != nil {
			return nil, err
		}
		output = append(output, member)
		p.skipOWS()
		if p.eol() {
			break
		}
		err = p.matchByte(',')
		if err != nil {
			return nil, err
		}

		p.skipOWS()
		if p.eol() {
			return nil, &ParseError{
				msg: "Unexpected end of string. Was there a trailing comma?",
				pos: p.pos,
			}
		}
	}
	return output, nil
}

func (p *Parser) parseMember() (Member, error) {
	if p.debug {
		log.Printf("parseMember enter, rest=%s", string(p.input[p.pos:]))
		defer log.Printf("parseMember exit, rest=%s", string(p.input[p.pos:]))
	}
	var value interface{}
	b, err := p.peekByte()
	if err != nil {
		return nil, err
	}
	if b == '(' {
		value, err = p.parseInnerList()
		if err != nil {
			return nil, err
		}
	} else {
		value, err = p.parseItem()
		if err != nil {
			return nil, err
		}
	}

	return &member{
		val: value,
	}, nil
}

func (p *Parser) parseInnerList() (InnerList, error) {
	err := p.matchByte('(')
	if err != nil {
		return nil, err
	}
	var items []Item
	for !p.eol() {
		p.skipOWS()
		b, err := p.peekByte()
		if err != nil {
			return nil, err
		}
		if b == ')' {
			p.advance()
			break
		}
		item, err := p.parseItem()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		b, err = p.peekByte()
		if err != nil {
			return nil, err
		}
		if b != ' ' && b != ')' {
			return nil, &ParseError{
				msg: "Malformed list. Expected whitespace or )",
				pos: p.pos,
			}
		}
	}
	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}
	return &innerList{
		items:  items,
		params: params,
	}, nil
}

func (p *Parser) parseItem() (Item, error) {
	if p.debug {
		log.Printf("parseItem enter, rest=%s", string(p.input[p.pos:]))
		defer func() { log.Printf("parseItem exit, rest=%s", string(p.input[p.pos:])) }()
	}

	bi, err := p.parseBareItem()
	if err != nil {
		return nil, err
	}

	params, err := p.parseParameters()
	if err != nil {
		return nil, err
	}

	return &item{
		bareItem: bi,
		params:   params,
	}, nil
}

func (p *Parser) parseParameters() (Parameters, error) {
	if p.debug {
		log.Printf("parseParameters enter, rest=%s", string(p.input[p.pos:]))
		defer func() { log.Printf("parseParameters exit, rest=%s", string(p.input[p.pos:])) }()
	}

	params := &parameters{}
	for !p.eol() {
		b, err := p.peekByte()
		if err != nil {
			return nil, err
		}
		if b != ';' {
			break
		}
		p.advance()
		p.skipOWS()
		paramKey, err := p.parseKey()
		if err != nil {
			return nil, err
		}
		if i := params.index(paramKey); i != -1 {
			return nil, &ParseError{
				msg: fmt.Sprintf("Duplicate parameter key: %s", paramKey),
				pos: p.pos,
			}
		}
		var paramValue BareItem
		if !p.eol() {
			b, err = p.peekByte()
			if err != nil {
				return nil, err
			}
			if b == '=' {
				p.advance()
				paramValue, err = p.parseBareItem()
				if err != nil {
					return nil, err
				}
			}
		}
		params.Store(paramKey, paramValue)
	}
	return params, nil
}

func (p *Parser) parseBareItem() (BareItem, error) {
	if p.debug {
		log.Printf("parseBareItem enter, rest=%s", string(p.input[p.pos:]))
		defer func() { log.Printf("parseBareItem exit, rest=%s", string(p.input[p.pos:])) }()
	}

	b, err := p.peekByte()
	if err != nil {
		return nil, err
	}
	switch {
	case b == '"':
		v, err := p.parseString()
		if err != nil {
			return nil, err
		}
		return &bareItem{val: v}, nil
	case b == '*':
		v, err := p.parseByteSeq()
		if err != nil {
			return nil, err
		}
		return &bareItem{val: v}, nil
	case b == '?':
		v, err := p.parseBoolean()
		if err != nil {
			return nil, err
		}
		return &bareItem{val: v}, nil
	case ('0' <= b && b <= '9') || b == '-':
		v, err := p.parseNumber()
		if err != nil {
			return nil, err
		}
		return &bareItem{val: v}, nil
	case ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z'):
		v, err := p.parseToken()
		if err != nil {
			return nil, err
		}
		return &bareItem{val: v}, nil
	}
	return nil, &ParseError{
		msg: fmt.Sprintf("Unexpected character: %c on position %d", b, p.pos),
		pos: p.pos,
	}
}

func (p *Parser) parseString() (string, error) {
	var out []byte
	p.advance()
	for {
		b, err := p.getByte()
		if err != nil {
			return "", err
		}
		switch b {
		case '\\':
			b2, err := p.getByte()
			if err != nil {
				return "", err
			}
			if b2 != '"' && b2 != '\\' {
				return "", &ParseError{
					msg: fmt.Sprintf(`Expected a " or \ on position: %d`, p.pos-1),
					pos: p.pos - 1,
				}
			}
			out = append(out, b2)
		case '"':
			return string(out), nil
		default:
			if b < ' ' || b > '~' {
				return "", &ParseError{
					msg: "Character outside of ASCII range",
					pos: p.pos - 1,
				}
			}
			out = append(out, b)
		}
	}
}

var tokenRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-\.\:\%\*\/]*`)

func (p *Parser) parseToken() (Token, error) {
	m := tokenRegex.Find(p.input[p.pos:])
	if len(m) == 0 {
		return "", &ParseError{
			msg: fmt.Sprintf("Expected token identifier on position %d", p.pos),
			pos: p.pos,
		}
	}
	p.pos += len(m)
	return Token(m), nil
}

var keyRegex = regexp.MustCompile(`^[a-z][a-z0-9_\-\*]{0,254}`)

func (p *Parser) parseKey() (string, error) {
	if p.debug {
		log.Printf("parseKey enter, rest=%s", string(p.input[p.pos:]))
		defer func() { log.Printf("parseKey exit, rest=%s", string(p.input[p.pos:])) }()
	}

	m := keyRegex.Find(p.input[p.pos:])
	if len(m) == 0 {
		return "", &ParseError{
			msg: fmt.Sprintf("Expected key identifier on position %d", p.pos),
			pos: p.pos,
		}
	}
	p.pos += len(m)
	return string(m), nil
}

var byteSeqRegex = regexp.MustCompile(`^([A-Za-z0-9\\+\\/=]*)\*`)

func (p *Parser) parseByteSeq() ([]byte, error) {
	if err := p.matchByte('*'); err != nil {
		return nil, err
	}
	m := byteSeqRegex.FindSubmatch(p.input[p.pos:])
	if len(m) == 0 {
		return nil, &ParseError{
			msg: fmt.Sprintf("Couldn't parse byte sequence at position %d", p.pos),
			pos: p.pos,
		}
	}
	// encodedLen := len(m[1])
	// if encodedLen%4 != 0 {
	// 	return nil, &ParseError{
	// 		msg: fmt.Sprintf("Base64 strings should always have a length that's a multiple of 4. Did you forget padding at position %d?", p.pos),
	// 		pos: p.pos,
	// 	}
	// }
	p.pos += len(m[0])

	src := m[1]
	dst, err := p.decodeBase64(src, base64.StdEncoding)
	if err != nil {
		dst, err = p.decodeBase64(src, base64.RawStdEncoding)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func (p *Parser) decodeBase64(src []byte, enc *base64.Encoding) ([]byte, error) {
	encodedLen := len(src)
	dst := make([]byte, enc.DecodedLen(encodedLen))
	n, err := enc.Decode(dst, src)
	if err != nil {
		return nil, &ParseError{
			msg: fmt.Sprintf("Invalid base64 strings at position %d?", p.pos),
			pos: p.pos,
		}
	}
	return dst[:n], nil
}

func (p *Parser) parseBoolean() (bool, error) {
	if err := p.matchByte('?'); err != nil {
		return false, err
	}
	b, err := p.getByte()
	if err != nil {
		return false, err
	}
	switch b {
	case '0':
		return false, nil
	case '1':
		return true, nil
	default:
		return false, &ParseError{
			msg: `A "?" must be followed by "0" or "1"`,
			pos: p.pos - 1,
		}
	}
}

var numberPartRegex = regexp.MustCompile(`^[0-9-]([0-9])*(\.[0-9]{1,6})?`)

func (p *Parser) parseNumber() (interface{}, error) {
	m := numberPartRegex.Find(p.input[p.pos:])
	if len(m) == 0 {
		return nil, &ParseError{
			msg: fmt.Sprintf("Expected number on position %d", p.pos),
			pos: p.pos,
		}
	}
	p.pos += len(m)
	if bytes.IndexByte(m, '.') != -1 {
		v, err := strconv.ParseFloat(string(m), 64)
		if err != nil {
			return nil, &ParseError{
				msg: fmt.Sprintf("Expected float number on position %d", p.pos),
				pos: p.pos,
			}
		}
		return v, nil
	}
	if len(m) > 16 || (m[0] != '-' && len(m) > 15) {
		return nil, &ParseError{
			msg: "Integers must not have more than 15 digits",
			pos: p.pos,
		}
	}
	v, err := strconv.ParseInt(string(m), 10, 64)
	if err != nil {
		return nil, &ParseError{
			msg: fmt.Sprintf("Expected integer number on position %d", p.pos),
			pos: p.pos,
		}
	}
	return v, nil
}

func (p *Parser) matchByte(match byte) error {
	b, err := p.getByte()
	if err != nil {
		return err
	}
	if b != match {
		return &ParseError{
			msg: fmt.Sprintf("Expected %c on position %d", match, p.pos-1),
			pos: p.pos - 1,
		}
	}
	return nil
}

func (p *Parser) getByte() (byte, error) {
	b, err := p.peekByte()
	if err != nil {
		return 0, err
	}
	p.advance()
	return b, nil
}

func (p *Parser) peekByte() (byte, error) {
	if len(p.input[p.pos:]) == 0 {
		// panic("Unexpected end of string in peekByte")
		return 0, &ParseError{
			msg: "Unexpected end of string in peekByte",
			pos: p.pos,
		}
	}
	return p.input[p.pos], nil
}

func (p *Parser) advance() {
	p.pos++
}

func (p *Parser) end() error {
	p.skipOWS()
	if !p.eol() {
		return &ParseError{
			msg: "Expected end of the string, but found more data instead",
			pos: p.pos,
		}
	}
	return nil
}

func (p *Parser) skipOWS() {
	for len(p.input[p.pos:]) > 0 {
		b := p.input[p.pos]
		if b == ' ' || b == '\t' {
			p.advance()
		} else {
			break
		}
	}
}

func (p *Parser) eol() bool {
	return p.pos >= len(p.input)
}
