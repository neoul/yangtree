package yangtree

import (
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
		if IsUniqueList(c.schema) {
			c.key = *key
		}
	case *DataLeaf:
		c.parent = parent
		c.key = *key
	}
}

func resetParent(node DataNode) {
	switch c := node.(type) {
	case *DataBranch:
		c.parent = nil
		c.key = ""
	case *DataLeaf:
		c.parent = nil
		c.key = ""
	}
}

// indexRange() returns the index of a child related to the key
func indexRange(parent *DataBranch, key string, prefixKey bool) (i, max int) {
	length := len(parent.children)
	i = sort.Search(length,
		func(j int) bool {
			return key <= parent.children[j].Key()
		})
	if prefixKey {
		max = i
		for ; max < length; max++ {
			if !strings.HasPrefix(parent.children[max].Key(), key) {
				break
			}
		}
		return
	}
	max = i
	for ; max < length; max++ {
		if parent.children[max].Key() != key {
			break
		}
	}
	return
}

// keyToIndex() returns the index of a child related to the key
func keyToIndex(parent *DataBranch, key string) int {
	length := len(parent.children)
	i := sort.Search(length,
		func(j int) bool {
			return key <= parent.children[j].Key()
		})
	return i
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

type DataBranch struct {
	schema   *yang.Entry
	parent   *DataBranch
	key      string
	children []DataNode
	metadata map[string]DataNode
}

func (branch *DataBranch) IsYangDataNode()     {}
func (branch *DataBranch) IsNil() bool         { return branch == nil }
func (branch *DataBranch) IsBranch() bool      { return true }
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

func (branch *DataBranch) New(key string, value ...string) (DataNode, error) {
	pathnode, err := ParsePath(&key)
	if err != nil {
		return nil, err
	}
	if len(pathnode) == 0 {
		return nil, fmt.Errorf("invalid key %q inserted", key)
	}
	if len(pathnode) > 1 {
		return nil, fmt.Errorf("invalid key %q inserted", key)
	}
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	key, pmap, err := keyGen(cschema, pathnode[0])
	if err != nil {
		return nil, err
	}
	return newChild(branch, cschema, pmap, value...)
}

func (branch *DataBranch) Update(key string, value ...string) error {
	pathnode, err := ParsePath(&key)
	if err != nil {
		return err
	}
	if len(pathnode) == 0 {
		return fmt.Errorf("invalid key %q inserted", key)
	}
	if len(pathnode) > 1 {
		return fmt.Errorf("invalid key %q inserted", key)
	}
	if err := setValue(branch, pathnode[:1], value...); err != nil {
		return err
	}
	return nil
}

func (branch *DataBranch) Set(value ...string) error {
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
	for i := range value {
		err := branch.UnmarshalJSON([]byte(value[i]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (branch *DataBranch) Remove(value ...string) error {
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

type InsertToFirst struct{}
type InsertToLast struct{}
type InsertToBefore struct {
	Key string
}
type InsertToAfter struct {
	Key string
}

func (o InsertToFirst) IsOption()  {}
func (o InsertToLast) IsOption()   {}
func (o InsertToBefore) IsOption() {}
func (o InsertToAfter) IsOption()  {}

func (branch *DataBranch) Insert(child DataNode, option ...Option) error {
	if !IsValid(child) {
		return fmt.Errorf("invalid child data node")
	}
	if child.Parent() != nil {
		return fmt.Errorf("%q is already inserted", child)
	}
	schema := child.Schema()
	if branch.Schema() != GetPresentParentSchema(schema) {
		return fmt.Errorf("unable to insert %q because it is not a child of %s", child, branch)
	}
	duplicatable := IsDuplicatable(schema)
	orderedbyuser := IsOrderedByUser(schema)

	var insertOption Option
	if orderedbyuser || duplicatable {
		insertOption = InsertToLast{}
		for i := range option {
			switch option[i].(type) {
			case InsertToFirst, InsertToAfter, InsertToBefore:
				insertOption = option[i]
			}
		}
	}
	// meta := GetSchemaMeta(schema)
	// index := 0

	length := len(branch.children)
	key := child.Key()
	if key == "" {
		return fmt.Errorf("unable to insert %q because it doesn't have a key", child.Schema().Name)
	}
	i := sort.Search(length,
		func(j int) bool {
			return key <= branch.children[j].Key()
		})

	// replace the existent data node if it is not a duplicatable node.
	if !duplicatable {
		if i < length && branch.children[i].Key() == key {
			resetParent(branch.children[i])
			branch.children[i] = child
			setParent(child, branch, &key)
			return nil
		}
	}
	// insert the new child data node.
	switch o := insertOption.(type) {
	case nil:
		// get the best position (ordered-by system)
		for ; i < length; i++ {
			if key < branch.children[i].Key() {
				break
			}
		}
	case InsertToLast:
		for ; i < length; i++ {
			if schema != branch.children[i].Schema() {
				break
			}
		}
	case InsertToFirst:
		name := child.Name()
		i = sort.Search(length, func(j int) bool { return name <= branch.children[j].Key() })
	case InsertToBefore:
		target := child.Name() + o.Key
		i = sort.Search(length, func(j int) bool { return target <= branch.children[j].Key() })
	case InsertToAfter:
		target := child.Name() + o.Key
		i = sort.Search(length, func(j int) bool { return target <= branch.children[j].Key() })
		if i < length {
			i++
		}
	}
	branch.children = append(branch.children, nil)
	copy(branch.children[i+1:], branch.children[i:])
	branch.children[i] = child
	setParent(child, branch, &key)
	return nil
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

	length := len(branch.children)
	key := child.Key()
	i := sort.Search(length,
		func(j int) bool {
			return key <= branch.children[j].Key()
		})
	if i < length && branch.children[i] == child {
		c := branch.children[i]
		branch.children = append(branch.children[:i], branch.children[i+1:]...)
		resetParent(c)
		return nil
	}
	for i := range branch.children {
		if branch.children[i] == child {
			branch.children = append(branch.children[:i], branch.children[i+1:]...)
			resetParent(child)
			return nil
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
	length := len(branch.children)
	i := sort.Search(length,
		func(j int) bool {
			return key <= branch.children[j].Key()
		})
	if i < length {
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
		length := len(branch.children)
		i := sort.Search(length,
			func(j int) bool {
				return key <= branch.children[j].Key()
			})
		if i < length && key == branch.children[i].Key() {
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
		i, max := indexRange(branch, key, false)
		if i < max {
			return branch.children[i:max]
		}
	}
	return nil
}

func (branch *DataBranch) GetValue(key string) interface{} {
	switch key {
	case ".", "..", "*", "...":
		return nil
	default:
		length := len(branch.children)
		i := sort.Search(length,
			func(j int) bool {
				return key <= branch.children[j].Key()
			})
		if i < length && key == branch.children[i].Key() {
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
		length := len(branch.children)
		i := sort.Search(length,
			func(j int) bool {
				return key <= branch.children[j].Key()
			})
		if i < length && key == branch.children[i].Key() {
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
		i, max := indexRange(branch, prefix, true)
		if i < max {
			return branch.children[i:max]
		}
	}
	return nil
}

func (branch *DataBranch) Child(index int) DataNode {
	if index < len(branch.children) {
		return branch.children[index]
	}
	return nil
}

func (branch *DataBranch) Index(key string) (int, int) {
	return indexRange(branch, key, false)
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
	case IsUniqueList(branch.schema):
		keyname := GetKeynames(branch.schema)
		key := make([]string, 0, len(keyname)+1)
		key = append(key, branch.schema.Name)
		for i := range keyname {
			found := false
			for j := range branch.children {
				// [FIXME]
				if branch.children[j].Key() == keyname[i] {
					key = append(key, "["+keyname[i]+"="+branch.children[j].ValueString()+"]")
					found = true
				}
			}
			if !found {
				return ""
			}
		}
		return strings.Join(key, "")
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
func (leaf *DataLeaf) IsBranch() bool      { return false }
func (leaf *DataLeaf) IsLeaf() bool        { return leaf.schema.IsLeaf() }
func (leaf *DataLeaf) IsLeafList() bool    { return leaf.schema.IsLeafList() }
func (leaf *DataLeaf) Schema() *yang.Entry { return leaf.schema }
func (leaf *DataLeaf) Parent() DataNode {
	if leaf.parent == nil {
		return nil
	}
	return leaf.parent
}
func (leaf *DataLeaf) String() string { return leaf.schema.Name }

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

func (leaf *DataLeaf) New(key string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("new is not supported on %q", leaf)
}

func (leaf *DataLeaf) Update(key string, value ...string) error {
	return fmt.Errorf("update is not supported %q", leaf)
}

func (leaf *DataLeaf) Set(value ...string) error {
	if IsKeyNode(leaf.schema) && leaf.parent != nil {
		return nil
		// [FIXME]
		// ignore key update
		// return fmt.Errorf("unable to update key node %q if used", leaf)
	}
	if len(value) == 0 && leaf.schema.Default != "" {
		if err := leaf.Set(leaf.schema.Default); err != nil {
			return err
		}
	}

	for i := range value {
		v, err := StringToValue(leaf.schema, leaf.schema.Type, value[i])
		if err != nil {
			return err
		}
		leaf.value = v
	}
	// fmt.Printf("\n##leaf.value Type %T %v\n", leaf.value, leaf.value)
	return nil
}

func (leaf *DataLeaf) Remove(value ...string) error {
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

func (leaf *DataLeaf) Index(key string) (int, int) {
	return 0, 0
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
	if leaf.schema.IsLeaf() {
		return leaf.schema.Name
	}
	// leaf-list key format LEAF[.=VALUE]
	if leaf.parent != nil {
		if leaf.key == "" {
			return leaf.schema.Name
		}
		return leaf.key
	}
	return leaf.schema.Name + `[.=` + ValueToString(leaf.value) + `]`
}

func newChild(parent *DataBranch, cschema *yang.Entry, pmap map[string]interface{}, value ...string) (DataNode, error) {
	child, err := New(cschema, value...)
	if err != nil {
		return nil, err
	}
	switch {
	case IsUniqueList(cschema):
		keyname := GetKeynames(cschema)
		for i := range keyname {
			v, ok := pmap[keyname[i]]
			if !ok {
				continue
			}
			delete(pmap, keyname[i])
			kn, err := New(GetSchema(cschema, keyname[i]), v.(string))
			if err != nil {
				return nil, err
			}
			if err := child.Insert(kn); err != nil {
				return nil, err
			}
		}
		fallthrough
	default:
		for k, v := range pmap {
			if strings.HasPrefix(k, "@") {
				continue
			}
			if k == "." {
				if err := child.Set(v.(string)); err != nil {
					return nil, err
				}
				continue
			}
			kn, err := New(GetSchema(cschema, k), v.(string))
			if err != nil {
				return nil, err
			}
			if err := child.Insert(kn); err != nil {
				return nil, err
			}
		}
	}
	if err := parent.Insert(child); err != nil {
		return nil, err
	}
	return child, nil
}

// UpdateChild() updates the data node using pmap (path predicate map) and string values.
// The pmap is a map has {child key : value} pairs.
func UpdateChild(node DataNode, pmap map[string]interface{}) error {
	schema := node.Schema()
	switch {
	case IsUniqueList(schema):
		keyname := GetKeynames(schema)
		for i := range keyname {
			v, ok := pmap[keyname[i]]
			if !ok {
				continue
			}
			delete(pmap, keyname[i])
			kn, err := New(GetSchema(schema, keyname[i]), v.(string))
			if err != nil {
				return err
			}
			if err := node.Insert(kn); err != nil {
				return err
			}
		}
		fallthrough
	default:
		for k, v := range pmap {
			if strings.HasPrefix(k, "@") {
				continue
			}
			if k == "." {
				if err := node.Set(v.(string)); err != nil {
					return err
				}
				continue
			}
			kn, err := New(GetSchema(schema, k), v.(string))
			if err != nil {
				return err
			}
			if err := node.Insert(kn); err != nil {
				return err
			}
		}
	}
	return nil
}

// New() creates a new DataNode (one of *DataBranch, *DataLeaf and *DataLeaflist) according to the schema
// with its values. The values should be a string if the creating DataNode is *DataLeaf or *DataLeafList.
// If the creating DataNode is *DataBranch, the values should be JSON encoded bytes.
func New(schema *yang.Entry, value ...string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	var err error
	var newdata DataNode
	switch {
	case schema.Dir == nil: // leaf, leaf-list
		leaf := &DataLeaf{
			schema: schema,
		}
		err = leaf.Set(value...)
		if err != nil {
			return nil, err
		}
		newdata = leaf
	default: // list, container
		branch := &DataBranch{
			schema:   schema,
			children: []DataNode{},
		}
		err = branch.Set(value...)
		if err != nil {
			return nil, err
		}
		newdata = branch
	}
	return newdata, err
}

func setValue(root DataNode, pathnode []*PathNode, value ...string) error {
	if len(pathnode) == 0 {
		return root.Set(value...)
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return setValue(root, pathnode[1:], value...)
	case NodeSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("unknown parent node selected from %q", root)
		}
		root = root.Parent()
		return setValue(root, pathnode[1:], value...)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		branch, ok := root.(*DataBranch)
		if !ok {
			return fmt.Errorf("select children from non-branch node %q", root)
		}
		for i := 0; i < len(branch.children); i++ {
			if err := setValue(branch.Child(i), pathnode[1:], value...); err != nil {
				return err
			}
		}
		return nil
	case NodeSelectAll:
		if err := setValue(root, pathnode[1:], value...); err != nil {
			return err
		}
		branch, ok := root.(*DataBranch)
		if !ok {
			return fmt.Errorf("select children from non-branch node %q", root)
		}
		for i := 0; i < len(branch.children); i++ {
			if err := setValue(root.Child(i), pathnode, value...); err != nil {
				return err
			}
		}
		return nil
	}

	if pathnode[0].Name == "" {
		return root.Set(value...)
	}
	// [FIXME] - metadata
	// if strings.HasPrefix(pathnode[0].Name, "@") {
	// 	return root.SetMeta(value...)
	// }

	branch, ok := root.(*DataBranch)
	if !ok {
		return fmt.Errorf("unable to find children from %q", root)
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}

	if LeafListValueAsKey && cschema.IsLeafList() {
		if len(pathnode) == 2 {
			value = append(value, pathnode[1].Name)
			child, err := New(cschema, value...)
			if err != nil {
				return err
			}
			return root.Insert(child)
		}
	}

	var first, last int
	key, pmap, err := keyGen(cschema, pathnode[0])
	if err != nil {
		return err
	}
	if index, ok := pmap["@index"]; ok {
		first = keyToIndex(branch, key)
		first, last, err = jumpToIndex(branch, first, index.(int))
		if err != nil {
			return err
		}
	} else {
		_, prefixmatch := pmap["@prefix"]
		first, last = indexRange(branch, key, prefixmatch)
		if IsDuplicatedList(cschema) {
			first = last
		}
	}
	// newly adds a node
	if first == last {
		var child DataNode
		if len(pathnode)-1 == 0 {
			_, err = newChild(branch, cschema, pmap, value...)
		} else {
			child, err = newChild(branch, cschema, pmap)
			if err != nil {
				return err
			}
			err = setValue(child, pathnode[1:], value...)
			if err != nil {
				child.Remove()
			}
		}
		return err
	}
	_, needToUpdate := pmap["@need-to-update"]

	// updates existent nodes
	if !cschema.IsDir() { // predicate for self value ==> [.=VALUE]
		if v, ok := pmap["."]; ok {
			value = append(value, v.(string))
		}
	}
	for ; first < last; first++ {
		child := root.Child(first)
		if needToUpdate {
			if err := UpdateChild(child, pmap); err != nil {
				return err
			}
		}
		if err := setValue(child, pathnode[1:], value...); err != nil {
			return err
		}
	}
	return nil
}

// Set sets a value or values to the target DataNode in the path.
// If the target DataNode is a branch node, the value must be json or json_ietf bytes.
// If the target data node is a leaf or a leaf-list node, the value should be string.
func Set(root DataNode, path string, value ...string) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return setValue(root, pathnode, value...)
}

func replace(root DataNode, pathnode []*PathNode, node DataNode) error {
	if len(pathnode) == 0 {
		return root.Insert(node)
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return replace(root, pathnode[1:], node)
	case NodeSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("unknown parent node selected from %q", root)
		}
		root = root.Parent()
		return replace(root, pathnode[1:], node)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		return fmt.Errorf("unable to specify the node position inserted")
	case NodeSelectAll:
		return fmt.Errorf("unable to specify the node position inserted")
	}

	branch, ok := root.(*DataBranch)
	if !ok {
		return fmt.Errorf("unable to find a child from %q", root)
	}
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}

	var first, last int
	key, pmap, err := keyGen(cschema, pathnode[0])
	if err != nil {
		return err
	}
	if index, ok := pmap["@index"]; ok {
		first = keyToIndex(branch, key)
		first, last, err = jumpToIndex(branch, first, index.(int))
		if err != nil {
			return err
		}
	} else {
		_, prefixmatch := pmap["@prefix"]
		first, last = indexRange(branch, key, prefixmatch)
		if IsDuplicatedList(cschema) {
			first = last
		}
	}
	if cschema == node.Schema() {
		if len(pathnode) > 1 {
			return fmt.Errorf("invalid long tail path: %q", pathnode[1].Name)
		}
		err := UpdateChild(node, pmap)
		if err != nil {
			return err
		}
		return branch.Insert(node)
	}
	// newly adds a node
	if first == last {
		child, err := newChild(branch, cschema, pmap)
		if err != nil {
			return err
		}
		err = replace(child, pathnode[1:], node)
		if err != nil {
			child.Remove()
		}
		return err
	}
	// updates existent nodes
	if first+1 == last {
		return replace(branch.Child(first), pathnode[1:], node)
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
	return replace(root, pathnode, new)
}

func deleteValue(root DataNode, pathnode []*PathNode, value ...string) error {
	if len(pathnode) == 0 {
		return root.Remove(value...)
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return deleteValue(root, pathnode[1:], value...)
	case NodeSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("unknown parent node selected from %q", root)
		}
		root = root.Parent()
		return deleteValue(root, pathnode[1:], value...)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		branch, ok := root.(*DataBranch)
		if !ok {
			return fmt.Errorf("select children from non-branch node %q", root)
		}
		for i := 0; i < len(branch.children); i++ {
			if err := deleteValue(root.Child(i), pathnode[1:], value...); err != nil {
				return err
			}
		}
		return nil
	case NodeSelectAll:
		if err := deleteValue(root, pathnode[1:], value...); err != nil {
			return err
		}
		branch, ok := root.(*DataBranch)
		if !ok {
			return fmt.Errorf("select children from non-branch node %q", root)
		}
		for i := 0; i < len(branch.children); i++ {
			if err := deleteValue(root.Child(i), pathnode, value...); err != nil {
				return err
			}
		}
		return nil
	}

	if pathnode[0].Name == "" {
		return root.Remove(value...)
	}
	if LeafListValueAsKey {
		if root.Schema().IsLeafList() {
			value = append(value, pathnode[0].Name)
			return root.Remove(value...)
		}
	}
	branch, ok := root.(*DataBranch)
	if !ok {
		return fmt.Errorf("select children from non-branch node %q", root)
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("schema %q not found from %q", pathnode[0].Name, root.Schema().Name)
	}
	var first, last int
	key, pmap, err := keyGen(cschema, pathnode[0])
	if err != nil {
		return err
	}
	if index, ok := pmap["@index"]; ok {
		first = keyToIndex(branch, key)
		first, last, err = jumpToIndex(branch, first, index.(int))
		if err != nil {
			return err
		}
	} else {
		_, prefixmatch := pmap["@prefix"]
		first, last = indexRange(branch, key, prefixmatch)
		if IsDuplicatedList(cschema) {
			if first < last {
				last = first + 1
			}
		}
	}
	if !cschema.IsDir() {
		if v, ok := pmap["."]; ok {
			value = append(value, v.(string))
		}
	}
	if first >= last {
		return nil
	}
	for _, node := range branch.children[first:last] {
		if err := deleteValue(node, pathnode[1:], value...); err != nil {
			return err
		}
	}
	return nil
}

// Delete() deletes the target data node in the path if the value is not specified.
// If the value is specified, only the value is deleted.
func Delete(root DataNode, path string, value ...string) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return deleteValue(root, pathnode, value...)
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
		// if leaflist, ok := root.(*DataLeafList); ok {
		// 	if leaflist.Exist(pathnode[0].Name) {
		// 		return returnFound(root, option...)
		// 	}
		// 	return nil
		// }
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return nil
	}
	branch, ok := root.(*DataBranch)
	if !ok {
		return nil
	}
	var first, last int
	key, pmap, err := keyGen(cschema, pathnode[0])
	if err != nil {
		return nil
	}
	_, prefixsearch := pmap["@prefix"]
	first, last = indexRange(branch, key, prefixsearch)
	if _, ok := pmap["@find-in-order"]; ok {
		node, _ = findByPredicates(branch.children[first:last], pathnode[0].Predicates)
	} else {
		if index, ok := pmap["@index"]; ok {
			first, last, err = jumpToIndex(branch, first, index.(int))
			if err != nil {
				return nil
			}
		}
		node = branch.children[first:last]
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

func findValue(root DataNode, pathnode []*PathNode) []interface{} {
	if len(pathnode) == 0 {
		if root.IsBranch() {
			return nil
		}
		return []interface{}{root.Value()}
	}
	var childvalues []interface{}
	var node []DataNode
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return findValue(root, pathnode[1:])
	case NodeSelectParent:
		if root.Parent() == nil {
			return nil
		}
		root = root.Parent()
		return findValue(root, pathnode[1:])
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
			childvalues = append(childvalues, findValue(root.Child(i), pathnode[1:])...)
		}
		return childvalues
	case NodeSelectAll:
		childvalues = append(childvalues, findValue(root, pathnode[1:])...)
		branch, ok := root.(*DataBranch)
		if !ok {
			return childvalues
		}
		for i := 0; i < len(branch.children); i++ {
			childvalues = append(childvalues, findValue(root.Child(i), pathnode)...)
		}
		return childvalues
	}

	if pathnode[0].Name == "" {
		if root.IsBranch() {
			return nil
		}
		return []interface{}{root.Value()}
	}
	// [FIXME]
	if LeafListValueAsKey {
		// if leaflist, ok := root.(*DataLeafList); ok {
		// 	if leaflist.Exist(pathnode[0].Name) {
		// 		return []interface{}{pathnode[0].Name}
		// 	}
		// 	return nil
		// }
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return nil
	}
	branch, ok := root.(*DataBranch)
	if !ok {
		return nil
	}
	var first, last int
	key, pmap, err := keyGen(cschema, pathnode[0])
	if err != nil {
		return nil
	}
	_, prefixsearch := pmap["@prefix"]
	first, last = indexRange(branch, key, prefixsearch)
	if _, ok := pmap["@find-in-order"]; ok {
		node, _ = findByPredicates(branch.children[first:last], pathnode[0].Predicates)
	} else {
		if index, ok := pmap["@index"]; ok {
			first, last, err = jumpToIndex(branch, first, index.(int))
			if err != nil {
				return nil
			}
		}
		node = branch.children[first:last]
	}

	for i := range node {
		if node[i].IsLeaf() {
			if v, ok := pmap["."]; ok {
				if node[i].ValueString() == v {
					childvalues = append(childvalues, node[i].ValueString())
				}
				return childvalues
			}
		}
		childvalues = append(childvalues, findValue(node[i], pathnode[1:])...)
	}
	return childvalues
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
	vlist := findValue(root, pathnode)
	slist := make([]string, 0, len(vlist))
	for i := range vlist {
		slist = append(slist, ValueToString(vlist[i]))
	}
	return slist, nil
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
	vlist := findValue(root, pathnode)
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
			if IsDuplicatedList(schema) {
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
		err := Set(root, path)
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
	if IsDuplicatedList(branch.schema) {
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
