package yangtree

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

// The node structure of yangtree for container and list data nodes.
type DataBranch struct {
	schema   *SchemaNode
	parent   *DataBranch
	id       string
	children []DataNode
	metadata map[string]DataNode
}

func (branch *DataBranch) IsDataNode()              {}
func (branch *DataBranch) IsNil() bool              { return branch == nil }
func (branch *DataBranch) IsBranchNode() bool       { return true }
func (branch *DataBranch) IsLeafNode() bool         { return false }
func (branch *DataBranch) IsLeaf() bool             { return false }
func (branch *DataBranch) IsLeafList() bool         { return false }
func (branch *DataBranch) IsList() bool             { return branch.schema.IsList() }
func (branch *DataBranch) IsContainer() bool        { return branch.schema.IsContainer() }
func (branch *DataBranch) IsDuplicatableNode() bool { return branch.schema.IsDuplicatable() }
func (branch *DataBranch) IsListableNode() bool     { return branch.schema.IsListable() }
func (branch *DataBranch) IsStateNode() bool        { return branch.schema.IsState }
func (branch *DataBranch) HasStateNode() bool       { return branch.schema.HasState }
func (branch *DataBranch) HasMultipleValues() bool  { return false }

func (branch *DataBranch) Schema() *SchemaNode { return branch.schema }
func (branch *DataBranch) Parent() DataNode {
	if branch.parent == nil {
		return nil
	}
	return branch.parent
}
func (branch *DataBranch) Children() []DataNode { return branch.children }
func (branch *DataBranch) Value() interface{} {
	ynode := &yamlNode{
		DataNode: branch,
	}
	m, err := ynode.toMap(false)
	if err != nil {
		return nil
	}
	return m
}
func (branch *DataBranch) Values() []interface{} {
	m := branch.Value()
	if m != nil {
		return []interface{}{m}
	}
	return nil
}
func (branch *DataBranch) ValueString() string {
	b, err := branch.MarshalJSON()
	if err != nil {
		return ""
	}
	return string(b)
}
func (branch *DataBranch) HasValueString(value string) bool {
	return false
}

func (branch *DataBranch) Path() string {
	if branch == nil {
		return ""
	}
	if branch.parent != nil {
		return branch.parent.Path() + "/" + branch.ID()
	}
	if branch.schema.IsRoot {
		return ""
	}
	return "/" + branch.ID()
}

func (branch *DataBranch) PathTo(descendant DataNode) string {
	if descendant == nil || branch == descendant {
		return ""
	}
	p := []string{}
	for n := descendant; n != nil; n = n.Parent() {
		if n == branch {
			var buf strings.Builder
			for i := len(p) - 1; i >= 0; i-- {
				buf.WriteString(p[i])
				buf.WriteString("/")
			}
			return buf.String()
		}
		p = append(p, n.ID())
	}
	return ""
}

func (branch *DataBranch) String() string {
	if branch == nil {
		return ""
	}
	return branch.ID()
}

// copyDataNodeList clones the src nodes.
func copyDataNodeList(src []DataNode) []DataNode {
	if len(src) > 0 {
		result := make([]DataNode, len(src))
		copy(result, src)
		return result
	}
	return nil
}

// find() is used to find child data nodes using the id internally.
func (branch *DataBranch) find(cschema *SchemaNode, id *string, groupSearch, valueSearch bool, pmap map[string]interface{}) []DataNode {
	i := indexFirst(branch, id)
	if i < len(branch.children) && cschema != branch.children[i].Schema() {
		if !strings.HasPrefix(branch.children[i].ID(), *id) {
			return nil
		}
		if *id != cschema.Name {
			return nil
		}
		for ; i < len(branch.children); i++ {
			if cschema == branch.children[i].Schema() {
				break
			}
		}
	}
	if i >= len(branch.children) {
		return nil
	}
	if pmap != nil {
		if index, ok := pmap["@index"]; ok {
			j := i + index.(int)
			if j < len(branch.children) && cschema == branch.children[j].Schema() {
				return branch.children[j : j+1]
			}
			return nil
		}
		if _, ok := pmap["@last"]; ok {
			last := i
			for ; i < len(branch.children); i++ {
				if cschema == branch.children[i].Schema() {
					last = i
				} else {
					break
				}
			}
			return branch.children[last : last+1]
		}
	}
	max := i
	var matched func() bool
	switch {
	case cschema.IsList() && cschema.Key == "":
		matched = func() bool {
			return true
		}
	case valueSearch:
		v, ok := pmap["."]
		if !ok {
			return nil
		}
		matched = func() bool {
			return branch.children[max].HasValueString(v.(string))
		}
	case groupSearch:
		matched = func() bool {
			return strings.HasPrefix(branch.children[max].ID(), *id)
		}
	default:
		matched = func() bool {
			return branch.children[max].ID() == *id
		}
	}

	if cschema.IsOrderedByUser() || cschema.IsDuplicatable() {
		var node []DataNode
		for ; max < len(branch.children); max++ {
			if cschema != branch.children[max].Schema() {
				break
			}
			if matched() {
				node = append(node, branch.children[max])
			}
		}
		return node
	}

	for ; max < len(branch.children); max++ {
		if cschema != branch.children[max].Schema() {
			break
		}
		if !matched() {
			break
		}
	}
	return branch.children[i:max]
}

