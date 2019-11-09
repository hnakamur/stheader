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

type Parameters interface {
	Delete(name string)
	Load(name string) (value BareItem, ok bool)
	Range(f func(name string, value BareItem) bool)
	Store(name string, value BareItem)
	Len() int
}

type MemberType int

const (
	MemberTypeInvalid MemberType = iota
	MemberTypeItem
	MemberTypeInnerList
)

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

type Dictionary interface {
	Delete(name string)
	Load(name string) (value Member, ok bool)
	Range(f func(name string, value Member) bool)
	Store(name string, value Member)
	Len() int
}

type bareItem struct {
	val interface{}
}

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
