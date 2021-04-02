package yangtree

import (
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

type DataNode interface {
	IsYangData()
	Schema() *yang.Entry
	GetParent() *DataBranch
	SetParent(parent *DataBranch, key ...string)

	Set(value ...string) error
	Remove() error

	Insert(key string, data DataNode) error
	Delete(key string) error

	Find(key string) DataNode
}

type DataBranch struct {
	schema   *yang.Entry
	parent   *DataBranch
	key      string
	Children map[string]DataNode
}

func (branch *DataBranch) IsYangData()            {}
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
		return ""
	}
	return "branch " + branch.schema.Name
}

func (branch *DataBranch) Set(value ...string) error {
	return nil
}

func (branch *DataBranch) Remove() error {
	delete(branch.parent.Children, branch.key)
	branch.parent = nil
	return nil
}

func (branch *DataBranch) Insert(key string, data DataNode) error {
	cschema, err := FindSchema(branch.schema, key)
	if err != nil {
		return err
	}
	if data == nil {
		data = New(cschema)
	} else {
		if cschema != data.Schema() {
			return fmt.Errorf("yangtree: unexpected schema '%v' inserted", data.Schema().Name)
		}
	}

	switch {
	case branch.schema.IsList():
		// [FIXME] check key validation

		fallthrough
	default:
		branch.Children[key] = data
		data.SetParent(branch, key)
	}
	return nil
}

func (branch *DataBranch) Delete(key string) error {
	if c := branch.Children[key]; c != nil {
		delete(branch.Children, key)
		c.SetParent(nil, "")
	}
	return nil
}

func (branch *DataBranch) Find(key string) DataNode {
	return branch.Children[key]
}

type DataLeaf struct {
	schema *yang.Entry
	parent *DataBranch
	Value  string
}

func (leaf *DataLeaf) IsYangData()                                 {}
func (leaf *DataLeaf) Schema() *yang.Entry                         { return leaf.schema }
func (leaf *DataLeaf) SetParent(parent *DataBranch, key ...string) { leaf.parent = parent }
func (leaf *DataLeaf) GetParent() *DataBranch                      { return leaf.parent }
func (leaf *DataLeaf) String() string {
	if leaf == nil {
		return ""
	}
	return "leaf " + leaf.schema.Name
}

func (leaf *DataLeaf) Set(value ...string) error {
	for i := range value {
		// check the validation of the value[i]
		// set value
		leaf.Value = value[i]
	}
	return nil
}

func (leaf *DataLeaf) Remove() error {
	delete(leaf.parent.Children, leaf.schema.Name)
	leaf.parent = nil
	return nil
}

func (leaf *DataLeaf) Insert(key string, data DataNode) error {
	return fmt.Errorf("yangtree: %v is not a branch node", leaf)
}

func (leaf *DataLeaf) Delete(key string) error {
	return fmt.Errorf("yangtree: %v is not a branch node", leaf)
}

func (leaf *DataLeaf) Find(key string) DataNode {
	return nil
}

type DataLeafList struct {
	schema *yang.Entry
	parent *DataBranch
	Value  []string
}

func (leaflist *DataLeafList) IsYangData()                                 {}
func (leaflist *DataLeafList) Schema() *yang.Entry                         { return leaflist.schema }
func (leaflist *DataLeafList) SetParent(parent *DataBranch, key ...string) { leaflist.parent = parent }
func (leaflist *DataLeafList) GetParent() *DataBranch                      { return leaflist.parent }
func (leaflist *DataLeafList) String() string {
	if leaflist == nil {
		return ""
	}
	return "leaf-list " + leaflist.schema.Name
}

func (leaflist *DataLeafList) Set(value ...string) error {
	for i := range value {
		// check the validation of the value[i]
		// set value
		leaflist.Value = append(leaflist.Value, value[i])
	}
	return nil
}

func (leaflist *DataLeafList) Remove() error {
	delete(leaflist.parent.Children, leaflist.schema.Name)
	leaflist.parent = nil
	return nil
}

func (leaflist *DataLeafList) Insert(key string, data DataNode) error {
	return fmt.Errorf("yangtree: %v is not a branch node", leaflist)
}

func (leaflist *DataLeafList) Delete(key string) error {
	return fmt.Errorf("yangtree: %v is not a branch node", leaflist)
}

func (leaflist *DataLeafList) Find(key string) DataNode {
	return nil
}

func New(schema *yang.Entry, value ...string) DataNode {
	if schema == nil {
		return nil
	}
	var newdata DataNode
	switch {
	case schema.Dir == nil && schema.ListAttr != nil: // leaf-list
		newdata = &DataLeafList{
			schema: schema,
			Value:  value,
		}
	case schema.Dir == nil: // leaf
		leaf := &DataLeaf{
			schema: schema,
		}
		for i := range value {
			leaf.Value = value[i]
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
	return newdata
}

func Insert(root DataNode, path string, value ...string) error {
	if root == nil {
		return fmt.Errorf("yangtree: nil root data node")
	}
	key, err := SplitPath(root.Schema(), path)
	if err != nil {
		return err
	}
	for i := range key {
		found := root.Find(key[i])
		if found == nil {
			cschema, err := FindSchema(root.Schema(), key[i])
			if err != nil {
				return err
			}
			found = New(cschema)
			if err := root.Insert(key[i], found); err != nil {
				return err
			}
		}
		root = found
	}
	return root.Set(value...)
}

func Delete(root DataNode, path string, value ...string) error {
	if root == nil {
		return fmt.Errorf("yangtree: nil root data node")
	}
	key, err := SplitPath(root.Schema(), path)
	if err != nil {
		return err
	}

	for i := range key {
		found := root.Find(key[i])
		if found == nil {
			return fmt.Errorf("yangtree: data node %v not found", key[:i])
		}
		root = found
	}
	return root.Remove()
}
