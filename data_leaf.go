package yangtree

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/goccy/go-json"

	"github.com/openconfig/goyang/pkg/yang"
)

// DataLeaf - The node structure of yangtree for list or leaf-list nodes.
type DataLeaf struct {
	schema   *SchemaNode
	parent   *DataBranch
	value    interface{}
	id       string
	metadata map[string]DataNode
}

func (leaf *DataLeaf) IsDataNode()              {}
func (leaf *DataLeaf) IsNil() bool              { return leaf == nil }
func (leaf *DataLeaf) IsBranchNode() bool       { return false }
func (leaf *DataLeaf) IsLeafNode() bool         { return true }
func (leaf *DataLeaf) IsLeaf() bool             { return leaf.schema.IsLeaf() }
func (leaf *DataLeaf) IsLeafList() bool         { return leaf.schema.IsLeafList() }
func (leaf *DataLeaf) IsList() bool             { return leaf.schema.IsList() }
func (leaf *DataLeaf) IsContainer() bool        { return leaf.schema.IsContainer() }
func (leaf *DataLeaf) IsDuplicatableNode() bool { return leaf.schema.IsDuplicatable() }
func (leaf *DataLeaf) IsListableNode() bool     { return leaf.schema.IsListable() }
func (leaf *DataLeaf) IsStateNode() bool        { return leaf.schema.IsState }
func (leaf *DataLeaf) HasStateNode() bool       { return leaf.schema.HasState }
func (leaf *DataLeaf) HasMultipleValues() bool  { return false }

func (leaf *DataLeaf) Schema() *SchemaNode { return leaf.schema }
func (leaf *DataLeaf) Parent() DataNode {
	if leaf.parent == nil {
		return nil
	}
	return leaf.parent
}
func (leaf *DataLeaf) Children() []DataNode { return nil }
func (leaf *DataLeaf) String() string {
	return leaf.schema.Name
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
	if c, ok := leaf.value.(func(cur DataNode) interface{}); ok {
		return c(leaf)
	}
	return leaf.value
}

func (leaf *DataLeaf) Values() []interface{} {
	if c, ok := leaf.value.(func(cur DataNode) interface{}); ok {
		if v := c(leaf); v != nil {
			return []interface{}{v}
		}
		return nil
	}
	return []interface{}{leaf.value}
}

func (leaf *DataLeaf) ValueString() string {
	if c, ok := leaf.value.(func(cur DataNode) interface{}); ok {
		return ValueToValueString(c(leaf))
	}
	return ValueToValueString(leaf.value)
}

func (leaf *DataLeaf) HasValueString(value string) bool {
	return leaf.ValueString() == value
}

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (leaf *DataLeaf) GetOrNew(id string, insert InsertOption) (DataNode, bool, error) {
	return nil, false, fmt.Errorf("leaf node doesn't support GetOrNew")
}

func (leaf *DataLeaf) Create(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("new is not supported on %s", leaf)
}

func (leaf *DataLeaf) Update(id string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("update is not supported %s", leaf)
}

func (leaf *DataLeaf) SetValue(value ...interface{}) error {
	if len(value) > 1 {
		return fmt.Errorf("more than one value cannot be set to a leaf node %s", leaf)
	}
	if leaf.parent != nil {
		if leaf.IsLeafList() && value[0] != leaf.Value() {
			return fmt.Errorf("value update is not supported for multiple leaf-list")
		}
		if leaf.schema.IsKey {
			// ignore id update
			// return fmt.Errorf("unable to update id node %s if used", leaf)
			return nil
		}
	}

	for i := range value {
		if c, ok := value[i].(func(cur DataNode) interface{}); ok {
			// ReadCallback doesn't support the type validation.
			// It must be checked by ReadCallback.
			if leaf.schema.IsKey {
				return fmt.Errorf("unable to set readcallback to key leaf node")
			}
			if leaf.schema.IsLeafList() {
				return fmt.Errorf("unable to set readcallback to multiple leaflist node")
			}
			leaf.value = c
			break
		}
		v, err := ValueToValidTypeValue(leaf.schema, leaf.schema.Type, value[i])
		if err != nil {
			return err
		}
		leaf.value = v
	}
	return nil
}

func (leaf *DataLeaf) SetValueSafe(value ...interface{}) error {
	return leaf.SetValue(value...)
}

func (leaf *DataLeaf) unsetValue() error {
	if leaf.parent != nil {
		if leaf.IsLeafList() {
			return fmt.Errorf("leaf-list %s must be inserted or deleted", leaf)
		}
		if leaf.schema.IsKey {
			// ignore id update
			// return fmt.Errorf("unable to update id node %s if used", leaf)
			return nil
		}
	}

	if IsCreatedWithDefault(leaf.schema) && leaf.schema.Default != "" {
		v, err := ValueStringToValue(leaf.schema, leaf.schema.Type, leaf.schema.Default)
		if err != nil {
			return err
		}
		leaf.value = v
	} else {
		leaf.value = nil
	}
	return nil
}

func (leaf *DataLeaf) UnsetValue(value ...interface{}) error {
	return leaf.unsetValue()
}

func (leaf *DataLeaf) SetValueString(value ...string) error {
	if len(value) > 1 {
		return fmt.Errorf("more than one value cannot be set to a leaf node %s", leaf)
	}
	if leaf.parent != nil {
		if leaf.IsLeafList() && !leaf.HasValueString(value[0]) {
			return fmt.Errorf("value update is not supported for multiple leaf-list")
		}
		if leaf.schema.IsKey {
			// ignore id update
			// return fmt.Errorf("unable to update id node %s if used", leaf)
			return nil
		}
	}
	for i := range value {
		v, err := ValueStringToValue(leaf.schema, leaf.schema.Type, value[i])
		if err != nil {
			return err
		}
		leaf.value = v
	}
	return nil
}

