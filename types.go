package stheader

type Token string

type ItemType int

const (
	ItemTypeInvalid ItemType = iota
	ItemTypeString
	ItemTypeByteSeq
	ItemTypeBool
	ItemTypeInt
	ItemTypeFloat
	ItemTypeToken
)

type BareItem interface {
	Type() ItemType
	AsString() string
	AsByteSeq() []byte
	AsBool() bool
	AsInt() int64
	AsFloat() float64
	AsToken() Token
}

type Item interface {
	BareItem() BareItem
	Parameters() Parameters
}

type MemberType int

const (
	MemberTypeInvalid MemberType = iota
	MemberTypeItem
	MemberTypeInnerList
)

type Parameters map[string]BareItem

type Member interface {
	Type() MemberType
	AsItem() Item
	AsInnerList() InnerList
}

type InnerList interface {
	Items() []Item
	Parameters() Parameters
}

type List []Member

type Dictionary map[string]Member

type bareItem struct {
	val interface{}
}

func (i *bareItem) Type() ItemType {
	switch i.val.(type) {
	case string:
		return ItemTypeString
	case []byte:
		return ItemTypeByteSeq
	case bool:
		return ItemTypeBool
	case int64:
		return ItemTypeInt
	case float64:
		return ItemTypeFloat
	case Token:
		return ItemTypeToken
	default:
		return ItemTypeInvalid
	}
}

func (i *bareItem) AsString() string {
	return i.val.(string)
}

func (i *bareItem) AsByteSeq() []byte {
	return i.val.([]byte)
}

func (i *bareItem) AsBool() bool {
	return i.val.(bool)
}

func (i *bareItem) AsInt() int64 {
	return i.val.(int64)
}

func (i *bareItem) AsFloat() float64 {
	return i.val.(float64)
}

func (i *bareItem) AsToken() Token {
	return i.val.(Token)
}

type item struct {
	bareItem BareItem
	params   Parameters
}

func (i *item) BareItem() BareItem {
	return i.bareItem
}

func (i *item) Parameters() Parameters {
	return i.params
}

type innerList struct {
	items  []Item
	params Parameters
}

func (l *innerList) Items() []Item {
	return l.items
}

func (l *innerList) Parameters() Parameters {
	return l.params
}

type member struct {
	val interface{}
}

func (m *member) Type() MemberType {
	switch m.val.(type) {
	case Item:
		return MemberTypeItem
	case InnerList:
		return MemberTypeInnerList
	default:
		return MemberTypeInvalid
	}
}

func (m *member) AsItem() Item {
	return m.val.(Item)
}

func (m *member) AsInnerList() InnerList {
	return m.val.(InnerList)
}

func (t ItemType) String() string {
	switch t {
	case ItemTypeString:
		return "string"
	case ItemTypeByteSeq:
		return "byteSeq"
	case ItemTypeBool:
		return "bool"
	case ItemTypeInt:
		return "int"
	case ItemTypeFloat:
		return "float"
	case ItemTypeToken:
		return "token"
	default:
		panic("invalidItemType")
	}
}

func (t MemberType) String() string {
	switch t {
	case MemberTypeItem:
		return "item"
	case MemberTypeInnerList:
		return "innerList"
	default:
		panic("invalidMemberType")
	}
}
