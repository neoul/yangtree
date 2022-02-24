package yangtree

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
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

// EditOperation [EditMerge, EditCreate, EditReplace, EditDelete, EditRemove] for yangtree
type EditOp int

const (
	EditMerge   EditOp = iota // similar to NETCONF edit-config: merge operation
	EditCreate                // similar to NETCONF edit-config: create operation
	EditReplace               // similar to NETCONF edit-config: replace operation
	EditDelete                // similar to NETCONF edit-config: delete operation
	EditRemove                // similar to NETCONF edit-config: remove operation
)

func (op EditOp) String() string {
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

func (op EditOp) IsOption() {}

// EditOption is used for yangtree set operation.
type EditOption struct {
	EditOp                                                     // EditOperation
	InsertOption                                               // Insert option for ordered-by yang option
	Callback        func(op EditOp, old, new []DataNode) error // Callback is invoked upon the node changes.
	FailureRecovery bool
}

func (edit *EditOption) String() string {
	if edit == nil {
		return ""
	}
	if edit.InsertOption == nil {
		return `operation=` + edit.EditOp.String()
	}
	return `operation=` + edit.EditOp.String() + `,` + edit.GetInsertOption().String()
}

func (edit *EditOption) GetOperation() EditOp {
	if edit == nil {
		return EditMerge
	}
	return edit.EditOp
}

func (edit *EditOption) GetInsertOption() InsertOption {
	if edit == nil {
		return nil
	}
	return edit.InsertOption
}

func (edit *EditOption) GetFailureRecovery() bool {
	if edit == nil {
		return false
	}
	return edit.FailureRecovery
}

func (edit *EditOption) GetCallback() func(EditOp, []DataNode, []DataNode) error {
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
	case *DataLeafList:
		c.parent = parent
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
	case *DataLeafList:
		c.parent = nil
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

// indexRangeBySchema() returns the index of a child related to the node id
func indexRangeBySchema(parent *DataBranch, target *SchemaNode) (i, max int) {
	i = sort.Search(len(parent.children),
		func(j int) bool {
			return target.Name <= parent.children[j].ID()
		})
	for ; i < len(parent.children); i++ {
		if parent.children[i].Schema() == target {
			break
		}
	}
	for max = i; max < len(parent.children); max++ {
		if target != parent.children[max].Schema() {
			break
		}
	}
	return
}

// insert() insert a child node to the branch node according to the operation and insert option.
// It returns a data node that becomes replaced.
func (branch *DataBranch) insert(child DataNode, iopt InsertOption) (DataNode, error) {
	if child.Parent() != nil {
		if child.Parent() == branch {
			return nil, nil
		}
		// allow to move the child to another node.
		// return fmt.Errorf("child node %s is already inserted to %s", child, child.Parent())
		child.Remove()
	}
	schema := child.Schema()
	if !branch.schema.IsAnyData() && !branch.schema.ContainAny {
		if branch.Schema() != schema.Parent {
			return nil, fmt.Errorf("unable to insert %s because it is not a child of %s", child, branch)
		}
	}

	// duplicatable nodes: read-only leaf-list and non-key list nodes.
	duplicatable := schema.IsDuplicatable()
	orderedByUser := schema.IsOrderedByUser()

	id := child.ID()
	i := indexFirst(branch, &id)
	if !duplicatable {
		// find and replace the node if it is not a duplicatable node.
		if i < len(branch.children) && id == branch.children[i].ID() {
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
		if schema.IsDuplicatableList() {
			return nil, Errorf(ETagOperationNotSupported,
				"insert option (before) not supported for non-key list")
		}
		target := child.Name() + o.Key
		i = sort.Search(len(branch.children),
			func(j int) bool { return target <= branch.children[j].ID() })
	case InsertToAfter:
		if schema.IsDuplicatableList() {
			return nil, Errorf(ETagOperationNotSupported,
				"insert option (after) not supported for non-key list")
		}
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

// NewCollector() creates a fake node that can be used to collect all kindes of data nodes.
// Any of data nodes can be contained to the collector data node.
func NewCollector() DataNode {
	node, _ := NewWithValueString(collector)
	return node
}

// New() creates a new DataNode (*DataBranch or *DataLeaf) according to the schema.
func New(schema *SchemaNode) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	return newDataNode(schema)
}

// NewWithValue() creates a new DataNode (*DataBranch or *DataLeaf) according to the schema
// with its values. The values should be a string if the new DataNode is *DataLeaf, *DataLeafList.
// The values should be JSON encoded bytes if the node is *DataBranch.
func NewWithValue(schema *SchemaNode, value ...interface{}) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	node, err := newDataNode(schema)
	if err != nil {
		return nil, err
	}
	if err = node.SetValue(value...); err != nil {
		return nil, err
	}
	return node, err
}

// NewWithValueString() creates a new DataNode (*DataBranch or *DataLeaf) according to the schema
// with its values. The values should be a string if the new DataNode is *DataLeaf, *DataLeafList.
// The values should be JSON encoded bytes if the node is *DataBranch.
func NewWithValueString(schema *SchemaNode, value ...string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	node, err := newDataNode(schema)
	if err != nil {
		return nil, err
	}
	for i := range value {
		if err = node.SetValueString(value[i]); err != nil {
			return nil, err
		}
	}
	return node, err
}

// NewWithID() creates a new DataNode using id (NODE_NAME or NODE_NAME[KEY=VALUE])
func NewWithID(schema *SchemaNode, id string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	pathnode, err := ParsePath(&id)
	if err != nil {
		return nil, err
	}
	if len(pathnode) == 0 || len(pathnode) > 1 {
		return nil, fmt.Errorf("invalid id %s inserted", id)
	}
	node, err := newDataNode(schema)
	if err != nil {
		return nil, err
	}
	pmap, err := pathnode[0].ToMap()
	if err != nil {
		return nil, err
	}
	if err := node.UpdateByMap(pmap); err != nil {
		return nil, err
	}
	return node, nil
}

func newDataNode(schema *SchemaNode) (DataNode, error) {
	var err error
	var newdata DataNode
	soption := schema.Option
	switch {
	case schema.IsLeaf() || schema.IsLeafList(): // leaf, leaf-list
		if soption.SingleLeafList && schema.ListAttr != nil {
			leaflist := &DataLeafList{
				schema: schema,
			}
			if schema.Default != "" && soption.CreatedWithDefault {
				if err := leaflist.SetValueString(schema.Default); err != nil {
					return nil, err
				}
			}
			newdata = leaflist
		} else {
			leaf := &DataLeaf{
				schema: schema,
			}
			if schema.Default != "" && soption.CreatedWithDefault {
				if err := leaf.SetValueString(schema.Default); err != nil {
					return nil, err
				}
			}
			newdata = leaf
		}
	default: // list, container, anydata
		branch := &DataBranch{
			schema:   schema,
			children: []DataNode{},
		}
		if soption.CreatedWithDefault {
			for _, s := range schema.Children {
				if !s.IsDir() && s.Default != "" {
					c, err := NewWithValueString(s)
					if err != nil {
						return nil, err
					}
					_, err = branch.insert(c, nil)
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

// merge and report changed child nodes.
func mergeChildren(dest DataNode, mergedChildren []DataNode, edit *EditOption) ([]DataNode, []DataNode, error) {
	var err error
	var n DataNode
	var before []DataNode
	var after []DataNode
	d := dest.(*DataBranch)
	for i := range mergedChildren {
		schema := mergedChildren[i].Schema()
		if schema.IsDuplicatable() {
			n = Clone(mergedChildren[i])
			_, err = d.insert(n, edit.GetInsertOption())
			if err != nil {
				break
			}
			after = append(after, n)
		} else {
			dchild := d.Get(mergedChildren[i].ID())
			if dchild != nil {
				before = append(before, Clone(dchild))
				if err = merge(dchild, mergedChildren[i]); err != nil {
					break
				}
				after = append(after, dchild)
			} else {
				n = Clone(mergedChildren[i])
				_, err = d.insert(n, edit.GetInsertOption())
				if err != nil {
					break
				}
				after = append(after, n)
			}
		}
	}
	return before, after, err
}

func setGroupValue(branch *DataBranch, cschema *SchemaNode, oldnodes []DataNode, edit *EditOption, value interface{}) error {
	var err error
	var newnodes []DataNode
	op := edit.GetOperation()
	iop := edit.GetInsertOption()
	switch op {
	case EditDelete, EditRemove:
		if callback := edit.GetCallback(); callback != nil {
			if err = callback(op, oldnodes, nil); err != nil {
				return err
			}
		}
		for i := range oldnodes {
			branch.Delete(oldnodes[i])
		}
		return nil
	case EditReplace, EditCreate:
		var new *DataNodeGroup
		switch v := value.(type) {
		case []string:
			if new, err = NewGroupWithValueString(cschema, v...); err != nil {
				return err
			}
		case []interface{}:
			if new, err = NewGroupWithValue(cschema, v...); err != nil {
				return err
			}
		}
		newnodes = new.Nodes
		for i := range oldnodes {
			branch.Delete(oldnodes[i])
		}
		for i := range newnodes {
			_, err = branch.insert(newnodes[i], iop)
			if err != nil {
				break
			}
		}
		if err == nil {
			if callback := edit.GetCallback(); callback != nil {
				if err = callback(op, oldnodes, newnodes); err != nil {
					break
				}
			}
		}
	case EditMerge:
		var new *DataNodeGroup
		switch v := value.(type) {
		case []string:
			if new, err = NewGroupWithValueString(cschema, v...); err != nil {
				return err
			}
		case []interface{}:
			if new, err = NewGroupWithValue(cschema, v...); err != nil {
				return err
			}
		}
		oldnodes, newnodes, err = mergeChildren(branch, new.Nodes, edit)
		if err == nil {
			if callback := edit.GetCallback(); callback != nil {
				if err = callback(op, oldnodes, newnodes); err != nil {
					break
				}
			}
		}
	}
	if err != nil {
		for i := range newnodes {
			branch.Delete(newnodes[i])
		}
		for i := range oldnodes {
			branch.insert(oldnodes[i], nil)
		}
	}
	return err
}

// setValue() create or update a target data node using the value string.
//  // - EditOption (create): create a node. It returns data-exists error if it exists.
//  // - EditOption (replace): replace the node to the new node.
//  // - EditOption (merge): update the node. (default)
//  // - EditOption (delete): delete the node. It returns data-missing error if it doesn't exist.
//  // - EditOption (remove): delete the node. It doesn't return data-missing error.
func setValue(root DataNode, pathnode []*PathNode, eopt *EditOption, value interface{}) error {
	var valnum int
	isValueString := false
	switch v := value.(type) {
	case []interface{}:
		valnum = len(v)
	case []string:
		isValueString = true
		valnum = len(v)
	}
	op := eopt.GetOperation()
	if len(pathnode) == 0 {
		switch op {
		case EditCreate:
			return Errorf(ETagDataExists, "data node %s already exists", root.ID())
		case EditDelete, EditRemove:
			if cb := eopt.GetCallback(); cb != nil {
				if err := cb(op, []DataNode{root}, nil); err != nil {
					return err
				}
			}
			if err := root.Remove(); err != nil {
				return err
			}
		default: // replace, merge
			if value == nil {
				return nil
			}
			if cb := eopt.GetCallback(); cb != nil {
				var err error
				backup := Clone(root)
				if isValueString {
					err = root.SetValueString(value.([]string)...)
				} else {
					err = root.SetValue(value.([]interface{})...)
				}
				if err == nil {
					err = cb(op, []DataNode{backup}, []DataNode{root})
				}
				if err != nil {
					recover(root, backup)
				}
				return err
			} else { // without callback
				if eopt.GetFailureRecovery() {
					if isValueString {
						return root.SetValueStringSafe(value.([]string)...)
					} else {
						return root.SetValueSafe(value.([]interface{})...)
					}
				}
				if isValueString {
					return root.SetValueString(value.([]string)...)
				} else {
					return root.SetValue(value.([]interface{})...)
				}

			}
		}
		return nil
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return setValue(root, pathnode[1:], eopt, value)
	case NodeSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("unknown parent node selected from %s", root)
		}
		root = root.Parent()
		return setValue(root, pathnode[1:], eopt, value)
	case NodeSelectFromRoot:
		for root.Parent() != nil {
			root = root.Parent()
		}
	case NodeSelectAllChildren:
		branch, ok := root.(*DataBranch)
		if !ok {
			return fmt.Errorf("select children from non-branch node %s", root)
		}
		for i := 0; i < len(branch.children); i++ {
			err := setValue(branch.Child(i), pathnode[1:], eopt, value)
			if err != nil {
				return err
			}
		}
		return nil
	case NodeSelectAll:
		err := setValue(root, pathnode[1:], eopt, value)
		if err != nil {
			return err
		}
		branch, ok := root.(*DataBranch)
		if !ok {
			return fmt.Errorf("select children from non-branch node %s", root)
		}
		for i := 0; i < len(branch.children); i++ {
			err := setValue(root.Child(i), pathnode, eopt, value)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// processing metadata
	reachToEnd := len(pathnode) == 1
	if strings.HasPrefix(pathnode[0].Name, "@") {
		if !reachToEnd {
			return fmt.Errorf("longtail path for metadata %s", pathnode[0].Name)
		}
		switch op {
		case EditDelete, EditRemove:
			return root.UnsetMetadata(pathnode[0].Name)
		default:
			if isValueString {
				return root.SetMetadataString(pathnode[0].Name, value.([]string)...)
			}
			return root.SetMetadata(pathnode[0].Name, value)
		}
	}

	branch, ok := root.(*DataBranch)
	if !ok {
		return fmt.Errorf("unable to find children from %s", root)
	}
	cschema := root.Schema().GetSchema(pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("schema %s not found from %s", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].ToMap()
	if err != nil {
		return err
	}

	switch {
	case cschema.IsLeafList():
		if cschema.Option.LeafListValueAsKey && len(pathnode) > 1 {
			pmap["."] = pathnode[1].Name
			pathnode = pathnode[:1]
			value = nil
			valnum = 0
		}
		if cschema.IsSingleLeafList() {
			delete(pmap, ".")
		} else { // multiple leaf-list
			if _, ok := pmap["."]; ok {
				if valnum > 1 {
					return Errorf(EAppTagInvalidArg,
						"more than one value cannot be set to multiple leaf-list node")
				}
				// if (isValueString && v.(string) != (value.([]string))[0]) ||
				// 	(!isValueString && v.(string) != ValueToValueString((value.([]interface{}))[0])) {
				// 	return Errorf(EAppTagInvalidArg,
				// 		"the setting value must equal to the indicating value of the path element %s[.=%s]", cschema.Name, v)
				// }
			}
		}
	}

	id, nodeGroup, valueSearch := cschema.GenerateID(pmap)
	children := branch.find(cschema, &id, nodeGroup, valueSearch, pmap)
	if len(children) == 0 {
		switch op {
		case EditDelete:
			return Errorf(ETagDataMissing, "data node %s not found", id)
		case EditRemove:
			return nil
		}
	} else {
		if reachToEnd && op == EditCreate && len(children) > 0 {
			return Errorf(EAppTagDataNodeExists, "data node %s exits", id)
		}
	}

	if reachToEnd && nodeGroup {
		return setGroupValue(branch, cschema, copyDataNodeList(children), eopt, value)
	}

	switch len(children) {
	case 0:
		child, err := NewWithValueString(cschema)
		if err != nil {
			return err
		}
		if err = child.UpdateByMap(pmap); err != nil {
			return err
		}
		if reachToEnd && valnum > 0 {
			if isValueString {
				err = child.SetValueString(value.([]string)...)
			} else {
				err = child.SetValue(value.([]interface{})...)
			}
			if err != nil {
				return err
			}
		}
		if _, err = branch.insert(child, eopt.GetInsertOption()); err != nil {
			return err
		}
		if reachToEnd {
			if cb := eopt.GetCallback(); cb != nil {
				if err := cb(op, nil, []DataNode{child}); err != nil {
					return err
				}
			}
			return nil
		}

		if err := setValue(child, pathnode[1:], eopt, value); err != nil {
			child.Remove()
			return err
		}
		return nil

	default:
		for _, child := range children {
			if err := setValue(child, pathnode[1:], eopt, value); err != nil {
				return err
			}
		}
		return nil
	}
}

// SetValueString sets a value to the target DataNode in the path.
// If the target DataNode is a branch node, the value must be json or json_ietf bytes.
// If the target data node is a leaf or a leaf-list node, the value should be string.
func SetValueString(root DataNode, path string, opt *EditOption, value ...string) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return setValue(root, pathnode, opt, value)
}

// SetValue sets a value to the target DataNode in the path.
// If the target DataNode is a branch node, the value must be map[interface{}]interface{} or map[string]interface{}.
// If the target data node is a leaf or a leaf-list node, the value should be the value.
func SetValue(root DataNode, path string, opt *EditOption, value ...interface{}) error {
	if !IsValid(root) {
		return fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return err
	}
	return setValue(root, pathnode, opt, value)
}

func replaceNode(root DataNode, pathnode []*PathNode, node DataNode) error {
	branch, ok := root.(*DataBranch)
	if !ok {
		return fmt.Errorf("%s is not a branch", root)
	}
	if len(pathnode) == 0 {
		_, err := branch.insert(node, nil)
		return err
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return replaceNode(root, pathnode[1:], node)
	case NodeSelectParent:
		if root.Parent() == nil {
			return fmt.Errorf("unknown parent node selected from %s", root)
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

	cschema := branch.schema.GetSchema(pathnode[0].Name)
	if cschema == nil {
		return fmt.Errorf("schema %s not found from %s", pathnode[0].Name, branch.schema.Name)
	}
	pmap, err := pathnode[0].ToMap()
	if err != nil {
		return err
	}
	switch {
	case cschema.IsLeafList():
		if cschema.Option.LeafListValueAsKey && len(pathnode) > 1 {
			pmap["."] = pathnode[1].Name
			pathnode = pathnode[:1]
		}
	}
	if cschema == node.Schema() {
		if len(pathnode) > 1 {
			return fmt.Errorf("invalid long tail path: %s", pathnode[1].Name)
		}
		err := node.UpdateByMap(pmap)
		if err != nil {
			return err
		}
		if _, err := branch.insert(node, nil); err != nil {
			return err
		}
		return nil
	}

	id, groupSearch, valueSearch := cschema.GenerateID(pmap)
	children := branch.find(cschema, &id, groupSearch, valueSearch, pmap)
	if len(children) == 0 { // create
		child, err := NewWithValueString(cschema)
		if err != nil {
			return err
		}
		if err = child.UpdateByMap(pmap); err != nil {
			return err
		}
		if _, err = branch.insert(child, nil); err != nil {
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
	// n, c, err := GetOrNew(root, path)
	// if err == nil {
	// 	err = n.Replace(new)
	// }
	// if err != nil {
	// 	if c != nil {
	// 		c.Remove()
	// 	}
	// 	return err
	// }
	// return nil
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
	return setValue(root, pathnode, &EditOption{EditOp: EditRemove}, nil)
}

func returnFound(node DataNode, option ...Option) []DataNode {
	for i := range option {
		switch option[i].(type) {
		case ConfigOnly:
			if node.Schema().IsState {
				return nil
			}
			return []DataNode{node}
		case StateOnly:
			if node.Schema().IsState {
				return []DataNode{node}
			}
			return nil
		case HasState:
			s := node.Schema()
			if s.IsState {
				return []DataNode{node}
			} else if s.HasState {
				return []DataNode{node}
			}
			return nil
		}
	}
	return []DataNode{node}
}

func findNode(root DataNode, pathnode []*PathNode, useXPath bool, option ...Option) []DataNode {
	if len(pathnode) == 0 {
		return returnFound(root, option...)
	}
	var node, children []DataNode
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return findNode(root, pathnode[1:], useXPath, option...)
	case NodeSelectParent:
		if root.Parent() == nil {
			return nil
		}
		root = root.Parent()
		return findNode(root, pathnode[1:], useXPath, option...)
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
			children = append(children, findNode(root.Child(i), pathnode[1:], useXPath, option...)...)
		}
		return children
	case NodeSelectAll:
		children = append(children, findNode(root, pathnode[1:], useXPath, option...)...)
		branch, ok := root.(*DataBranch)
		if !ok {
			return children
		}
		for i := 0; i < len(branch.children); i++ {
			children = append(children, findNode(root.Child(i), pathnode, useXPath, option...)...)
		}
		return children
	}

	branch, ok := root.(*DataBranch)
	if !ok {
		return nil
	}
	cschema := branch.schema.GetSchema(pathnode[0].Name)
	if cschema == nil {
		return nil
	}
	pmap, err := pathnode[0].ToMap()
	if err != nil {
		return nil
	}
	switch {
	case root.IsLeafList():
		if root.Schema().Option.LeafListValueAsKey && len(pathnode) > 1 {
			pmap["."] = pathnode[1].Name
			pathnode = pathnode[:1]
		}
	}
	id, groupSearch, valueSearch := cschema.GenerateID(pmap)
	if _, ok := pmap["@evaluate-xpath"]; ok || useXPath {
		first, last := indexRangeBySchema(branch, cschema)
		node, err = findByPredicates(branch.children[first:last], pathnode[0].Predicates)
		if err != nil {
			return nil
		}
	} else {
		node = branch.find(cschema, &id, groupSearch, valueSearch, pmap)
	}
	for i := range node {
		children = append(children, findNode(node[i], pathnode[1:], useXPath, option...)...)
	}
	return children
}

type UseXPath struct{}

func (useXpath UseXPath) IsOption() {}

// Find() finds all data nodes in the path. xpath format can be used for the path as following example.
//   Find(root, "//path/to/data[name='xxx']", UseXPath{})
func Find(root DataNode, path string, option ...Option) ([]DataNode, error) {
	if !IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, err
	}
	useXPath := false
	for i := range option {
		if _, ok := option[i].(UseXPath); ok {
			useXPath = true
		}
	}
	return findNode(root, pathnode, useXPath, option...), nil
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
	// FIXME - use xpath?
	node := findNode(root, pathnode, false)
	if len(node) == 0 {
		return nil, nil
	}
	vlist := make([]string, 0, len(node))
	for i := range node {
		if node[i].IsLeafNode() {
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
	node := findNode(root, pathnode, true)
	if len(node) == 0 {
		return nil, nil
	}
	vlist := make([]interface{}, 0, len(node))
	for i := range node {
		if node[i].IsLeafNode() {
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
	case *DataLeafList:
		dnode := &DataLeafList{
			schema: node.schema,
		}
		if len(node.value) > 0 {
			dnode.value = make([]interface{}, len(node.value))
			copy(dnode.value, node.value)
		}
		dest = dnode
	case *DataLeaf:
		dest = &DataLeaf{
			schema: node.schema,
			value:  node.value,
		}
	}
	if destParent != nil {
		_, err := destParent.insert(dest, nil)
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
		if node.schema.IsListHasKey() {
			for _, c := range node.children {
				if c.Schema().IsKey {
					if _, err := clone(dnode, c); err != nil {
						return nil, err
					}
				}
			}
		}
		destnode = dnode
	case *DataLeafList:
		dnode := &DataLeafList{
			schema: node.schema,
		}
		if len(node.value) > 0 {
			dnode.value = make([]interface{}, len(node.value))
			copy(dnode.value, node.value)
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
	if dest.IsLeafNode() {
		return Errorf(EAppTagInvalidArg, "dest must be a branch node")
	}
	destbranch := dest.(*DataBranch)

	for n := src; n != nil; n = n.Parent() {
		p := n.Parent()
		if p == nil {
			break
		}
		if destbranch.schema == p.Schema() {
			_, err := destbranch.insert(n, nil)
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
	case *DataLeafList:
		d2 := node2.(*DataLeafList)
		if len(d1.value) != len(d2.value) {
			return false
		}
		for i := range d1.value {
			if d1.value[i] != d2.value[i] {
				return false
			}
		}
		return true
	case *DataLeaf:
		d2 := node2.(*DataLeaf)
		if _, ok := d2.value.(yang.Number); ok {
			r := cmp.Equal(d1.value, d2.value)
			if r {
				return r
			}
			return false
		}
		r := d1.value == d2.value
		if r {
			return r
		}
		return false
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
			if schema.IsDuplicatableList() {
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
	case *DataLeafList:
		d := dest.(*DataLeafList)
		return d.setValue(true, s.value)
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
		editopt := &EditOption{EditOp: EditMerge}
		err = SetValueString(root, path, editopt)
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
		return fmt.Errorf("failed to create and merge the nodes in %s", path)
	case 1:
		return merge(node[0], src)
	default:
		return fmt.Errorf("more than one data node found - cannot specify the merged node")
	}
}

// PathMap converts the data node list to a map using the path.
func PathMap(node []DataNode) map[string]DataNode {
	m := map[string]DataNode{}
	for i := range node {
		m[node[i].Path()] = node[i]
	}
	return m
}

// FindAllInRoute() find all parent nodes in the path.
// The path must indicate an unique node. (not support wildcard and multiple node selection)
func FindAllInRoute(path string) []DataNode {
	return nil
}

// Get key-value pairs of the list data node.
func GetKeyValues(node DataNode) ([]string, []string) {
	keynames := node.Schema().Keyname
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

// GetOrNew returns the target data node and the ancestor node that was created first along the path from the root.
func GetOrNew(root DataNode, path string) (node DataNode, created DataNode, err error) {
	if !IsValid(root) {
		return nil, nil, fmt.Errorf("invalid root data node")
	}
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil, nil, err
	}
	var ok bool
	var branch *DataBranch
	var pmap map[string]interface{}
	node = root
	for i := range pathnode {
		branch, ok = node.(*DataBranch)
		if !ok {
			err = fmt.Errorf("%s is not a branch", node)
			break
		}
		cschema := branch.schema.GetSchema(pathnode[i].Name)
		if cschema == nil {
			err = fmt.Errorf("schema %s not found from %s", pathnode[i].Name, branch.schema.Name)
			break
		}
		pmap, err = pathnode[i].ToMap()
		if err != nil {
			break
		}
		var children []DataNode
		id, groupSearch, valueSearch := cschema.GenerateID(pmap)
		children = branch.find(cschema, &id, groupSearch, valueSearch, pmap)
		if cschema.IsDuplicatableList() {
			children = nil // clear found nodes to create a new one.
		}

		switch len(children) {
		case 0:
			child, err := NewWithValueString(cschema)
			if err != nil {
				return nil, nil, err
			}
			if err = child.UpdateByMap(pmap); err != nil {
				return nil, nil, err
			}
			if _, err = branch.insert(child, nil); err != nil {
				return nil, nil, err
			}
			if created == nil {
				created = child
			}
			node = child
		case 1:
			node = children[0]
		default:
			return nil, nil, Errorf(ETagOperationNotSupported,
				"multiple nodes are selected for GetOrNew")
		}
	}
	if err != nil {
		if created != nil {
			created.Remove()
		}
		return nil, nil, err
	}
	return node, created, nil
}

// replace() replaces a node.
func replace(from, to DataNode) error {
	schema := from.Schema()
	if schema != to.Schema() {
		return fmt.Errorf("unable to replace the different schema nodes")
	}
	parent := from.Parent()
	if parent == nil {
		return fmt.Errorf("no parent node")
	}
	keynames := schema.Keyname
	for i := range keynames {
		keynode := from.Get(keynames[i])
		if keynode != nil {
			to.Insert(Clone(keynode), nil)
		}
	}

	idA := from.ID()
	idB := to.ID()
	if idA != idB {
		return fmt.Errorf("replaced child has different id %s, %s", idA, idB)
	}
	branch := parent.(*DataBranch)
	i := indexFirst(branch, &idA)
	if i < len(branch.children) && idA == branch.children[i].ID() {
		for ; i < len(branch.children); i++ {
			if branch.children[i] == from {
				resetParent(branch.children[i])
				branch.children[i] = to
				setParent(to, branch, &idB)
				return nil
			}
		}
	}
	return fmt.Errorf("the node to replace not found")
}

// recover recovers the target node using the backup data node.
func recover(target, backup DataNode) error {
	switch t := target.(type) {
	case *DataBranch:
		b, ok := backup.(*DataBranch)
		if !ok {
			return fmt.Errorf("different type data node inserted for recovery")
		}
		for i := range t.children {
			resetParent(t.children[i])
			t.children[i] = nil
		}
		t.children = make([]DataNode, len(b.children))
		for i := range b.children {
			id := b.children[i].ID()
			t.children[i] = b.children[i]
			setParent(b.children[i], t, &id)
		}
	case *DataLeaf:
		b, ok := backup.(*DataLeaf)
		if !ok {
			return fmt.Errorf("different type data node inserted for recovery")
		}
		t.value = b.value
	case *DataLeafList:
		b, ok := backup.(*DataLeafList)
		if !ok {
			return fmt.Errorf("different type data node inserted for recovery")
		}
		t.value = make([]interface{}, len(b.value))
		copy(t.value, b.value)
	}
	return nil
}
