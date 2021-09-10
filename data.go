package yangtree

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// LeafListValueAsKey - leaf-list value can be represented to a path or a key if it is set to true.
	LeafListValueAsKey bool = true
)

// ConfigOnly option is used to find config data nodes that have "config true" statement.
type ConfigOnly struct{}

// StateOnly option is used to find state data nodes that have "config false" statement.
type StateOnly struct{}

// HasState option is used to find state data nodes and data nodes having state data nodes.
type HasState struct{}

func (f ConfigOnly) IsOption() {}
func (f StateOnly) IsOption()  {}
func (f HasState) IsOption()   {}

func (f ConfigOnly) String() string { return "config-only" }
func (f StateOnly) String() string  { return "state-only" }
func (f HasState) String() string   { return "has-state" }

type Operation int

const (
	SetMerge   Operation = iota // netconf edit-config: merge
	SetCreate                   // netconf edit-config: create
	SetReplace                  // netconf edit-config: replace
	SetDelete                   // netconf edit-config: delete
	SetRemove                   // netconf edit-config: remove
)

func (op Operation) String() string {
	switch op {
	case SetMerge:
		return "merge"
	case SetCreate:
		return "create"
	case SetReplace:
		return "replace"
	case SetDelete:
		return "delete"
	case SetRemove:
		return "remove"
	default:
		return "unknown"
	}
}

func (op Operation) IsOption() {}

type EditOption struct {
	Operation
	InsertOption
}

func (edit *EditOption) GetOperation() Operation {
	if edit == nil {
		return SetMerge
	}
	return edit.Operation
}
func (edit *EditOption) GetInsertOption() InsertOption {
	if edit == nil {
		return nil
	}
	return edit.InsertOption
}
func (edit EditOption) IsOption() {}

type InsertToFirst struct{}
type InsertToLast struct{}
type InsertToBefore struct {
	Key string
}
type InsertToAfter struct {
	Key string
}
type InsertOption interface {
	GetInsertKey() string
}

func (o InsertToFirst) GetInsertKey() string  { return "" }
func (o InsertToLast) GetInsertKey() string   { return "" }
func (o InsertToBefore) GetInsertKey() string { return o.Key }
func (o InsertToAfter) GetInsertKey() string  { return o.Key }

func IsValid(node DataNode) bool {
	if node == nil {
		return false
	}
	if node.IsNil() {
		return false
	}
	if node.Schema() == nil {
		return false
	}
	return true
}

func setParent(node DataNode, parent *DataBranch, key *string) {
	switch c := node.(type) {
	case *DataBranch:
		c.parent = parent
		if c.schema.Name != *key {
			c.key = *key
		}
	case *DataLeaf:
		c.parent = parent
		if c.schema.Name != *key {
			c.key = *key
		}
	}
}

func resetParent(node DataNode) {
	switch c := node.(type) {
	case *DataBranch:
		c.parent = nil
		if c.key != "" {
			c.key = ""
		}
	case *DataLeaf:
		c.parent = nil
		if c.key != "" {
			c.key = ""
		}
	}
}

// indexFirst() returns the index of a child related to the key
func indexFirst(parent *DataBranch, key *string) int {
	i := sort.Search(len(parent.children),
		func(j int) bool {
			return *key <= parent.children[j].Key()
		})
	return i
}

func indexMatched(parent *DataBranch, index int, key *string) bool {
	if index < len(parent.children) && *key == parent.children[index].Key() {
		return true
	}
	return false
}

// indexRangeBySchema() returns the index of a child related to the key
func indexRangeBySchema(parent *DataBranch, key *string) (i, max int) {
	i = indexFirst(parent, key)
	max = i
	for ; max < len(parent.children); max++ {
		if parent.children[i].Schema() != parent.children[max].Schema() {
			break
		}
	}
	return
}

// jumpToIndex() updates the node index using offset
func jumpToIndex(parent *DataBranch, index, offset int) (int, int, error) {
	length := len(parent.children)
	if index >= length {
		return length, length, nil
	}
	j := index + offset
	if j < length {
		if parent.Child(index).Schema() != parent.Child(j).Schema() {
			return length, length, fmt.Errorf("invalid data node selected")
		}
		return j, j + 1, nil
	}
	return length, length, nil
}

func (branch *DataBranch) insert(child DataNode, op Operation, iopt InsertOption) error {
	if child.Parent() != nil {
		if child.Parent() == branch {
			return nil
		}
		return fmt.Errorf("%q is already inserted", child)
	}
	schema := child.Schema()
	if branch.Schema() != GetPresentParentSchema(schema) {
		return fmt.Errorf("unable to insert %q because it is not a child of %s", child, branch)
	}

	// duplicatable nodes: read-only leaf-list and non-key list
	duplicatable := IsDuplicatable(schema)
	orderedByUser := IsOrderedByUser(schema)

	key := child.Key()
	i := indexFirst(branch, &key)
	if !duplicatable {
		// find and replace the node that has the same key.
		if i < len(branch.children) && key == branch.children[i].Key() {
			if op == SetCreate {
				return fmt.Errorf("data node %q exists", key)
			}
			resetParent(branch.children[i])
			branch.children[i] = child
			setParent(child, branch, &key)
			return nil
		}
	}
	if !orderedByUser && !duplicatable { // ignore insert option
		iopt = nil
	}

	// insert the new child data node.
	switch o := iopt.(type) {
	case nil:
		// get the best position (ordered-by system)
		for ; i < len(branch.children); i++ {
			if key < branch.children[i].Key() {
				break
			}
		}
	case InsertToLast:
		for ; i < len(branch.children); i++ {
			if schema != branch.children[i].Schema() {
				break
			}
		}
	case InsertToFirst:
		name := child.Name()
		i = sort.Search(len(branch.children),
			func(j int) bool { return name <= branch.children[j].Key() })
	case InsertToBefore:
		target := child.Name() + o.Key
		i = sort.Search(len(branch.children),
			func(j int) bool { return target <= branch.children[j].Key() })
	case InsertToAfter:
		target := child.Name() + o.Key
		i = sort.Search(len(branch.children),
			func(j int) bool { return target <= branch.children[j].Key() })
		if i < len(branch.children) {
			i++
		}
	}
	branch.children = append(branch.children, nil)
	copy(branch.children[i+1:], branch.children[i:])
	branch.children[i] = child
	setParent(child, branch, &key)
	return nil
}

