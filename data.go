package yangtree

import (
	"fmt"
	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// LeafListValueAsKey - leaf-list value can be represented to a path if it is set to true.
	LeafListValueAsKey bool = true
)

// ConfigOnly option is used to find config data nodes that have "config true" statement.
type ConfigOnly struct{}

// StateOnly option is used to find state data nodes that have "config false" statement.
type StateOnly struct{}

// HasState option is used to find state data nodes and data nodes having state data nodes.
type HasState struct{}

func (f ConfigOnly) IsOption() {}
func (f StateOnly) IsOption()  {}
func (f HasState) IsOption()   {}

func (f ConfigOnly) String() string { return "config-only" }
func (f StateOnly) String() string  { return "state-only" }
func (f HasState) String() string   { return "has-state" }

type Operation int

const (
	EditMerge   Operation = iota // netconf edit-config: merge
	EditCreate                   // netconf edit-config: create
	EditReplace                  // netconf edit-config: replace
	EditDelete                   // netconf edit-config: delete
	EditRemove                   // netconf edit-config: remove
)

func (op Operation) String() string {
	switch op {
	case EditMerge:
		return "merge"
	case EditCreate:
		return "create"
	case EditReplace:
		return "replace"
	case EditDelete:
		return "delete"
	case EditRemove:
		return "remove"
	default:
		return "unknown"
	}
}

func (op Operation) IsOption() {}

type EditOption struct {
	Operation
	InsertOption
	Callback func(op Operation, new, old DataNodeGroup) error
}

func (edit *EditOption) String() string {
	if edit == nil {
		return ""
	}
	if edit.InsertOption == nil {
		return `operation=` + edit.Operation.String()
	}
	return `operation=` + edit.Operation.String() + `,` + edit.GetInsertOption().String()
}

func (edit *EditOption) GetOperation() Operation {
	if edit == nil {
		return EditMerge
	}
	return edit.Operation
}
func (edit *EditOption) GetInsertOption() InsertOption {
	if edit == nil {
		return nil
	}
	return edit.InsertOption
}

func (edit *EditOption) GetCallback() func(Operation, DataNodeGroup, DataNodeGroup) error {
	if edit == nil {
		return nil
	}
	return edit.Callback
}

func (edit EditOption) IsOption() {}

type InsertToFirst struct{}
type InsertToLast struct{}
type InsertToBefore struct {
	Key string
}
type InsertToAfter struct {
	Key string
}
type InsertOption interface {
	GetInsertKey() string
	String() string
}

func (o InsertToFirst) GetInsertKey() string  { return "" }
func (o InsertToLast) GetInsertKey() string   { return "" }
func (o InsertToBefore) GetInsertKey() string { return o.Key }
func (o InsertToAfter) GetInsertKey() string  { return o.Key }

func (o InsertToFirst) String() string  { return "insert=first" }
func (o InsertToLast) String() string   { return "insert=last" }
func (o InsertToBefore) String() string { return "insert=before,value=" + o.Key }
func (o InsertToAfter) String() string  { return "insert=after,value=" + o.Key }

// IsValid() return true if it is a valid data node.
func IsValid(node DataNode) bool {
	if node == nil {
		return false
	}
	if node.IsNil() {
		return false
	}
	if node.Schema() == nil {
		return false
	}
	return true
}

// setParent() set the id and parent of the data node.
func setParent(node DataNode, parent *DataBranch, id *string) {
	switch c := node.(type) {
	case *DataBranch:
		c.parent = parent
		if c.schema.Name != *id {
			c.id = *id
		}
	case *DataLeaf:
		c.parent = parent
		if c.schema.Name != *id {
			c.id = *id
		}
	}
}

// resetParent() reset the id and parent of the data node.
func resetParent(node DataNode) {
	switch c := node.(type) {
	case *DataBranch:
		c.parent = nil
		if c.id != "" {
			c.id = ""
		}
	case *DataLeaf:
		c.parent = nil
		if c.id != "" {
			c.id = ""
		}
	}
}

// indexFirst() returns the index of a child related to the id
func indexFirst(parent *DataBranch, id *string) int {
	i := sort.Search(len(parent.children),
		func(j int) bool {
			return *id <= parent.children[j].ID()
		})
	return i
}

