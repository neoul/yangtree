package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
	"gopkg.in/yaml.v2"
)

// DataLeaf - The node structure of yangtree for list or leaf-list nodes.
type DataLeaf struct {
	schema *SchemaNode
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
func (leaf *DataLeaf) Schema() *SchemaNode { return leaf.schema }
func (leaf *DataLeaf) Parent() DataNode {
	if leaf.parent == nil {
		return nil
	}
	return leaf.parent
}
func (leaf *DataLeaf) Children() []DataNode { return nil }
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

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (leaf *DataLeaf) GetOrNew(id string, opt *EditOption) (DataNode, bool, error) {
	return nil, false, fmt.Errorf("leaf node doesn't support GetOrNew")
}

func (leaf *DataLeaf) NewDataNode(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("new is not supported on %q", leaf)
}

func (leaf *DataLeaf) Update(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("update is not supported %q", leaf)
}

func (leaf *DataLeaf) Set(value ...string) error {
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
	for i := range value {
		v, err := StringToValue(leaf.schema, leaf.schema.Type, value[i])
		if err != nil {
			return err
		}
		leaf.value = v
	}
	return nil
}

func (leaf *DataLeaf) SetSafe(value ...string) error {
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
	backup := leaf.value
	for i := range value {
		v, err := StringToValue(leaf.schema, leaf.schema.Type, value[i])
		if err != nil {
			leaf.value = backup
			return err
		}
		leaf.value = v
	}
	return nil
}

func (leaf *DataLeaf) Unset(value ...string) error {
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

	if IsCreatedWithDefault(leaf.schema) && leaf.schema.Default != "" {
		v, err := StringToValue(leaf.schema, leaf.schema.Type, leaf.schema.Default)
		if err != nil {
			return err
		}
		leaf.value = v
	}
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

func (leaf *DataLeaf) GetAll(id string) []DataNode {
	return nil
}

func (leaf *DataLeaf) GetValue(id string) interface{} {
	return nil
}

func (leaf *DataLeaf) GetValueString(id string) string {
	return ""
}

func (leaf *DataLeaf) Lookup(prefix string) []DataNode {
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

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
func (leaf *DataLeaf) UpdateByMap(pmap map[string]interface{}) error {
	if v, ok := pmap["."]; ok {
		if leaf.ValueString() == v.(string) {
			return nil
		}
		if err := leaf.Set(v.(string)); err != nil {
			return err
		}
	}
	return nil
}

func (leaf *DataLeaf) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(leaf, jval) // merge
}

func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: leaf}
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (leaf *DataLeaf) MarshalJSON_RFC7951() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: leaf}
	jnode.rfc7951s = rfc7951Enabled
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// UnmarshalYAML updates the leaf data node using YAML-encoded data.
func (leaf *DataLeaf) UnmarshalYAML(in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(leaf, ydata)
}

// MarshalYAML encodes the leaf data node to a YAML document.
func (leaf *DataLeaf) MarshalYAML() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: leaf, indentStr: " "}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalYAML_RFC7951 encodes the leaf data node to a YAML document using RFC7951 namespace-qualified name.
// RFC7951 is the encoding specification for JSON. So, MarshalYAML_RFC7951 only utilizes the RFC7951 namespace-qualified name for YAML encoding.
func (leaf *DataLeaf) MarshalYAML_RFC7951() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: leaf, indentStr: " ", rfc7951s: rfc7951Enabled}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// Replace() replaces itself to the src node.
func (leaf *DataLeaf) Replace(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return replace(leaf, src)
}

// Merge() merges the src data node to the leaf data node.
func (leaf *DataLeaf) Merge(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return merge(leaf, src)
}