type DataBranch struct {
	schema   *yang.Entry
	parent   *DataBranch
	key      string
	children []DataNode
	metadata map[string]DataNode
}

func (branch *DataBranch) IsYangDataNode()     {}
func (branch *DataBranch) IsNil() bool         { return branch == nil }
func (branch *DataBranch) IsDataBranch() bool  { return true }
func (branch *DataBranch) IsDataLeaf() bool    { return false }
func (branch *DataBranch) IsLeaf() bool        { return false }
func (branch *DataBranch) IsLeafList() bool    { return false }
func (branch *DataBranch) Schema() *yang.Entry { return branch.schema }
func (branch *DataBranch) Parent() DataNode {
	if branch.parent == nil {
		return nil
	}
	return branch.parent
}
func (branch *DataBranch) Value() interface{} { return nil }

func (branch *DataBranch) ValueString() string {
	b, err := branch.MarshalJSON()
	if err != nil {
		return ""
	}
	return string(b)
}

func (branch *DataBranch) Path() string {
	if branch == nil {
		return ""
	}
	if branch.parent != nil {
		return branch.parent.Path() + "/" + branch.Key()
	}
	if IsRootSchema(branch.schema) {
		return ""
	}
	return "/" + branch.Key()
}

func (branch *DataBranch) PathTo(descendant DataNode) string {
	if descendant == nil || branch == descendant {
		return ""
	}
	p := []string{}
	for n := descendant; n != nil; n = n.Parent() {
		if n == branch {
			var buf strings.Builder
			for i := len(p) - 1; i >= 0; i-- {
				buf.WriteString(p[i])
				buf.WriteString("/")
			}
			return buf.String()
		}
		p = append(p, n.Key())
	}
	return ""
}

func (branch *DataBranch) String() string {
	if branch == nil {
		return ""
	}
	return branch.Key()
}

func (branch *DataBranch) New(key string) (DataNode, error) {
	pathnode, err := ParsePath(&key)
	if err != nil {
		return nil, err
	}
	if len(pathnode) == 0 || len(pathnode) > 1 {
		return nil, fmt.Errorf("invalid key %q inserted", key)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return nil, err
	}
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	child, err := New(cschema)
	if err != nil {
		return nil, err
	}
	err = UpdateByMap(child, pmap)
	if err != nil {
		return nil, err
	}
	err = branch.Insert(child)
	if err != nil {
		return nil, err
	}
	return child, nil
}

func (branch *DataBranch) Update(key string, value string) (DataNode, error) {
	pathnode, err := ParsePath(&key)
	if err != nil {
		return nil, err
	}
	if len(pathnode) == 0 || len(pathnode) > 1 {
		return nil, fmt.Errorf("invalid key %q inserted", key)
	}
	nodes, err := setValue(branch, pathnode, value, &EditOption{})
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("reach to unexpected case ... there is not a updated node in %q", branch)
	}
	return nodes[0], nil
}

func (branch *DataBranch) Set(value string) error {
	if IsCreatedWithDefault(branch.schema) {
		for _, s := range branch.schema.Dir {
			if !s.IsDir() && s.Default != "" {
				if branch.Get(s.Name) != nil {
					continue
				}
				c, err := New(s)
				if err != nil {
					return err
				}
				err = branch.Insert(c)
				if err != nil {
					return err
				}
			}
		}
	}
	if value == "" {
		return nil
	}
	err := branch.UnmarshalJSON([]byte(value))
	if err != nil {
		return err
	}
	return nil
}

func (branch *DataBranch) Remove() error {
	if branch.parent == nil {
		return nil
	}
	parent := branch.parent
	length := len(parent.children)
	key := branch.Key()
	i := sort.Search(length,
		func(j int) bool {
			return key <= parent.children[j].Key()
		})
	if i < length && branch == parent.children[i] {
		parent.children = append(parent.children[:i], parent.children[i+1:]...)
		resetParent(branch)
		return nil
	}
	for i := range parent.children {
		if parent.children[i] == branch {
			parent.children = append(parent.children[:i], parent.children[i+1:]...)
			resetParent(branch)
			return nil
		}
	}
	return nil
}

func (branch *DataBranch) Insert(child DataNode, option ...Option) error {
	if !IsValid(child) {
		return fmt.Errorf("invalid child data node")
	}
	for i := range option {
		switch o := option[i].(type) {
		case EditOption:
			return branch.insert(child, o.Operation, o.InsertOption)
		case Operation:
			return branch.insert(child, o, nil)
		}
	}
	return branch.insert(child, SetMerge, nil)
}

