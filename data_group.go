package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

// A set of data nodes that have the same schema.
type DataNodeGroup struct {
	schema *SchemaNode
	Nodes  []DataNode
}

// NewDataNodeGroup() creates a set of new data nodes (DataNodeGroup) having the same schema.
// To create a set of data nodes, the value must be encoded to a JSON object or a JSON array of the data.
// It is useful to create multiple list or leaf-list nodes.
//    // e.g.
//    groups, err := NewDataNodeGroup(schema, `["leaf-list-value1", "leaf-list-value2"]`)
//    for _, node := range groups {
//         // Process the created nodes ("leaf-list-value1" and "leaf-list-value2") here.
//    }
func NewDataNodeGroup(schema *SchemaNode, value ...string) (*DataNodeGroup, error) {
	vv := make([]*string, len(value))
	for i := range value {
		vv[i] = &(value[i])
	}
	collector, err := newDataNodes(schema, vv...)
	if err != nil {
		return nil, err
	}
	group := &DataNodeGroup{
		schema: schema,
		Nodes:  copyDataNodeList(collector.children),
	}
	return group, nil
}

// ConvertToDataNodeGroup() creates a set of new data nodes (DataNodeGroup) having the same schema.
// To create a set of data nodes, the value must be encoded to a JSON object or a JSON array of the data.
// It is useful to create multiple list or leaf-list nodes.
//    // e.g.
//    groups, err := NewDataNodeGroup(schema, `["leaf-list-value1", "leaf-list-value2"]`)
//    for _, node := range groups {
//         // Process the created nodes ("leaf-list-value1" and "leaf-list-value2") here.
//    }
func ConvertToDataNodeGroup(schema *SchemaNode, nodes []DataNode) (*DataNodeGroup, error) {
	if len(nodes) == 0 {
		if schema == nil {
			return nil, fmt.Errorf("nil schema")
		}
	} else {
		if schema == nil {
			schema = nodes[0].Schema()
		}
		for i := range nodes {
			if schema != nodes[i].Schema() {
				return nil, fmt.Errorf("converted data nodes doesn't have the same schema")
			}
		}
	}
	group := &DataNodeGroup{
		schema: schema,
		Nodes:  copyDataNodeList(nodes),
	}
	return group, nil
}

// func ValidateDataNodeGroup(nodes []DataNode) bool {
// 	if len(nodes) == 0 {
// 		return false
// 	}
// 	parent := nodes[0].Parent()
// 	schema := nodes[0].Schema()
// 	if parent == nil {
// 		return false
// 	}
// 	for i := 1; i < len(nodes); i++ {
// 		if schema != nodes[i].Schema() {
// 			return false
// 		}
// 		if parent != nodes[i].Parent() {
// 			return false
// 		}
// 	}
// 	return true
// }

func newDataNodes(schema *SchemaNode, value ...*string) (*DataBranch, error) {
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
			if schema.IsDuplicatableList() {
				return nil, fmt.Errorf("non-key list %q must have the array format of RFC7951", schema.Name)
			}
			kname := schema.Keyname
			kval := make([]string, 0, len(kname))
			if err := c.unmarshalJSONList(schema, kname, kval, jdata); err != nil {
				return nil, err
			}
		case []interface{}:
			if err := c.unmarshalJSONListable(schema, schema.Keyname, jdata); err != nil {
				return nil, err
			}
		default:
			n, err := NewDataNode(schema, *(value[i]))
			if err != nil {
				return nil, err
			}
			if _, err := c.insert(n, nil); err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}

func (group *DataNodeGroup) IsDataNode()              {}
func (group *DataNodeGroup) IsNil() bool              { return group == nil }
func (group *DataNodeGroup) IsBranchNode() bool       { return group.schema.IsDir() }
func (group *DataNodeGroup) IsLeafNode() bool         { return !group.schema.IsDir() }
func (group *DataNodeGroup) IsLeaf() bool             { return group.schema.IsLeaf() }
func (group *DataNodeGroup) IsLeafList() bool         { return group.schema.IsLeafList() }
func (group *DataNodeGroup) IsList() bool             { return group.schema.IsList() }
func (group *DataNodeGroup) IsContainer() bool        { return group.schema.IsContainer() }
func (group *DataNodeGroup) IsDuplicatableNode() bool { return group.schema.IsDuplicatable() }
func (group *DataNodeGroup) IsListableNode() bool     { return group.schema.IsListable() }
func (group *DataNodeGroup) Schema() *SchemaNode      { return group.schema }
func (group *DataNodeGroup) Parent() DataNode         { return nil }
func (group *DataNodeGroup) Children() []DataNode {
	if group.schema.IsDir() {
		return group.Nodes
	}
	return nil
}
func (group *DataNodeGroup) String() string                    { return "group" + group.schema.Name }
func (group *DataNodeGroup) Path() string                      { return "" }
func (group *DataNodeGroup) PathTo(descendant DataNode) string { return "" }
func (group *DataNodeGroup) Value() interface{}                { return nil }
func (group *DataNodeGroup) Values() []interface{} {
	if !group.schema.IsDir() {
		if len(group.Nodes) > 0 {
			values := make([]interface{}, 0, len(group.Nodes))
			for i := range group.Nodes {
				values = append(values, group.Nodes[i].Values()...)
			}
			return values
		}
	}
	return nil
}
func (group *DataNodeGroup) ValueString() string { return "" }

