package yangtree

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// ManualKeyCreation - The key data nodes of list nodes are automatically created if set to false.
	ManualKeyCreation bool = false
)

type DataNode interface {
	IsYangDataNode()
	Key() string
	Schema() *yang.Entry
	Parent() DataNode

	Insert(child DataNode) error
	Delete(child DataNode) error

	Set(value ...string) error
	Remove(value ...string) error

	New(key string, value ...string) (DataNode, error)

	Get(key string) DataNode // Get an child having the key or being close

	Len() int                    // Len() returns the length of children
	Index(key string) (int, int) // Index() finds all children and returns the indexes of them.
	Child(index int) DataNode    // Child() gets the child of the index.

	Find(path string) DataNode // Find an exact data node

	// Find all matched data nodes with wildcards (*, ...) and trace back strings (./ and ../)
	// It also allows the namespace-qualified form.
	// https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md
	Retrieve(path string) ([]DataNode, error)

	String() string
	Path() string
	Value() interface{}
	ValueString() string

	MarshalJSON() ([]byte, error)      // Encoding to JSON
	MarshalJSON_IETF() ([]byte, error) // Encoding to JSON_IETF (rfc7951)

	UnmarshalJSON([]byte) error // Assembling DataNode using JSON or JSON_IETF (rfc7951) input

	// internal interfaces
	unmarshalJSON(jtree interface{}) error
	setParent(parent DataNode, key string)
}

func SearchInOrder(n int, f func(int) bool) int {
	i := 0
	for ; i < n; i++ {
		if f(i) {
			break
		}
	}
	return i
}

func isValid(node DataNode) bool {
	if node == nil {
		return false
	}
	if node.Schema() == nil {
		return false
	}
	return true
}

// isDuplicatable() checks the data nodes can be duplicated.
func isDuplicatable(schema *yang.Entry) bool {
	return schema.IsList() && schema.Key == ""
}

// _update() updates the first matched node or replaces it to the child if replace is true.
func _update(parent *DataBranch, child DataNode) error {
	if !isValid(child) {
		return fmt.Errorf("yangtree: invalid child node")
	}
	if !isValid(parent) {
		return fmt.Errorf("yangtree: invalid parent node")
	}
	if child.Parent() != nil {
		return fmt.Errorf("yangtree: the node is already appended to a parent")
	}
	if parent.schema != GetPresentParentSchema(child.Schema()) {
		return fmt.Errorf("yangtree: '%s' is not a child of %s", child, parent)
	}
	length := len(parent.children)
	key := child.Key()
	i := sort.Search(length,
		func(j int) bool {
			return key <= parent.children[j].Key()
		})
	// it just upates the first matched node.
	// if not matched, it inserts the child to the proper location.
	if i < length && parent.children[i].Key() == key && !isDuplicatable(parent.children[i].Schema()) {
		dest := parent.children[i]
		if dest.Schema() != child.Schema() {
			return fmt.Errorf("yangtree: unable to update different schema data")
		}
		switch node := child.(type) {
		case *DataLeaf, *DataLeafList:
			return dest.Set(child.ValueString())
		case *DataBranch:
			d, ok := dest.(*DataBranch)
			if !ok {
				return fmt.Errorf("yangtree: unable to update type mismatched node")
			}
			for j := range node.children {
				if err := _update(d, node.children[j]); err != nil {
					return err
				}
			}
		}
		return nil
	} else {
		for ; i < length; i++ {
			if parent.children[i].Key() > key {
				break
			}
		}
	}
	parent.children = append(parent.children, nil)
	copy(parent.children[i+1:], parent.children[i:])
	parent.children[i] = child
	child.setParent(parent, key)
	return nil
}

func _delete(parent DataNode, child DataNode) error {
	if isValid(child) {
		return fmt.Errorf("yangtree: invalid child node")
	}
	if isValid(parent) {
		return fmt.Errorf("yangtree: invalid parent node")
	}
	if child.Parent() == nil {
		return fmt.Errorf("yangtree: '%s' is already removed from a parent", child)
	}
	p, ok := parent.(*DataBranch)
	if !ok {
		return fmt.Errorf("yangtree: unable to remove a child a non-branch node")
	}
	if p.schema != GetPresentParentSchema(child.Schema()) {
		return fmt.Errorf("yangtree: '%s' is not a child of %s", child, p)
	}
	length := len(p.children)
	key := child.Key()
	i := sort.Search(length,
		func(j int) bool {
			return key <= p.children[j].Key()
		})
	if i < length {
		if p.children[i] == child {
			c := p.children[i]
			p.children = append(p.children[:i], p.children[:i+1]...)
			c.setParent(nil, "")
			return nil
		}
	}
	return fmt.Errorf("yangtree: %s not found on delete", child)
}