// GetOrNew() gets or creates a node having the id and returns the found or created node
// with the boolean value that indicates the returned node is created.
func (branch *DataBranch) GetOrNew(id string, insert InsertOption) (DataNode, bool, error) {
	pathnode, err := ParsePath(&id)
	if err != nil {
		return nil, false, err
	}
	if len(pathnode) == 0 || len(pathnode) > 1 {
		return nil, false, fmt.Errorf("invalid node id %q inserted", id)
	}

	cschema := branch.schema.GetSchema(pathnode[0].Name)
	if cschema == nil {
		return nil, false, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].ToMap()
	if err != nil {
		return nil, false, err
	}
	var children []DataNode
	id, groupSearch, valueSearch := cschema.GenerateID(pmap)
	children = branch.find(cschema, &id, groupSearch, valueSearch, pmap)
	if cschema.IsDuplicatableList() {
		switch insert.(type) {
		case InsertToAfter, InsertToBefore:
			return nil, false, Errorf(ETagOperationNotSupported,
				"insert option (after, before) not supported for non-key list")
		}
		children = nil // clear found nodes
	}
	if len(children) > 0 {
		return children[0], false, nil
	}
	child, err := NewWithValueString(cschema)
	if err != nil {
		return nil, false, err
	}
	if err = child.UpdateByMap(pmap); err != nil {
		return nil, false, err
	}
	if _, err = branch.insert(child, insert); err != nil {
		return nil, false, err
	}
	return child, true, nil
}

func (branch *DataBranch) Create(id string, value ...string) (DataNode, error) {
	if len(value) > 1 {
		return nil, Errorf(ETagInvalidValue, "a single value can only be set at a time")
	}
	pathnode, err := ParsePath(&id)
	if err != nil {
		return nil, err
	}
	if len(pathnode) == 0 || len(pathnode) > 1 {
		return nil, fmt.Errorf("invalid id %q inserted", id)
	}
	cschema := branch.schema.GetSchema(pathnode[0].Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].ToMap()
	if err != nil {
		return nil, err
	}
	n, err := NewWithValueString(cschema, value...)
	if err != nil {
		return nil, err
	}
	if err := n.UpdateByMap(pmap); err != nil {
		return nil, err
	}
	if _, err := branch.insert(n, nil); err != nil {
		return nil, err
	}
	return n, nil
}

func (branch *DataBranch) Update(id string, value ...string) (DataNode, error) {
	if len(value) > 1 {
		return nil, Errorf(ETagInvalidValue, "a single value can only be set at a time")
	}
	pathnode, err := ParsePath(&id)
	if err != nil {
		return nil, err
	}
	if len(pathnode) == 0 || len(pathnode) > 1 {
		return nil, fmt.Errorf("invalid id %q inserted", id)
	}
	cschema := branch.schema.GetSchema(pathnode[0].Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].ToMap()
	if err != nil {
		return nil, err
	}
	n, err := NewWithValueString(cschema, value...)
	if err != nil {
		return nil, err
	}
	if err := n.UpdateByMap(pmap); err != nil {
		return nil, err
	}
	if _, err := branch.insert(n, nil); err != nil {
		return nil, err
	}
	return n, nil
}

func (branch *DataBranch) SetValue(value ...interface{}) error {
	var err error
	for i := range value {
		switch v := value[i].(type) {
		case map[interface{}]interface{}, map[string]interface{}, []interface{}:
			err = unmarshalYAML(branch, branch.schema, v)
		default:
			return Errorf(EAppTagInvalidArg, "invalid value inserted for branch node %q", branch)
		}
	}
	return err
}

