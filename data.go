package yangtree

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// LeafListValueAsKey - leaf-list value can be represented to the path if it is set to true.
	LeafListValueAsKey bool = true
)

type DataNode interface {
	IsYangDataNode()
	Key() string
	Schema() *yang.Entry
	Parent() DataNode

	Insert(child DataNode) error
	Delete(child DataNode) error

	Set(value ...string) error
	Remove(value ...string) error

	// New() creates a cild using a key and values
	// key is the element of xpath combined with xpath predicates.
	// For example, /interfaces/interface[name=VALUE]
	New(key string, value ...string) (DataNode, error)

	Get(key string) []DataNode       // Get children having the key
	Lookup(prefix string) []DataNode // Get all children that starts with prefix

	Len() int                    // Len() returns the length of children
	Index(key string) (int, int) // Index() finds all children and returns the indexes of them.
	Child(index int) DataNode    // Child() gets the child of the index.

	String() string
	Path() string
	Value() interface{}
	ValueString() string

	MarshalJSON() ([]byte, error)      // Encoding to JSON
	MarshalJSON_IETF() ([]byte, error) // Encoding to JSON_IETF (rfc7951)

	UnmarshalJSON([]byte) error // Assembling DataNode using JSON or JSON_IETF (rfc7951) input
}

func SearchInOrder(n int, f func(int) bool) int {
	i := 0
	for ; i < n; i++ {
		if f(i) {
			break
		}
	}
	return i
}

func isValid(node DataNode) bool {
	if node == nil {
		return false
	}
	if node.Schema() == nil {
		return false
	}
	return true
}

func setParent(node DataNode, parent *DataBranch, key string) {
	switch c := node.(type) {
	case *DataBranch:
		c.parent = parent
		c.key = key
	case *DataLeaf:
		c.parent = parent
	case *DataLeafList:
		c.parent = parent
	}
}

// updateNode() updates the first matched node or replaces it to the child if replace is true.
func updateNode(parent *DataBranch, child DataNode) error {
	if !isValid(child) {
		return fmt.Errorf("yangtree: invalid child node")
	}
	if !isValid(parent) {
		return fmt.Errorf("yangtree: invalid parent node")
	}
	if child.Parent() != nil {
		return fmt.Errorf("yangtree: the node is already appended to a parent")
	}
	if parent.schema != GetPresentParentSchema(child.Schema()) {
		return fmt.Errorf("yangtree: '%s' is not a child of %s", child, parent)
	}
	length := len(parent.children)
	key := child.Key()
	i := sort.Search(length,
		func(j int) bool {
			return key <= parent.children[j].Key()
		})
	// it just upates the first matched node.
	// if not matched, it inserts the child to the proper location.
	if i < length && parent.children[i].Key() == key && !IsDuplicatedList(parent.children[i].Schema()) {
		dest := parent.children[i]
		if dest.Schema() != child.Schema() {
			return fmt.Errorf("yangtree: unable to update different schema data")
		}
		switch node := child.(type) {
		case *DataLeaf, *DataLeafList:
			return dest.Set(child.ValueString())
		case *DataBranch:
			d, ok := dest.(*DataBranch)
			if !ok {
				return fmt.Errorf("yangtree: unable to update type mismatched node")
			}
			for j := range node.children {
				if err := updateNode(d, node.children[j]); err != nil {
					return err
				}
			}
		}
		return nil
	} else {
		for ; i < length; i++ {
			if parent.children[i].Key() > key {
				break
			}
		}
	}
	parent.children = append(parent.children, nil)
	copy(parent.children[i+1:], parent.children[i:])
	parent.children[i] = child
	setParent(child, parent, key)
	return nil
}

func deleteNode(parent DataNode, child DataNode) error {
	if isValid(child) {
		return fmt.Errorf("yangtree: invalid child node")
	}
	if isValid(parent) {
		return fmt.Errorf("yangtree: invalid parent node")
	}
	if child.Parent() == nil {
		return fmt.Errorf("yangtree: '%s' is already removed from a parent", child)
	}
	p, ok := parent.(*DataBranch)
	if !ok {
		return fmt.Errorf("yangtree: unable to remove a child a non-branch node")
	}
	if p.schema != GetPresentParentSchema(child.Schema()) {
		return fmt.Errorf("yangtree: '%s' is not a child of %s", child, p)
	}
	length := len(p.children)
	key := child.Key()
	i := sort.Search(length,
		func(j int) bool {
			return key <= p.children[j].Key()
		})
	if i < length {
		if p.children[i] == child {
			c := p.children[i]
			p.children = append(p.children[:i], p.children[:i+1]...)
			setParent(c, nil, "")
			return nil
		}
	}
	return fmt.Errorf("yangtree: %s not found on delete", child)
}