func (branch *DataBranch) Delete(child DataNode) error {
	// if !IsValid(child) {
	// 	return fmt.Errorf("invalid child node")
	// }

	// if child.Parent() == nil {
	// 	return fmt.Errorf("'%s' is already removed from a branch", child)
	// }
	if IsKeyNode(child.Schema()) && branch.parent != nil {
		// return fmt.Errorf("key node %q must not be deleted", child)
		return nil
	}

	key := child.Key()
	i := indexFirst(branch, &key)
	if i < len(branch.children) && key == branch.children[i].Key() {
		for ; i < len(branch.children); i++ {
			if branch.children[i] == child {
				branch.children = append(branch.children[:i], branch.children[i+1:]...)
				resetParent(child)
				return nil
			}
		}
	}
	return fmt.Errorf("%q not found on %q", child, branch)
}

// [FIXME] - metadata
// SetMeta() sets metadata key-value pairs.
//   e.g. node.SetMeta(map[string]string{"operation": "replace", "last-modified": "2015-06-18T17:01:14+02:00"})
func (branch *DataBranch) SetMeta(meta ...map[string]string) error {
	sm := GetSchemaMeta(branch.schema)
	if sm.Option == nil {
		return fmt.Errorf("no metadata schema")
	}
	// metaschema := sm.Option.ExtensionSchema
	// for i := range meta {
	// 	for name, value := range meta[i] {
	// 		schema := metaschema[name]
	// 		if schema == nil {
	// 			return fmt.Errorf("metadata schema %q not found", name)
	// 		}
	// 		if branch.metadata == nil {
	// 			branch.metadata = map[string]DataNode{}
	// 		}

	// 		metanode, err := New(schema, value)
	// 		if err != nil {
	// 			return fmt.Errorf("error in seting metadata: %v", err)
	// 		}
	// 		branch.metadata[name] = metanode
	// 	}
	// }
	return nil
}

func (branch *DataBranch) Exist(key string) bool {
	i := indexFirst(branch, &key)
	if i < len(branch.children) {
		return key == branch.children[i].Key()
	}
	return false
}

func (branch *DataBranch) Get(key string) DataNode {
	switch key {
	case ".":
		return branch
	case "..":
		return branch.parent
	case "*":
		if len(branch.children) > 0 {
			return branch.children[0]
		}
		return nil
	case "...":
		n := findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: NodeSelectAll}})
		if len(n) > 0 {
			return n[0]
		}
		return nil
	default:
		i := indexFirst(branch, &key)
		if i < len(branch.children) && key == branch.children[i].Key() {
			return branch.children[i]
		}
		return nil
	}
}

func (branch *DataBranch) GetAll(key string) []DataNode {
	switch key {
	case ".":
		return []DataNode{branch}
	case "..":
		return []DataNode{branch.parent}
	case "*":
		return branch.children
	case "...":
		return findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: NodeSelectAll}})
	default:
		i := indexFirst(branch, &key)
		node := make([]DataNode, 0, len(branch.children)-i+1)
		for max := i; max < len(branch.children); max++ {
			if branch.children[i].Schema() != branch.children[max].Schema() {
				break
			}
			if branch.children[max].Key() == key {
				node = append(node, branch.children[max])
			}
		}
		if len(node) == 0 {
			return nil
		}
		return node
	}
	return nil
}

func (branch *DataBranch) GetValue(key string) interface{} {
	switch key {
	case ".", "..", "*", "...":
		return nil
	default:
		i := indexFirst(branch, &key)
		if i < len(branch.children) && key == branch.children[i].Key() {
			return branch.children[i].Value()
		}
		return nil
	}
}

func (branch *DataBranch) GetValueString(key string) string {
	switch key {
	case ".", "..", "*", "...":
		return ""
	default:
		i := indexFirst(branch, &key)
		if i < len(branch.children) && key == branch.children[i].Key() {
			return branch.children[i].ValueString()
		}
		return ""
	}
}

func (branch *DataBranch) Lookup(prefix string) []DataNode {
	switch prefix {
	case ".":
		return []DataNode{branch}
	case "..":
		return []DataNode{branch.parent}
	case "*":
		return branch.children
	case "...":
		return findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: NodeSelectAll}})
	default:
		i := indexFirst(branch, &prefix)
		node := make([]DataNode, 0, len(branch.children)-i+1)
		for max := i; max < len(branch.children); max++ {
			if strings.HasPrefix(branch.children[max].Key(), prefix) {
				node = append(node, branch.children[max])
			}
		}
		if len(node) == 0 {
			return nil
		}
		return node
	}
}

func (branch *DataBranch) Child(index int) DataNode {
	if index >= 0 && index < len(branch.children) {
		return branch.children[index]
	}
	return nil
}

func (branch *DataBranch) Index(key string) int {
	return indexFirst(branch, &key)
}

func (branch *DataBranch) Len() int {
	return len(branch.children)
}

func (branch *DataBranch) Name() string {
	return branch.schema.Name
}

func (branch *DataBranch) Key() string {
	if branch.parent != nil {
		if branch.key == "" {
			return branch.schema.Name
		}
		return branch.key
	}
	switch {
	case IsListHasKey(branch.schema):
		var keybuffer strings.Builder
		keyname := GetKeynames(branch.schema)
		keybuffer.WriteString(branch.schema.Name)
		for i := range keyname {
			j := indexFirst(branch, &keyname[i])
			if j < len(branch.children) && keyname[i] == branch.children[j].Key() {
				keybuffer.WriteString(`[`)
				keybuffer.WriteString(keyname[i])
				keybuffer.WriteString(`=`)
				keybuffer.WriteString(branch.children[j].ValueString())
				keybuffer.WriteString(`]`)
			} else {
				return keybuffer.String()
			}
		}
		return keybuffer.String()
	default:
		return branch.schema.Name
	}
}