func (branch *DataBranch) SetValueSafe(value ...interface{}) error {
	var err error
	backup := Clone(branch)
	for i := range value {
		switch v := value[i].(type) {
		case map[interface{}]interface{}:
			err = unmarshalYAML(branch, branch.schema, v)
		case map[string]interface{}:
			err = unmarshalJSON(branch, branch.schema, v)
		default:
			return Errorf(EAppTagInvalidArg, "invalid value inserted for branch node %q", branch)
		}
	}
	if err != nil {
		recover(branch, backup)
		return err
	}
	return nil
}

func (branch *DataBranch) UnsetValue(value ...interface{}) error {
	return Errorf(ETagOperationNotSupported, "branch data node doesn't support unset")
}

func (branch *DataBranch) setValueString(safe bool, value []string) error {
	var err error
	var backup DataNode
	if safe {
		backup = Clone(branch)
	}
	if IsCreatedWithDefault(branch.schema) {
		for _, s := range branch.schema.Children {
			if !s.IsDir() && s.Default != "" {
				if branch.Get(s.Name) != nil {
					continue
				}
				var c DataNode
				c, err = NewWithValueString(s)
				if err != nil {
					break
				}
				_, err = branch.insert(c, nil)
				if err != nil {
					break
				}
			}
		}
	}
	if err == nil {
		for i := range value {
			if value[i] == "" {
				continue
			}
			err = branch.UnmarshalJSON([]byte(value[i]))
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		if safe {
			recover(branch, backup)
		}
		return err
	}
	return nil
}

func (branch *DataBranch) SetValueString(value ...string) error {
	return branch.setValueString(false, value)
}

func (branch *DataBranch) SetValueStringSafe(value ...string) error {
	return branch.setValueString(true, value)
}

func (branch *DataBranch) UnsetValueString(value ...string) error {
	return Errorf(ETagOperationNotSupported, "branch data node doesn't support unset")
}

func (branch *DataBranch) Remove() error {
	if branch.parent == nil {
		return nil
	}
	parent := branch.parent
	length := len(parent.children)
	id := branch.ID()
	i := sort.Search(length,
		func(j int) bool {
			return id <= parent.children[j].ID()
		})
	if i < length && branch == parent.children[i] {
		parent.children = append(parent.children[:i], parent.children[i+1:]...)
		resetParent(branch)
		return nil
	}
	for i := range parent.children {
		if parent.children[i] == branch {
			parent.children = append(parent.children[:i], parent.children[i+1:]...)
			resetParent(branch)
			return nil
		}
	}
	return nil
}

func (branch *DataBranch) Insert(child DataNode, insert InsertOption) (DataNode, error) {
	if !IsValid(child) {
		return nil, fmt.Errorf("invalid child data node")
	}
	return branch.insert(child, insert)
}

func (branch *DataBranch) Delete(child DataNode) error {
	if !IsValid(child) {
		return fmt.Errorf("invalid child node")
	}

	// if child.Parent() == nil {
	// 	return fmt.Errorf("'%s' is already removed from a branch", child)
	// }
	if child.Schema().IsKey && branch.parent != nil {
		// return fmt.Errorf("id node %q must not be deleted", child)
		return nil
	}

	id := child.ID()
	i := indexFirst(branch, &id)
	if i < len(branch.children) && id == branch.children[i].ID() {
		for ; i < len(branch.children); i++ {
			if branch.children[i] == child {
				branch.children = append(branch.children[:i], branch.children[i+1:]...)
				resetParent(child)
				return nil
			}
		}
	}
	return fmt.Errorf("%q not found on %q", child, branch)
}

// SetMetadata() sets a metadata. for example, the following last-modified is set to the node as a metadata.
//   node.SetMetadata("last-modified", "2015-06-18T17:01:14+02:00")
func (branch *DataBranch) SetMetadata(name string, value ...interface{}) error {
	name = strings.TrimPrefix(name, "@")
	mschema := branch.schema.MetadataSchema[name]
	if mschema == nil {
		return fmt.Errorf("metadata schema %q not found", name)
	}

	meta, err := NewWithValue(mschema, value...)
	if err != nil {
		return err
	}
	if branch.metadata == nil {
		branch.metadata = map[string]DataNode{}
	}
	branch.metadata[name] = meta
	return nil
}

// SetMetadataString() sets a metadata. for example, the following last-modified is set to the node as a metadata.
//   node.SetMetadataString("last-modified", "2015-06-18T17:01:14+02:00")
func (branch *DataBranch) SetMetadataString(name string, value ...string) error {
	name = strings.TrimPrefix(name, "@")
	mschema := branch.schema.MetadataSchema[name]
	if mschema == nil {
		return fmt.Errorf("metadata schema %q not found", name)
	}
	meta, err := NewWithValueString(mschema, value...)
	if err != nil {
		return err
	}
	if branch.metadata == nil {
		branch.metadata = map[string]DataNode{}
	}
	branch.metadata[name] = meta
	return nil
}

// UnsetMetadata() remove a metadata.
func (branch *DataBranch) UnsetMetadata(name string) error {
	name = strings.TrimPrefix(name, "@")
	// mschema := branch.schema.MetadataSchema[name]
	// if mschema == nil {
	// 	return fmt.Errorf("metadata schema %q not found", name)
	// }
	if branch.metadata != nil {
		delete(branch.metadata, name)
	}
	return nil
}

func (branch *DataBranch) Metadata() map[string]DataNode {
	return branch.metadata
}

func (branch *DataBranch) Exist(id string) bool {
	i := indexFirst(branch, &id)
	if i < len(branch.children) {
		return id == branch.children[i].ID()
	}
	return false
}

func (branch *DataBranch) Get(id string) DataNode {
	switch id {
	case ".":
		return branch
	case "..":
		return branch.parent
	case "*":
		if len(branch.children) > 0 {
			return branch.children[0]
		}
		return nil
	case "...":
		n := findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: NodeSelectAll}}, false)
		if len(n) > 0 {
			return n[0]
		}
		return nil
	default:
		i := indexFirst(branch, &id)
		if i < len(branch.children) && id == branch.children[i].ID() {
			return branch.children[i]
		}
		return nil
	}
}

