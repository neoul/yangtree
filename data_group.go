package yangtree

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/goccy/go-json"
)

// A set of data nodes that have the same schema.
type DataNodeGroup struct {
	schema *SchemaNode
	Nodes  []DataNode
}

// NewGroupWithValue() creates a set of new data nodes (*DataNodeGroup) having the same schema.
// To create a set of data nodes, the value must be encoded to a JSON object or a JSON array of the data.
// It is useful to create multiple list or leaf-list nodes.
//    // e.g.
//    groups, err := NewGroupWithValue(schema, `["leaf-list-value1", "leaf-list-value2"]`)
//    for _, node := range groups.Nodes {
//         // Process the created nodes ("leaf-list-value1" and "leaf-list-value2") here.
//    }
func NewGroupWithValue(schema *SchemaNode, value ...interface{}) (*DataNodeGroup, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	switch {
	case schema.IsSingleLeafList():
		break
	case schema.IsLeafList(): // multiple leaf-list node
		collector := NewCollector().(*DataBranch)
		if len(value) == 1 {
			if v, ok := value[0].([]interface{}); ok {
				for i := range v {
					node, err := NewWithValue(schema, v[i])
					if err != nil {
						return nil, err
					}
					collector.insert(node, nil)
				}
				return &DataNodeGroup{
					schema: schema,
					Nodes:  copyDataNodeList(collector.children),
				}, nil
			}
		}
		for i := range value {
			node, err := NewWithValue(schema, value[i])
			if err != nil {
				return nil, err
			}
			if _, err := collector.insert(node, nil); err != nil {
				return nil, err
			}
		}
		return &DataNodeGroup{
			schema: schema,
			Nodes:  copyDataNodeList(collector.children),
		}, nil
	case schema.IsList():
		collector := NewCollector().(*DataBranch)
		for i := range value {
			switch entry := value[i].(type) {
			case map[interface{}]interface{}, map[string]interface{}:
				kval := make([]interface{}, 0, len(schema.Keyname))
				if err := unmarshalYAMLListNode(collector, schema, schema.Keyname, kval, entry); err != nil {
					return nil, Error(EAppTagYAMLParsing, err)
				}
			case []interface{}:
				for i := range entry {
					node, err := NewWithValue(schema, entry[i])
					if err != nil {
						return nil, err
					}
					if _, err := collector.insert(node, nil); err != nil {
						return nil, err
					}
				}
			default:
				return nil, Errorf(EAppTagYAMLParsing, "unexpected value %s inserted for %s", value[i], schema)
			}
		}
		return &DataNodeGroup{
			schema: schema,
			Nodes:  copyDataNodeList(collector.children),
		}, nil
	}
	node, err := NewWithValue(schema, value...)
	if err != nil {
		return nil, err
	}
	return &DataNodeGroup{
		schema: schema,
		Nodes:  []DataNode{node},
	}, nil
}

// NewGroupWithValueString() creates a set of new data nodes (*DataNodeGroup) having the same schema.
// To create a set of data nodes, the value must be encoded to a JSON object or a JSON array of the data.
// It is useful to create multiple list or leaf-list nodes.
//    // e.g.
//    groups, err := NewGroupWithValueString(schema, `["leaf-list-value1", "leaf-list-value2"]`)
//    for _, node := range groups.Nodes {
//         // Process the created nodes ("leaf-list-value1" and "leaf-list-value2") here.
//    }
func NewGroupWithValueString(schema *SchemaNode, value ...string) (*DataNodeGroup, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	switch {
	case schema.IsSingleLeafList():
		break
	case schema.IsLeafList():
		collector := NewCollector().(*DataBranch)
		if len(value) == 1 {
			if strings.HasPrefix(value[0], "[") && strings.HasSuffix(value[0], "]") {
				var jval interface{}
				if err := json.Unmarshal([]byte(value[0]), &jval); err != nil {
					return nil, err
				}
				array, ok := jval.([]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid value inserted for %s", schema.Name)
				}
				if err := unmarshalJSONListableNode(collector, schema, schema.Keyname, array, nil); err != nil {
					return nil, err
				}
				return &DataNodeGroup{
					schema: schema,
					Nodes:  copyDataNodeList(collector.children),
				}, nil
			}
		}
		for i := range value {
			node, err := NewWithValueString(schema, value[i])
			if err != nil {
				return nil, err
			}
			if _, err := collector.insert(node, nil); err != nil {
				return nil, err
			}
		}
		return &DataNodeGroup{
			schema: schema,
			Nodes:  copyDataNodeList(collector.children),
		}, nil
	case schema.IsList():
		collector := NewCollector().(*DataBranch)
		for i := range value {
			var jval interface{}
			if err := json.Unmarshal([]byte(value[i]), &jval); err != nil {
				return nil, err
			}
			switch jdata := jval.(type) {
			case map[string]interface{}:
				if schema.IsDuplicatableList() {
					return nil, fmt.Errorf("non-key list %s must have the array format of RFC7951", schema.Name)
				}
				kname := schema.Keyname
				kval := make([]string, 0, len(kname))
				if err := unmarshalJSONListNode(collector, schema, kname, kval, jdata); err != nil {
					return nil, err
				}
			case []interface{}:
				if err := unmarshalJSONListableNode(collector, schema, schema.Keyname, jdata, nil); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("invalid value inserted for %s", schema.Name)
			}
		}
		return &DataNodeGroup{
			schema: schema,
			Nodes:  copyDataNodeList(collector.children),
		}, nil
	}
	node, err := NewWithValueString(schema, value...)
	if err != nil {
		return nil, err
	}
	return &DataNodeGroup{
		schema: schema,
		Nodes:  []DataNode{node},
	}, nil
}