type DataLeaf struct {
	schema *yang.Entry
	parent *DataBranch
	value  interface{}
	key    string
}

func (leaf *DataLeaf) IsYangDataNode()     {}
func (leaf *DataLeaf) IsNil() bool         { return leaf == nil }
func (leaf *DataLeaf) IsDataBranch() bool  { return false }
func (leaf *DataLeaf) IsDataLeaf() bool    { return true }
func (leaf *DataLeaf) IsLeaf() bool        { return leaf.schema.IsLeaf() }
func (leaf *DataLeaf) IsLeafList() bool    { return leaf.schema.IsLeafList() }
func (leaf *DataLeaf) Schema() *yang.Entry { return leaf.schema }
func (leaf *DataLeaf) Parent() DataNode {
	if leaf.parent == nil {
		return nil
	}
	return leaf.parent
}
func (leaf *DataLeaf) String() string {
	if leaf.schema.IsLeaf() {
		return leaf.schema.Name
	}
	return leaf.schema.Name + `[.=` + ValueToString(leaf.value) + `]`
}

func (leaf *DataLeaf) Path() string {
	if leaf == nil {
		return ""
	}
	if leaf.parent != nil {
		return leaf.parent.Path() + "/" + leaf.Key()
	}
	return "/" + leaf.Key()
}

func (leaf *DataLeaf) PathTo(descendant DataNode) string {
	return ""
}

func (leaf *DataLeaf) Value() interface{} {
	return leaf.value
}

func (leaf *DataLeaf) ValueString() string {
	return ValueToString(leaf.value)
}

func (leaf *DataLeaf) New(key string) (DataNode, error) {
	return nil, fmt.Errorf("new is not supported on %q", leaf)
}

func (leaf *DataLeaf) Update(key string, value string) (DataNode, error) {
	return nil, fmt.Errorf("update is not supported %q", leaf)
}

func (leaf *DataLeaf) Set(value string) error {
	if leaf.parent != nil {
		if leaf.IsLeafList() {
			return fmt.Errorf("leaf-list %q must be inserted or deleted", leaf)
		}
		if IsKeyNode(leaf.schema) {
			// ignore key update
			// return fmt.Errorf("unable to update key node %q if used", leaf)
			return nil
		}
	}

	v, err := StringToValue(leaf.schema, leaf.schema.Type, value)
	if err != nil {
		return err
	}
	leaf.value = v
	// fmt.Printf("\n##leaf.value Type %T %v\n", leaf.value, leaf.value)
	return nil
}

func (leaf *DataLeaf) Remove() error {
	if leaf.parent == nil {
		return nil
	}
	if branch := leaf.parent; branch != nil {
		return branch.Delete(leaf)
	}
	return nil
}

func (leaf *DataLeaf) Insert(child DataNode, option ...Option) error {
	return fmt.Errorf("insert is not supported on %q", leaf)
}

func (leaf *DataLeaf) Delete(child DataNode) error {
	return fmt.Errorf("delete is not supported on %q", leaf)
}

// [FIXME] - metadata
// SetMeta() sets metadata key-value pairs.
//   e.g. node.SetMeta(map[string]string{"operation": "replace", "last-modified": "2015-06-18T17:01:14+02:00"})
func (leaf *DataLeaf) SetMeta(meta ...map[string]string) error {
	return nil
}

func (leaf *DataLeaf) Exist(key string) bool {
	return false
}

func (leaf *DataLeaf) Get(key string) DataNode {
	return nil
}

func (leaf *DataLeaf) GetAll(key string) []DataNode {
	return nil
}

func (leaf *DataLeaf) GetValue(key string) interface{} {
	return nil
}

func (leaf *DataLeaf) GetValueString(key string) string {
	return ""
}

func (leaf *DataLeaf) Lookup(prefix string) []DataNode {
	return nil
}

func (leaf *DataLeaf) Child(index int) DataNode {
	return nil
}

func (leaf *DataLeaf) Index(key string) int {
	return 0
}

func (leaf *DataLeaf) Len() int {
	if leaf.schema.Type.Kind == yang.Yempty {
		return 1
	}
	if leaf.value == nil {
		return 0
	}
	return 1
}

func (leaf *DataLeaf) Name() string {
	return leaf.schema.Name
}