func (branch *DataBranch) GetAll(id string) []DataNode {
	switch id {
	case ".":
		return []DataNode{branch}
	case "..":
		return []DataNode{branch.parent}
	case "*":
		return branch.children
	case "...":
		return findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: NodeSelectAll}}, false)
	default:
		i := indexFirst(branch, &id)
		node := make([]DataNode, 0, len(branch.children)-i+1)
		for max := i; max < len(branch.children); max++ {
			if branch.children[i].Schema() != branch.children[max].Schema() {
				break
			}
			if branch.children[max].ID() == id {
				node = append(node, branch.children[max])
			}
		}
		if len(node) == 0 {
			return nil
		}
		return node
	}
	return nil
}

func (branch *DataBranch) GetValue(id string) interface{} {
	switch id {
	case ".", "..", "*", "...":
		return nil
	default:
		i := indexFirst(branch, &id)
		if i < len(branch.children) && id == branch.children[i].ID() {
			return branch.children[i].Value()
		}
		return nil
	}
}

func (branch *DataBranch) GetValueString(id string) string {
	switch id {
	case ".", "..", "*", "...":
		return ""
	default:
		i := indexFirst(branch, &id)
		if i < len(branch.children) && id == branch.children[i].ID() {
			return branch.children[i].ValueString()
		}
		return ""
	}
}

func (branch *DataBranch) Lookup(prefix string) []DataNode {
	switch prefix {
	case ".":
		return []DataNode{branch}
	case "..":
		return []DataNode{branch.parent}
	case "*":
		return branch.children
	case "...":
		return findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: NodeSelectAll}}, false)
	default:
		i := indexFirst(branch, &prefix)
		node := make([]DataNode, 0, len(branch.children)-i+1)
		for max := i; max < len(branch.children); max++ {
			if strings.HasPrefix(branch.children[max].ID(), prefix) {
				node = append(node, branch.children[max])
			}
		}
		if len(node) == 0 {
			return nil
		}
		return node
	}
}

func (branch *DataBranch) Child(index int) DataNode {
	if index >= 0 && index < len(branch.children) {
		return branch.children[index]
	}
	return nil
}

func (branch *DataBranch) Index(id string) int {
	return indexFirst(branch, &id)
}

func (branch *DataBranch) Len() int {
	return len(branch.children)
}

func (branch *DataBranch) Name() string {
	return branch.schema.Name
}

func (branch *DataBranch) QName(rfc7951 bool) (string, bool) {
	return branch.schema.GetQName(rfc7951)
}

func (branch *DataBranch) ID() string {
	if branch.parent != nil {
		if branch.id == "" {
			return branch.schema.Name
		}
		return branch.id
	}
	switch {
	case branch.schema.IsListHasKey():
		var keybuffer strings.Builder
		keyname := branch.schema.Keyname
		keybuffer.WriteString(branch.schema.Name)
		for i := range keyname {
			j := indexFirst(branch, &keyname[i])
			if j < len(branch.children) && keyname[i] == branch.children[j].ID() {
				keybuffer.WriteString(`[`)
				keybuffer.WriteString(keyname[i])
				keybuffer.WriteString(`=`)
				keybuffer.WriteString(branch.children[j].ValueString())
				keybuffer.WriteString(`]`)
			} else {
				return keybuffer.String()
			}
		}
		return keybuffer.String()
	default:
		return branch.schema.Name
	}
}

