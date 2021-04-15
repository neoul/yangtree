package yangtree

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// ManualKeyCreation - The key data nodes of list nodes are automatically created if set to false.
	ManualKeyCreation bool = false
)

type DataNode interface {
	IsYangDataNode()
	Key() string
	Schema() *yang.Entry
	GetParent() DataNode
	SetParent(parent DataNode, key ...string)

	Set(value ...string) error
	Remove(value ...string) error

	Insert(key string, data DataNode) error
	Delete(key string) error

	Get(key string) DataNode   // Get an child having the key.
	Find(path string) DataNode // Find an exact data node

	// Find all matched data nodes with wildcards (*, ...) and trace back strings (./ and ../)
	// It also allows the namespace-qualified form.
	// https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md
	Retrieve(path string) ([]DataNode, error)

	String() string
	Path() string
	Value() interface{}

	MarshalJSON() ([]byte, error)      // Encoding to JSON
	MarshalJSON_IETF() ([]byte, error) // Encoding to JSON_IETF (rfc7951)

	UnmarshalJSON([]byte) error // Assembling DataNode using JSON or JSON_IETF (rfc7951) input

	// internal interfaces
	unmarshalJSON(jtree interface{}) error
}

func parsePath(path *string, pos, length int) (prefix, pathelem string, end int, testAll bool, err error) {
	begin := pos
	end = pos
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0
	beginBracket := 0
	switch (*path)[end] {
	case '/':
		begin++
	case '=': // ignore data string in path
		end = length
		return
	case '[', ']':
		end = length
		err = fmt.Errorf("yangtree: path '%s' starts with bracket", *path)
		return
	}
	end++
	for end < length {
		switch (*path)[end] {
		case '/':
			if insideBrackets <= 0 {
				if pathelem == "" {
					pathelem = (*path)[begin:end]
				}
				end++
				return
			}
		case '[':
			if (*path)[end-1] != '\\' {
				if insideBrackets <= 0 {
					beginBracket = end
				}
				insideBrackets++
			}
		case ']':
			if (*path)[end-1] != '\\' {
				insideBrackets--
			}
		case '*':
			if insideBrackets == 1 {
				if end+1 < length && (*path)[end-1:end+2] == "=*]" { // * wildcard inside key value
					pathelem = (*path)[begin:beginBracket]
					testAll = true
				}
			}
		case '=':
			if insideBrackets <= 0 {
				if pathelem == "" {
					pathelem = (*path)[begin:end]
				}
				end = length
				return
			}
		case ':':
			if insideBrackets <= 0 {
				prefix = (*path)[begin:end]
				begin = end + 1
			}
		}
		end++
	}
	if pathelem == "" {
		pathelem = (*path)[begin:end]
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
func (branch *DataBranch) GetParent() DataNode { return branch.parent }
func (branch *DataBranch) SetParent(parent DataNode, key ...string) {
	branch.parent = parent
	for i := range key {
		branch.key = key[i]
	}
}

func (branch *DataBranch) Value() interface{} {
	return nil
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
		return "branch.null"
	}
	return "branch." + branch.schema.Name
}

func (branch *DataBranch) Set(value ...string) error {
	// A value become a key upon branch

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

func (branch *DataBranch) Insert(key string, data DataNode) error {
	var err error
	var cschema *yang.Entry

	if data == nil {
		cschema, err = FindSchema(branch.schema, key)
		if err != nil {
			return err
		}
		data, err = New(cschema)
		if err != nil {
			return err
		}
	} else {
		cschema = data.Schema()
		if branch.schema != GetPresentParentSchema(cschema) {
			return fmt.Errorf("yangtree: the schema found by '%s' is not a child of %s", key, branch)
		}
	}

	switch {
	case cschema.IsList():
		if !ManualKeyCreation {
			keyname := strings.Split(cschema.Key, " ")
			keyval, err := ExtractKeyValues(keyname, key)
			if err != nil {
				return err
			}
			for i := range keyval {
				keyschema := cschema.Dir[keyname[i]]
				keynode, err := New(keyschema, keyval[i])
				if err != nil {
					return fmt.Errorf("yangtree: failed to set leaf.%s to '%s'", keyname[i], keyval[i])
				}
				if err := data.Insert(keyname[i], keynode); err != nil {
					return err
				}
			}
		}
	}
	iindex := sort.Search(len(branch.children),
		func(i int) bool { return branch.children[i].Key() >= key })
	branch.children = append(branch.children, nil)
	copy(branch.children[iindex+1:], branch.children[iindex:])
	branch.children[iindex] = data
	data.SetParent(branch, key)
	return nil
}

func (branch *DataBranch) Delete(key string) error {
	length := len(branch.children)
	iindex := sort.Search(length,
		func(i int) bool { return branch.children[i].Key() >= key })
	if iindex < length && branch.children[iindex].Key() == key {
		c := branch.children[iindex]
		branch.children = append(branch.children[:iindex], branch.children[:iindex+1]...)
		c.SetParent(nil, "")
	}
	return nil
}

func (branch *DataBranch) Get(key string) DataNode {
	if key == ".." {
		return branch.parent
	} else if key == "." {
		return branch
	}
	iindex := sort.Search(len(branch.children),
		func(i int) bool { return branch.children[i].Key() >= key })
	if iindex < len(branch.children) && branch.children[iindex].Key() == key {
		return branch.children[iindex]
	}
	return nil
}

func (branch *DataBranch) Find(path string) DataNode {
	if branch == nil {
		return nil
	}
	var err error
	var pos int
	var pathelem string
	var found DataNode
	pathlen := len(path)
	found = branch
Loop:
	for {
		_, pathelem, pos, _, err = parsePath(&path, pos, pathlen)
		if err != nil {
			return nil
		}
		switch pathelem {
		case "":
			if pos >= pathlen {
				break Loop
			}
			return nil
		}

		found := found.Get(pathelem)
		if found == nil {
			return nil
		}
		if pos >= pathlen {
			break
		}
	}
	return found
}

func (branch *DataBranch) Retrieve(path string) ([]DataNode, error) {
	if branch == nil {
		return nil, fmt.Errorf("yangtree: %s found on retrieve", branch)
	}
	pathlen := len(path)
	if pathlen == 0 {
		return []DataNode{branch}, nil
	}
	testAllDescendant := false
	_, pathelem, pos, testAllChildren, err := parsePath(&path, 0, pathlen)
	if err != nil {
		return nil, err
	}
	switch pathelem {
	case "":
		return nil, fmt.Errorf("yangtree: invalid path %s", path)
	case ".":
		return branch.Retrieve(path[pos:])
	case "..":
		return branch.parent.Retrieve(path[pos:])
	case "...":
		testAllDescendant = true
		fallthrough
	case "*":
		testAllChildren = true
		pathelem = ""
	default:
		// exact matching
		cschema, err := FindSchema(branch.schema, pathelem)
		if err != nil {
			return nil, nil
		}
		if cschema.IsList() && cschema.Name == pathelem {
			testAllChildren = true
		}
	}

	// wildcard maching
	if testAllChildren || testAllDescendant {
		var nodes []DataNode
		for i, child := range branch.children {
			if strings.HasPrefix(branch.children[i].Key(), pathelem) {
				n, err := child.Retrieve(path[pos:])
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, n...)
			}
		}
		if testAllDescendant {
			for _, child := range branch.children {
				n, err := child.Retrieve(path)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, n...)
			}
		}
		return nodes, nil
	}

	node := branch.Get(pathelem)
	if node == nil {
		return nil, nil
	}
	return node.Retrieve(path[pos:])
}