func (leaf *DataLeaf) Key() string {
	if leaf.key != "" {
		return leaf.key
	}
	if leaf.schema.IsLeaf() {
		return leaf.schema.Name
	}
	// leaf-list key format LEAF[.=VALUE]
	return leaf.schema.Name + `[.=` + ValueToString(leaf.value) + `]`
}

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
// The pmap is a map has {child key : string value} pairs.
func UpdateByMap(node DataNode, pmap map[string]interface{}) error {
	schema := node.Schema()
	for k, v := range pmap {
		if !strings.HasPrefix(k, "@") {
			if vstr, ok := v.(string); ok {
				if k == "." {
					if err := node.Set(vstr); err != nil {
						return err
					}
				} else if found := node.Get(k); found == nil {
					newnode, err := NewWithValue(GetSchema(schema, k), vstr)
					if err != nil {
						return err
					}
					if err := node.Insert(newnode); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// New() creates a new DataNode (*DataBranch or *DataLeaf) according to the schema
func New(schema *yang.Entry) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	return newDataNode(schema, IsCreatedWithDefault(schema))
}

// NewWithValue() creates a new DataNode (*DataBranch or *DataLeaf) according to the schema
// with its values. The values should be a string if the new DataNode is *DataLeaf.
// The values should be JSON encoded bytes if the node is *DataBranch.
func NewWithValue(schema *yang.Entry, value string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	node, err := newDataNode(schema, false)
	if err != nil {
		return nil, err
	}
	if err = node.Set(value); err != nil {
		return nil, err
	}
	return node, err
}

func newDataNode(schema *yang.Entry, withDefault bool) (DataNode, error) {
	var err error
	var newdata DataNode
	switch {
	case schema.Dir == nil: // leaf, leaf-list
		leaf := &DataLeaf{
			schema: schema,
		}
		if withDefault && schema.Default != "" {
			if err := leaf.Set(schema.Default); err != nil {
				return nil, err
			}
		}
		newdata = leaf
	default: // list, container
		branch := &DataBranch{
			schema:   schema,
			children: []DataNode{},
		}
		if withDefault {
			for _, s := range schema.Dir {
				if !s.IsDir() && s.Default != "" {
					c, err := New(s)
					if err != nil {
						return nil, err
					}
					err = branch.Insert(c)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		newdata = branch
	}
	return newdata, err
}

func setLeafListValues(branch *DataBranch, cschema *yang.Entry, value string, opt *EditOption) ([]DataNode, error) {
	op := opt.GetOperation()
	iopt := opt.GetInsertOption()
	var jval interface{}
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, fmt.Errorf(`leaf-list %q must be set using json arrary e.g. ["a", "b"]`, cschema.Name)
	}
	err := json.Unmarshal([]byte(value), &jval)
	if err != nil {
		return nil, err
	}
	switch jdata := jval.(type) {
	case []interface{}:
		nodes := make([]DataNode, 0, len(jdata))
		for i := range jdata {
			valstr, err := JSONValueToString(jdata[i])
			if err != nil {
				return nil, err
			}
			node, err := New(cschema)
			if err != nil {
				return nil, err
			}
			if err = node.Set(valstr); err != nil {
				return nil, err
			}
			if err = branch.insert(node, op, iopt); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
		return nodes, nil
	default:
		return nil, fmt.Errorf(`leaf-list %q must be set using json arrary e.g. ["a", "b"]`, cschema.Name)
	}
}

func setListValues(branch *DataBranch, cschema *yang.Entry, value string, opt *EditOption) ([]DataNode, error) {
	// op := opt.GetOperation()
	// iopt := opt.GetInsertOption()
	var jval interface{}
	if value == "" {
		return nil, nil
	}
	err := json.Unmarshal([]byte(value), &jval)
	if err != nil {
		return nil, err
	}
	switch jdata := jval.(type) {
	case map[string]interface{}:
		if IsDuplicatableList(cschema) {
			return nil, fmt.Errorf("non-key list %q must have json array format", cschema.Name)
		}
		kname := GetKeynames(cschema)
		kval := make([]string, 0, len(kname))
		return branch.unmarshalJSONList(cschema, kname, kval, jdata)
	case []interface{}:
		return branch.unmarshalJSONListable(cschema, GetKeynames(cschema), jdata)
	default:
		return nil, fmt.Errorf(`leaf-list %q must be set using json arrary e.g. ["ABC", "EFG"]`, cschema.Name)
	}
}

// setValue() create or update a target data node using the value.
//  // - EditOption (create): create a node. It returns data-exists error if it exists.
//  // - EditOption (replace): replace the node to the new node.
//  // - EditOption (merge): update the node. (default)
//  // - EditOption (delete): delete the node. It returns data-missing error if it doesn't exist.
//  // - EditOption (remove): delete the node. It doesn't return data-missing error.
func setValue(root DataNode, pathnode []*PathNode, value string, opt *EditOption) ([]DataNode, error) {
	if len(pathnode) == 0 || pathnode[0].Name == "" {
		if err := root.Set(value); err != nil {
			return nil, err
		}
		return []DataNode{root}, nil
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return setValue(root, pathnode[1:], value, opt)
	case NodeSelectParent:
		if root.Parent() == nil {
			return nil, fmt.Errorf("unknown parent node selected from %q", root)
		}
		root = root.Parent()
		return setValue(root, pathnode[1:], value, opt)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		branch, ok := root.(*DataBranch)
		if !ok {
			return nil, fmt.Errorf("select children from non-branch node %q", root)
		}
		var nodes []DataNode
		for i := 0; i < len(branch.children); i++ {
			rnodes, err := setValue(branch.Child(i), pathnode[1:], value, opt)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, rnodes...)
		}
		return nodes, nil
	case NodeSelectAll:
		nodes, err := setValue(root, pathnode[1:], value, opt)
		if err != nil {
			return nil, err
		}
		branch, ok := root.(*DataBranch)
		if !ok {
			return nil, fmt.Errorf("select children from non-branch node %q", root)
		}
		for i := 0; i < len(branch.children); i++ {
			rnodes, err := setValue(root.Child(i), pathnode, value, opt)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, rnodes...)
		}
		return nodes, nil
	}

	// [FIXME] - metadata
	// if strings.HasPrefix(pathnode[0].Name, "@") {
	// 	return root.SetMeta(value)
	// }

	branch, ok := root.(*DataBranch)
	if !ok {
		return nil, fmt.Errorf("unable to find children from %q", root)
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return nil, err
	}

	var children []DataNode
	var diableSearch bool
	switch {
	case cschema.IsLeafList():
		if LeafListValueAsKey && len(pathnode) == 2 {
			value = pathnode[1].Name
			pmap["."] = pathnode[1].Name
			pathnode = pathnode[:1]
		}
		if v, ok := pmap["."]; ok && v.(string) != value {
			return nil, fmt.Errorf(`value %q must be equal to xpath predicate %s[.=%s]`,
				value, cschema.Name, pmap["."].(string))
		}
	case cschema.IsList() && cschema.Key == "": // non-key list
		diableSearch = true
	}
	reachToEnd := len(pathnode) == 1

	if !diableSearch {
		key, searchAndSetAll := GenerateKey(cschema, pmap)
		children = _find(branch, cschema, &key, searchAndSetAll, pmap, false)
		// setting listable nodes
		if reachToEnd && searchAndSetAll {
			switch {
			case cschema.IsLeafList():
				return setLeafListValues(branch, cschema, value, opt)
			case cschema.IsList():
				return setListValues(branch, cschema, value, opt)
			}
		}
	}

	switch len(children) {
	case 0:
		child, err := New(cschema)
		if err != nil {
			return nil, err
		}
		if err = UpdateByMap(child, pmap); err != nil {
			return nil, err
		}
		if reachToEnd {
			if err = child.Set(value); err != nil {
				return nil, err
			}
		}
		if err = branch.insert(child, opt.GetOperation(), opt.GetInsertOption()); err != nil {
			return nil, err
		}
		if reachToEnd {
			return []DataNode{child}, nil
		}
		node, err := setValue(child, pathnode[1:], value, opt)
		if err != nil {
			child.Remove()
			return nil, err
		}
		return node, nil
	case 1:
		return setValue(children[0], pathnode[1:], value, opt)
	default:
		var nodes []DataNode
		for _, child := range children {
			rnodes, err := setValue(child, pathnode[1:], value, opt)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, rnodes...)
		}
		return nodes, nil
	}
}

// Set sets a value to the target DataNode in the path.
// If the target DataNode is a branch node, the value must be json or json_ietf bytes.
// If the target data node is a leaf or a leaf-list node, the value should be string.
func Set(root DataNode, path string, value string) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	_, err = setValue(root, pathnode, value, nil)
	return err
}

// EditConfig sets a value to the target DataNode in the path.
// If the target DataNode is a branch node, the value must be json or json_ietf bytes.
// If the target data node is a leaf or a leaf-list node, the value should be string.
func EditConfig(root DataNode, path string, value string, opt EditOption) ([]DataNode, error) {
	if !IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	return setValue(root, pathnode, value, &opt)
}

func replaceNode(root DataNode, pathnode []*PathNode, node DataNode) error {
	if len(pathnode) == 0 {
		return root.Insert(node)
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return replaceNode(root, pathnode[1:], node)
	case NodeSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("unknown parent node selected from %q", root)
		}
		root = root.Parent()
		return replaceNode(root, pathnode[1:], node)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		return fmt.Errorf("unable to specify the node position replaced")
	case NodeSelectAll:
		return fmt.Errorf("unable to specify the node position replaced")
	}

	branch, ok := root.(*DataBranch)
	if !ok {
		return fmt.Errorf("unable to find a child from %q", root)
	}
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return err
	}
	switch {
	case IsDuplicatableList(cschema):
		key, _ := GenerateKey(cschema, pmap)
		first := indexFirst(branch, &key)
		if indexMatched(branch, first, &key) {
			return replaceNode(branch.children[first], pathnode[1:], node)
		}
		return nil
	case cschema.IsLeafList():
		if LeafListValueAsKey && len(pathnode) == 2 {
			pmap["."] = pathnode[1].Name
			pathnode = pathnode[:1]
		}
	}
	if cschema == node.Schema() {
		if len(pathnode) > 1 {
			return fmt.Errorf("invalid long tail path: %q", pathnode[1].Name)
		}
		err := UpdateByMap(node, pmap)
		if err != nil {
			return err
		}
		return branch.Insert(node)
	}

	key, prefixmatch := GenerateKey(cschema, pmap)
	children := _find(branch, cschema, &key, prefixmatch, pmap, true)
	if len(children) == 0 { // create
		child, err := New(cschema)
		if err != nil {
			return err
		}
		if err = UpdateByMap(child, pmap); err != nil {
			return err
		}
		if err = branch.Insert(child); err != nil {
			return err
		}
		err = replaceNode(child, pathnode[1:], node)
		if err != nil {
			child.Remove()
		}
		return err
	}

	// updates existent nodes
	if len(children) == 1 {
		return replaceNode(children[0], pathnode[1:], node)
	}
	return fmt.Errorf("unable to specify the node position inserted")
}

// Replace() replaces the target data node to the new data node in the path.
func Replace(root DataNode, path string, new DataNode) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	if !IsValid(new) {
		return fmt.Errorf("invalid new data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return replaceNode(root, pathnode, new)
}

func deleteValue(root DataNode, pathnode []*PathNode) ([]DataNode, error) {
	if len(pathnode) == 0 || pathnode[0].Name == "" {
		if err := root.Remove(); err != nil {
			return nil, err
		}
		return []DataNode{root}, nil
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return deleteValue(root, pathnode[1:])
	case NodeSelectParent:
		if root.Parent() == nil {
			return nil, fmt.Errorf("unknown parent node selected from %q", root)
		}
		root = root.Parent()
		return deleteValue(root, pathnode[1:])
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		branch, ok := root.(*DataBranch)
		if !ok {
			return nil, fmt.Errorf("select children from non-branch node %q", root)
		}
		var deletedNodes []DataNode
		for i := 0; i < len(branch.children); i++ {
			_nodes, err := deleteValue(root.Child(i), pathnode[1:])
			if err != nil {
				return nil, err
			}
			deletedNodes = append(deletedNodes, _nodes...)
		}
		return deletedNodes, nil
	case NodeSelectAll:
		deletedNodes, err := deleteValue(root, pathnode[1:])
		if err != nil {
			return nil, err
		}
		branch, ok := root.(*DataBranch)
		if !ok {
			return nil, fmt.Errorf("select children from non-branch node %q", root)
		}
		for i := 0; i < len(branch.children); i++ {
			_nodes, err := deleteValue(root.Child(i), pathnode)
			if err != nil {
				return nil, err
			}
			deletedNodes = append(deletedNodes, _nodes...)
		}
		return deletedNodes, nil
	}

	branch, ok := root.(*DataBranch)
	if !ok {
		return nil, fmt.Errorf("select children from non-branch node %q", root)
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, root.Schema().Name)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return nil, err
	}
	switch {
	case IsDuplicatableList(cschema):
		key, _ := GenerateKey(cschema, pmap)
		first := indexFirst(branch, &key)
		if indexMatched(branch, first, &key) {
			return deleteValue(branch.children[first], pathnode[1:])
		}
		// [FIXME] Is it an error if the node is not found?
		// return nil, fmt.Errorf("the node deleting not found")
		return nil, nil
	case cschema.IsLeafList():
		if LeafListValueAsKey && len(pathnode) == 2 {
			pmap["."] = pathnode[1].Name
			pathnode = pathnode[:1]
		}
	}
	key, prefixmatch := GenerateKey(cschema, pmap)
	children := _find(branch, cschema, &key, prefixmatch, pmap, true)
	switch len(children) {
	case 0:
		// [FIXME] Is it an error if the node is not found?
		// return nil, fmt.Errorf("the node deleting not found")
		return nil, nil
	case 1:
		return deleteValue(children[0], pathnode[1:])
	default:
		var deletedNodes []DataNode
		for _, node := range children {
			_nodes, err := deleteValue(node, pathnode[1:])
			if err != nil {
				return nil, err
			}
			deletedNodes = append(deletedNodes, _nodes...)
		}
		return deletedNodes, nil
	}
}

// Delete() deletes the target data node in the path if the value is not specified.
// If the value is specified, only the value is deleted.
func Delete(root DataNode, path string) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	_, err = deleteValue(root, pathnode)
	return err
}

