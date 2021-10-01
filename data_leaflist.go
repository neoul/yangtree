package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"gopkg.in/yaml.v2"
)

// DataLeafList - The node structure of yangtree for leaf-list nodes.
// By default, it is not used for the data node representation of the leaf-list nodes.
// It will be only used when SchemaOption.SingleLeafList is enabled.
type DataLeafList struct {
	schema *yang.Entry
	parent *DataBranch
	value  []interface{}
}

func (leaflist *DataLeafList) IsYangDataNode()     {}
func (leaflist *DataLeafList) IsNil() bool         { return leaflist == nil }
func (leaflist *DataLeafList) IsDataBranch() bool  { return false }
func (leaflist *DataLeafList) IsDataLeaf() bool    { return true }
func (leaflist *DataLeafList) IsLeaf() bool        { return leaflist.schema.IsLeaf() }
func (leaflist *DataLeafList) IsLeafList() bool    { return leaflist.schema.IsLeafList() }
func (leaflist *DataLeafList) Schema() *yang.Entry { return leaflist.schema }
func (leaflist *DataLeafList) Parent() DataNode {
	if leaflist.parent == nil {
		return nil
	}
	return leaflist.parent
}
func (leaflist *DataLeafList) Children() DataNodeGroup { return nil }
func (leaflist *DataLeafList) String() string {
	if leaflist.schema.IsLeaf() {
		return leaflist.schema.Name
	}
	return leaflist.schema.Name + `[.=` + ValueToString(leaflist.value) + `]`
}

func (leaflist *DataLeafList) Path() string {
	if leaflist == nil {
		return ""
	}
	if leaflist.parent != nil {
		return leaflist.parent.Path() + "/" + leaflist.ID()
	}
	return "/" + leaflist.ID()
}

func (leaflist *DataLeafList) PathTo(descendant DataNode) string {
	return ""
}

func (leaflist *DataLeafList) Value() interface{} {
	return leaflist.value
}

func (leaflist *DataLeafList) ValueString() string {
	return ValueToString(leaflist.value)
}

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (leaflist *DataLeafList) GetOrNew(id string, opt *EditOption) (DataNode, bool, error) {
	return nil, false, fmt.Errorf("leaf-list node doesn't support GetOrNew")
}

func (leaflist *DataLeafList) NewDataNode(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("new is not supported on %q", leaflist)
}

func (leaflist *DataLeafList) Update(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("update is not supported %q", leaflist)
}

func (leaflist *DataLeafList) Set(value ...string) error {
	if leaflist.parent != nil {
		// leaflist allows the set operation
		// if leaflist.IsLeafList() {
		// 	return fmt.Errorf("leaflist-list %q must be inserted or deleted", leaflist)
		// }
		if IsKeyNode(leaflist.schema) {
			// ignore id update
			// return fmt.Errorf("unable to update id node %q if used", leaflist)
			return nil
		}
	}
	for i := range value {
		if strings.HasPrefix(value[i], "[") && strings.HasSuffix(value[i], "]") {
			err := leaflist.UnmarshalJSON([]byte(value[i]))
			if err != nil {
				return err
			}
		} else {
			var index int
			if IsConfig(leaflist.schema) {
				index = sort.Search(len(leaflist.value),
					func(j int) bool {
						return ValueToString(leaflist.value[j]) >= value[i]
					})
				if index < len(leaflist.value) && ValueToString(leaflist.value[index]) == value[i] {
					continue
				}
			} else {
				index = len(leaflist.value)
			}
			v, err := StringToValue(leaflist.schema, leaflist.schema.Type, value[i])
			if err != nil {
				return err
			}
			leaflist.value = append(leaflist.value, nil)
			copy(leaflist.value[index+1:], leaflist.value[index:])
			leaflist.value[index] = v
		}
	}
	return nil
}

