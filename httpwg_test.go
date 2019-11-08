package stheader

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
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

func TestHTTPWG(t *testing.T) {
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
		// "token-semicolon",
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
				parser := NewParser(strings.Join(test.Raw, ","))
				var hadError bool
				var caughtErr error
				var result interface{}
				switch test.HeaderType {
				case "item":
					item, err := parser.parseItem()
					if err != nil {
						hadError = true
						caughtErr = err
					} else {
						result = convertItemToExpected(item)
					}
				case "list":
					list, err := parser.parseList()
					if err != nil {
						hadError = true
						caughtErr = err
					} else {
						result = convertListToExpected(list)
					}
				case "dictionary":
					dict, err := parser.parseDictionary()
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
					t.Errorf("unmatch result, got=%+v (%T), want=%+v (%T)",
						got, got, want, want)
					got2, gotOK := got.([]interface{})
					want2, wantOK := want.([]interface{})
					if gotOK && wantOK {
						t.Logf("both []interface{}, got=%p, len(got)=%d, cap(got)=%d, want=%p, len(want)=%d, cap(want)=%d", got2, len(got2), cap(got2), want2, len(want2), cap(want2))
					}
				}
			})
		}
	}
}

func convertItemToExpected(item Item) []interface{} {
	var ret []interface{}
	switch item.Type() {
	case ItemTypeBool:
		return append(ret, item.AsBool(),
			convertParametersToExpected(item.Parameters()))
	case ItemTypeString:
		return append(ret, item.AsString(),
			convertParametersToExpected(item.Parameters()))
	case ItemTypeByteSeq:
		return append(ret, base32.StdEncoding.EncodeToString(item.AsByteSeq()),
			convertParametersToExpected(item.Parameters()))
	case ItemTypeInt:
		return append(ret, float64(item.AsInt()),
			convertParametersToExpected(item.Parameters()))
	case ItemTypeFloat:
		return append(ret, item.AsFloat(),
			convertParametersToExpected(item.Parameters()))
	case ItemTypeToken:
		return append(ret, string(item.AsToken()),
			convertParametersToExpected(item.Parameters()))
	default:
		panic("invalid Item type")
	}
}

func convertDictionaryToExpected(dict Dictionary) interface{} {
	ret := make(map[string]interface{})
	for key, member := range dict {
		ret[key] = convertListItemToExpected(member)
	}
	return ret
}

func convertListToExpected(list List) interface{} {
	var ret []interface{}
	for _, li := range list {
		ret = append(ret, convertListItemToExpected(li))
	}
	return ret
}

func convertListItemToExpected(listItem ListItem) interface{} {
	var ret []interface{}
	switch listItem.Type() {
	case ListItemTypeItem:
		return append(ret, convertItemToExpected(listItem.AsItem())[0],
			convertParametersToExpected(listItem.Parameters()))
	case ListItemTypeInnerList:
		return append(ret, convertInnerListToExpected(listItem.AsInnerList()),
			convertParametersToExpected(listItem.Parameters()))
	default:
		panic("invalid ListItem type")
	}
}

func convertInnerListToExpected(list InnerList) interface{} {
	var ret []interface{}
	for _, item := range []Item(list) {
		ret = append(ret, convertItemToExpected(item))
	}
	return ret
}

func convertParametersToExpected(params Parameters) interface{} {
	ret := make(map[string]interface{})
	for key, val := range params {
		switch v := val.(type) {
		case Item:
			ret[key] = convertItemToExpected(v)[0]
		case nil:
			ret[key] = nil
		default:
			panic("invalid param value type")
		}
	}
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
