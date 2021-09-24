package yangtree

import (
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

type DataLeaf struct {
	schema *yang.Entry
	parent *DataBranch
	value  interface{}
	id     string
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
func (leaf *DataLeaf) Children() DataNodeGroup { return nil }
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
		return leaf.parent.Path() + "/" + leaf.ID()
	}
	return "/" + leaf.ID()
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

func (leaf *DataLeaf) NewDataNode(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("new is not supported on %q", leaf)
}

func (leaf *DataLeaf) Update(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("update is not supported %q", leaf)
}

func (leaf *DataLeaf) Set(value string) error {
	if leaf.parent != nil {
		if leaf.IsLeafList() {
			return fmt.Errorf("leaf-list %q must be inserted or deleted", leaf)
		}
		if IsKeyNode(leaf.schema) {
			// ignore id update
			// return fmt.Errorf("unable to update id node %q if used", leaf)
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

func (leaf *DataLeaf) Insert(child DataNode, edit *EditOption) (DataNode, error) {
	return nil, fmt.Errorf("insert is not supported on %q", leaf)
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

func (leaf *DataLeaf) Exist(id string) bool {
	return false
}

func (leaf *DataLeaf) Get(id string) DataNode {
	return nil
}

func (leaf *DataLeaf) GetAll(id string) DataNodeGroup {
	return nil
}

func (leaf *DataLeaf) GetValue(id string) interface{} {
	return nil
}

func (leaf *DataLeaf) GetValueString(id string) string {
	return ""
}

func (leaf *DataLeaf) Lookup(prefix string) DataNodeGroup {
	return nil
}

func (leaf *DataLeaf) Child(index int) DataNode {
	return nil
}

func (leaf *DataLeaf) Index(id string) int {
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

func (leaf *DataLeaf) ID() string {
	if leaf.id != "" {
		return leaf.id
	}
	if leaf.schema.IsLeaf() {
		return leaf.schema.Name
	}
	// leaf-list id format: LEAF[.=VALUE]
	return leaf.schema.Name + `[.=` + ValueToString(leaf.value) + `]`
}