// indexNode() returns the index of a child related to the key
func indexNode(parent *DataBranch, key string, prefixKey bool) (i, max int) {
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

type DataBranch struct {
	schema   *yang.Entry
	parent   DataNode
	key      string
	children []DataNode
}

func (branch *DataBranch) IsYangDataNode()     {}
func (branch *DataBranch) Schema() *yang.Entry { return branch.schema }
func (branch *DataBranch) Parent() DataNode    { return branch.parent }
func (branch *DataBranch) Value() interface{} {
	return nil
}

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
	if branch.key == "" {
		return ""
	}
	if branch.parent != nil {
		return branch.parent.Path() + "/" + branch.key
	}
	return "/" + branch.key
}

func (branch *DataBranch) String() string {
	if branch == nil {
		return "branch.nil"
	}
	return "branch." + branch.Key()
}

func (branch *DataBranch) New(key string, value ...string) (DataNode, error) {
	if !isValid(branch) {
		return nil, fmt.Errorf("yangtree: invalid branch node")
	}
	pathnode, err := ParsePath(&key)
	if err != nil {
		return nil, err
	}
	if len(pathnode) == 0 {
		return nil, fmt.Errorf("yangtree: invalid key inserted for new")
	}
	return newChild(branch, pathnode[0], value...)
}

func (branch *DataBranch) Set(value ...string) error {
	for i := range value {
		err := branch.UnmarshalJSON([]byte(value[i]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (branch *DataBranch) Remove(value ...string) error {
	if branch == nil {
		return nil
	}
	if branch.parent == nil {
		return nil
	}
	switch p := branch.parent.(type) {
	case *DataBranch:
		// [FIXME] need the performance improvement
		for i := range p.children {
			if p.children[i] == branch {
				p.children = append(p.children[:i], p.children[i+1:]...)
				break
			}
		}
		branch.parent = nil
	}
	return nil
}

func (branch *DataBranch) Insert(child DataNode) error {
	if !isValid(child) {
		return fmt.Errorf("yangtree: invalid child node")
	}
	if !isValid(branch) {
		return fmt.Errorf("yangtree: invalid parent node")
	}
	if child.Parent() != nil {
		return fmt.Errorf("yangtree: the node is already appended to a parent")
	}
	if branch.Schema() != GetPresentParentSchema(child.Schema()) {
		return fmt.Errorf("yangtree: '%s' is not a child of %s", child, branch)
	}
	length := len(branch.children)
	key := child.Key()
	i := sort.Search(length,
		func(j int) bool {
			return key <= branch.children[j].Key()
		})

	// replace the data node if it is exists or add the child.
	if i < length && branch.children[i].Key() == key &&
		!IsDuplicatedList(branch.children[i].Schema()) {
		setParent(branch.children[i], nil, "")
		branch.children[i] = child
		setParent(child, branch, key)
		return nil
	} else {
		for ; i < length; i++ {
			if branch.children[i].Key() > key {
				break
			}
		}
	}
	branch.children = append(branch.children, nil)
	copy(branch.children[i+1:], branch.children[i:])
	branch.children[i] = child
	setParent(child, branch, key)
	return nil
}

func (branch *DataBranch) Delete(child DataNode) error {
	return deleteNode(branch, child)
}

func (branch *DataBranch) Get(key string) []DataNode {
	switch key {
	case ".":
		return []DataNode{branch}
	case "..":
		return []DataNode{branch.parent}
	case "*":
		return branch.children
	case "...":
		return findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: PathSelectAllMatched}})
	default:
		i, max := indexNode(branch, key, false)
		if i < max {
			return branch.children[i:max]
		}
	}
	return nil
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
			&PathNode{Name: "...", Select: PathSelectAllMatched}})
	default:
		i, max := indexNode(branch, prefix, true)
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
	return indexNode(branch, key, false)
}

func (branch *DataBranch) Len() int {
	return len(branch.children)
}

func (branch *DataBranch) Key() string {
	if !isValid(branch) {
		return ""
	}
	if branch.parent != nil {
		return branch.key
	}
	switch {
	case IsUniqueList(branch.schema):
		keyname := strings.Split(branch.schema.Key, " ")
		key := make([]string, 0, len(keyname)+1)
		key = append(key, branch.schema.Name)
		for i := range keyname {
			for j := range branch.children {
				// [FIXME]
				if branch.children[j].Key() == keyname[i] {
					key = append(key, "["+keyname[i]+"="+branch.children[j].ValueString()+"]")
				}
			}
		}
		return strings.Join(key, "")
	default:
		return branch.schema.Name
	}
}