// _index() returns the index of a child related to the key
func _index(parent *DataBranch, key string, searchInOrder bool) (i, max int) {
	length := len(parent.children)
	if searchInOrder {
		i = SearchInOrder(length,
			func(j int) bool {
				return key <= parent.children[j].Key()
			})
	} else {
		i = sort.Search(length,
			func(j int) bool {
				return key <= parent.children[j].Key()
			})
	}
	max = i
	for ; max < length; max++ {
		if parent.children[max].Key() != key {
			break
		}
	}
	return
}

// ParseXPath parses the input xpath and return a single element with its attrs.
func ParseXPath(path *string, pos, length int) (prefix, elem string, attrs map[string]string, end int, err error) {
	begin := pos
	end = pos
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0
	switch (*path)[end] {
	case '/':
		begin++
	case '=': // ignore data string in path
		end = length
		return
	case '[', ']':
		end = length
		err = fmt.Errorf("yangtree: path '%s' starts with bracket", *path)
		return
	}
	attrname := ""
	attrs = make(map[string]string)
	end++
	for end < length {
		switch (*path)[end] {
		case '/':
			if insideBrackets <= 0 {
				if elem == "" {
					elem = (*path)[begin:end]
				}
				end++
				return
			}
		case '[':
			if (*path)[end-1] != '\\' {
				if insideBrackets <= 0 {
					if elem == "" {
						elem = (*path)[begin:end]
					}
					begin = end + 1
				}
				insideBrackets++
			}
		case ']':
			if (*path)[end-1] != '\\' {
				insideBrackets--
				if insideBrackets <= 0 {
					attrs[attrname] = (*path)[begin:end]
					attrname = ""
					begin = end + 1
				}
			}
		case '=':
			if insideBrackets <= 0 {
				if elem == "" {
					elem = (*path)[begin:end]
				}
				end = length
				return
			} else if insideBrackets == 1 && attrname == "" {
				attrname = (*path)[begin:end]
				begin = end + 1
			}
		case ':':
			if insideBrackets <= 0 {
				prefix = (*path)[begin:end]
				begin = end + 1
			}
		}
		end++
	}
	if elem == "" {
		elem = (*path)[begin:end]
	}
	return
}

func GenerateKey(schema *yang.Entry, attrs map[string]string) (string, int) {
	switch {
	case schema.IsList():
		keyname := strings.Split(schema.Key, " ")
		key := make([]string, 0, len(keyname)+1)
		key = append(key, schema.Name)
		for i := range keyname {
			if a, ok := attrs[keyname[i]]; ok {
				key = append(key, "["+keyname[i]+"="+a+"]")
			} else {
				break
			}
		}
		return strings.Join(key, ""), len(attrs) - len(keyname)
	default:
		return schema.Name, 0
	}
}

func parsePath(path *string, pos, length int) (prefix, pathelem string, end int, testAll bool, err error) {
	begin := pos
	end = pos
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0
	beginBracket := 0
	switch (*path)[end] {
	case '/':
		begin++
	case '=': // ignore data string in path
		end = length
		return
	case '[', ']':
		end = length
		err = fmt.Errorf("yangtree: path '%s' starts with bracket", *path)
		return
	}
	end++
	for end < length {
		switch (*path)[end] {
		case '/':
			if insideBrackets <= 0 {
				if pathelem == "" {
					pathelem = (*path)[begin:end]
				}
				end++
				return
			}
		case '[':
			if (*path)[end-1] != '\\' {
				if insideBrackets <= 0 {
					beginBracket = end
				}
				insideBrackets++
			}
		case ']':
			if (*path)[end-1] != '\\' {
				insideBrackets--
			}
		case '*':
			if insideBrackets == 1 {
				if end+1 < length && (*path)[end-1:end+2] == "=*]" { // * wildcard inside key value
					pathelem = (*path)[begin:beginBracket]
					testAll = true
				}
			}
		case '=':
			if insideBrackets <= 0 {
				if pathelem == "" {
					pathelem = (*path)[begin:end]
				}
				end = length
				return
			}
		case ':':
			if insideBrackets <= 0 {
				prefix = (*path)[begin:end]
				begin = end + 1
			}
		}
		end++
	}
	if pathelem == "" {
		pathelem = (*path)[begin:end]
	}
	return
}

