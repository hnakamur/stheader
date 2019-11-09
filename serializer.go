package stheader

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
)

// Serialize return an ASCII string suitable for use in a HTTP header value.
// It panics if value is neither Dictionary, List nor Item.
func Serialize(value interface{}) (string, error) {
	switch v := value.(type) {
	case Dictionary:
		return serializeDictionary(v)
	case List:
		return serializeList(v)
	case Item:
		return serializeItem(v)
	default:
		panic("invalid value type")
	}
}

func serializeDictionary(dict Dictionary) (string, error) {
	var b []byte
	b, err := appendDictionary(b, dict)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func serializeList(list List) (string, error) {
	var b []byte
	b, err := appendList(b, list)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func serializeItem(item Item) (string, error) {
	var b []byte
	b, err := appendItem(b, item)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func appendDictionary(b []byte, dict Dictionary) ([]byte, error) {
	if dict == nil || dict.Len() == 0 {
		return b, nil
	}
	var err error
	i := -1
	dict.Range(func(name string, val Member) bool {
		i++
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = appendKey(b, name)
		if err != nil {
			return false
		}
		b = append(b, '=')
		b, err = appendMember(b, val)
		if err != nil {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

func appendMember(b []byte, m Member) ([]byte, error) {
	var err error
	switch m.Type() {
	case MemberTypeInnerList:
		b, err = appendInnerList(b, m.AsInnerList())
		if err != nil {
			return nil, err
		}
	case MemberTypeItem:
		b, err = appendItem(b, m.AsItem())
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func appendList(b []byte, list List) ([]byte, error) {
	var err error
	for i, m := range []Member(list) {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = appendMember(b, m)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func appendInnerList(b []byte, list InnerList) ([]byte, error) {
	b = append(b, '(')
	var err error
	for i, it := range list.Items() {
		if i > 0 {
			b = append(b, ' ')
		}
		b, err = appendItem(b, it)
		if err != nil {
			return nil, err
		}
	}
	b = append(b, ')')
	b, err = appendParameters(b, list.Parameters())
	if err != nil {
		return nil, err
	}
	return b, nil
}

func appendItem(b []byte, item Item) ([]byte, error) {
	b, err := appendBareItem(b, item.BareItem())
	if err != nil {
		return nil, err
	}

	b, err = appendParameters(b, item.Parameters())
	if err != nil {
		return nil, err
	}

	return b, err
}

func appendParameters(b []byte, params Parameters) ([]byte, error) {
	if params == nil || params.Len() == 0 {
		return b, nil
	}
	var err error
	params.Range(func(name string, val BareItem) bool {
		b = append(b, ';')
		b, err = appendKey(b, name)
		if err != nil {
			return false
		}
		if val != nil {
			b = append(b, '=')
			b, err = appendBareItem(b, val)
			if err != nil {
				return false
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

func appendBareItem(b []byte, bi BareItem) ([]byte, error) {
	switch bi.Type() {
	case ItemTypeString:
		return appendBareItemString(b, bi.AsString())
	case ItemTypeByteSeq:
		return appendBareItemByteSeq(b, bi.AsByteSeq())
	case ItemTypeBool:
		return appendBareItemBool(b, bi.AsBool())
	case ItemTypeInt:
		return appendBareItemInt(b, bi.AsInt())
	case ItemTypeFloat:
		return appendBareItemFloat(b, bi.AsFloat())
	case ItemTypeToken:
		return appendBareItemToken(b, bi.AsToken())
	}
	panic("invalid item type")
}

func appendBareItemInt(b []byte, v int64) ([]byte, error) {
	if v < -999_999_999_999_999 || 999_999_999_999_999 < v {
		return nil, errors.New("Integers may not be larger than 15 digits")
	}
	return strconv.AppendInt(b, v, 10), nil
}

func appendBareItemFloat(b []byte, v float64) ([]byte, error) {
	formatted := strconv.FormatFloat(v, 'f', -1, 64)
	parts := strings.Split(formatted, ".")
	if len(parts[0]) > 15 || (v > 0 && len(parts[0]) > 14) {
		return nil, errors.New("When serializing floats, the integer part may not be larger than 14 digits")
	}
	b = append(b, parts[0]...)
	b = append(b, '.')
	if len(parts) <= 1 {
		b = append(b, '0')
	} else {
		fracLen := len(parts[1])
		if fracLen > 15-len(parts[0]) {
			fracLen = 15 - len(parts[0])
		}
		b = append(b, parts[1][:fracLen]...)
	}
	return b, nil
}

func appendBareItemString(b []byte, val string) ([]byte, error) {
	b = append(b, '"')
	for _, c := range []byte(val) {
		if c < ' ' || c > '~' {
			return nil, errors.New("invalid character in string")
		}
		if c == '\\' || c == '"' {
			b = append(b, '\\')
		}
		b = append(b, c)
	}
	b = append(b, '"')
	return b, nil
}

func appendBareItemToken(b []byte, token Token) ([]byte, error) {
	m := tokenRegex.FindStringIndex(string(token))
	if len(m) == 0 || m[1] != len(string(token)) {
		return nil, errors.New("invalid token value")
	}
	return append(b, token...), nil
}

func appendBareItemByteSeq(b []byte, data []byte) ([]byte, error) {
	b = append(b, '*')
	b = append(b, base64.StdEncoding.EncodeToString(data)...)
	b = append(b, '*')
	return b, nil
}

func appendBareItemBool(b []byte, v bool) ([]byte, error) {
	b = append(b, '?')
	if v {
		b = append(b, '1')
	} else {
		b = append(b, '0')
	}
	return b, nil
}

func appendKey(b []byte, key string) ([]byte, error) {
	m := keyRegex.FindStringIndex(key)
	if len(m) == 0 || m[1] != len(key) {
		return nil, errors.New("keys must start with a-z and only contain a-z0-9_-")
	}
	return append(b, key...), nil
}
