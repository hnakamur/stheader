package stheader_test

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"gihtub.com/hnakamur/stheader"
)

type httpwgTest struct {
	Name       string      `json:"name"`
	Raw        []string    `json:"raw"`
	Expected   interface{} `json:"expected"`
	HeaderType string      `json:"header_type"`
	MustFail   bool        `json:"must_fail"`
	CanFail    bool        `json:"can_fail"`
	Canonical  []string    `json:"canonical"`
}

type httpwgTestGroup []httpwgTest

func TestParseHTTPWG(t *testing.T) {
	groupNames := []string{
		"binary",
		"boolean",
		"number",
		"string",
		"token",

		"item",

		"list",
		"listlist",
		"dictionary",
		"param-list",

		"key-generated",
		"large-generated",
		"string-generated",
		"token-generated",
	}
	for _, groupName := range groupNames {
		filename := fmt.Sprintf("structured-header-tests/%s.json", groupName)
		group, err := readHTTPWGTestGroupFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range group {
			subTestName := fmt.Sprintf("%s_%s", groupName, test.Name)
			t.Run(subTestName, func(t *testing.T) {
				parser := stheader.NewParser(strings.Join(test.Raw, ","))
				var hadError bool
				var caughtErr error
				var result interface{}
				switch test.HeaderType {
				case "item":
					item, err := parser.ParseItem()
					if err != nil {
						hadError = true
						caughtErr = err
					} else {
						result = convertItemToExpected(item)
					}
				case "list":
					list, err := parser.ParseList()
					if err != nil {
						hadError = true
						caughtErr = err
					} else {
						result = convertListToExpected(list)
					}
				case "dictionary":
					dict, err := parser.ParseDictionary()
					if err != nil {
						hadError = true
						caughtErr = err
					} else {
						result = convertDictionaryToExpected(dict)
					}
				default:
					t.Fatalf("Unsupported header type: %s", test.HeaderType)
				}

				if test.MustFail {
					if !hadError {
						t.Errorf("unmatch MustFail, got=%v, want=%v", hadError, test.MustFail)
					}
					return
				}
				if hadError {
					if !test.CanFail {
						t.Errorf("should not have failed, but got error=%s", caughtErr)
						return
					}
				}
				if got, want := result, fixEmptyListExpected(test.Expected); !reflect.DeepEqual(got, want) {
					t.Errorf("unmatch result, got=%+v, want=%+v",
						got, want)
				}
			})
		}
	}
}

func TestSerializeHTTPWG(t *testing.T) {
	groupNames := []string{
		"binary",
		"boolean",
		"number",
		"string",
		"token",

		"item",

		"list",
		"listlist",
		"dictionary",
		"param-list",

		"key-generated",
		"large-generated",
		"string-generated",
		"token-generated",
	}
	debug := false
	if debug {
		groupNames = []string{"error"}
	}
	for _, groupName := range groupNames {
		filename := fmt.Sprintf("structured-header-tests/%s.json", groupName)
		group, err := readHTTPWGTestGroupFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range group {
			if test.MustFail {
				continue
			}
			subTestName := fmt.Sprintf("%s_%s", groupName, test.Name)
			t.Run(subTestName, func(t *testing.T) {
				parser := stheader.NewParser(strings.Join(test.Raw, ","))
				var got string
				switch test.HeaderType {
				case "item":
					item, err := parser.ParseItem()
					if err != nil {
						t.Fatal("parse error")
					}
					got, err = stheader.Serialize(item)
					if err != nil {
						t.Fatalf("serialize: %s", err)
					}
				case "list":
					list, err := parser.ParseList()
					if err != nil {
						t.Fatal("parse error")
					}
					got, err = stheader.Serialize(list)
					if err != nil {
						t.Fatalf("serialize: %s", err)
					}
				case "dictionary":
					dict, err := parser.ParseDictionary()
					if err != nil {
						t.Fatal("parse error")
					}
					got, err = stheader.Serialize(dict)
					if err != nil {
						t.Fatalf("serialize: %s", err)
					}
				default:
					t.Fatalf("Unsupported header type: %s", test.HeaderType)
				}
				var want string
				if len(test.Canonical) > 0 {
					want = test.Canonical[0]
				} else {
					want = test.Raw[0]
				}
				if got != want {
					t.Errorf("Unmatch, got=%q, want=%q", got, want)
				}
			})
		}
	}
}

func convertBareItemToExpected(bi stheader.BareItem) interface{} {
	switch bi.Type() {
	case stheader.ItemTypeBool:
		return bi.AsBool()
	case stheader.ItemTypeString:
		return bi.AsString()
	case stheader.ItemTypeByteSeq:
		return base32.StdEncoding.EncodeToString(bi.AsByteSeq())
	case stheader.ItemTypeInt:
		return float64(bi.AsInt())
	case stheader.ItemTypeFloat:
		return bi.AsFloat()
	case stheader.ItemTypeToken:
		return string(bi.AsToken())
	default:
		panic("invalid BareItem type")
	}
}

func convertItemToExpected(item stheader.Item) []interface{} {
	return []interface{}{
		convertBareItemToExpected(item.BareItem()),
		convertParametersToExpected(item.Parameters()),
	}
}

func convertDictionaryToExpected(dict stheader.Dictionary) interface{} {
	ret := make(map[string]interface{})
	dict.Range(func(key string, val stheader.Member) bool {
		ret[key] = convertMemberToExpected(val)
		return true
	})
	return ret
}

func convertListToExpected(list stheader.List) interface{} {
	var ret []interface{}
	for _, li := range list {
		ret = append(ret, convertMemberToExpected(li))
	}
	return ret
}

func convertMemberToExpected(li stheader.Member) interface{} {
	switch li.Type() {
	case stheader.MemberTypeItem:
		return convertItemToExpected(li.AsItem())
	case stheader.MemberTypeInnerList:
		return convertInnerListToExpected(li.AsInnerList())
	default:
		panic("invalid Member type")
	}
}

func convertInnerListToExpected(list stheader.InnerList) interface{} {
	var ret []interface{}
	var items []interface{}
	for _, item := range list.Items() {
		items = append(items, convertItemToExpected(item))
	}
	return append(ret, items, convertParametersToExpected(list.Parameters()))
}

func convertParametersToExpected(params stheader.Parameters) interface{} {
	ret := make(map[string]interface{})
	params.Range(func(key string, val stheader.BareItem) bool {
		switch v := val.(type) {
		case stheader.BareItem:
			ret[key] = convertBareItemToExpected(v)
		case nil:
			ret[key] = nil
		default:
			panic("invalid param value type")
		}
		return true
	})
	return ret
}

func readHTTPWGTestGroupFile(filename string) (httpwgTestGroup, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readHTTPWGTestGroup(file)
}

func readHTTPWGTestGroup(r io.Reader) (httpwgTestGroup, error) {
	dec := json.NewDecoder(r)
	g := httpwgTestGroup{}
	if err := dec.Decode(&g); err != nil {
		return nil, err
	}
	return g, nil
}

func fixEmptyListExpected(expected interface{}) interface{} {
	if arr, ok := expected.([]interface{}); ok {
		if len(arr) == 0 {
			var nilarr []interface{}
			return nilarr
		}
		for i := range arr {
			arr[i] = fixEmptyListExpected(arr[i])
		}
		return arr
	}
	if dict, ok := expected.(map[string]interface{}); ok {
		for k, v := range dict {
			dict[k] = fixEmptyListExpected(v)
		}
		return dict
	}
	return expected
}