// indexMatched() return true if the child data node indexed in the parent has the same node id.
func indexMatched(parent *DataBranch, index int, id *string) bool {
	if index < len(parent.children) && *id == parent.children[index].ID() {
		return true
	}
	return false
}

// indexRangeBySchema() returns the index of a child related to the node id
func indexRangeBySchema(parent *DataBranch, id *string) (i, max int) {
	i = indexFirst(parent, id)
	max = i
	for ; max < len(parent.children); max++ {
		if parent.children[i].Schema() != parent.children[max].Schema() {
			break
		}
	}
	return
}

// insert() insert a child node to the branch node according to the operation and insert option.
// It returns a data node that becomes replaced.
func (branch *DataBranch) insert(child DataNode, op Operation, iopt InsertOption) (DataNode, error) {
	if child.Parent() != nil {
		if child.Parent() == branch {
			return nil, nil
		}
		// allow to move the child to another node.
		// return fmt.Errorf("child node %q is already inserted to %q", child, child.Parent())
		child.Remove()
	}
	schema := child.Schema()
	if !IsAnyData(branch.schema) {
		if branch.Schema() != GetPresentParentSchema(schema) {
			return nil, fmt.Errorf("unable to insert %q because it is not a child of %s", child, branch)
		}
	}

	// duplicatable nodes: read-only leaf-list and non-key list nodes.
	duplicatable := IsDuplicatable(schema)
	orderedByUser := IsOrderedByUser(schema)

	id := child.ID()
	i := indexFirst(branch, &id)
	if !duplicatable {
		// find and replace the node if it is not a duplicatable node.
		if i < len(branch.children) && id == branch.children[i].ID() {
			if op == EditCreate {
				return nil, fmt.Errorf("data node %q exists", id)
			}
			old := branch.children[i]
			resetParent(branch.children[i])
			branch.children[i] = child
			setParent(child, branch, &id)
			return old, nil
		}
	}
	if !orderedByUser && !duplicatable { // ignore insert option
		iopt = nil
	}

	// insert the new child data node.
	switch o := iopt.(type) {
	case nil:
		// get the best position (ordered-by system)
		for ; i < len(branch.children); i++ {
			if id < branch.children[i].ID() {
				break
			}
		}
	case InsertToLast:
		for ; i < len(branch.children); i++ {
			if schema != branch.children[i].Schema() {
				break
			}
		}
	case InsertToFirst:
		name := child.Name()
		i = sort.Search(len(branch.children),
			func(j int) bool { return name <= branch.children[j].ID() })
	case InsertToBefore:
		target := child.Name() + o.Key
		i = sort.Search(len(branch.children),
			func(j int) bool { return target <= branch.children[j].ID() })
	case InsertToAfter:
		target := child.Name() + o.Key
		i = sort.Search(len(branch.children),
			func(j int) bool { return target <= branch.children[j].ID() })
		if i < len(branch.children) {
			i++
		}
	}
	branch.children = append(branch.children, nil)
	copy(branch.children[i+1:], branch.children[i:])
	branch.children[i] = child
	setParent(child, branch, &id)
	return nil, nil
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

// NewDataNode() creates a new DataNode (*DataBranch or *DataLeaf) according to the schema
// with its values. The values should be a string if the new DataNode is *DataLeaf.
// The values should be JSON encoded bytes if the node is *DataBranch.
func NewDataNode(schema *yang.Entry, value ...string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	node, err := newDataNode(schema, IsCreatedWithDefault(schema))
	if err != nil {
		return nil, err
	}
	for i := range value {
		if err = node.Set(value[i]); err != nil {
			return nil, err
		}
	}
	return node, err
}

func newDataNode(schema *yang.Entry, withDefault bool) (DataNode, error) {
	var err error
	var newdata DataNode
	switch {
	case schema.Dir == nil: // leaf, leaf-list
		leaf := &DataLeaf{
			schema: schema,
		}
		if withDefault && schema.Default != "" {
			if err := leaf.Set(schema.Default); err != nil {
				return nil, err
			}
		}
		newdata = leaf
	default: // list, container, anydata
		branch := &DataBranch{
			schema:   schema,
			children: DataNodeGroup{},
		}
		if withDefault {
			for _, s := range schema.Dir {
				if !s.IsDir() && s.Default != "" {
					c, err := NewDataNode(s)
					if err != nil {
						return nil, err
					}
					_, err = branch.insert(c, EditMerge, nil)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		newdata = branch
	}
	return newdata, err
}

func setGroupValue(branch *DataBranch, old DataNodeGroup, new DataNodeGroup, option *EditOption) error {
	op := option.GetOperation()
	switch op {
	case EditDelete, EditRemove:
		if callback := option.GetCallback(); callback != nil {
			if err := callback(op, old, new); err != nil {
				return err
			}
		}
		for i := range old {
			branch.Delete(old[i])
		}
	case EditMerge, EditReplace, EditCreate:
		for i := range old {
			branch.Delete(old[i])
		}
		for i := range new {
			branch.insert(new[i], op, option.GetInsertOption())
		}
		if callback := option.GetCallback(); callback != nil {
			if err := callback(op, old, new); err != nil {
				return err
			}
		}
	}
	return nil
}

// setValue() create or update a target data node using the value.
//  // - EditOption (create): create a node. It returns data-exists error if it exists.
//  // - EditOption (replace): replace the node to the new node.
//  // - EditOption (merge): update the node. (default)
//  // - EditOption (delete): delete the node. It returns data-missing error if it doesn't exist.
//  // - EditOption (remove): delete the node. It doesn't return data-missing error.
func setValue(root DataNode, pathnode []*PathNode, option *EditOption, value ...string) error {
	op := option.GetOperation()
	if len(pathnode) == 0 || pathnode[0].Name == "" {
		switch op {
		case EditCreate:
			return Errorf(ETagDataExists, "data node %q already exists", root.ID())
		case EditDelete, EditRemove:
			if callback := option.GetCallback(); callback != nil {
				if err := callback(op, DataNodeGroup{root}, nil); err != nil {
					return err
				}
			}
			if err := root.Remove(); err != nil {
				return err
			}
		default: // replace, merge
			if len(value) == 0 {
				return nil
			}
			if callback := option.GetCallback(); callback != nil {
				old := Clone(root)
				if err := root.Set(value[0]); err != nil {
					return err
				}
				return callback(op, DataNodeGroup{old}, DataNodeGroup{root})
			}
			if err := root.Set(value[0]); err != nil {
				return err
			}
		}
		return nil
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return setValue(root, pathnode[1:], option, value...)
	case NodeSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("unknown parent node selected from %q", root)
		}
		root = root.Parent()
		return setValue(root, pathnode[1:], option, value...)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		branch, ok := root.(*DataBranch)
		if !ok {
			return fmt.Errorf("select children from non-branch node %q", root)
		}
		for i := 0; i < len(branch.children); i++ {
			err := setValue(branch.Child(i), pathnode[1:], option, value...)
			if err != nil {
				return err
			}
		}
		return nil
	case NodeSelectAll:
		err := setValue(root, pathnode[1:], option, value...)
		if err != nil {
			return err
		}
		branch, ok := root.(*DataBranch)
		if !ok {
			return fmt.Errorf("select children from non-branch node %q", root)
		}
		for i := 0; i < len(branch.children); i++ {
			err := setValue(root.Child(i), pathnode, option, value...)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// [FIXME] - metadata
	// if strings.HasPrefix(pathnode[0].Name, "@") {
	// 	return root.SetMeta(value)
	// }

	branch, ok := root.(*DataBranch)
	if !ok {
		return fmt.Errorf("unable to find children from %q", root)
	}
	cschema := GetSchema(root.Schema(), pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return err
	}

	switch {
	case cschema.IsLeafList():
		if LeafListValueAsKey && len(pathnode) == 2 {
			value = nil
			pmap["."] = pathnode[1].Name
			pathnode = pathnode[:1]
		}
		fallthrough
	case cschema.IsList():
		if len(value) > 0 {
			if v, ok := pmap["."]; ok {
				if v.(string) != value[0] {
					return fmt.Errorf(`value %q must be equal with the xpath predicate of %s[.=%s]`,
						value, cschema.Name, pmap["."].(string))
				}
			}
		}
	}
	reachToEnd := len(pathnode) == 1
	id, nodeGroup := GenerateID(cschema, pmap)
	children := branch.find(cschema, &id, nodeGroup, pmap)
	if len(children) == 0 {
		switch op {
		case EditDelete:
			return Errorf(ETagDataMissing, "data node %q not found", id)
		case EditRemove:
			return nil
		}
	} else {
		if reachToEnd && op == EditCreate && len(children) > 0 {
			return Errorf(EAppTagDataNodeExists, "data node %q exits", id)
		}
	}

	if reachToEnd && nodeGroup {
		var new DataNodeGroup
		switch op {
		case EditMerge:
			new, err = NewDataGroup(cschema, children, value...)
		case EditReplace, EditCreate:
			new, err = NewDataGroup(cschema, nil, value...)
		}
		if err != nil {
			return err
		}
		return setGroupValue(branch, copyDataNodeGroup(children), new, option)
	}

	switch len(children) {
	case 0:
		child, err := NewDataNode(cschema)
		if err != nil {
			return err
		}
		if err = child.UpdateByMap(pmap); err != nil {
			return err
		}
		if reachToEnd && len(value) > 0 {
			if err = child.Set(value[0]); err != nil {
				return err
			}
		}
		if _, err = branch.insert(child, op, option.GetInsertOption()); err != nil {
			return err
		}
		if reachToEnd {
			if callback := option.GetCallback(); callback != nil {
				if err := callback(op, nil, DataNodeGroup{child}); err != nil {
					return err
				}
			}
			return nil
		}

		if err := setValue(child, pathnode[1:], option, value...); err != nil {
			child.Remove()
			return err
		}
		return nil

	default:
		for _, child := range children {
			if err := setValue(child, pathnode[1:], option, value...); err != nil {
				return err
			}
		}
		return nil
	}
}

// Set sets a value to the target DataNode in the path.
// If the target DataNode is a branch node, the value must be json or json_ietf bytes.
// If the target data node is a leaf or a leaf-list node, the value should be string.
func Set(root DataNode, path string, value ...string) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return setValue(root, pathnode, nil, value...)
}

// Edit sets a value to the target DataNode in the path.
// If the target DataNode is a branch node, the value must be json or json_ietf bytes.
// If the target data node is a leaf or a leaf-list node, the value should be string.
func Edit(opt *EditOption, root DataNode, path string, value ...string) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	if len(value) > 1 {
		return Errorf(ETagInvalidValue, "a single value can only be set at a time")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	if err := setValue(root, pathnode, opt, value...); err != nil {
		return err
	}
	// if opt != nil {
	// 	for i := range opt.Created {
	// 		fmt.Printf("Created %s\n", opt.Created[i].Path())
	// 	}
	// 	for i := range opt.Replaced {
	// 		fmt.Printf("Replaced %s\n", opt.Replaced[i].Path())
	// 	}
	// 	for i := range opt.Deleted {
	// 		fmt.Printf("Deleted %s\n", opt.Deleted[i].Path())
	// 	}
	// }
	return err
}

func replaceNode(root DataNode, pathnode []*PathNode, node DataNode) error {
	branch, ok := root.(*DataBranch)
	if !ok {
		return fmt.Errorf("%q is not a branch", root)
	}
	if len(pathnode) == 0 {
		_, err := branch.insert(node, EditReplace, nil)
		return err
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return replaceNode(root, pathnode[1:], node)
	case NodeSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("unknown parent node selected from %q", root)
		}
		root = root.Parent()
		return replaceNode(root, pathnode[1:], node)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		return fmt.Errorf("unable to specify the node position replaced")
	case NodeSelectAll:
		return fmt.Errorf("unable to specify the node position replaced")
	}

	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("schema %q not found from %q", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return err
	}
	switch {
	// case IsDuplicatableList(cschema):
	// 	id, _ := GenerateID(cschema, pmap)
	// 	first := indexFirst(branch, &id)
	// 	if indexMatched(branch, first, &id) {
	// 		return replaceNode(branch.children[first], pathnode[1:], node)
	// 	}
	// 	return nil
	case cschema.IsLeafList():
		if LeafListValueAsKey && len(pathnode) == 2 {
			pmap["."] = pathnode[1].Name
			pathnode = pathnode[:1]
		}
	}
	if cschema == node.Schema() {
		if len(pathnode) > 1 {
			return fmt.Errorf("invalid long tail path: %q", pathnode[1].Name)
		}
		err := node.UpdateByMap(pmap)
		if err != nil {
			return err
		}
		if _, err := branch.insert(node, EditReplace, nil); err != nil {
			return err
		}
		return nil
	}

	id, groupSearch := GenerateID(cschema, pmap)
	children := branch.find(cschema, &id, groupSearch, pmap)
	if len(children) == 0 { // create
		child, err := NewDataNode(cschema)
		if err != nil {
			return err
		}
		if err = child.UpdateByMap(pmap); err != nil {
			return err
		}
		if _, err = branch.insert(child, EditReplace, nil); err != nil {
			return err
		}
		err = replaceNode(child, pathnode[1:], node)
		if err != nil {
			child.Remove()
		}
		return err
	}
	// updates existent nodes
	if len(children) == 1 {
		return replaceNode(children[0], pathnode[1:], node)
	}
	return fmt.Errorf("unable to specify the node position inserted")
}

// Replace() replaces the target data node to the new data node in the path.
func Replace(root DataNode, path string, new DataNode) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	if !IsValid(new) {
		return fmt.Errorf("invalid new data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return replaceNode(root, pathnode, new)
}

// Delete() deletes the target data node in the path if the value is not specified.
// If the value is specified, only the value is deleted.
func Delete(root DataNode, path string) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return setValue(root, pathnode, &EditOption{Operation: EditRemove})
}

func returnFound(node DataNode, option ...Option) DataNodeGroup {
	for i := range option {
		switch option[i].(type) {
		case ConfigOnly:
			meta := GetSchemaMeta(node.Schema())
			if meta.IsState {
				return nil
			}
			return DataNodeGroup{node}
		case StateOnly:
			meta := GetSchemaMeta(node.Schema())
			if meta.IsState {
				return DataNodeGroup{node}
			}
			return nil
		case HasState:
			meta := GetSchemaMeta(node.Schema())
			if meta.IsState {
				return DataNodeGroup{node}
			} else if meta.HasState {
				return DataNodeGroup{node}
			}
			return nil
		}
	}
	return DataNodeGroup{node}
}

func findNode(root DataNode, pathnode []*PathNode, option ...Option) DataNodeGroup {
	if len(pathnode) == 0 {
		return returnFound(root, option...)
	}
	var node, children DataNodeGroup
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return findNode(root, pathnode[1:], option...)
	case NodeSelectParent:
		if root.Parent() == nil {
			return nil
		}
		root = root.Parent()
		return findNode(root, pathnode[1:], option...)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		branch, ok := root.(*DataBranch)
		if !ok {
			return nil
		}
		for i := 0; i < len(branch.children); i++ {
			children = append(children, findNode(root.Child(i), pathnode[1:], option...)...)
		}
		return children
	case NodeSelectAll:
		children = append(children, findNode(root, pathnode[1:], option...)...)
		branch, ok := root.(*DataBranch)
		if !ok {
			return children
		}
		for i := 0; i < len(branch.children); i++ {
			children = append(children, findNode(root.Child(i), pathnode, option...)...)
		}
		return children
	}

	if pathnode[0].Name == "" {
		return returnFound(root, option...)
	}
	// [FIXME]
	if LeafListValueAsKey {
		if root.IsDataLeaf() {
			if pathnode[0].Name == root.ValueString() {
				return DataNodeGroup{root}
			}
			return nil
		}
	}
	branch, ok := root.(*DataBranch)
	if !ok {
		return nil
	}
	cschema := GetSchema(branch.schema, pathnode[0].Name)
	if cschema == nil {
		return nil
	}
	pmap, err := pathnode[0].PredicatesToMap()
	if err != nil {
		return nil
	}
	id, groupSearch := GenerateID(cschema, pmap)
	if _, ok := pmap["@evaluate-xpath"]; ok {
		first, last := indexRangeBySchema(branch, &id)
		node, err = findByPredicates(branch.children[first:last], pathnode[0].Predicates)
		if err != nil {
			return nil
		}
	} else {
		node = branch.find(cschema, &id, groupSearch, pmap)
	}
	for i := range node {
		children = append(children, findNode(node[i], pathnode[1:], option...)...)
	}
	return children
}

// Find() finds all data nodes in the path. xpath format is used for the path.
func Find(root DataNode, path string, option ...Option) (DataNodeGroup, error) {
	if !IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	return findNode(root, pathnode, option...), nil
}

// FindValueString() finds all data in the path and then returns their values by string.
func FindValueString(root DataNode, path string) ([]string, error) {
	if !IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	node := findNode(root, pathnode)
	if len(node) == 0 {
		return nil, nil
	}
	vlist := make([]string, 0, len(node))
	for i := range node {
		if node[i].IsDataLeaf() {
			vlist = append(vlist, node[i].ValueString())
		}
	}
	return vlist, nil
}

// FindValue() finds all data in the path and then returns their values.
func FindValue(root DataNode, path string) ([]interface{}, error) {
	if !IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	node := findNode(root, pathnode)
	if len(node) == 0 {
		return nil, nil
	}
	vlist := make([]interface{}, 0, len(node))
	for i := range node {
		if node[i].IsDataLeaf() {
			vlist = append(vlist, node[i].Value())
		}
	}
	return vlist, nil
}

func clone(destParent *DataBranch, src DataNode) (DataNode, error) {
	var dest DataNode
	switch node := src.(type) {
	case *DataBranch:
		b := &DataBranch{
			schema: node.schema,
		}
		for i := range node.children {
			if _, err := clone(b, node.children[i]); err != nil {
				return nil, err
			}
		}
		dest = b
	case *DataLeaf:
		dest = &DataLeaf{
			schema: node.schema,
			value:  node.value,
		}
	}
	if destParent != nil {
		_, err := destParent.insert(dest, EditMerge, nil)
		if err != nil {
			return nil, err
		}
	}
	return dest, nil
}

func cloneUp(destChild DataNode, src DataNode) (DataNode, error) {
	var destnode DataNode
	if src == nil {
		return nil, nil
	}
	switch node := src.(type) {
	case *DataBranch:
		dnode := &DataBranch{
			schema: node.schema,
		}
		if IsListHasKey(node.schema) {
			for _, c := range node.children {
				if IsKeyNode(c.Schema()) {
					if _, err := clone(dnode, c); err != nil {
						return nil, err
					}
				}
			}
		}
		destnode = dnode
	case *DataLeaf:
		destnode = &DataLeaf{
			schema: node.schema,
			value:  node.value,
		}
	}
	if destChild != nil {
		if _, err := destnode.Insert(destChild, nil); err != nil {
			return nil, err
		}
	}
	if _, err := cloneUp(destnode, src.Parent()); err != nil {
		return nil, err
	}
	return destnode, nil
}

// Clone() copies the src data node with its all decendants.
func Clone(src DataNode) DataNode {
	if IsValid(src) {
		dest, _ := clone(nil, src)
		return dest
	}
	return nil
}

// Move() moves the src data node to the dest node.
// The dest node must have the same schema of the src parent nodes.
func Move(src, dest DataNode) error {
	if !IsValid(src) {
		return Errorf(EAppTagDataNodeMissing, "no src data node")
	}
	parent := src.Parent()
	if parent == nil {
		return nil
	}
	parent, err := cloneUp(nil, parent)
	if err != nil {
		return err
	}
	if _, err := parent.Insert(src, nil); err != nil {
		return err
	}

	if dest == nil {
		return nil
	}
	if dest.IsDataLeaf() {
		return Errorf(EAppTagInvalidArg, "dest must be a branch node")
	}
	destbranch := dest.(*DataBranch)

	for n := src; n != nil; n = n.Parent() {
		p := n.Parent()
		if p == nil {
			break
		}
		if destbranch.schema == p.Schema() {
			_, err := destbranch.insert(n, EditMerge, nil)
			return err
		}
	}
	return Errorf(EAppTagDataNodeMissing, "matched dest node not found from src parent nodes")
}

// Equal() returns true if node1 and node2 have the same data tree and values.
func Equal(node1, node2 DataNode) bool {
	if node1 == node2 {
		return true
	}
	if node1 == nil || node2 == nil {
		return false
	}
	if node1.Schema() != node2.Schema() {
		return false
	}
	switch d1 := node1.(type) {
	case *DataBranch:
		d2 := node2.(*DataBranch)
		if d1.Len() != d2.Len() {
			return false
		}
		for i := range d1.children {
			if equal := Equal(d1.children[i], d2.children[i]); !equal {
				return false
			}
		}
		return true
	case *DataLeaf:
		d2 := node2.(*DataLeaf)
		if _, ok := d2.value.(yang.Number); ok {
			return cmp.Equal(d1.value, d2.value)
		}
		return d1.value == d2.value
	}
	return false
}

func merge(dest, src DataNode) error {
	if dest.Schema() != src.Schema() {
		return fmt.Errorf("unable to merge different schema (%s, %s)", dest, src)
	}
	switch s := src.(type) {
	case *DataBranch:
		d := dest.(*DataBranch)
		for i := range s.children {
			schema := s.children[i].Schema()
			if IsDuplicatableList(schema) {
				if _, err := clone(d, s.children[i]); err != nil {
					return err
				}
			} else {
				dchild := d.GetAll(s.children[i].ID())
				if len(dchild) > 0 {
					for j := range dchild {
						if err := merge(dchild[j], s.children[i]); err != nil {
							return err
						}
					}
				} else {
					if _, err := clone(d, s.children[i]); err != nil {
						return err
					}
				}
			}
		}
	case *DataLeaf:
		d := dest.(*DataLeaf)
		d.value = s.value
	default:
		return fmt.Errorf("invalid data node type: %T", s)
	}
	return nil
}

// Merge() merges the src data node to the target data node in the path.
// The target data node is updated using the src data node.
func Merge(root DataNode, path string, src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	node, err := Find(root, path)
	if err != nil {
		return err
	}
	switch len(node) {
	case 0:
		editopt := &EditOption{Operation: EditMerge}
		err = Edit(editopt, root, path)
		if err != nil {
			return err
		}
		node, err = Find(root, path)
		if err != nil {
			return err
		}
		if len(node) > 1 {
			return fmt.Errorf("more than one data node found - cannot specify the merged node")
		} else if len(node) == 1 {
			return merge(node[0], src)
		}
		return fmt.Errorf("failed to create and merge the nodes in %q", path)
	case 1:
		return merge(node[0], src)
	default:
		return fmt.Errorf("more than one data node found - cannot specify the merged node")
	}
}

// Merge() merges the src data node to the branch data node.
func (branch *DataBranch) Merge(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return merge(branch, src)
}

// Merge() merges the src data node to the leaf data node.
func (leaf *DataLeaf) Merge(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	return merge(leaf, src)
}

// Replace() replaces itself to the src node.
func (branch *DataBranch) Replace(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	if branch.parent == nil {
		return fmt.Errorf("no parent node")
	}
	if branch.schema != src.Schema() {
		return fmt.Errorf("unable to replace the different schema nodes")
	}
	if IsDuplicatableList(branch.schema) {
		return fmt.Errorf("replace is not supported for non-key list")
	}
	_, err := branch.parent.insert(src, EditReplace, nil)
	return err
}

// Replace() replaces itself to the src node.
func (leaf *DataLeaf) Replace(src DataNode) error {
	if !IsValid(src) {
		return fmt.Errorf("invalid src data node")
	}
	if leaf.schema != src.Schema() {
		return fmt.Errorf("unable to replace the different schema nodes")
	}
	if leaf.parent == nil {
		return fmt.Errorf("no parent node")
	}
	_, err := leaf.parent.insert(src, EditReplace, nil)
	return err
}

// Map converts the data node list to a map using the path.
func Map(node DataNodeGroup) map[string]DataNode {
	m := map[string]DataNode{}
	for i := range node {
		m[node[i].Path()] = node[i]
	}
	return m
}

// FindAllInRoute() find all parent nodes in the path.
// The path must indicate an unique node. (not support wildcard and multiple node selection)
func FindAllInRoute(path string) DataNodeGroup {
	return nil
}

// Get key-value pairs of the list data node.
func GetKeyValues(node DataNode) ([]string, []string) {
	keynames := GetKeynames(node.Schema())
	keyvals := make([]string, 0, len(keynames))
	for i := range keynames {
		keynode := node.Get(keynames[i])
		if keynode == nil {
			return keynames, keyvals
		}
		keyvals = append(keyvals, keynode.ValueString())
	}
	return keynames, keyvals
}
