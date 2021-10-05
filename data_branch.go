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

// The node structure of yangtree for container and list data nodes.
type DataBranch struct {
	schema   *yang.Entry
	parent   *DataBranch
	id       string
	children DataNodeGroup
	metadata map[string]DataNode
}

func (branch *DataBranch) IsYangDataNode()     {}
func (branch *DataBranch) IsNil() bool         { return branch == nil }
func (branch *DataBranch) IsDataBranch() bool  { return true }
func (branch *DataBranch) IsDataLeaf() bool    { return false }
func (branch *DataBranch) IsLeaf() bool        { return false }
func (branch *DataBranch) IsLeafList() bool    { return false }
func (branch *DataBranch) Schema() *yang.Entry { return branch.schema }
func (branch *DataBranch) Parent() DataNode {
	if branch.parent == nil {
		return nil
	}
	return branch.parent
}
func (branch *DataBranch) Children() DataNodeGroup { return branch.children }
func (branch *DataBranch) Value() interface{}      { return nil }

func (branch *DataBranch) ValueString() string {
	b, err := branch.MarshalJSON()
	if err != nil {
		return ""
	}
	return string(b)
}

func (branch *DataBranch) Path() string {
	if branch == nil {
		return ""
	}
	if branch.parent != nil {
		return branch.parent.Path() + "/" + branch.ID()
	}
	if IsRootSchema(branch.schema) {
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

// copyDataNodeGroup clones the src nodes.
func copyDataNodeGroup(src DataNodeGroup) DataNodeGroup {
	if len(src) > 0 {
		result := make(DataNodeGroup, len(src))
		copy(result, src)
		return result
	}
	return nil
}

// find() is used to find child data nodes using the id internally.
func (branch *DataBranch) find(cschema *yang.Entry, id *string, groupSearch bool, pmap map[string]interface{}) DataNodeGroup {
	i := indexFirst(branch, id)
	if i >= len(branch.children) ||
		(i < len(branch.children) && cschema != branch.children[i].Schema()) {
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
	case groupSearch:
		matched = func() bool {
			return strings.HasPrefix(branch.children[max].ID(), *id)
		}
	default:
		matched = func() bool {
			return branch.children[max].ID() == *id
		}
	}

	if IsOrderedByUser(cschema) || IsDuplicatable(cschema) {
		var node DataNodeGroup
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
func (branch *DataBranch) GetOrNew(id string, opt *EditOption) (DataNode, bool, error) {
	op := opt.GetOperation()
	if op == EditRemove || op == EditDelete {
		return nil, false, Errorf(ETagOperationNotSupported, "delete or remove is not supported for GetOrNew")
	}
	iopt := opt.GetInsertOption()

	pathnode, err := ParsePath(&id)
	if err != nil {
		return nil, false, err
	}
	if len(pathnode) == 0 || len(pathnode) > 1 {
		return nil, false, fmt.Errorf("invalid node id %q inserted", id)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return nil, false, err
	}
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return nil, false, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	var children DataNodeGroup
	id, groupSearch := GenerateID(cschema, pmap)
	children = branch.find(cschema, &id, groupSearch, pmap)
	if IsDuplicatableList(cschema) {
		switch iopt.(type) {
		case InsertToAfter, InsertToBefore:
			return nil, false, Errorf(ETagOperationNotSupported,
				"insert option (after, before) not supported for non-key list")
		}
		children = nil // clear found nodes
	}
	if len(children) > 0 {
		return children[0], false, nil
	}
	child, err := NewDataNode(cschema)
	if err != nil {
		return nil, false, err
	}
	if err = child.UpdateByMap(pmap); err != nil {
		return nil, false, err
	}
	if _, err = branch.insert(child, op, iopt); err != nil {
		return nil, false, err
	}
	return child, true, nil
}

func (branch *DataBranch) NewDataNode(id string, value ...string) (DataNode, error) {
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
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return nil, err
	}
	n, err := NewDataNode(cschema, value...)
	if err != nil {
		return nil, err
	}
	if err := n.UpdateByMap(pmap); err != nil {
		return nil, err
	}
	if _, err := branch.insert(n, EditCreate, nil); err != nil {
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
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return nil, err
	}
	n, err := NewDataNode(cschema, value...)
	if err != nil {
		return nil, err
	}
	if err := n.UpdateByMap(pmap); err != nil {
		return nil, err
	}
	if _, err := branch.insert(n, EditMerge, nil); err != nil {
		return nil, err
	}
	return n, nil
}

func (branch *DataBranch) Set(value ...string) error {
	if IsCreatedWithDefault(branch.schema) {
		for _, s := range branch.schema.Dir {
			if !s.IsDir() && s.Default != "" {
				if branch.Get(s.Name) != nil {
					continue
				}
				c, err := NewDataNode(s)
				if err != nil {
					return err
				}
				_, err = branch.insert(c, EditMerge, nil)
				if err != nil {
					return err
				}
			}
		}
	}
	for i := range value {
		if value[i] == "" {
			continue
		}
		err := branch.UnmarshalJSON([]byte(value[i]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (branch *DataBranch) SetSafe(value ...string) error {
	var err error
	backup := Clone(branch)
	if IsCreatedWithDefault(branch.schema) {
		for _, s := range branch.schema.Dir {
			if !s.IsDir() && s.Default != "" {
				if branch.Get(s.Name) != nil {
					continue
				}
				var c DataNode
				c, err = NewDataNode(s)
				if err != nil {
					break
				}
				_, err = branch.insert(c, EditMerge, nil)
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
		recover(branch, backup)
	}
	return nil
}

func (branch *DataBranch) Unset(value ...string) error {
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

func (branch *DataBranch) Insert(child DataNode, edit *EditOption) (DataNode, error) {
	if !IsValid(child) {
		return nil, fmt.Errorf("invalid child data node")
	}
	return branch.insert(child, edit.GetOperation(), edit.GetInsertOption())
}

func (branch *DataBranch) Delete(child DataNode) error {
	if !IsValid(child) {
		return fmt.Errorf("invalid child node")
	}

	// if child.Parent() == nil {
	// 	return fmt.Errorf("'%s' is already removed from a branch", child)
	// }
	if IsKeyNode(child.Schema()) && branch.parent != nil {
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

// [FIXME] - metadata
// SetMeta() sets metadata key-value pairs.
//   e.g. node.SetMeta(map[string]string{"operation": "replace", "last-modified": "2015-06-18T17:01:14+02:00"})
func (branch *DataBranch) SetMeta(meta ...map[string]string) error {
	sm := GetSchemaMeta(branch.schema)
	if sm.Option == nil {
		return fmt.Errorf("no metadata schema")
	}
	// metaschema := sm.Option.ExtensionSchema
	// for i := range meta {
	// 	for name, value := range meta[i] {
	// 		schema := metaschema[name]
	// 		if schema == nil {
	// 			return fmt.Errorf("metadata schema %q not found", name)
	// 		}
	// 		if branch.metadata == nil {
	// 			branch.metadata = map[string]DataNode{}
	// 		}

	// 		metanode, err := NewDataNode(schema, value)
	// 		if err != nil {
	// 			return fmt.Errorf("error in seting metadata: %v", err)
	// 		}
	// 		branch.metadata[name] = metanode
	// 	}
	// }
	return nil
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
			&PathNode{Name: "...", Select: NodeSelectAll}})
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

func (branch *DataBranch) GetAll(id string) DataNodeGroup {
	switch id {
	case ".":
		return DataNodeGroup{branch}
	case "..":
		return DataNodeGroup{branch.parent}
	case "*":
		return branch.children
	case "...":
		return findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: NodeSelectAll}})
	default:
		i := indexFirst(branch, &id)
		node := make(DataNodeGroup, 0, len(branch.children)-i+1)
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

func (branch *DataBranch) Lookup(prefix string) DataNodeGroup {
	switch prefix {
	case ".":
		return DataNodeGroup{branch}
	case "..":
		return DataNodeGroup{branch.parent}
	case "*":
		return branch.children
	case "...":
		return findNode(branch, []*PathNode{
			&PathNode{Name: "...", Select: NodeSelectAll}})
	default:
		i := indexFirst(branch, &prefix)
		node := make(DataNodeGroup, 0, len(branch.children)-i+1)
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

func (branch *DataBranch) ID() string {
	if branch.parent != nil {
		if branch.id == "" {
			return branch.schema.Name
		}
		return branch.id
	}
	switch {
	case IsListHasKey(branch.schema):
		var keybuffer strings.Builder
		keyname := GetKeynames(branch.schema)
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

// UpdateByMap() updates the data node using pmap (path predicate map) and string values.
func (branch *DataBranch) UpdateByMap(pmap map[string]interface{}) error {
	for k, v := range pmap {
		if !strings.HasPrefix(k, "@") {
			if vstr, ok := v.(string); ok {
				if k == "." {
					continue
				} else if found := branch.Get(k); found == nil {
					newnode, err := NewDataNode(GetSchema(branch.Schema(), k), vstr)
					if err != nil {
						return err
					}
					if _, err := branch.insert(newnode, EditMerge, nil); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (branch *DataBranch) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(branch, jval) // merge
}

func (branch *DataBranch) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: branch}
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (branch *DataBranch) MarshalJSON_IETF() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: branch}
	jnode.rfc7951s = rfc7951Enabled
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// UnmarshalYAML updates the branch data node using YAML-encoded data.
func (branch *DataBranch) UnmarshalYAML(in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(branch, ydata)
}

// MarshalYAML encodes the branch data node to a YAML document.
func (branch *DataBranch) MarshalYAML() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: branch, indentStr: " ", iformat: true}
	// ynode := &yDataNode{DataNode: branch, indentStr: " "}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalYAML_RFC7951 encodes the branch data node to a YAML document using RFC7951 namespace-qualified name.
// RFC7951 is the encoding specification for JSON. So, MarshalYAML_RFC7951 only utilizes the RFC7951 namespace-qualified name for YAML encoding.
func (branch *DataBranch) MarshalYAML_RFC7951() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: branch, indentStr: " ", rfc7951s: rfc7951Enabled}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
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