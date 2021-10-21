package yangtree

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

// DataLeafList - The node structure of yangtree for leaf-list nodes.
// By default, it is not used for the data node representation of the leaf-list nodes.
// It will be only used when SchemaOption.SingleLeafList is enabled.
type DataLeafList struct {
	schema *SchemaNode
	parent *DataBranch
	value  []interface{}
}

func (leaflist *DataLeafList) IsDataNode()              {}
func (leaflist *DataLeafList) IsNil() bool              { return leaflist == nil }
func (leaflist *DataLeafList) IsBranchNode() bool       { return false }
func (leaflist *DataLeafList) IsLeafNode() bool         { return true }
func (leaflist *DataLeafList) IsLeaf() bool             { return leaflist.schema.IsLeaf() }
func (leaflist *DataLeafList) IsLeafList() bool         { return leaflist.schema.IsLeafList() }
func (leaflist *DataLeafList) IsList() bool             { return leaflist.schema.IsList() }
func (leaflist *DataLeafList) IsContainer() bool        { return leaflist.schema.IsContainer() }
func (leaflist *DataLeafList) IsDuplicatableNode() bool { return leaflist.schema.IsDuplicatable() }
func (leaflist *DataLeafList) IsListableNode() bool     { return leaflist.schema.IsListable() }
func (leaflist *DataLeafList) IsStateNode() bool        { return leaflist.schema.IsState }
func (leaflist *DataLeafList) HasStateNode() bool       { return leaflist.schema.HasState }
func (leaflist *DataLeafList) HasMultipleValues() bool  { return true }

func (leaflist *DataLeafList) Schema() *SchemaNode { return leaflist.schema }
func (leaflist *DataLeafList) Parent() DataNode {
	if leaflist.parent == nil {
		return nil
	}
	return leaflist.parent
}
func (leaflist *DataLeafList) Children() []DataNode { return nil }
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

func (leaflist *DataLeafList) Value() interface{}    { return leaflist.value }
func (leaflist *DataLeafList) Values() []interface{} { return leaflist.value }
func (leaflist *DataLeafList) ValueString() string {
	return ValueToString(leaflist.value)
}
func (leaflist *DataLeafList) QValue(rfc7951format bool) interface{} {
	return leaflist.QValues(rfc7951format)
}
func (leaflist *DataLeafList) QValues(rfc7951format bool) []interface{} {
	if len(leaflist.value) == 0 {
		return nil
	}
	rvalues := make([]interface{}, 0, len(leaflist.value))
	for i := range leaflist.value {
		v, _ := leaflist.schema.ValueToQValue(leaflist.schema.Type, leaflist.value[i], rfc7951format)
		rvalues = append(rvalues, v)
	}
	return rvalues
}

func (leaflist *DataLeafList) HasValue(value string) bool {
	for i := range leaflist.value {
		if v := ValueToString(leaflist.value[i]); v == value {
			return true
		}
	}
	return false
}

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (leaflist *DataLeafList) GetOrNew(id string, opt *EditOption) (DataNode, bool, error) {
	return nil, false, fmt.Errorf("leaf-list node doesn't support GetOrNew")
}

func (leaflist *DataLeafList) Create(id string, value ...string) (DataNode, error) {
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
		if leaflist.schema.IsKey {
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
			if !leaflist.schema.IsState {
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

func (leaflist *DataLeafList) SetSafe(value ...string) error {
	if leaflist.parent != nil {
		// leaflist allows the set operation
		// if leaflist.IsLeafList() {
		// 	return fmt.Errorf("leaflist-list %q must be inserted or deleted", leaflist)
		// }
		if leaflist.schema.IsKey {
			// ignore id update
			// return fmt.Errorf("unable to update id node %q if used", leaflist)
			return nil
		}
	}

	var backup []interface{}
	if len(leaflist.value) > 0 {
		backup = make([]interface{}, len(leaflist.value))
		copy(backup, leaflist.value)
	}
	for i := range value {
		if strings.HasPrefix(value[i], "[") && strings.HasSuffix(value[i], "]") {
			err := leaflist.UnmarshalJSON([]byte(value[i]))
			if err != nil {
				leaflist.value = backup
				return err
			}
		} else {
			var index int
			if !leaflist.schema.IsState {
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
				leaflist.value = backup
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
		if leaflist.schema.IsKey {
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

func (leaflist *DataLeafList) Insert(child DataNode, insert InsertOption) (DataNode, error) {
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

func (leaflist *DataLeafList) GetAll(id string) []DataNode {
	return nil
}

func (leaflist *DataLeafList) GetValue(id string) interface{} {
	return nil
}

func (leaflist *DataLeafList) GetValueString(id string) string {
	return ""
}

func (leaflist *DataLeafList) Lookup(prefix string) []DataNode {
	return nil
}

func (leaflist *DataLeafList) Child(index int) DataNode {
	return nil
}

func (leaflist *DataLeafList) Index(id string) int {
	return 0
}

func (leaflist *DataLeafList) Len() int {
	return len(leaflist.value)
}

func (leaflist *DataLeafList) Name() string {
	return leaflist.schema.Name
}

func (leaflist *DataLeafList) QName(rfc7951 bool) (string, bool) {
	return leaflist.schema.GetQName(rfc7951)
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

// Replace() replaces itself to the src node.
func (leaflist *DataLeafList) Replace(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return replace(leaflist, src)
}

// Merge() merges the src data node to the leaflist data node.
func (leaflist *DataLeafList) Merge(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return merge(leaflist, src)
}

func (leaflist *DataLeafList) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(leaflist, leaflist.schema, jval) // merge
}

func (leaflist *DataLeafList) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jsonNode{DataNode: leaflist}
	err := jnode.marshalJSON(&buffer, false)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (leaflist *DataLeafList) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	boundary := false
	if start.Name.Local != leaflist.schema.Name {
		boundary = true
	} else if leaflist.schema.Qboundary {
		boundary = true
	}
	if boundary {
		ns := leaflist.schema.Module.Namespace
		if ns != nil {
			start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: ns.Name})
			start.Name.Local = leaflist.schema.Name
		}
	} else {
		start = xml.StartElement{Name: xml.Name{Local: leaflist.schema.Name}}
	}
	// if err := e.EncodeToken(xml.Comment(leaflist.ID())); err != nil {
	// 	return err
	// }
	for i := range leaflist.value {
		if err := e.EncodeElement(ValueToString(leaflist.value[i]), start); err != nil {
			return err
		}
	}
	return nil
}

func (leaflist *DataLeafList) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	_, name := SplitQName(&(start.Name.Local))
	// FIXME - prefix (namesapce) must be checked.
	if name != leaflist.schema.Name {
		return fmt.Errorf("invalid element %q inserted for %q", name, leaflist.ID())
	}
	var value string
	d.DecodeElement(&value, &start)
	return leaflist.Set(value)
}

func (leaflist *DataLeafList) MarshalYAML() (interface{}, error) {
	ynode := &yamlNode{
		DataNode: leaflist,
	}
	return ynode.MarshalYAML()
}

func (leaflist *DataLeafList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ydata interface{}
	err := unmarshal(&ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(leaflist, ydata)
}