func returnFound(node DataNode, option ...Option) []DataNode {
	for i := range option {
		switch option[i].(type) {
		case ConfigOnly:
			meta := GetSchemaMeta(node.Schema())
			if meta.IsState {
				return nil
			}
			return []DataNode{node}
		case StateOnly:
			meta := GetSchemaMeta(node.Schema())
			if meta.IsState {
				return []DataNode{node}
			}
			return nil
		case HasState:
			meta := GetSchemaMeta(node.Schema())
			if meta.IsState {
				return []DataNode{node}
			} else if meta.HasState {
				return []DataNode{node}
			}
			return nil
		}
	}
	return []DataNode{node}
}

func findNode(root DataNode, pathnode []*PathNode, option ...Option) []DataNode {
	if len(pathnode) == 0 {
		return returnFound(root, option...)
	}
	var node, children []DataNode
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return findNode(root, pathnode[1:], option...)
	case NodeSelectParent:
		if root.Parent() == nil {
			return nil
		}
		root = root.Parent()
		return findNode(root, pathnode[1:], option...)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		branch, ok := root.(*DataBranch)
		if !ok {
			return nil
		}
		for i := 0; i < len(branch.children); i++ {
			children = append(children, findNode(root.Child(i), pathnode[1:], option...)...)
		}
		return children
	case NodeSelectAll:
		children = append(children, findNode(root, pathnode[1:], option...)...)
		branch, ok := root.(*DataBranch)
		if !ok {
			return children
		}
		for i := 0; i < len(branch.children); i++ {
			children = append(children, findNode(root.Child(i), pathnode, option...)...)
		}
		return children
	}

	if pathnode[0].Name == "" {
		return returnFound(root, option...)
	}
	// [FIXME]
	if LeafListValueAsKey {
		if root.IsDataLeaf() {
			if pathnode[0].Name == root.ValueString() {
				return []DataNode{root}
			}
			return nil
		}
	}
	branch, ok := root.(*DataBranch)
	if !ok {
		return nil
	}
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return nil
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return nil
	}
	key, prefixmatch := GenerateKey(cschema, pmap)
	if _, ok := pmap["@evaluate-xpath"]; ok {
		first, last := indexRangeBySchema(branch, &key)
		node, err = findByPredicates(branch.children[first:last], pathnode[0].Predicates)
		if err != nil {
			return nil
		}
	} else {
		node = _find(branch, cschema, &key, prefixmatch, pmap, false)
	}
	for i := range node {
		children = append(children, findNode(node[i], pathnode[1:], option...)...)
	}
	return children
}