func (leaflist *DataLeafList) Unset(value ...string) error {
	if leaflist.parent != nil {
		if IsKeyNode(leaflist.schema) {
			// ignore id update
			// return fmt.Errorf("unable to update id node %q if used", leaflist)
			return nil
		}
	}
	for i := range value {
		length := len(leaflist.value)
		index := sort.Search(length,
			func(j int) bool {
				return ValueToString(leaflist.value[j]) >= value[i]
			})
		if index < length && ValueToString(leaflist.value[index]) == value[i] {
			leaflist.value = append(leaflist.value[:index], leaflist.value[index+1:]...)
		}
	}
	return nil
}

func (leaflist *DataLeafList) Remove() error {
	if leaflist.parent == nil {
		return nil
	}
	if branch := leaflist.parent; branch != nil {
		return branch.Delete(leaflist)
	}
	return nil
}

func (leaflist *DataLeafList) Insert(child DataNode, edit *EditOption) (DataNode, error) {
	return nil, fmt.Errorf("insert is not supported on %q", leaflist)
}

func (leaflist *DataLeafList) Delete(child DataNode) error {
	return fmt.Errorf("delete is not supported on %q", leaflist)
}

// [FIXME] - metadata
// SetMeta() sets metadata key-value pairs.
//   e.g. node.SetMeta(map[string]string{"operation": "replace", "last-modified": "2015-06-18T17:01:14+02:00"})
func (leaflist *DataLeafList) SetMeta(meta ...map[string]string) error {
	return nil
}

func (leaflist *DataLeafList) Exist(id string) bool {
	return false
}

func (leaflist *DataLeafList) Get(id string) DataNode {
	return nil
}

func (leaflist *DataLeafList) GetAll(id string) DataNodeGroup {
	return nil
}

func (leaflist *DataLeafList) GetValue(id string) interface{} {
	return nil
}

func (leaflist *DataLeafList) GetValueString(id string) string {
	return ""
}

func (leaflist *DataLeafList) Lookup(prefix string) DataNodeGroup {
	return nil
}

func (leaflist *DataLeafList) Child(index int) DataNode {
	return nil
}

func (leaflist *DataLeafList) Index(id string) int {
	return 0
}

func (leaflist *DataLeafList) Len() int {
	if leaflist.schema.Type.Kind == yang.Yempty {
		return 1
	}
	if leaflist.value == nil {
		return 0
	}
	return 1
}

func (leaflist *DataLeafList) Name() string {
	return leaflist.schema.Name
}

func (leaflist *DataLeafList) ID() string {
	return leaflist.schema.Name
}

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
func (leaflist *DataLeafList) UpdateByMap(pmap map[string]interface{}) error {
	if v, ok := pmap["."]; ok {
		if leaflist.ValueString() == v.(string) {
			return nil
		}
		if err := leaflist.Set(v.(string)); err != nil {
			return err
		}
	}
	return nil
}

func (leaflist *DataLeafList) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(leaflist, jval) // merge
}

func (leaflist *DataLeafList) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: leaflist}
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (leaflist *DataLeafList) MarshalJSON_IETF() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: leaflist}
	jnode.rfc7951s = rfc7951Enabled
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// UnmarshalYAML updates the leaflist data node using YAML-encoded data.
func (leaflist *DataLeafList) UnmarshalYAML(in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(leaflist, ydata)
}

// MarshalYAML encodes the leaflist data node to a YAML document.
func (leaflist *DataLeafList) MarshalYAML() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: leaflist, indentStr: " "}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalYAML_RFC7951 encodes the leaflist data node to a YAML document using RFC7951 namespace-qualified name.
// RFC7951 is the encoding specification for JSON. So, MarshalYAML_RFC7951 only utilizes the RFC7951 namespace-qualified name for YAML encoding.
func (leaflist *DataLeafList) MarshalYAML_RFC7951() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: leaflist, indentStr: " ", rfc7951s: rfc7951Enabled}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// Replace() replaces itself to the src node.
func (leaflist *DataLeafList) Replace(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	if leaflist.schema != src.Schema() {
		return fmt.Errorf("unable to replace the different schema nodes")
	}
	if leaflist.parent == nil {
		return fmt.Errorf("no parent node")
	}
	_, err := leaflist.parent.insert(src, EditReplace, nil)
	return err
}

// Merge() merges the src data node to the leaflist data node.
func (leaflist *DataLeafList) Merge(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return merge(leaflist, src)
}