func (leaf *DataLeaf) SetValueStringSafe(value ...string) error {
	return leaf.SetValueString(value...)
}

func (leaf *DataLeaf) UnsetValueString(value ...string) error {
	return leaf.unsetValue()
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

func (leaf *DataLeaf) Insert(child DataNode, insert InsertOption) (DataNode, error) {
	return nil, fmt.Errorf("insert is not supported on %s", leaf)
}

func (leaf *DataLeaf) Delete(child DataNode) error {
	return fmt.Errorf("delete is not supported on %s", leaf)
}

// SetMetadata() sets a metadata. for example, the following last-modified is set to the node as a metadata.
//   node.SetMetadata("last-modified", "2015-06-18T17:01:14+02:00")
func (leaf *DataLeaf) SetMetadata(name string, value ...interface{}) error {
	name = strings.TrimPrefix(name, "@")
	mschema := leaf.schema.MetadataSchema[name]
	if mschema == nil {
		return fmt.Errorf("metadata schema %s not found", name)
	}
	meta, err := NewWithValue(mschema, value...)
	if err != nil {
		return err
	}
	if leaf.metadata == nil {
		leaf.metadata = map[string]DataNode{}
	}
	leaf.metadata[name] = meta
	return nil
}

// SetMetadataString() sets a metadata. for example, the following last-modified is set to the node as a metadata.
//   node.SetMetadataString("last-modified", "2015-06-18T17:01:14+02:00")
func (leaf *DataLeaf) SetMetadataString(name string, value ...string) error {
	name = strings.TrimPrefix(name, "@")
	mschema := leaf.schema.MetadataSchema[name]
	if mschema == nil {
		return fmt.Errorf("metadata schema %s not found", name)
	}
	meta, err := NewWithValueString(mschema, value...)
	if err != nil {
		return err
	}
	if leaf.metadata == nil {
		leaf.metadata = map[string]DataNode{}
	}
	leaf.metadata[name] = meta
	return nil
}

// UnsetMetadata() remove a metadata.
func (leaf *DataLeaf) UnsetMetadata(name string) error {
	name = strings.TrimPrefix(name, "@")
	// mschema := leaf.schema.MetadataSchema[name]
	// if mschema == nil {
	// 	return fmt.Errorf("metadata schema %s not found", name)
	// }
	if leaf.metadata != nil {
		delete(leaf.metadata, name)
	}
	return nil
}

func (leaf *DataLeaf) Metadata() map[string]DataNode {
	return leaf.metadata
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
	if leaf.Value() == nil {
		return 0
	}
	return 1
}

func (leaf *DataLeaf) Name() string {
	return leaf.schema.Name
}

func (leaf *DataLeaf) QName(rfc7951 bool) (string, bool) {
	return leaf.schema.GetQName(rfc7951)
}

func (leaf *DataLeaf) ID() string {
	if leaf.id != "" {
		return leaf.id
	}
	if leaf.schema.IsLeaf() {
		return leaf.schema.Name
	}
	// leaf-list id format: LEAF[.=VALUE]
	return leaf.schema.Name + `[.=` + leaf.ValueString() + `]`
}

// CreateByMap() updates the data node using pmap (path predicate map) and string values.
func (leaf *DataLeaf) CreateByMap(pmap map[string]interface{}) error {
	if v, ok := pmap["."]; ok {
		if leaf.ValueString() == v.(string) {
			return nil
		}
		if err := leaf.SetValueString(v.(string)); err != nil {
			return err
		}
	}
	return nil
}

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
func (leaf *DataLeaf) UpdateByMap(pmap map[string]interface{}) error {
	if v, ok := pmap["."]; ok {
		if leaf.ValueString() == v.(string) {
			return nil
		}
		if err := leaf.SetValueString(v.(string)); err != nil {
			return err
		}
	}
	return nil
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

func (leaf *DataLeaf) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(leaf, leaf.schema, jval) // merge
}

func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jsonNode{DataNode: leaf}
	_, err := jnode.marshalJSON(&buffer, false, false, false)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (leaf *DataLeaf) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	boundary := false
	if start.Name.Local != leaf.schema.Name {
		boundary = true
	} else if leaf.schema.Qboundary {
		boundary = true
	}
	if boundary {
		ns := leaf.schema.Module.Namespace
		if ns != nil {
			start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: ns.Name})
			start.Name.Local = leaf.schema.Name
		}
	} else {
		start = xml.StartElement{Name: xml.Name{Local: leaf.schema.Name}}
	}
	// if err := e.EncodeToken(xml.Comment(leaf.ID())); err != nil {
	// 	return err
	// }
	vstr, err := value2XMLString(leaf.schema, leaf.schema.Type, leaf.Value())
	if err != nil {
		return err
	}
	if err := e.EncodeElement(vstr, start); err != nil {
		return err
	}
	return nil
}

func (leaf *DataLeaf) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	_, name := SplitQName(&(start.Name.Local))

	if name != leaf.schema.Name {
		return fmt.Errorf("invalid element %s inserted for %s", name, leaf.ID())
	}
	if start.Name.Space != leaf.Schema().Module.Namespace.Name {
		return fmt.Errorf("unknown namespace %s", start.Name.Space)
	}

	var value string
	d.DecodeElement(&value, &start)
	return leaf.SetValueString(value)
}

func (leaf *DataLeaf) MarshalYAML() (interface{}, error) {
	ynode := &yamlNode{
		DataNode: leaf,
	}
	return ynode.MarshalYAML()
}

func (leaf *DataLeaf) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ydata interface{}
	err := unmarshal(&ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(leaf, leaf.schema, ydata)
}