type DataBranch struct {
	schema   *yang.Entry
	parent   DataNode
	key      string
	children []DataNode
}

func (branch *DataBranch) IsYangDataNode()     {}
func (branch *DataBranch) Schema() *yang.Entry { return branch.schema }
func (branch *DataBranch) Parent() DataNode    { return branch.parent }
func (branch *DataBranch) setParent(parent DataNode, key string) {
	branch.parent = parent
	branch.key = key
}

func (branch *DataBranch) Value() interface{} {
	return nil
}

func (branch *DataBranch) ValueString() string {
	return ""
}

func (branch *DataBranch) Path() string {
	if branch == nil {
		return ""
	}
	if branch.key == "" {
		return ""
	}
	if branch.parent != nil {
		return branch.parent.Path() + "/" + branch.key
	}
	return "/" + branch.key
}

func (branch *DataBranch) String() string {
	if branch == nil {
		return "branch.null"
	}
	return "branch." + branch.schema.Name
}

func (branch *DataBranch) New(key string, value ...string) (DataNode, error) {
	if !isValid(branch) {
		return nil, fmt.Errorf("yangtree: invalid branch node")
	}
	_, elem, attrs, _, err := ParseXPath(&key, 0, len(key))
	if err != nil {
		return nil, err
	}
	cschema := GetSchema(branch.schema, elem)
	if cschema == nil {
		return nil, fmt.Errorf("yangtree: schema '%s' not fond from %s", elem, branch)
	}
	child, err := New(cschema, value...)
	if err != nil {
		return nil, err
	}
	switch {
	case cschema.IsList():
		keyname := strings.Split(cschema.Key, " ")
		for i := range keyname {
			knode, err := New(GetSchema(cschema, keyname[i]), attrs[keyname[i]])
			if err != nil {
				return nil, fmt.Errorf("yangtree: failed to set leaf.%s to '%s'", keyname[i], attrs[keyname[i]])
			}
			if err := child.Insert(knode); err != nil {
				return nil, err
			}
		}
	}
	if err := branch.Insert(child); err != nil {
		return nil, err
	}
	return child, nil
}