func (group *DataNodeGroup) HasValue(value string) bool { return false }

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (group *DataNodeGroup) GetOrNew(id string, opt *EditOption) (DataNode, bool, error) {
	return nil, false, fmt.Errorf("data node group doesn't support GetOrNew")
}

func (group *DataNodeGroup) Create(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support Create")
}

func (group *DataNodeGroup) Update(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support Update")
}

func (group *DataNodeGroup) Set(value ...string) error {
	return fmt.Errorf("data node group doesn't support Set")
}

func (group *DataNodeGroup) SetSafe(value ...string) error {
	return fmt.Errorf("data node group doesn't support SetSafe")
}

func (group *DataNodeGroup) Unset(value ...string) error {
	return fmt.Errorf("data node group doesn't support Unset")
}

func (group *DataNodeGroup) Remove() error {
	return fmt.Errorf("data node group doesn't support Remove")
}

func (group *DataNodeGroup) Insert(child DataNode, insert InsertOption) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support Insert")
}

func (group *DataNodeGroup) Delete(child DataNode) error {
	return fmt.Errorf("data node group doesn't support Delete")
}

// [FIXME] - metadata
// SetMeta() sets metadata key-value pairs.
//   e.g. node.SetMeta(map[string]string{"operation": "replace", "last-modified": "2015-06-18T17:01:14+02:00"})
func (group *DataNodeGroup) SetMeta(meta ...map[string]string) error {
	return nil
}

func (group *DataNodeGroup) Exist(id string) bool {
	return false
}

func (group *DataNodeGroup) Get(id string) DataNode {
	// FIXME - search the data node having the id.
	return nil
}

func (group *DataNodeGroup) GetAll(id string) []DataNode {
	return group.Nodes
}

func (group *DataNodeGroup) GetValue(id string) interface{} {
	return nil
}

func (group *DataNodeGroup) GetValueString(id string) string {
	return ""
}

func (group *DataNodeGroup) Lookup(prefix string) []DataNode {
	return nil
}

func (group *DataNodeGroup) Child(index int) DataNode {
	return nil
}

func (group *DataNodeGroup) Index(id string) int {
	return 0
}

func (group *DataNodeGroup) Len() int {
	return len(group.Nodes)
}

func (group *DataNodeGroup) Name() string {
	return group.schema.Name
}

func (group *DataNodeGroup) QName(rfc7951 bool) (string, bool) {
	return group.schema.GetQName(rfc7951)
}

func (group *DataNodeGroup) ID() string {
	return group.schema.Name
}

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
func (group *DataNodeGroup) UpdateByMap(pmap map[string]interface{}) error {
	return fmt.Errorf("data node group doesn't support UpdateByMap")
}

func (group *DataNodeGroup) UnmarshalJSON(jbytes []byte) error {
	return nil
}

func (group *DataNodeGroup) MarshalJSON() ([]byte, error) {
	return group.marshalJSON()
}

func (group *DataNodeGroup) MarshalJSON_RFC7951() ([]byte, error) {
	return group.marshalJSON(RFC7951Format{})
}

// Replace() replaces itself to the src node.
func (group *DataNodeGroup) Replace(src DataNode) error {
	return nil
}

// Merge() merges the src data node to the leaf data node.
func (group *DataNodeGroup) Merge(src DataNode) error {
	return nil
}

