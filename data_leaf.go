package yangtree

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

// DataLeaf - The node structure of yangtree for list or leaf-list nodes.
type DataLeaf struct {
	schema *SchemaNode
	parent *DataBranch
	value  interface{}
	id     string
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

func (leaf *DataLeaf) Value() interface{}    { return leaf.value }
func (leaf *DataLeaf) Values() []interface{} { return []interface{}{leaf.value} }
func (leaf *DataLeaf) ValueString() string   { return ValueToString(leaf.value) }
func (leaf *DataLeaf) HasValue(value string) bool {
	v := ValueToString(leaf.value)
	return v == value
}

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (leaf *DataLeaf) GetOrNew(id string, insert InsertOption) (DataNode, bool, error) {
	return nil, false, fmt.Errorf("leaf node doesn't support GetOrNew")
}

func (leaf *DataLeaf) Create(id string, value ...string) (DataNode, error) {
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
		if leaf.schema.IsKey {
			// ignore id update
			// return fmt.Errorf("unable to update id node %q if used", leaf)
			return nil
		}
	}
	if len(value) > 1 {
		return fmt.Errorf("data node %q is single value node", leaf)
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
		if leaf.schema.IsKey {
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
		if leaf.schema.IsKey {
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
	} else {
		leaf.value = nil
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

func (leaf *DataLeaf) Insert(child DataNode, insert InsertOption) (DataNode, error) {
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
	return leaf.schema.Name + `[.=` + ValueToString(leaf.value) + `]`
}

// CreateByMap() updates the data node using pmap (path predicate map) and string values.
func (leaf *DataLeaf) CreateByMap(pmap map[string]interface{}) error {
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
	err := jnode.marshalJSON(&buffer, false)
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
	vstr, err := value2String(leaf.schema, leaf.schema.Type, leaf.value)
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
	// FIXME - prefix (namesapce) must be checked.
	if name != leaf.schema.Name {
		return fmt.Errorf("invalid element %q inserted for %q", name, leaf.ID())
	}
	var value string
	d.DecodeElement(&value, &start)
	return leaf.Set(value)
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
	return unmarshalYAML(leaf, ydata)
}
