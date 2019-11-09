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

type Item interface {
	Type() ItemType
	AsString() string
	AsByteSeq() []byte
	AsBool() bool
	AsInt() int64
	AsFloat() float64
	AsToken() Token

	Parameters() Parameters
}

type ListItemType int

const (
	ListItemTypeInvalid ListItemType = iota
	ListItemTypeItem
	ListItemTypeInnerList
)

type Parameters map[string]Item

type ListItem interface {
	Type() ListItemType
	AsItem() Item
	AsInnerList() *InnerList
}

type InnerList struct {
	Items      []Item
	Parameters Parameters
}

type List []ListItem

type Dictionary map[string]ListItem

type item struct {
	val    interface{}
	params Parameters
}

func (i *item) Type() ItemType {
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

func (i *item) AsString() string {
	return i.val.(string)
}

func (i *item) AsByteSeq() []byte {
	return i.val.([]byte)
}

func (i *item) AsBool() bool {
	return i.val.(bool)
}

func (i *item) AsInt() int64 {
	return i.val.(int64)
}

func (i *item) AsFloat() float64 {
	return i.val.(float64)
}

func (i *item) AsToken() Token {
	return i.val.(Token)
}

func (i *item) Parameters() Parameters {
	return i.params
}

type listItem struct {
	val interface{}
}

func (i *listItem) Type() ListItemType {
	switch i.val.(type) {
	case Item:
		return ListItemTypeItem
	case *InnerList:
		return ListItemTypeInnerList
	default:
		return ListItemTypeInvalid
	}
}

func (i *listItem) AsItem() Item {
	return i.val.(Item)
}

func (i *listItem) AsInnerList() *InnerList {
	return i.val.(*InnerList)
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

func (t ListItemType) String() string {
	switch t {
	case ListItemTypeItem:
		return "item"
	case ListItemTypeInnerList:
		return "innerList"
	default:
		panic("invalidListItemType")
	}
}