func (branch *DataBranch) Set(value ...string) error {
	for i := range value {
		err := branch.UnmarshalJSON([]byte(value[i]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (branch *DataBranch) Remove(value ...string) error {
	if branch == nil {
		return nil
	}
	if branch.parent == nil {
		return nil
	}
	switch p := branch.parent.(type) {
	case *DataBranch:
		// [FIXME] need the performance improvement
		for i := range p.children {
			if p.children[i] == branch {
				p.children = append(p.children[:i], p.children[i+1:]...)
				break
			}
		}
		branch.parent = nil
	}
	return nil
}

func (branch *DataBranch) Insert(child DataNode) error {
	if !isValid(child) {
		return fmt.Errorf("yangtree: invalid child node")
	}
	if !isValid(branch) {
		return fmt.Errorf("yangtree: invalid parent node")
	}
	if child.Parent() != nil {
		return fmt.Errorf("yangtree: the node is already appended to a parent")
	}
	if branch.Schema() != GetPresentParentSchema(child.Schema()) {
		return fmt.Errorf("yangtree: '%s' is not a child of %s", child, branch)
	}
	length := len(branch.children)
	key := child.Key()
	i := sort.Search(length,
		func(j int) bool {
			return key <= branch.children[j].Key()
		})
	// it just upates the first matched node.
	// if not matched, it inserts the child to the proper location.
	if i < length && branch.children[i].Key() == key && !isDuplicatable(branch.children[i].Schema()) {
		branch.children[i].setParent(nil, "")
		branch.children[i] = child
		child.setParent(branch, key)
		return nil
	} else {
		for ; i < length; i++ {
			if branch.children[i].Key() > key {
				break
			}
		}
	}
	branch.children = append(branch.children, nil)
	copy(branch.children[i+1:], branch.children[i:])
	branch.children[i] = child
	child.setParent(branch, key)
	return nil
}

func (branch *DataBranch) Delete(child DataNode) error {
	return _delete(branch, child)
}

func (branch *DataBranch) Get(key string) DataNode {
	if key == ".." {
		return branch.parent
	} else if key == "." {
		return branch
	}
	iindex := sort.Search(len(branch.children),
		func(i int) bool { return branch.children[i].Key() >= key })
	// if iindex < len(branch.children) {
	// 	return branch.children[iindex]
	// }
	if iindex < len(branch.children) && branch.children[iindex].Key() == key {
		return branch.children[iindex]
	}
	return nil
}

func (branch *DataBranch) Child(index int) DataNode {
	if index < len(branch.children) {
		return branch.children[index]
	}
	return nil
}

func (branch *DataBranch) Index(key string) (int, int) {
	return _index(branch, key, false)
}

func (branch *DataBranch) Len() int {
	return len(branch.children)
}

func (branch *DataBranch) Find(path string) DataNode {
	if branch == nil {
		return nil
	}
	var err error
	var pos int
	var pathelem string
	var found DataNode
	pathlen := len(path)
	found = branch
Loop:
	for {
		_, pathelem, pos, _, err = parsePath(&path, pos, pathlen)
		if err != nil {
			return nil
		}
		switch pathelem {
		case "":
			if pos >= pathlen {
				break Loop
			}
			return nil
		}

		found := found.Get(pathelem)
		if found == nil {
			return nil
		}
		if pos >= pathlen {
			break
		}
	}
	return found
}

func (branch *DataBranch) Retrieve(path string) ([]DataNode, error) {
	if branch == nil {
		return nil, fmt.Errorf("yangtree: %s found on retrieve", branch)
	}
	pathlen := len(path)
	if pathlen == 0 {
		return []DataNode{branch}, nil
	}
	testAllDescendant := false
	_, pathelem, pos, testAllChildren, err := parsePath(&path, 0, pathlen)
	if err != nil {
		return nil, err
	}
	switch pathelem {
	case "":
		return nil, fmt.Errorf("yangtree: invalid path %s", path)
	case ".":
		return branch.Retrieve(path[pos:])
	case "..":
		return branch.parent.Retrieve(path[pos:])
	case "...":
		testAllDescendant = true
		fallthrough
	case "*":
		testAllChildren = true
		pathelem = ""
	default:
		// exact matching
		cschema, err := FindSchema(branch.schema, pathelem)
		if err != nil {
			return nil, nil
		}
		if cschema.IsList() && cschema.Name == pathelem {
			testAllChildren = true
		}
	}

	// wildcard maching
	if testAllChildren || testAllDescendant {
		var nodes []DataNode
		for i, child := range branch.children {
			if strings.HasPrefix(branch.children[i].Key(), pathelem) {
				n, err := child.Retrieve(path[pos:])
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, n...)
			}
		}
		if testAllDescendant {
			for _, child := range branch.children {
				n, err := child.Retrieve(path)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, n...)
			}
		}
		return nodes, nil
	}

	node := branch.Get(pathelem)
	if node == nil {
		return nil, nil
	}
	return node.Retrieve(path[pos:])
}

func (branch *DataBranch) Key() string {
	if !isValid(branch) {
		return ""
	}
	if branch.parent != nil {
		return branch.key
	}
	switch {
	case branch.schema.IsList():
		keyname := strings.Split(branch.schema.Key, " ")
		key := make([]string, 0, len(keyname)+1)
		key = append(key, branch.schema.Name)
		for i := range keyname {
			for j := range branch.children {
				// [FIXME]
				if branch.children[j].Key() == keyname[i] {
					key = append(key, "["+keyname[i]+"="+branch.children[j].ValueString()+"]")
				}
			}
		}
		return strings.Join(key, "")
	default:
		return branch.schema.Name
	}
}

type DataLeaf struct {
	schema *yang.Entry
	parent DataNode
	value  interface{}
}

func (leaf *DataLeaf) IsYangDataNode()                       {}
func (leaf *DataLeaf) Schema() *yang.Entry                   { return leaf.schema }
func (leaf *DataLeaf) setParent(parent DataNode, key string) { leaf.parent = parent }

func (leaf *DataLeaf) Parent() DataNode { return leaf.parent }
func (leaf *DataLeaf) String() string {
	if leaf == nil {
		return "leaf.null"
	}
	return "leaf." + leaf.schema.Name
}

func (leaf *DataLeaf) Path() string {
	if leaf == nil {
		return ""
	}
	if leaf.parent != nil {
		return leaf.parent.Path() + "/" + leaf.Key()
	}
	return "/" + leaf.Key()
}

func (leaf *DataLeaf) Value() interface{} {
	return leaf.value
}

func (leaf *DataLeaf) ValueString() string {
	return ValueToString(leaf.value)
}

func (leaf *DataLeaf) New(key string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("yangtree: insert not supported for %s", leaf)
}

func (leaf *DataLeaf) Set(value ...string) error {
	for i := range value {
		v, err := StringToValue(leaf.schema, leaf.schema.Type, value[i])
		if err != nil {
			return err
		}
		leaf.value = v
	}
	// fmt.Printf("\n##leaf.value Type %T %v\n", leaf.value, leaf.value)
	return nil
}

func (leaf *DataLeaf) Remove(value ...string) error {
	if leaf.parent == nil {
		return nil
	}
	switch p := leaf.parent.(type) {
	case *DataBranch:
		// [FIXME] need the performance improvement
		for i := range p.children {
			if p.children[i] == leaf {
				p.children = append(p.children[:i], p.children[i+1:]...)
				break
			}
		}
		leaf.parent = nil
	}
	return nil
}

func (leaf *DataLeaf) Insert(child DataNode) error {
	return fmt.Errorf("yangtree: insert not supported for %s", leaf)
}

func (leaf *DataLeaf) Delete(child DataNode) error {
	return fmt.Errorf("yangtree: delete not supported for %s", leaf)
}

func (leaf *DataLeaf) Get(key string) DataNode {
	return nil
}

func (leaf *DataLeaf) Child(index int) DataNode {
	return nil
}

func (leaf *DataLeaf) Index(key string) (int, int) {
	return 0, 0
}

func (leaf *DataLeaf) Len() int {
	return 0
}

func (leaf *DataLeaf) Find(path string) DataNode {
	return nil
}

func (leaf *DataLeaf) Retrieve(path string) ([]DataNode, error) {
	if leaf == nil {
		return nil, fmt.Errorf("yangtree: %s found on retrieve", leaf)
	}
	pathlen := len(path)
	if pathlen == 0 {
		return []DataNode{leaf}, nil
	}
	_, pathelem, pos, _, err := parsePath(&path, 0, pathlen)
	if err != nil {
		return nil, err
	}
	switch pathelem {
	case "":
		return nil, fmt.Errorf("yangtree: invalid path %s", path)
	case ".":
		return leaf.Retrieve(path[pos:])
	case "..":
		if leaf.parent != nil {
			return leaf.parent.Retrieve(path[pos:])
		}
		fallthrough
	case "...", "*":
		return nil, nil
	default:
		return nil, nil
	}
}

func (leaf *DataLeaf) Key() string {
	if !isValid(leaf) {
		return ""
	}
	return leaf.schema.Name
}

// DataLeafList (leaf-list data node)
// It can be set by the key
type DataLeafList struct {
	schema *yang.Entry
	parent DataNode
	value  []interface{}
}

func (leaflist *DataLeafList) IsYangDataNode() {}
func (leaflist *DataLeafList) Schema() *yang.Entry {
	if leaflist == nil {
		return nil
	}
	return leaflist.schema
}
func (leaflist *DataLeafList) setParent(parent DataNode, key string) { leaflist.parent = parent }

func (leaflist *DataLeafList) Parent() DataNode {
	if leaflist == nil {
		return nil
	}
	return leaflist.parent
}
func (leaflist *DataLeafList) String() string {
	if leaflist == nil {
		return "leaf-list.null"
	}
	return "leaf-list." + leaflist.schema.Name
}

func (leaflist *DataLeafList) Path() string {
	if leaflist == nil {
		return ""
	}
	if leaflist.parent != nil {
		return leaflist.parent.Path() + "/" + leaflist.Key()
	}
	return "/" + leaflist.Key()
}

func (leaflist *DataLeafList) Value() interface{} {
	return leaflist.value
}

func (leaflist *DataLeafList) ValueString() string {
	return ValueToString(leaflist.value)
}

func (leaflist *DataLeafList) New(key string, value ...string) (DataNode, error) {
	return nil, fmt.Errorf("yangtree: insert not supported for %s", leaflist)
}

func (leaflist *DataLeafList) Set(value ...string) error {
	if leaflist == nil {
		return fmt.Errorf("yangtree: %s found on set", leaflist)
	}
	if len(value) == 1 {
		if strings.HasPrefix(value[0], "[") && strings.HasSuffix(value[0], "]") {
			return leaflist.UnmarshalJSON([]byte(value[0]))
		}
	}
	for i := range value {
		length := len(leaflist.value)
		iindex := sort.Search(length,
			func(j int) bool {
				return ValueToString(leaflist.value[j]) >= value[i]
			})
		v, err := StringToValue(leaflist.schema, leaflist.schema.Type, value[i])
		if err != nil {
			return err
		}
		leaflist.value = append(leaflist.value, nil)
		copy(leaflist.value[iindex+1:], leaflist.value[iindex:])
		leaflist.value[iindex] = v
	}
	return nil
}

func (leaflist *DataLeafList) Remove(value ...string) error {
	if leaflist == nil {
		return fmt.Errorf("yangtree: %s found on remove", leaflist)
	}
	for i := range value {
		length := len(leaflist.value)
		iindex := sort.Search(length,
			func(j int) bool {
				return ValueToString(leaflist.value[j]) >= value[i]
			})
		if iindex < length && ValueToString(leaflist.value[iindex]) == value[i] {
			leaflist.value = append(leaflist.value[:iindex], leaflist.value[:iindex+1]...)
		}
	}
	// remove itself if there is no value inserted.
	if len(value) == 0 {
		if leaflist.parent == nil {
			return nil
		}
		if branch, ok := leaflist.parent.(*DataBranch); ok {
			branch.Delete(leaflist)
		}
	}
	return nil
}

func (leaflist *DataLeafList) Insert(child DataNode) error {
	return fmt.Errorf("yangtree: insert not supported for %s", leaflist)
	// return leaflist.Set()
}

func (leaflist *DataLeafList) Delete(child DataNode) error {
	return fmt.Errorf("yangtree: delete not supported for %s", leaflist)
	// return leaflist.Remove(key)
}

// Get finds the key from its value.
// [FIXME] Should it be supported?
func (leaflist *DataLeafList) Get(key string) DataNode {
	// length := len(leaflist.value)
	// iindex := sort.Search(length,
	// 	func(j int) bool {
	// 		return ValueToString(leaflist.value[j]) >= key
	// 	})
	// if iindex < length && ValueToString(leaflist.value[iindex]) == key {
	// 	return leaflist
	// }
	return nil
}

func (leaflist *DataLeafList) Child(index int) DataNode {
	return nil
}

// [FIXME] Should it be supported?
func (leaflist *DataLeafList) Index(key string) (int, int) {
	return 0, 0
}

// [FIXME] Should it be supported?
func (leaflist *DataLeafList) Len() int {
	return 0
}

// Get finds the key from its value.
// [FIXME] Should it be supported?
func (leaflist *DataLeafList) Find(path string) DataNode {
	length := len(leaflist.value)
	iindex := sort.Search(length,
		func(j int) bool {
			return ValueToString(leaflist.value[j]) >= path
		})
	if iindex < length && ValueToString(leaflist.value[iindex]) == path {
		return leaflist
	}
	return nil
}

func (leaflist *DataLeafList) Retrieve(path string) ([]DataNode, error) {
	if leaflist == nil {
		return nil, fmt.Errorf("yangtree: %s found on retrieve", leaflist)
	}
	pathlen := len(path)
	if pathlen == 0 {
		return []DataNode{leaflist}, nil
	}
	_, pathelem, pos, _, err := parsePath(&path, 0, pathlen)
	if err != nil {
		return nil, err
	}
	switch pathelem {
	case "":
		return nil, fmt.Errorf("yangtree: invalid path %s", path)
	case ".":
		return leaflist.Retrieve(path[pos:])
	case "..":
		if leaflist.parent != nil {
			return leaflist.parent.Retrieve(path[pos:])
		}
		fallthrough
	case "...", "*":
		return nil, nil
	default:
		node := leaflist.Find(path[pos:])
		if node == nil {
			return nil, nil
		}
		return []DataNode{node}, nil
	}
}

func (leaflist *DataLeafList) Key() string {
	if !isValid(leaflist) {
		return ""
	}
	return leaflist.schema.Name
}

func New(schema *yang.Entry, value ...string) (DataNode, error) {
	if schema == nil {
		return nil, fmt.Errorf("yangtree: schema.null on new")
	}
	var err error
	var newdata DataNode
	switch {
	case schema.Dir == nil && schema.ListAttr != nil: // leaf-list
		leaflist := &DataLeafList{
			schema: schema,
		}
		err := leaflist.Set(value...)
		if err != nil {
			return nil, err
		}
		newdata = leaflist
	case schema.Dir == nil: // leaf
		leaf := &DataLeaf{
			schema: schema,
		}
		if len(value) > 0 {
			err = leaf.Set(value...)
		} else if schema.Default != "" {
			err = leaf.Set(schema.Default)
		}
		if err != nil {
			return nil, err
		}
		newdata = leaf
	case schema.ListAttr != nil: // list
		fallthrough
	default: // container, case, etc.
		branch := &DataBranch{
			schema:   schema,
			children: []DataNode{},
		}
		err = branch.Set(value...)
		if err != nil {
			return nil, err
		}
		newdata = branch
	}
	return newdata, nil
}

func Set(root DataNode, path string, value ...string) error {
	if !isValid(root) {
		return fmt.Errorf("yangtree: invalid root node")
	}
	var err error
	var pos int
	var elem string
	var attrs map[string]string
	var created, found DataNode
	pathlen := len(path)
Loop:
	for pos < pathlen {
		_, elem, attrs, pos, err = ParseXPath(&path, pos, pathlen)
		if err != nil {
			return err
		}
		if elem == "" {
			if pos >= pathlen {
				break Loop
			}
			return fmt.Errorf("yangtree: invalid path %s", path)
		}
		if root.Schema().IsLeafList() {
			value = append(value, elem)
			break Loop
		}
		cschema := GetSchema(root.Schema(), elem)
		if cschema == nil {
			return fmt.Errorf("yangtree: schema.%s not found", elem)
		}
		key, testmore := GenerateKey(cschema, attrs)
		if testmore != 0 {
			return fmt.Errorf("yangtree: invalid key attribute in '%s'", path)
		}
		i, _ := root.Index(key)
		found = root.Child(i)
		if found == nil || found.Key() != key {
			found, err = root.New(key)
			if err != nil {
				return err
			}
			if created == nil {
				created = found
			}
		}
		root = found
	}

	err = root.Set(value...)
	if err != nil {
		if created != nil {
			created.Remove()
		}
		return err
	}
	return nil
}

func Delete(root DataNode, path string, value ...string) error {
	if root == nil {
		return fmt.Errorf("yangtree: data node is null")
	}
	var err error
	var pos int
	var elem string
	var attrs map[string]string
	var found DataNode
	pathlen := len(path)
Loop:
	for pos < pathlen {
		_, elem, attrs, pos, err = ParseXPath(&path, pos, pathlen)
		if err != nil {
			return err
		}
		if elem == "" {
			if pos >= pathlen {
				break Loop
			}
			return fmt.Errorf("yangtree: invalid path %s", path)
		}
		if root.Schema().IsLeafList() {
			value = append(value, elem)
			break Loop
		}
		cschema := GetSchema(root.Schema(), elem)
		if cschema == nil {
			return fmt.Errorf("yangtree: schema.%s not found", elem)
		}
		key, testmore := GenerateKey(cschema, attrs)
		if testmore != 0 {
			return fmt.Errorf("yangtree: invalid key attribute in '%s'", path)
		}
		i, _ := root.Index(key)
		found = root.Child(i)
		if found == nil {
			return fmt.Errorf("yangtree: data node '%s' not found", path)
		}
		root = found
	}
	return root.Remove(value...)
}