type DataLeaf struct {
	schema *yang.Entry
	parent DataNode
	value  interface{}
}

func (leaf *DataLeaf) IsYangDataNode()     {}
func (leaf *DataLeaf) Schema() *yang.Entry { return leaf.schema }
func (leaf *DataLeaf) Parent() DataNode    { return leaf.parent }
func (leaf *DataLeaf) String() string {
	if leaf == nil {
		return "leaf.nil"
	}
	return "leaf." + leaf.schema.Name
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

func (leaf *DataLeaf) Value() interface{} {
	return leaf.value
}

func (leaf *DataLeaf) ValueString() string {
	return ValueToString(leaf.value)
}

func (leaf *DataLeaf) New(key string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("yangtree: insert not supported for %s", leaf)
}

func (leaf *DataLeaf) Set(value ...string) error {
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
	switch p := leaf.parent.(type) {
	case *DataBranch:
		// [FIXME] need the performance improvement
		for i := range p.children {
			if p.children[i] == leaf {
				p.children = append(p.children[:i], p.children[i+1:]...)
				break
			}
		}
		leaf.parent = nil
	}
	return nil
}

func (leaf *DataLeaf) Insert(child DataNode) error {
	return fmt.Errorf("yangtree: insert not supported for %s", leaf)
}

func (leaf *DataLeaf) Delete(child DataNode) error {
	return fmt.Errorf("yangtree: delete not supported for %s", leaf)
}

func (leaf *DataLeaf) Get(key string) []DataNode {
	return nil
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
	return 0
}

func (leaf *DataLeaf) Key() string {
	if !isValid(leaf) {
		return ""
	}
	return leaf.schema.Name
}

// DataLeafList (leaf-list data node)
// It can be set by the key
type DataLeafList struct {
	schema *yang.Entry
	parent DataNode
	value  []interface{}
}

func (leaflist *DataLeafList) IsYangDataNode() {}
func (leaflist *DataLeafList) Schema() *yang.Entry {
	if leaflist == nil {
		return nil
	}
	return leaflist.schema
}
func (leaflist *DataLeafList) Parent() DataNode {
	if leaflist == nil {
		return nil
	}
	return leaflist.parent
}
func (leaflist *DataLeafList) String() string {
	if leaflist == nil {
		return "leaf-list.nil"
	}
	return "leaf-list." + leaflist.schema.Name
}

func (leaflist *DataLeafList) Path() string {
	if leaflist == nil {
		return ""
	}
	if leaflist.parent != nil {
		return leaflist.parent.Path() + "/" + leaflist.Key()
	}
	return "/" + leaflist.Key()
}

func (leaflist *DataLeafList) Value() interface{} {
	return leaflist.value
}

func (leaflist *DataLeafList) ValueString() string {
	return ValueToString(leaflist.value)
}

func (leaflist *DataLeafList) New(key string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("yangtree: insert not supported for %s", leaflist)
}

func (leaflist *DataLeafList) Set(value ...string) error {
	if leaflist == nil {
		return fmt.Errorf("yangtree: %s found on set", leaflist)
	}
	if len(value) == 1 {
		if strings.HasPrefix(value[0], "[") && strings.HasSuffix(value[0], "]") {
			return leaflist.UnmarshalJSON([]byte(value[0]))
		}
	}
	for i := range value {
		length := len(leaflist.value)
		index := sort.Search(length,
			func(j int) bool {
				return ValueToString(leaflist.value[j]) >= value[i]
			})
		v, err := StringToValue(leaflist.schema, leaflist.schema.Type, value[i])
		if err != nil {
			return err
		}
		leaflist.value = append(leaflist.value, nil)
		copy(leaflist.value[index+1:], leaflist.value[index:])
		leaflist.value[index] = v
	}
	return nil
}

func (leaflist *DataLeafList) Remove(value ...string) error {
	if leaflist == nil {
		return fmt.Errorf("yangtree: %s found on remove", leaflist)
	}
	for i := range value {
		length := len(leaflist.value)
		index := sort.Search(length,
			func(j int) bool {
				return ValueToString(leaflist.value[j]) >= value[i]
			})
		if index < length && ValueToString(leaflist.value[index]) == value[i] {
			leaflist.value = append(leaflist.value[:index], leaflist.value[:index+1]...)
		}
	}
	// remove itself if there is no value inserted.
	if len(value) == 0 {
		if leaflist.parent == nil {
			return nil
		}
		if branch, ok := leaflist.parent.(*DataBranch); ok {
			branch.Delete(leaflist)
		}
	}
	return nil
}

func (leaflist *DataLeafList) Insert(child DataNode) error {
	return fmt.Errorf("yangtree: insert not supported for %s", leaflist)
}

func (leaflist *DataLeafList) Delete(child DataNode) error {
	return fmt.Errorf("yangtree: delete not supported for %s", leaflist)
}

func (leaflist *DataLeafList) Get(key string) []DataNode {
	return nil
}

func (leaflist *DataLeafList) Lookup(prefix string) []DataNode {
	return nil
}

func (leaflist *DataLeafList) Child(index int) DataNode {
	return nil
}

// [FIXME] Should it be supported?
func (leaflist *DataLeafList) Index(key string) (int, int) {
	return 0, 0
}

// [FIXME] Should it be supported?
func (leaflist *DataLeafList) Len() int {
	return 0
}

// Get finds the key from its value.
// [FIXME] Should it be supported?
func (leaflist *DataLeafList) Exist(value string) bool {
	length := len(leaflist.value)
	i := sort.Search(length,
		func(j int) bool {
			return ValueToString(leaflist.value[j]) >= value
		})
	return i < length && ValueToString(leaflist.value[i]) == value
}

func (leaflist *DataLeafList) Key() string {
	if !isValid(leaflist) {
		return ""
	}
	return leaflist.schema.Name
}

func newChild(parent *DataBranch, pathnode *PathNode, value ...string) (DataNode, error) {
	cschema := GetSchema(parent.Schema(), pathnode.Name)
	if cschema == nil {
		return nil, fmt.Errorf("yangtree: schema.%s not fond from schema.%s", pathnode.Name, parent.schema.Name)
	}
	m, err := PredicatesMap(pathnode.Predicates)
	if err != nil {
		return nil, err
	}
	child, err := New(cschema, value...)
	if err != nil {
		return nil, err
	}
	if IsUniqueList(cschema) {
		keyname := strings.Split(cschema.Key, " ")
		for i := range keyname {
			v, ok := m[keyname[i]]
			if !ok {
				return nil, fmt.Errorf("yangtree: %s of %s must be present in the path", keyname[i], cschema.Name)
			}
			kn, err := New(GetSchema(cschema, keyname[i]), v)
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

func New(schema *yang.Entry, value ...string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("yangtree: schema.nil inserted for new")
	}
	var err error
	var newdata DataNode
	switch {
	case schema.Dir == nil && schema.ListAttr != nil: // leaf-list
		leaflist := &DataLeafList{
			schema: schema,
		}
		err := leaflist.Set(value...)
		if err != nil {
			return nil, err
		}
		newdata = leaflist
	case schema.Dir == nil: // leaf
		leaf := &DataLeaf{
			schema: schema,
		}
		if len(value) > 0 {
			err = leaf.Set(value...)
		} else if schema.Default != "" {
			err = leaf.Set(schema.Default)
		}
		if err != nil {
			return nil, err
		}
		newdata = leaf
	case schema.ListAttr != nil: // list
		fallthrough
	default: // container, case, etc.
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
	return newdata, nil
}

func setValue(root DataNode, pathnode []*PathNode, value ...string) error {
	if len(pathnode) == 0 {
		return root.Set(value...)
	}
	switch pathnode[0].Select {
	case PathSelectSelf:
		return setValue(root, pathnode[1:], value...)
	case PathSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("yangtree: the parent of %s is not present in the path", root)
		}
		root = root.Parent()
		return setValue(root, pathnode[1:], value...)
	case PathSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case PathSelectAllChildren:
		i, max := 0, root.Len()
		for ; i < max; i++ {
			if err := setValue(root.Child(i), pathnode[1:], value...); err != nil {
				return err
			}
		}
		return nil
	case PathSelectAllMatched:
		i, max := 0, root.Len()
		for ; i < max; i++ {
			if err := setValue(root.Child(i), pathnode[1:], value...); err != nil {
				return err
			}
			if err := setValue(root.Child(i), pathnode, value...); err != nil {
				return err
			}
		}
		return nil
	}

	if pathnode[0].Name == "" {
		return root.Set(value...)
	}
	if LeafListValueAsKey {
		if root.Schema().IsLeafList() {
			value = append(value, pathnode[0].Name)
			return root.Set(value...)
		}
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("yangtree: schema.%s not found from schema.%s", pathnode[0].Name, root.Schema().Name)
	}
	key, err := KeyGen(cschema, pathnode[0].Predicates)
	if err != nil {
		return err
	}
	i, max := root.Index(key)
	switch {
	case IsDuplicatedList(cschema):
		i = max // always created
	}
	if i == max {
		child, err := newChild(root.(*DataBranch), pathnode[0])
		if err != nil {
			return err
		}
		err = setValue(child, pathnode[1:], value...)
		if err != nil {
			child.Remove()
		}
		return err
	}
	for ; i < max; i++ {
		if err := setValue(root.Child(i), pathnode[1:], value...); err != nil {
			return err
		}
	}
	return nil
}

// Set sets a value or values to the target DataNode in the path.
// If the target DataNode is a branch node, the value must be json or json_ietf bytes.
// If the target data node is a leaf or a leaf-list node, the value should be string.
func Set(root DataNode, path string, value ...string) error {
	if !isValid(root) {
		return fmt.Errorf("yangtree: invalid root node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return setValue(root, pathnode, value...)
}

func deleteValue(root DataNode, pathnode []*PathNode, value ...string) error {
	if len(pathnode) == 0 {
		return root.Remove(value...)
	}
	switch pathnode[0].Select {
	case PathSelectSelf:
		return deleteValue(root, pathnode[1:], value...)
	case PathSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("yangtree: the parent of %s is not present in the path", root)
		}
		root = root.Parent()
		return deleteValue(root, pathnode[1:], value...)
	case PathSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case PathSelectAllChildren:
		i, max := 0, root.Len()
		for ; i < max; i++ {
			if err := deleteValue(root.Child(i), pathnode[1:], value...); err != nil {
				return err
			}
		}
		return nil
	case PathSelectAllMatched:
		i, max := 0, root.Len()
		for ; i < max; i++ {
			if err := deleteValue(root.Child(i), pathnode[1:], value...); err != nil {
				return err
			}
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
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("yangtree: schema.%s not found from schema.%s", pathnode[0].Name, root.Schema().Name)
	}
	key, err := KeyGen(cschema, pathnode[0].Predicates)
	if err != nil {
		return err
	}
	i, max := root.Index(key)
	switch {
	case IsDuplicatedList(cschema):
		// always remove the first node
		if i < max {
			max = i + 1
		}
	}
	for ; i < max; i++ {
		if err := deleteValue(root.Child(i), pathnode[1:], value...); err != nil {
			return err
		}
	}
	return nil
}

func Delete(root DataNode, path string, value ...string) error {
	if !isValid(root) {
		return fmt.Errorf("yangtree: invalid root node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return deleteValue(root, pathnode, value...)
}

func findNode(root DataNode, pathnode []*PathNode) []DataNode {
	if len(pathnode) == 0 {
		return []DataNode{root}
	}
	var children []DataNode
	switch pathnode[0].Select {
	case PathSelectSelf:
		return findNode(root, pathnode[1:])
	case PathSelectParent:
		if root.Parent() == nil {
			return nil
		}
		root = root.Parent()
		return findNode(root, pathnode[1:])
	case PathSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case PathSelectAllChildren:
		i, max := 0, root.Len()
		for ; i < max; i++ {
			children = append(children, findNode(root.Child(i), pathnode[1:])...)
		}
		return children
	case PathSelectAllMatched:
		children = append(children, findNode(root, pathnode[1:])...)
		i, max := 0, root.Len()
		for ; i < max; i++ {
			children = append(children, findNode(root.Child(i), pathnode[1:])...)
			children = append(children, findNode(root.Child(i), pathnode)...)
		}
		return children
	}

	if pathnode[0].Name == "" {
		return []DataNode{root}
	}
	// [FIXME]
	if LeafListValueAsKey {
		if leaflist, ok := root.(*DataLeafList); ok {
			if leaflist.Exist(pathnode[0].Name) {
				return []DataNode{root}
			}
			return nil
		}
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return nil
	}
	key, err := KeyGen(cschema, pathnode[0].Predicates)
	if err != nil {
		return nil
	}
	branch := root.(*DataBranch)
	i, max := indexNode(branch, key, true)
	switch {
	case IsDuplicatedList(cschema):
		// [FIXME]
	}
	for ; i < max; i++ {
		children = append(children, findNode(root.Child(i), pathnode[1:])...)
	}
	return children
}

func Find(root DataNode, path string) ([]DataNode, error) {
	if !isValid(root) {
		return nil, fmt.Errorf("yangtree: invalid root node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	return findNode(root, pathnode), nil
}
