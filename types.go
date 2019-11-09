package stheader

// Token is the type of tokens, which is short textual words.
type Token string

// ItemType is the enumerated type of BareItem.
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

// BareItem is Item without Parameters.
// BareItem is one of "String", "Byte Sequence", "Boolean", "Integer",
// "Float", or "Token" value.
type BareItem interface {
	// Type returns the item type.
	Type() ItemType

	// AsString returns the "String" value.
	// It panics if item type is not ItemTypeString.
	AsString() string

	// AsByteSeq returns the "Byte Sequence" value.
	// It panics if item type is not ItemTypeByteSeq.
	AsByteSeq() []byte

	// AsBool returns the "Boolean" value.
	// It panics if item type is not ItemTypeBool.
	AsBool() bool

	// AsInt returns the "Integer" value.
	// It panics if item type is not ItemTypeInt.
	AsInt() int64

	// AsFloat returns the "Float" value.
	// It panics if item type is not ItemTypeFloat.
	AsFloat() float64

	// AsToken returns the "Token" value.
	// It panics if item type is not ItemTypeToken.
	AsToken() Token
}

// Item is BareItem with optional Parameters.
type Item interface {
	// BareItem returns the BareItem in Item.
	BareItem() BareItem

	// Parameters returns the optional parameters in Item.
	// It returns nil if Item has no parameters.
	Parameters() Parameters
}

// Parameters is an ordered map of string key to BareItem.
type Parameters interface {
	// Delete deletes a parameter of the specified name.
	Delete(name string)

	// Load returns the value and true if found,
	// nil and false otherwise.
	Load(name string) (value BareItem, ok bool)

	// Range calls f sequentially for each key and value present
	// in the parameters. If f returns false, range stops the iteration.
	//
	// Range does not necessarily correspond to any consistent snapshot
	// of the Map's contents: no name will be visited more than once,
	// but if the value for any name is stored or deleted concurrently,
	// Range may reflect any mapping for that name from any point during
	// the Range call.
	Range(f func(name string, value BareItem) bool)

	// Store sets the value for a name.
	Store(name string, value BareItem)

	// Len returns the count of mapping.
	// It returns 0 if the parameters is empty.
	Len() int
}

// MemberType is the enumerated type of Member.
type MemberType int

const (
	MemberTypeInvalid MemberType = iota
	MemberTypeItem
	MemberTypeInnerList
)

// Member is the type of item of List and also is the type of
// the value of an entry in Dictionary.
//
// Member is either Item or InnerList.
type Member interface {
	// Type returns the member type.
	Type() MemberType

	// AsItem returns the "Item" value.
	// It panics if item type is not MemberTypeItem.
	AsItem() Item

	// AsInnerList returns the "InnerList" value.
	// It panics if item type is not MemberTypeInnerList.
	AsInnerList() InnerList
}

// InnerList is the nested list in List.
type InnerList interface {
	// Items returns items in InnerList.
	Items() []Item

	// Parameters returns the optional parameters in Item.
	// It returns nil if Item has no parameters.
	Parameters() Parameters
}

// List is an ordered list of Member.
type List []Member

// Parameters is an ordered map of string key to Member.
type Dictionary interface {
	// Delete deletes a parameter of the specified name.
	Delete(name string)

	// Load returns the value and true if found,
	// nil and false otherwise.
	Load(name string) (value Member, ok bool)

	//  Range calls f sequentially for each key and value present
	// in the parameters. If f returns false, range stops the iteration.
	//
	// Range does not necessarily correspond to any consistent
	// snapshot of the Map's contents: no name will be visited more
	// than once, but if the value for any name is stored or deleted
	// concurrently, Range may reflect any mapping for that name from
	// any point during the Range call.
	Range(f func(name string, value Member) bool)

	// Store sets the value for a name.
	Store(name string, value Member)

	// Len returns the count of mapping.
	// It returns 0 if the parameters is empty.
	Len() int
}

type bareItem struct {
	val interface{}
}