func (branch *DataBranch) Key() string {
	return branch.key
}

type DataLeaf struct {
	schema *yang.Entry
	parent DataNode
	value  interface{}
}

func (leaf *DataLeaf) IsYangDataNode()                          {}
func (leaf *DataLeaf) Schema() *yang.Entry                      { return leaf.schema }
func (leaf *DataLeaf) SetParent(parent DataNode, key ...string) { leaf.parent = parent }
func (leaf *DataLeaf) GetParent() DataNode                      { return leaf.parent }
func (leaf *DataLeaf) String() string {
	if leaf == nil {
		return "leaf.null"
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

func (leaf *DataLeaf) Set(value ...string) error {
	for i := range value {
		v, err := StringValueToValue(leaf.schema, leaf.schema.Type, value[i])
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

func (leaf *DataLeaf) Insert(key string, data DataNode) error {
	return fmt.Errorf("yangtree: insert not supported for %s", leaf)
}

func (leaf *DataLeaf) Delete(key string) error {
	return fmt.Errorf("yangtree: delete not supported for %s", leaf)
}

func (leaf *DataLeaf) Get(key string) DataNode {
	return nil
}

func (leaf *DataLeaf) Find(path string) DataNode {
	return nil
}

func (leaf *DataLeaf) Retrieve(path string) ([]DataNode, error) {
	if leaf == nil {
		return nil, fmt.Errorf("yangtree: %s found on retrieve", leaf)
	}
	pathlen := len(path)
	if pathlen == 0 {
		return []DataNode{leaf}, nil
	}
	_, pathelem, pos, _, err := parsePath(&path, 0, pathlen)
	if err != nil {
		return nil, err
	}
	switch pathelem {
	case "":
		return nil, fmt.Errorf("yangtree: invalid path %s", path)
	case ".":
		return leaf.Retrieve(path[pos:])
	case "..":
		if leaf.parent != nil {
			return leaf.parent.Retrieve(path[pos:])
		}
		fallthrough
	case "...", "*":
		return nil, nil
	default:
		return nil, nil
	}
}

func (leaf *DataLeaf) Key() string {
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
func (leaflist *DataLeafList) SetParent(parent DataNode, key ...string) {
	if leaflist == nil {
		return
	}
	leaflist.parent = parent
}
func (leaflist *DataLeafList) GetParent() DataNode {
	if leaflist == nil {
		return nil
	}
	return leaflist.parent
}
func (leaflist *DataLeafList) String() string {
	if leaflist == nil {
		return "leaf-list.null"
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

func (leaflist *DataLeafList) Set(value ...string) error {
	if leaflist == nil {
		return fmt.Errorf("yangtree: %s found on set", leaflist)
	}
	for i := range value {
		v, err := StringValueToValue(leaflist.schema, leaflist.schema.Type, value[i])
		if err != nil {
			return err
		}
		insert := true
		for j := range leaflist.value {
			if leaflist.value[j] == v {
				insert = false
				break
			}
		}
		if insert {
			leaflist.value = append(leaflist.value, v)
		}
	}
	return nil
}

func (leaflist *DataLeafList) Remove(value ...string) error {
	if leaflist == nil {
		return fmt.Errorf("yangtree: %s found on remove", leaflist)
	}
	for i := range value {
		for j := range leaflist.value {
			if leaflist.value[j] == value[i] {
				leaflist.value = append(leaflist.value[:j], leaflist.value[j+1:]...)
				break
			}
		}
	}
	if len(value) == 0 {
		if leaflist.parent == nil {
			return nil
		}
		switch p := leaflist.parent.(type) {
		case *DataBranch:
			// [FIXME] need the performance improvement
			for i := range p.children {
				if p.children[i] == leaflist {
					p.children = append(p.children[:i], p.children[i+1:]...)
					break
				}
			}
			leaflist.parent = nil
		}
	}
	return nil
}

func (leaflist *DataLeafList) Insert(key string, data DataNode) error {
	// if other, ok := data.(*DataLeafList); ok && other != nil {
	// 	for i := range other.value {
	// 		insert := true
	// 		for j := range leaflist.value {
	// 			if other.value[i] == leaflist.value[j] {
	// 				insert = false
	// 				break
	// 			}
	// 		}
	// 		if insert {
	// 			leaflist.value = append(leaflist.value, other.value[i])
	// 		}
	// 	}
	// }
	return leaflist.Set(key)
}

func (leaflist *DataLeafList) Delete(key string) error {
	return leaflist.Remove(key)
}

// Get finds the key from its value.
func (leaflist *DataLeafList) Get(key string) DataNode {
	for i := range leaflist.value {
		if leaflist.value[i] == key {
			return leaflist
		}
	}
	return nil
}

// Get finds the key from its value.
func (leaflist *DataLeafList) Find(path string) DataNode {
	for i := range leaflist.value {
		if leaflist.value[i] == path {
			return leaflist
		}
	}
	return nil
}

func (leaflist *DataLeafList) Retrieve(path string) ([]DataNode, error) {
	if leaflist == nil {
		return nil, fmt.Errorf("yangtree: %s found on retrieve", leaflist)
	}
	pathlen := len(path)
	if pathlen == 0 {
		return []DataNode{leaflist}, nil
	}
	_, pathelem, pos, _, err := parsePath(&path, 0, pathlen)
	if err != nil {
		return nil, err
	}
	switch pathelem {
	case "":
		return nil, fmt.Errorf("yangtree: invalid path %s", path)
	case ".":
		return leaflist.Retrieve(path[pos:])
	case "..":
		if leaflist.parent != nil {
			return leaflist.parent.Retrieve(path[pos:])
		}
		fallthrough
	case "...", "*":
		return nil, nil
	default:
		node := leaflist.Find(path[pos:])
		if node == nil {
			return nil, nil
		}
		return []DataNode{node}, nil
	}
}

func (leaflist *DataLeafList) Key() string {
	return leaflist.schema.Name
}

func New(schema *yang.Entry, value ...string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("yangtree: schema.null on new")
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
		branch := &DataBranch{
			schema:   schema,
			children: []DataNode{},
		}
		// for key, cschema := range schema.Dir {
		// 	branch.Set(key)
		// }
		newdata = branch
	default: // container, case, etc.
		newdata = &DataBranch{
			schema:   schema,
			children: []DataNode{},
		}
	}
	return newdata, nil
}

func Insert(root DataNode, path string, value ...string) error {
	if root == nil {
		return fmt.Errorf("yangtree: data node is null")
	}
	var err error
	var pos int
	var pathelem string
	var created DataNode
	pathlen := len(path)
Loop:
	for {
		_, pathelem, pos, _, err = parsePath(&path, pos, pathlen)
		if err != nil {
			return err
		}
		switch pathelem {
		case "":
			if pos >= pathlen {
				break Loop
			}
			return fmt.Errorf("yangtree: invalid path %s", path)
		}
		found := root.Get(pathelem)
		if found == nil {
			schema := root.Schema()
			if !schema.IsLeafList() {
				schema, err = FindSchema(root.Schema(), pathelem)
				if err != nil {
					return err
				}
			}
			found, err = New(schema)
			if err != nil {
				return err
			}
			if err := root.Insert(pathelem, found); err != nil {
				return err
			}
			if created == nil {
				created = found
			}
		}
		root = found
		if pos >= pathlen {
			break
		}
	}

	err = root.Set(value...)
	if err != nil {
		if created != nil {
			created.Remove()
		}
		return err
	}
	return nil
}

func Delete(root DataNode, path string, value ...string) error {
	if root == nil {
		return fmt.Errorf("yangtree: data node is null")
	}
	var err error
	var pos int
	var pathelem string
	pathlen := len(path)
Loop:
	for {
		_, pathelem, pos, _, err = parsePath(&path, pos, pathlen)
		if err != nil {
			return err
		}
		switch pathelem {
		case "":
			if pos >= pathlen {
				break Loop
			}
			return fmt.Errorf("yangtree: invalid path %s", path)
		}

		found := root.Get(pathelem)
		if found == nil {
			if _, ok := root.(*DataLeafList); ok {
				value = append(value, pathelem)
			}
			break Loop
		}

		root = found
		if pos >= pathlen {
			break
		}
	}
	return root.Remove(value...)
}