// ConvertToGroup() creates a set of new data nodes (DataNodeGroup) having the same schema.
// To create a set of data nodes, the value must be encoded to a JSON object or a JSON array of the data.
// It is useful to create multiple list or leaf-list nodes.
//    // e.g.
//    groups, err := NewGroupWithValueString(schema, `["leaf-list-value1", "leaf-list-value2"]`)
//    for _, node := range groups {
//         // Process the created nodes ("leaf-list-value1" and "leaf-list-value2") here.
//    }
func ConvertToGroup(schema *SchemaNode, nodes []DataNode) (*DataNodeGroup, error) {
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
func (group *DataNodeGroup) IsStateNode() bool        { return group.schema.IsState }
func (group *DataNodeGroup) HasStateNode() bool       { return group.schema.HasState }
func (group *DataNodeGroup) HasMultipleValues() bool  { return !group.schema.IsDir() }

func (group *DataNodeGroup) Schema() *SchemaNode { return group.schema }
func (group *DataNodeGroup) Parent() DataNode    { return nil }
func (group *DataNodeGroup) Children() []DataNode {
	if group.schema.IsDir() {
		return group.Nodes
	}
	return nil
}
func (group *DataNodeGroup) String() string                    { return "group." + group.schema.Name }
func (group *DataNodeGroup) Path() string                      { return "" }
func (group *DataNodeGroup) PathTo(descendant DataNode) string { return "" }
func (group *DataNodeGroup) Value() interface{}                { return group.Values() }
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
func (group *DataNodeGroup) HasValueString(value string) bool {
	if !group.schema.IsDir() {
		for i := range group.Nodes {
			if group.Nodes[i].ValueString() == value {
				return true
			}
		}
	}
	return false
}

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (group *DataNodeGroup) GetOrNew(id string, insert InsertOption) (DataNode, bool, error) {
	return nil, false, fmt.Errorf("data node group doesn't support GetOrNew")
}

func (group *DataNodeGroup) Create(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support Create")
}

func (group *DataNodeGroup) Update(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support Update")
}

func (group *DataNodeGroup) SetValue(value ...interface{}) error {
	return fmt.Errorf("data node group doesn't support SetValue")
}

func (group *DataNodeGroup) SetValueSafe(value ...interface{}) error {
	return fmt.Errorf("data node group doesn't support SetValueSafe")
}

func (group *DataNodeGroup) UnsetValue(value ...interface{}) error {
	return fmt.Errorf("data node group doesn't support UnsetValue")
}

func (group *DataNodeGroup) SetValueString(value ...string) error {
	return fmt.Errorf("data node group doesn't support SetValueString")
}

func (group *DataNodeGroup) SetValueStringSafe(value ...string) error {
	return fmt.Errorf("data node group doesn't support SetValueStringSafe")
}

func (group *DataNodeGroup) UnsetValueString(value ...string) error {
	return fmt.Errorf("data node group doesn't support to unset a value")
}

func (group *DataNodeGroup) Remove() error {
	return fmt.Errorf("data node group doesn't support to remove a node itself")
}

func (group *DataNodeGroup) Insert(child DataNode, insert InsertOption) (DataNode, error) {
	return nil, fmt.Errorf("data node group doesn't support child insertion")
}

func (group *DataNodeGroup) Delete(child DataNode) error {
	return fmt.Errorf("data node group doesn't support child deletion")
}

// SetMetadata() sets a metadata. for example, the following last-modified is set to the node as a metadata.
//   node.SetMetadata("last-modified", "2015-06-18T17:01:14+02:00")
func (group *DataNodeGroup) SetMetadata(name string, value ...interface{}) error {
	return fmt.Errorf("data node group doesn't support metadata")
}

// SetMetadataString() sets a metadata. for example, the following last-modified is set to the node as a metadata.
//   node.SetMetadataString("last-modified", "2015-06-18T17:01:14+02:00")
func (group *DataNodeGroup) SetMetadataString(name string, value ...string) error {
	return fmt.Errorf("data node group doesn't support metadata")
}

// UnsetMetadata() remove a metadata.
func (group *DataNodeGroup) UnsetMetadata(name string) error {
	return fmt.Errorf("data node group doesn't support metadata")
}

func (group *DataNodeGroup) Metadata() map[string]DataNode {
	return nil
}

func (group *DataNodeGroup) Exist(id string) bool {
	for i := range group.Nodes {
		if group.Nodes[i].ID() == id {
			return true
		}
	}
	return false
}

func (group *DataNodeGroup) Get(id string) DataNode {
	for i := range group.Nodes {
		if group.Nodes[i].ID() == id {
			return group.Nodes[i]
		}
	}
	return nil
}

func (group *DataNodeGroup) GetAll(id string) []DataNode {
	nodes := []DataNode{}
	for i := range group.Nodes {
		if group.Nodes[i].ID() == id {
			nodes = append(nodes, group.Nodes[i])
		}
	}
	if len(nodes) > 0 {
		return nodes
	}
	return nil
}

func (group *DataNodeGroup) GetValue(id string) interface{} {
	if group.schema.IsDir() {
		return nil
	}
	for i := range group.Nodes {
		if group.Nodes[i].ID() == id {
			return group.Nodes[i].Value()
		}
	}
	return nil
}

func (group *DataNodeGroup) GetValueString(id string) string {
	if group.schema.IsDir() {
		return ""
	}
	for i := range group.Nodes {
		if group.Nodes[i].ID() == id {
			return group.Nodes[i].ValueString()
		}
	}
	return ""
}

func (group *DataNodeGroup) Lookup(prefix string) []DataNode {
	nodes := []DataNode{}
	for i := range group.Nodes {
		if strings.HasPrefix(group.Nodes[i].ID(), prefix) {
			nodes = append(nodes, group.Nodes[i])
		}
	}
	if len(nodes) > 0 {
		return nodes
	}
	return nil
}

func (group *DataNodeGroup) Child(index int) DataNode {
	if index >= 0 && index < len(group.Nodes) {
		return group.Nodes[index]
	}
	return nil
}

func (group *DataNodeGroup) Index(id string) int {
	for i := range group.Nodes {
		if group.Nodes[i].ID() == id {
			return i
		}
	}
	return len(group.Nodes)
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

// CreateByMap() updates the data node using pmap (path predicate map) and string values.
func (group *DataNodeGroup) CreateByMap(pmap map[string]interface{}) error {
	return fmt.Errorf("data node group doesn't support CreateByMap")
}

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
func (group *DataNodeGroup) UpdateByMap(pmap map[string]interface{}) error {
	return fmt.Errorf("data node group doesn't support UpdateByMap")
}

// Replace() replaces itself to the src node.
func (group *DataNodeGroup) Replace(src DataNode) error {
	return nil
}

// Merge() merges the src data node to the leaf data node.
func (group *DataNodeGroup) Merge(src DataNode) error {
	return nil
}

func (group *DataNodeGroup) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	for _, child := range group.Nodes {
		if err := e.EncodeElement(child, xml.StartElement{Name: xml.Name{Local: child.Name() + "?"}}); err != nil {
			return err
		}
	}
	return nil
}

func (group *DataNodeGroup) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return fmt.Errorf("data node group doesn't support to unmarshal xml")
}

func (group *DataNodeGroup) UnmarshalJSON(jbytes []byte) error {
	return nil
}

func (group *DataNodeGroup) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jsonNode{DataNode: group}
	_, err := jnode.marshalJSON(&buffer, false, false, false)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (group *DataNodeGroup) MarshalYAML() (interface{}, error) {
	ynode := &yamlNode{
		DataNode: group,
	}
	return ynode.toMap(true)
}

func (group *DataNodeGroup) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ydata interface{}
	err := unmarshal(&ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(group, group.schema, ydata)
}