// Find() finds all data nodes in the path. xpath format is used for the path.
func Find(root DataNode, path string, option ...Option) ([]DataNode, error) {
	if !IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	return findNode(root, pathnode, option...), nil
}

// FindValueString() finds all data in the path and then returns their values by string.
func FindValueString(root DataNode, path string) ([]string, error) {
	if !IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	node := findNode(root, pathnode)
	if len(node) == 0 {
		return nil, nil
	}
	vlist := make([]string, 0, len(node))
	for i := range node {
		if node[i].IsDataLeaf() {
			vlist = append(vlist, node[i].ValueString())
		}
	}
	return vlist, nil
}

// FindValue() finds all data in the path and then returns their values.
func FindValue(root DataNode, path string) ([]interface{}, error) {
	if !IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	node := findNode(root, pathnode)
	if len(node) == 0 {
		return nil, nil
	}
	vlist := make([]interface{}, 0, len(node))
	for i := range node {
		if node[i].IsDataLeaf() {
			vlist = append(vlist, node[i].Value())
		}
	}
	return vlist, nil
}

func clone(destParent *DataBranch, src DataNode) (DataNode, error) {
	var dest DataNode
	switch node := src.(type) {
	case *DataBranch:
		b := &DataBranch{
			schema: node.schema,
		}
		for i := range node.children {
			if _, err := clone(b, node.children[i]); err != nil {
				return nil, err
			}
		}
		dest = b
	case *DataLeaf:
		dest = &DataLeaf{
			schema: node.schema,
			value:  node.value,
		}
	}
	if destParent != nil {
		err := destParent.Insert(dest)
		if err != nil {
			return nil, err
		}
	}
	return dest, nil
}

