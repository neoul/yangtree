package yangtree

import (
	"encoding/json"
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

// A set of data nodes that have the same schema.
type DataNodeGroup []DataNode

// NewDataNodeGroup() creates a set of new data nodes (DataNodeGroup) having the same schema.
// To create a set of data nodes, the value must be encoded to a JSON object or a JSON array of the data.
// It is useful to create multiple list or leaf-list nodes.
//    // e.g.
//    groups, err := NewDataNodeGroup(schema, `["leaf-list-value1", "leaf-list-value2"]`)
//    for _, node := range groups {
//         // Process the created nodes ("leaf-list-value1" and "leaf-list-value2") here.
//    }
func NewDataNodeGroup(schema *yang.Entry, value ...string) (DataNodeGroup, error) {
	vv := make([]*string, len(value))
	for i := range value {
		vv[i] = &(value[i])
	}
	collector, err := newDataNodes(schema, vv...)
	if err != nil {
		return nil, err
	}
	return copyDataNodeList(collector.children), nil
}

func ValidateDataNodeGroup(nodes []DataNode) bool {
	if len(nodes) == 0 {
		return false
	}
	parent := nodes[0].Parent()
	schema := nodes[0].Schema()
	if parent == nil {
		return false
	}
	for i := 1; i < len(nodes); i++ {
		if schema != nodes[i].Schema() {
			return false
		}
		if parent != nodes[i].Parent() {
			return false
		}
	}
	return true
}

func newDataNodes(schema *yang.Entry, value ...*string) (*DataBranch, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	c := NewDataNodeCollector().(*DataBranch)
	for i := range value {
		var jval interface{}
		if err := json.Unmarshal([]byte(*(value[i])), &jval); err != nil {
			return nil, err
		}
		switch jdata := jval.(type) {
		case map[string]interface{}:
			if IsDuplicatableList(schema) {
				return nil, fmt.Errorf("non-key list %q must have the array format of RFC7951", schema.Name)
			}
			kname := GetKeynames(schema)
			kval := make([]string, 0, len(kname))
			if err := c.unmarshalJSONList(schema, kname, kval, jdata); err != nil {
				return nil, err
			}
		case []interface{}:
			if err := c.unmarshalJSONListable(schema, GetKeynames(schema), jdata); err != nil {
				return nil, err
			}
		default:
			n, err := NewDataNode(schema, *(value[i]))
			if err != nil {
				return nil, err
			}
			if _, err := c.insert(n, EditMerge, nil); err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}

func (group DataNodeGroup) IsYangDataNode()                   {}
func (group DataNodeGroup) IsNil() bool                       { return len(group) == 0 }
func (group DataNodeGroup) IsDataBranch() bool                { return false }
func (group DataNodeGroup) IsDataLeaf() bool                  { return false }
func (group DataNodeGroup) IsLeaf() bool                      { return false }
func (group DataNodeGroup) IsLeafList() bool                  { return false }
func (group DataNodeGroup) Schema() *yang.Entry               { return nil }
func (group DataNodeGroup) Parent() DataNode                  { return nil }
func (group DataNodeGroup) Children() []DataNode              { return nil }
func (group DataNodeGroup) String() string                    { return "" }
func (group DataNodeGroup) Path() string                      { return "" }
func (group DataNodeGroup) PathTo(descendant DataNode) string { return "" }
func (group DataNodeGroup) Value() interface{}                { return nil }
func (group DataNodeGroup) ValueString() string               { return "" }

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (group DataNodeGroup) GetOrNew(id string, opt *EditOption) (DataNode, bool, error) {
	return nil, false, fmt.Errorf("data node group doesn't support GetOrNew")
}

func (group DataNodeGroup) NewDataNode(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support NewDataNode")
}

func (group DataNodeGroup) Update(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support Update")
}

func (group DataNodeGroup) Set(value ...string) error {
	return fmt.Errorf("data node group doesn't support Set")
}

func (group DataNodeGroup) SetSafe(value ...string) error {
	return fmt.Errorf("data node group doesn't support SetSafe")
}

func (group DataNodeGroup) Unset(value ...string) error {
	return fmt.Errorf("data node group doesn't support Unset")
}

func (group DataNodeGroup) Remove() error {
	return fmt.Errorf("data node group doesn't support Remove")
}

func (group DataNodeGroup) Insert(child DataNode, edit *EditOption) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support Insert")
}

func (group DataNodeGroup) Delete(child DataNode) error {
	return fmt.Errorf("data node group doesn't support Delete")
}

// [FIXME] - metadata
// SetMeta() sets metadata key-value pairs.
//   e.g. node.SetMeta(map[string]string{"operation": "replace", "last-modified": "2015-06-18T17:01:14+02:00"})
func (group DataNodeGroup) SetMeta(meta ...map[string]string) error {
	return nil
}

func (group DataNodeGroup) Exist(id string) bool {
	return false
}

func (group DataNodeGroup) Get(id string) DataNode {
	return nil
}

func (group DataNodeGroup) GetAll(id string) []DataNode {
	return group
}

func (group DataNodeGroup) GetValue(id string) interface{} {
	return nil
}

func (group DataNodeGroup) GetValueString(id string) string {
	return ""
}

func (group DataNodeGroup) Lookup(prefix string) []DataNode {
	return nil
}

func (group DataNodeGroup) Child(index int) DataNode {
	return nil
}

func (group DataNodeGroup) Index(id string) int {
	return 0
}

func (group DataNodeGroup) Len() int {
	return len(group)
}

func (group DataNodeGroup) Name() string {
	return ""
}

func (group DataNodeGroup) ID() string {
	return ""
}

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
func (group DataNodeGroup) UpdateByMap(pmap map[string]interface{}) error {
	return fmt.Errorf("data node group doesn't support UpdateByMap")
}

func (group DataNodeGroup) UnmarshalJSON(jbytes []byte) error {
	return nil
}

func (group DataNodeGroup) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (group DataNodeGroup) MarshalJSON_IETF() ([]byte, error) {
	return nil, nil
}

// UnmarshalYAML updates the leaf data node using YAML-encoded data.
func (group DataNodeGroup) UnmarshalYAML(in []byte) error {
	return nil
}

// MarshalYAML encodes the leaf data node to a YAML document.
func (group DataNodeGroup) MarshalYAML() ([]byte, error) {
	return nil, nil
}

// MarshalYAML_RFC7951 encodes the leaf data node to a YAML document using RFC7951 namespace-qualified name.
// RFC7951 is the encoding specification for JSON. So, MarshalYAML_RFC7951 only utilizes the RFC7951 namespace-qualified name for YAML encoding.
func (group DataNodeGroup) MarshalYAML_RFC7951() ([]byte, error) {
	return nil, nil
}

// Replace() replaces itself to the src node.
func (group DataNodeGroup) Replace(src DataNode) error {
	return nil
}

// Merge() merges the src data node to the leaf data node.
func (group DataNodeGroup) Merge(src DataNode) error {
	return nil
}

// // MarshalJSON() encodes the data node group to a YAML document with a number of options.
// // The options available are [ConfigOnly, StateOnly, RFC7951Format].
// //   // usage:
// //   var node DataNodeGroup
// //   jsonbytes, err := DataNodeGroup(got).MarshalYAML()
// func (group DataNodeGroup) MarshalJSON(option ...Option) ([]byte, error) {
// 	var comma bool
// 	var buffer bytes.Buffer
// 	buffer.WriteString("[")
// 	configOnly := yang.TSUnset
// 	rfc7951s := rfc7951Enabled
// 	for i := range option {
// 		switch option[i].(type) {
// 		case HasState:
// 			return nil, fmt.Errorf("%v is not allowed for marshaling", option[i])
// 		case ConfigOnly:
// 			configOnly = yang.TSTrue
// 		case StateOnly:
// 			configOnly = yang.TSFalse
// 		case RFC7951Format:
// 			rfc7951s = rfc7951Enabled
// 		}
// 	}
// 	for _, n := range group {
// 		if comma {
// 			buffer.WriteString(",")
// 		}
// 		jnode := &jDataNode{DataNode: n, configOnly: configOnly, rfc7951s: rfc7951s}
// 		err := jnode.marshalJSON(&buffer)
// 		if err != nil {
// 			return nil, err
// 		}
// 		comma = true
// 	}
// 	buffer.WriteString("]")
// 	return buffer.Bytes(), nil
// }

// // MarshalYAML() encodes the data node group to a YAML document with a number of options.
// // The options available are [ConfigOnly, StateOnly, RFC7951Format, InternalFormat].
// //   // usage:
// //   var node DataNodeGroup
// //   yamlbytes, err := DataNodeGroup(got).MarshalYAML()
// func (group DataNodeGroup) MarshalYAML(option ...Option) ([]byte, error) {
// 	var buffer bytes.Buffer
// 	configOnly := yang.TSUnset
// 	rfc7951s := rfc7951Disabled
// 	iformat := false
// 	for i := range option {
// 		switch option[i].(type) {
// 		case HasState:
// 			return nil, fmt.Errorf("%v option can be used to find nodes", option[i])
// 		case ConfigOnly:
// 			configOnly = yang.TSTrue
// 		case StateOnly:
// 			configOnly = yang.TSFalse
// 		case RFC7951Format:
// 			rfc7951s = rfc7951Enabled
// 		case InternalFormat:
// 			iformat = true
// 		}
// 	}
// 	comma := false
// 	for _, n := range group {
// 		if comma {
// 			buffer.WriteString(", ")
// 		}
// 		if n.IsDataBranch() {
// 			buffer.WriteString("- ")
// 		} else {
// 			if !comma {
// 				buffer.WriteString("[")
// 			}
// 		}
// 		ynode := &yDataNode{DataNode: n, indentStr: " ",
// 			configOnly: configOnly, rfc7951s: rfc7951s, iformat: iformat}
// 		if err := ynode.marshalYAML(&buffer, 2, true); err != nil {
// 			return nil, err
// 		}
// 		if n.IsDataLeaf() {
// 			comma = true
// 		}
// 	}
// 	if comma {
// 		buffer.WriteString("]")
// 	}
// 	return buffer.Bytes(), nil
// }