// CreateByMap() updates the data node using pmap (path predicate map) and string values.
func (branch *DataBranch) CreateByMap(pmap map[string]interface{}) error {
	for k, v := range pmap {
		if !strings.HasPrefix(k, "@") {
			if vstr, ok := v.(string); ok {
				if k == "." {
					continue
				} else if found := branch.Get(k); found == nil {
					newnode, err := NewWithValueString(branch.Schema().GetSchema(k), vstr)
					if err != nil {
						return err
					}
					if _, err := branch.insert(newnode, nil); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
func (branch *DataBranch) UpdateByMap(pmap map[string]interface{}) error {
	for k, v := range pmap {
		if !strings.HasPrefix(k, "@") {
			if vstr, ok := v.(string); ok {
				if k == "." {
					continue
				}
				found := branch.Get(k)
				if found == nil {
					newnode, err := NewWithValueString(branch.Schema().GetSchema(k), vstr)
					if err != nil {
						return err
					}
					if _, err := branch.insert(newnode, nil); err != nil {
						return err
					}
				} else {
					if err := found.SetValueString(vstr); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// Replace() replaces itself to the src node.
func (branch *DataBranch) Replace(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return replace(branch, src)
}

// Merge() merges the src data node to the branch data node.
func (branch *DataBranch) Merge(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return merge(branch, src)
}

func (branch *DataBranch) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(branch, branch.schema, jval) // merge
}

func (branch *DataBranch) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jsonNode{DataNode: branch}
	_, err := jnode.marshalJSON(&buffer, false, false, false)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (branch *DataBranch) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	boundary := false
	if start.Name.Local != branch.schema.Name {
		boundary = true
	} else if branch.schema.Qboundary {
		boundary = true
	}
	if boundary {
		ns := branch.schema.Module.Namespace
		if ns != nil {
			start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: ns.Name})
			start.Name.Local = branch.schema.Name
		}
	} else {
		start = xml.StartElement{Name: xml.Name{Local: branch.schema.Name}}
	}
	if err := e.EncodeToken(xml.Token(start)); err != nil {
		return err
	}
	for _, child := range branch.children {
		if err := e.EncodeElement(child, xml.StartElement{Name: xml.Name{Local: child.Name()}}); err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.Token(xml.EndElement{Name: xml.Name{Local: branch.schema.Name}}))
}

func (branch *DataBranch) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	_, name := SplitQName(&(start.Name.Local))
	if name != branch.schema.Name {
		return fmt.Errorf("invalid element %q inserted for %q", name, branch.ID())
	}
	if start.Name.Space != branch.Schema().Module.Namespace.Name {
		return fmt.Errorf("unknown namespace %q", start.Name.Space)
	}

	schema := branch.schema
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		if tok == nil {
			break
		}
		switch e := tok.(type) {
		case xml.StartElement:
			_, name := SplitQName(&(e.Name.Local))
			cschema := schema.GetSchema(name)
			if cschema == nil {
				return fmt.Errorf("schema %q not found", e.Name.Local)
			}
			child, err := newDataNode(cschema)
			if err != nil {
				return err
			}
			if err := d.DecodeElement(child, &e); err != nil {
				return err
			}
			// fmt.Println("Branch", branch.ID(), "E", child.ID(), start.Attr)
			curchild := branch.Get(child.ID())
			if curchild == nil {
				if _, err := branch.insert(child, nil); err != nil {
					return err
				}
				curchild = child
			} else {
				if err := curchild.Merge(child); err != nil {
					return err
				}
			}
			for i := range e.Attr {
				if e.Attr[i].Name.Local != "xmlns" &&
					e.Attr[i].Name.Space != "xmlns" {
					// metadata
					curchild.SetMetadataString(e.Attr[i].Name.Local, e.Attr[i].Value)
					// if mschema := branch.schema.MetadataSchema[e.Attr[i].Name.Local]; mschema != nil {
					// 	if n, err := NewWithValueString(mschema, e.Attr[i].Value); err == nil {
					// 		branch.metadata[e.Attr[i].Name.Local] = n
					// 	}
					// }
				}
			}
		case xml.EndElement:
			return nil
		}
	}
	return nil
}

func (branch *DataBranch) MarshalYAML() (interface{}, error) {
	ynode := &yamlNode{
		DataNode: branch,
	}
	return ynode.MarshalYAML()
}

func (branch *DataBranch) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ydata interface{}
	err := unmarshal(&ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(branch, branch.schema, ydata)
}
