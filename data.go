package yangtree

import (
	"fmt"
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
	GetParent() *DataBranch
	SetParent(parent *DataBranch, key ...string)

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

	MarshalJSON() ([]byte, error)      // Encoding to JSON
	MarshalJSON_IETF() ([]byte, error) // Encoding to JSON_IETF
}

type DataBranch struct {
	schema   *yang.Entry
	parent   *DataBranch
	key      string
	Children map[string]DataNode
}

func (branch *DataBranch) IsYangDataNode()        {}
func (branch *DataBranch) Schema() *yang.Entry    { return branch.schema }
func (branch *DataBranch) GetParent() *DataBranch { return branch.parent }
func (branch *DataBranch) SetParent(parent *DataBranch, key ...string) {
	branch.parent = parent
	for i := range key {
		branch.key = key[i]
	}
}
func (branch *DataBranch) String() string {
	if branch == nil {
		return "branch.null"
	}
	return "branch." + branch.schema.Name
}

func (branch *DataBranch) Set(value ...string) error {
	return nil
}

func (branch *DataBranch) Remove(value ...string) error {
	if branch == nil {
		return nil
	}
	if branch.parent != nil {
		delete(branch.parent.Children, branch.key)
		branch.parent = nil
	}
	return nil
}

func (branch *DataBranch) Insert(key string, data DataNode) error {
	cschema, err := FindSchema(branch.schema, key)
	if err != nil {
		return err
	}
	if data == nil {
		data, err = New(cschema)
		if err != nil {
			return err
		}
	} else {
		if cschema != data.Schema() {
			return fmt.Errorf("yangtree: invalid %s inserted for '%s'",
				data, cschema.Name)
		}
	}

	switch {
	case cschema.IsList():
		if !ManualKeyCreation {
			keyname := strings.Split(cschema.Key, " ")
			keyval, err := ExtractKeys(keyname, key)
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
	branch.Children[key] = data
	data.SetParent(branch, key)
	return nil
}

func (branch *DataBranch) Delete(key string) error {
	if c := branch.Children[key]; c != nil {
		delete(branch.Children, key)
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
	return branch.Children[key]
}

func (branch *DataBranch) Find(path string) DataNode {
	if branch == nil {
		return nil
	}
	key, err := SplitPath(branch.Schema(), path)
	if err != nil {
		return nil
	}
	var node DataNode
	node = branch
	for i := range key {
		node = node.Get(key[i])
		if node == nil {
			return nil
		}
	}
	return node
}

// const metachars map[string]struct{} = map[string]struct{} {
// 	"../": struct{},
// 	"./": struct{},
// 	"*]": struct{},
// 	"/*": struct{},
// 	"/...": struct{},
// }

func parsePath(path *string, pos, length int) (string, int, error) {
	begin := pos
	end := pos
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0
	switch (*path)[end] {
	case '/':
		begin++
	case '=': // ignore data string in path
		return "", length, nil
	case '[', ']':
		return "", length, fmt.Errorf("yangtree: path '%s' starts with bracket", *path)
	}
	end++
	for end < length {
		switch (*path)[end] {
		case '/':
			if insideBrackets <= 0 {
				return (*path)[begin:end], end + 1, nil
			}
		case '[':
			if (*path)[end-1] != '\\' {
				insideBrackets++
			}
		case ']':
			if (*path)[end-1] != '\\' {
				insideBrackets--
			}
		case '=':
			if insideBrackets <= 0 {
				return (*path)[begin:end], end + 1, nil
			}
		}
		end++
	}
	return (*path)[begin:end], end + 1, nil
}

func (branch *DataBranch) Retrieve(path string) ([]DataNode, error) {
	return nil, nil
	// var node DataNode
	// if branch == nil {
	// 	return nil, fmt.Errorf("yangtree: %s found on retrieve", branch)
	// }
	// pos := 0
	// length := len(path)
	// schema := branch.schema
	// node = branch
	// for pos < length {
	// 	pathelem, next, err := parsePath(&path, pos, length)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	schema, err = FindSchema(schema, pathelem)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	node = node.Get(pathelem)
	// 	if node == nil {
	// 		return nil, fmt.Errorf("yangtree: '%s' not found in %s", pathelem, branch)
	// 	}
	// 	pos = next
	// }
	// key, err := SplitPath(branch.Schema(), path)
	// if err != nil {
	// 	return nil, err
	// }
	// var node DataNode
	// node = branch
	// for i := range key {
	// 	node = node.Get(key[i])
	// 	if node == nil {
	// 		return nil, fmt.Errorf("yangtree: '%s' not found in %s", key[i], branch)
	// 	}
	// }
	// return []DataNode{node}, nil
}

func (branch *DataBranch) Key() string {
	return branch.key
}

type DataLeaf struct {
	schema *yang.Entry
	parent *DataBranch
	Value  interface{}
}

func (leaf *DataLeaf) IsYangDataNode()                             {}
func (leaf *DataLeaf) Schema() *yang.Entry                         { return leaf.schema }
func (leaf *DataLeaf) SetParent(parent *DataBranch, key ...string) { leaf.parent = parent }
func (leaf *DataLeaf) GetParent() *DataBranch                      { return leaf.parent }
func (leaf *DataLeaf) String() string {
	if leaf == nil {
		return "leaf.null"
	}
	return "leaf." + leaf.schema.Name
}

func (leaf *DataLeaf) Set(value ...string) error {
	for i := range value {
		v, err := Set(leaf.schema, leaf.schema.Type, value[i])
		if err != nil {
			return err
		}
		leaf.Value = v
	}
	// fmt.Printf("\n##leaf.Value Type %T %v\n", leaf.Value, leaf.Value)
	return nil
}

func (leaf *DataLeaf) Remove(value ...string) error {
	delete(leaf.parent.Children, leaf.schema.Name)
	leaf.parent = nil
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
	return nil, fmt.Errorf("yangtree: retrieve not supported for %s", leaf)
}

func (leaf *DataLeaf) Key() string {
	return leaf.schema.Name
}

// DataLeafList (leaf-list data node)
// It can be set by the key
type DataLeafList struct {
	schema *yang.Entry
	parent *DataBranch
	Value  []interface{}
}

func (leaflist *DataLeafList) IsYangDataNode() {}
func (leaflist *DataLeafList) Schema() *yang.Entry {
	if leaflist == nil {
		return nil
	}
	return leaflist.schema
}
func (leaflist *DataLeafList) SetParent(parent *DataBranch, key ...string) {
	if leaflist == nil {
		return
	}
	leaflist.parent = parent
}
func (leaflist *DataLeafList) GetParent() *DataBranch {
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

func (leaflist *DataLeafList) Set(value ...string) error {
	if leaflist == nil {
		return fmt.Errorf("yangtree: %s found on set", leaflist)
	}
	for i := range value {
		v, err := Set(leaflist.schema, leaflist.schema.Type, value[i])
		if err != nil {
			return err
		}
		insert := true
		for j := range leaflist.Value {
			if leaflist.Value[j] == v {
				insert = false
				break
			}
		}
		if insert {
			leaflist.Value = append(leaflist.Value, v)
		}
	}
	return nil
}

func (leaflist *DataLeafList) Remove(value ...string) error {
	if leaflist == nil {
		return fmt.Errorf("yangtree: %s found on remove", leaflist)
	}
	for i := range value {
		for j := range leaflist.Value {
			if leaflist.Value[j] == value[i] {
				leaflist.Value = append(leaflist.Value[:j], leaflist.Value[j+1:]...)
				break
			}
		}
	}
	if len(value) == 0 {
		if leaflist.parent != nil {
			delete(leaflist.parent.Children, leaflist.schema.Name)
			leaflist.parent = nil
		}
	}
	return nil
}

func (leaflist *DataLeafList) Insert(key string, data DataNode) error {
	if other, ok := data.(*DataLeafList); ok && other != nil {
		for i := range other.Value {
			insert := true
			for j := range leaflist.Value {
				if other.Value[i] == leaflist.Value[j] {
					insert = false
					break
				}
			}
			if insert {
				leaflist.Value = append(leaflist.Value, other.Value[i])
			}
		}
	}
	return leaflist.Set(key)
}

func (leaflist *DataLeafList) Delete(key string) error {
	return leaflist.Remove(key)
}

// Get finds the key from its value.
func (leaflist *DataLeafList) Get(key string) DataNode {
	for i := range leaflist.Value {
		if leaflist.Value[i] == key {
			return leaflist
		}
	}
	return nil
}

// Get finds the key from its value.
func (leaflist *DataLeafList) Find(path string) DataNode {
	for i := range leaflist.Value {
		if leaflist.Value[i] == path {
			return leaflist
		}
	}
	return nil
}

func (leaflist *DataLeafList) Retrieve(path string) ([]DataNode, error) {
	node := leaflist.Find(path)
	if node == nil {
		return nil, nil
	}
	return []DataNode{node}, nil
}

func (leaflist *DataLeafList) Key() string {
	return leaflist.schema.Name
}

func New(schema *yang.Entry, value ...string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("yangtree: schema.null on new")
	}
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
		err := leaf.Set(value...)
		if err != nil {
			return nil, err
		}
		newdata = leaf
	case schema.ListAttr != nil: // list
		newdata = &DataBranch{
			schema:   schema,
			Children: map[string]DataNode{},
		}
	default: // container, case, etc.
		newdata = &DataBranch{
			schema:   schema,
			Children: map[string]DataNode{},
		}
	}
	return newdata, nil
}

func Insert(root DataNode, path string, value ...string) error {
	if root == nil {
		return fmt.Errorf("yangtree: data node is null")
	}
	key, err := SplitPath(root.Schema(), path)
	if err != nil {
		return err
	}
	var createdTop DataNode
	for i := range key {
		found := root.Get(key[i])
		if found == nil {
			schema := root.Schema()
			if !schema.IsLeafList() {
				schema, err = FindSchema(root.Schema(), key[i])
				if err != nil {
					return err
				}
			}
			found, err = New(schema)
			if err != nil {
				return err
			}
			if err := root.Insert(key[i], found); err != nil {
				return err
			}
			if createdTop == nil {
				createdTop = found
			}
		}
		root = found
	}
	err = root.Set(value...)
	if err != nil {
		if createdTop != nil {
			createdTop.Remove()
		}
		return err
	}
	return nil
}

func Delete(root DataNode, path string, value ...string) error {
	if root == nil {
		return fmt.Errorf("yangtree: data node is null")
	}
	key, err := SplitPath(root.Schema(), path)
	if err != nil {
		return err
	}

	for i := range key {
		if _, ok := root.(*DataLeafList); ok {
			value = append(value, key[i:]...)
		}
		found := root.Get(key[i])
		if found == nil {
			return fmt.Errorf("yangtree: '%s' not found in %s", key[i], root)
		}
		root = found
	}
	return root.Remove(value...)
}