// NewBareItem creates a new BareItem.
// It panics if value type is not one of the return value type
// of BareItem As* methods.
func NewBareItem(val interface{}) BareItem {
	bi := &bareItem{val: val}
	// Do type check
	bi.Type()
	return bi
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
		panic("invalid BareItem type")
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

// NewItem creates a new Item.
func NewItem(bareItem BareItem, params Parameters) Item {
	return &item{
		bareItem: bareItem,
		params:   params,
	}
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

// NewInnerList creates a new InnerList.
func NewInnerList(items []Item, params Parameters) InnerList {
	return &innerList{
		items:  items,
		params: params,
	}
}

func (l *innerList) Items() []Item {
	return l.items
}

func (l *innerList) Parameters() Parameters {
	return l.params
}

type paramItem struct {
	name  string
	value BareItem
}

type parameters struct {
	items []paramItem
}

// NewParameters creates an empty parameters.
func NewParameters() Parameters {
	return &parameters{}
}

func (p *parameters) Delete(name string) {
	i := p.index(name)
	if i == -1 {
		return
	}

	// https://github.com/golang/go/wiki/SliceTricks
	if i < len(p.items)-1 {
		copy(p.items[i:], p.items[i+1:])
	}
	p.items[len(p.items)-1] = paramItem{}
	p.items = p.items[:len(p.items)-1]
}

func (p *parameters) Load(name string) (value BareItem, ok bool) {
	i := p.index(name)
	if i == -1 {
		return nil, false
	}
	return p.items[i].value, true
}

func (p *parameters) Range(f func(name string, value BareItem) bool) {
	for _, it := range p.items {
		if !f(it.name, it.value) {
			return
		}
	}
}

func (p *parameters) Store(name string, value BareItem) {
	i := p.index(name)
	if i == -1 {
		p.items = append(p.items, paramItem{name: name, value: value})
		return
	}
	p.items[i].value = value
}

func (p *parameters) Len() int {
	return len(p.items)
}

func (p *parameters) index(name string) int {
	for i, it := range p.items {
		if it.name == name {
			return i
		}
	}
	return -1
}

type member struct {
	val interface{}
}

// NewMember creates a new member.
// It panics if value type is not one of the return value type
// of Member As* methods.
func NewMember(val interface{}) Member {
	m := &member{val: val}
	// Do type check
	m.Type()
	return m
}

func (m *member) Type() MemberType {
	switch m.val.(type) {
	case Item:
		return MemberTypeItem
	case InnerList:
		return MemberTypeInnerList
	default:
		panic("invalid Member type")
	}
}

func (m *member) AsItem() Item {
	return m.val.(Item)
}

func (m *member) AsInnerList() InnerList {
	return m.val.(InnerList)
}

type dictItem struct {
	name  string
	value Member
}

type dictionary struct {
	items []dictItem
}

// NewDictionary creates an empty dictionary.
func NewDictionary() Dictionary {
	return &dictionary{}
}

func (d *dictionary) Delete(name string) {
	i := d.index(name)
	if i == -1 {
		return
	}

	// https://github.com/golang/go/wiki/SliceTricks
	if i < len(d.items)-1 {
		copy(d.items[i:], d.items[i+1:])
	}
	d.items[len(d.items)-1] = dictItem{}
	d.items = d.items[:len(d.items)-1]
}

func (d *dictionary) Load(name string) (value Member, ok bool) {
	i := d.index(name)
	if i == -1 {
		return nil, false
	}
	return d.items[i].value, true
}

func (d *dictionary) Range(f func(name string, value Member) bool) {
	for _, it := range d.items {
		if !f(it.name, it.value) {
			return
		}
	}
}

func (d *dictionary) Store(name string, value Member) {
	i := d.index(name)
	if i == -1 {
		d.items = append(d.items, dictItem{name: name, value: value})
		return
	}
	d.items[i].value = value
}

func (d *dictionary) Len() int {
	return len(d.items)
}

func (d *dictionary) index(name string) int {
	for i, it := range d.items {
		if it.name == name {
			return i
		}
	}
	return -1
}

// String returns the string representation for ItemType
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

// String returns the string representation for MemberType
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