// MarshalJSON() encodes the data node group to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format].
//   // usage:
//   var node DataNodeGroup
//   jsonbytes, err := DataNodeGroup(got).MarshalYAML()
func (group *DataNodeGroup) marshalJSON(option ...Option) ([]byte, error) {
	var buffer bytes.Buffer
	configOnly := yang.TSUnset
	RFC7951S := RFC7951Enabled
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v is not allowed for marshaling", option[i])
		case ConfigOnly:
			configOnly = yang.TSTrue
		case StateOnly:
			configOnly = yang.TSFalse
		case RFC7951Format:
			RFC7951S = RFC7951Enabled
		}
	}
	switch configOnly {
	case yang.TSTrue:
		if group.schema.IsState {
			if RFC7951S != RFC7951Disabled {
				buffer.WriteString("[]")
			} else {
				buffer.WriteString("{}")
			}
			return buffer.Bytes(), nil
		}
	case yang.TSFalse: // stateOnly
		if !group.schema.IsState && !group.schema.HasState {
			if RFC7951S != RFC7951Disabled {
				buffer.WriteString("[]")
			} else {
				buffer.WriteString("{}")
			}
			return buffer.Bytes(), nil
		}
	}
	if RFC7951S != RFC7951Disabled || group.schema.IsDuplicatableList() || group.schema.IsLeafList() {
		nodelist := make([]interface{}, 0, len(group.Nodes))
		for _, n := range group.Nodes {
			jnode := &jDataNode{DataNode: n, configOnly: configOnly, RFC7951S: RFC7951S}
			nodelist = append(nodelist, jnode)
		}
		if err := marshalJNodeTree(&buffer, nodelist); err != nil {
			return nil, err
		}
		return buffer.Bytes(), nil
	}
	nodemap := map[string]interface{}{}
	for i := range group.Nodes {
		jnode := &jDataNode{DataNode: group.Nodes[i],
			configOnly: configOnly, RFC7951S: RFC7951S}
		keyname, keyval := GetKeyValues(jnode.DataNode)
		if len(keyname) != len(keyval) {
			return nil, fmt.Errorf("list %q doesn't have key value pairs", group.schema.Name)
		}
		m := nodemap
		for x := range keyval {
			if x < len(keyname)-1 {
				if n := m[keyval[x]]; n == nil {
					n := map[string]interface{}{}
					m[keyval[x]] = n
					m = n
				} else {
					m = n.(map[string]interface{})
				}
			} else {
				m[keyval[x]] = jnode
			}
		}
	}
	if err := marshalJNodeTree(&buffer, nodemap); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalYAML() encodes the data node group to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format, InternalFormat].
//   // usage:
//   var node DataNodeGroup
//   yamlbytes, err := DataNodeGroup(got).MarshalYAML()
func (group *DataNodeGroup) marshalYAML(indent int, indentStr string, option ...Option) ([]byte, error) {
	var buffer bytes.Buffer
	schema := group.schema
	ynode := &yamlNode{IndentStr: indentStr}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v option can be used to find nodes", option[i])
		case ConfigOnly:
			ynode.ConfigOnly = yang.TSTrue
		case StateOnly:
			ynode.ConfigOnly = yang.TSFalse
		case RFC7951Format:
			ynode.RFC7951S = RFC7951Enabled
		case InternalFormat:
			ynode.InternalFormat = true
		}
	}
	switch ynode.ConfigOnly {
	case yang.TSTrue:
		if group.schema.IsState {
			return buffer.Bytes(), nil
		}
	case yang.TSFalse: // stateOnly
		if !group.schema.IsState && !group.schema.HasState {
			return buffer.Bytes(), nil
		}
	}
	if ynode.RFC7951S != RFC7951Disabled || schema.IsDuplicatableList() || schema.IsLeafList() {
		// writeIndent(&buffer, indent, indentStr, disableFirstIndent)
		// buffer.WriteString(ynode.getQname())
		// buffer.WriteString(":\n")
		// indent++
		for i := range group.Nodes {
			ynode.DataNode = group.Nodes[i]
			writeIndent(&buffer, indent, ynode.IndentStr, false)
			buffer.WriteString("-")
			writeIndent(&buffer, 1, ynode.IndentStr, false)
			err := ynode.marshalYAML(&buffer, indent+2, true)
			if err != nil {
				return nil, err
			}
			if ynode.IsLeafList() {
				buffer.WriteString("\n")
			}
		}
		return buffer.Bytes(), nil
	}
	var lastKeyval []string
	for i := range group.Nodes {
		ynode.DataNode = group.Nodes[i]
		if ynode.InternalFormat {
			writeIndent(&buffer, indent, ynode.IndentStr, false)
			buffer.WriteString(ynode.getQname())
			buffer.WriteString(":\n")
			err := ynode.marshalYAML(&buffer, indent+1, false)
			if err != nil {
				return nil, err
			}
		} else {
			keyname, keyval := GetKeyValues(ynode.DataNode)
			if len(keyname) != len(keyval) {
				return nil, fmt.Errorf("list %q doesn't have a id value", schema.Name)
			}
			for j := range keyval {
				if len(lastKeyval) > 0 && keyval[j] == lastKeyval[j] {
					continue
				}
				writeIndent(&buffer, indent+j, ynode.IndentStr, false)
				buffer.WriteString(keyval[j])
				buffer.WriteString(":\n")
			}
			err := ynode.marshalYAML(&buffer, indent+len(keyval), false)
			if err != nil {
				return nil, err
			}
			lastKeyval = keyval
		}
	}
	return buffer.Bytes(), nil
}

func (group *DataNodeGroup) MarshalYAML() (interface{}, error) {
	ynode := &yamlNode{
		DataNode: group,
	}
	return ynode.MarshalYAML()
}

func (group *DataNodeGroup) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ydata interface{}
	err := unmarshal(&ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(group, ydata)
}