// Clone() makes a new data node copied from the src data node.
func Clone(src DataNode) DataNode {
	if IsValid(src) {
		dest, _ := clone(nil, src)
		return dest
	}
	return nil
}

// Equal() returns true if node1 and node2 have the same data tree and values.
func Equal(node1, node2 DataNode) bool {
	if node1 == node2 {
		return true
	}
	if node1 == nil || node2 == nil {
		return false
	}
	if node1.Schema() != node2.Schema() {
		return false
	}
	switch d1 := node1.(type) {
	case *DataBranch:
		d2 := node2.(*DataBranch)
		if d1.Len() != d2.Len() {
			return false
		}
		for i := range d1.children {
			if equal := Equal(d1.children[i], d2.children[i]); !equal {
				return false
			}
		}
		return true
	case *DataLeaf:
		d2 := node2.(*DataLeaf)
		if _, ok := d2.value.(yang.Number); ok {
			return cmp.Equal(d1.value, d2.value)
		}
		return d1.value == d2.value
	}
	return false
}

func merge(dest, src DataNode) error {
	if dest.Schema() != src.Schema() {
		return fmt.Errorf("unable to merge different schema (%s, %s)", dest, src)
	}
	switch s := src.(type) {
	case *DataBranch:
		d := dest.(*DataBranch)
		for i := range s.children {
			schema := s.children[i].Schema()
			if IsDuplicatableList(schema) {
				if _, err := clone(d, s.children[i]); err != nil {
					return err
				}
			} else {
				dchild := d.GetAll(s.children[i].Key())
				if len(dchild) > 0 {
					for j := range dchild {
						if err := merge(dchild[j], s.children[i]); err != nil {
							return err
						}
					}
				} else {
					if _, err := clone(d, s.children[i]); err != nil {
						return err
					}
				}
			}
		}
	case *DataLeaf:
		d := dest.(*DataLeaf)
		d.value = s.value
	default:
		return fmt.Errorf("invalid data node type: %T", s)
	}
	return nil
}

// Merge() merges the src data node to the target data node in the path.
// The target data node is updated using the src data node.
func Merge(root DataNode, path string, src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	node, err := Find(root, path)
	if err != nil {
		return err
	}
	switch len(node) {
	case 0:
		err := Set(root, path, "")
		if err != nil {
			return err
		}
		node, err = Find(root, path)
		if err != nil {
			return err
		}
		if len(node) > 1 {
			return fmt.Errorf("more than one data node found - cannot specify the merged node")
		} else if len(node) == 1 {
			return merge(node[0], src)
		}
		return fmt.Errorf("failed to create and merge the nodes in %q", path)
	case 1:
		return merge(node[0], src)
	}
	return fmt.Errorf("more than one data node found - cannot specify the merged node")
}

// Merge() merges the src data node to the branch data node.
func (branch *DataBranch) Merge(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return merge(branch, src)
}

// Merge() merges the src data node to the leaf data node.
func (leaf *DataLeaf) Merge(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return merge(leaf, src)
}

// Replace() replaces itself to the src node.
func (branch *DataBranch) Replace(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	if branch.parent == nil {
		return fmt.Errorf("no parent node")
	}
	if branch.schema != src.Schema() {
		return fmt.Errorf("unable to replace the different schema nodes")
	}
	if IsDuplicatableList(branch.schema) {
		return fmt.Errorf("replace is not supported for non-key list")
	}
	return branch.parent.Insert(src)
}

// Replace() replaces itself to the src node.
func (leaf *DataLeaf) Replace(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	if leaf.schema != src.Schema() {
		return fmt.Errorf("unable to replace the different schema nodes")
	}
	if leaf.parent == nil {
		return fmt.Errorf("no parent node")
	}
	return leaf.parent.Insert(src)
}

// Map converts the data node list to a map using the path.
func Map(node []DataNode) map[string]DataNode {
	m := map[string]DataNode{}
	for i := range node {
		m[node[i].Path()] = node[i]
	}
	return m
}

// FindAllInRoute() find all parent nodes in the path.
// The path must indicate an unique node. (not support wildcard and multiple node selection)
func FindAllInRoute(path string) []DataNode {
	return nil
}

// Get KeyValues if a key list.
func GetKeyValues(node DataNode) ([]string, []string) {
	keynames := GetKeynames(node.Schema())
	keyvals := make([]string, 0, len(keynames))
	for i := range keynames {
		keynode := node.Get(keynames[i])
		if keynode == nil {
			return keynames, keyvals
		}
		keyvals = append(keyvals, keynode.ValueString())
	}
	return keynames, keyvals
}
