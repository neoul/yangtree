package yangtree

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

// SchemaOption is used to store global yangtree schema options.
type SchemaOption struct {
	// If SingleLeafList is enabled, leaf-list data represents to a single leaf-list node that contains several values.
	// If disabled, leaf-list data represents to multiple leaf-list nodes that contains each single value.
	SingleLeafList     bool
	CreatedWithDefault bool   // DataNode (data node) is created with the default value of the schema if set.
	YANGLibrary2016    bool   // Load ietf-yang-library@2016-06-21
	YANGLibrary2019    bool   // Load ietf-yang-library@2019-01-04
	SchemaSetName      string // The name of the schema set
}

func (schemaoption SchemaOption) IsOption() {}

// SchemaNode - The node structure of yangtree schema.
type SchemaNode struct {
	*yang.Entry
	Parent        *SchemaNode             // The parent schema node of the schema node
	Module        *yang.Module            // The module of the schema node
	Children      []*SchemaNode           // The child schema nodes of the schema node
	Directory     map[string]*SchemaNode  // used to store the children of the schema node with all schema node's aliases
	Enum          map[string]int64        // used to store all enumeration string
	Identityref   map[string]*yang.Module // used to store all identity values of the schema node
	Keyname       []string                // used to store key list
	QName         string                  // The namespace-qualified name of RFC7951
	Qboundary     bool                    // used to indicate the boundary of the namespace-qualified name of RFC7951
	IsRoot        bool                    // used to indicate the schema is the root of the schema tree.
	IsKey         bool                    // used to indicate the schema node is a key node of a list.
	IsState       bool                    // used to indicate the schema node is state node.
	HasState      bool                    // used to indicate the schema node has a state node at least.
	OrderedByUser bool                    // used to indicate the ordering of the list or the leaf-list nodes.
	Option        *SchemaOption
	Extension     map[string]*SchemaNode
	Modules       *yang.Modules
}

func buildSchemaNode(e *yang.Entry, baseModule *yang.Module, parent *SchemaNode, option *SchemaOption, ext map[string]*SchemaNode, ms *yang.Modules) (*SchemaNode, error) {
	n := &SchemaNode{
		Entry:     e,
		Parent:    parent,
		Directory: map[string]*SchemaNode{},
		Option:    option,
		Extension: ext,
		Modules:   ms,
	}
	n.Directory["."] = n
	n.Module = getModule(e, baseModule, ms)
	orderedByUser := false
	if e.ListAttr != nil {
		if e.ListAttr.OrderedBy != nil {
			if e.ListAttr.OrderedBy.Name == "user" {
				orderedByUser = true
			}
		}
		n.OrderedByUser = orderedByUser
	}

	// namespace-qualified name of RFC 7951
	qname := strings.Join([]string{n.Module.Name, ":", e.Name}, "")
	n.QName = qname
	n.Qboundary = true

	// set keyname
	if e.Key != "" {
		n.Keyname = strings.Split(e.Key, " ")
	}

	if parent != nil {
		switch {
		case parent.IsChoice(), parent.IsCase():
			for parent.Parent != nil {
				parent = parent.Parent
				if !parent.IsChoice() && !parent.IsCase() {
					break
				}
			}
			if parent == nil {
				return nil, fmt.Errorf("no parent found ... updating schema tree failed")
			}
		}
		if parent.Module == n.Module {
			n.Qboundary = false
		}
		n.Parent = parent
		parent.Directory[e.Prefix.Name+":"+e.Name] = n
		parent.Directory[n.Module.Name+":"+e.Name] = n
		parent.Directory[e.Name] = n
		parent.Directory[".."] = parent
		parent.Children = append(parent.Children, n)

		for i := range parent.Keyname {
			if parent.Keyname[i] == e.Name {
				n.IsKey = true
			}
		}
		var isconfig yang.TriState
		for s := e; s != nil; s = s.Parent {
			isconfig = s.Config
			if isconfig != yang.TSUnset {
				break
			}
		}
		if isconfig == yang.TSFalse {
			n.IsState = true
		}
		if e.Config == yang.TSFalse {
			for p := parent; p != nil; p = p.Parent {
				p.HasState = true
			}
		}
	}
	if err := updatType(n, e.Type); err != nil {
		return nil, err
	}
	for _, ce := range e.Dir {
		if _, err := buildSchemaNode(ce, n.Module, n, option, ext, ms); err != nil {
			return nil, err
		}
	}
	return n, nil
}

func resolveGlobs(globs []string) ([]string, error) {
	results := make([]string, 0, len(globs))
	for _, pattern := range globs {
		for _, p := range strings.Split(pattern, ",") {
			if strings.ContainsAny(p, `*?[`) {
				// is a glob pattern
				matches, err := filepath.Glob(p)
				if err != nil {
					return nil, err
				}
				results = append(results, matches...)
			} else {
				// is not a glob pattern ( file or dir )
				results = append(results, p)
			}
		}
	}
	return results, nil
}

func walkDir(path, ext string) ([]string, error) {
	fs := make([]string, 0)
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fi, err := os.Stat(path)
			if err != nil {
				return err
			}
			switch mode := fi.Mode(); {
			case mode.IsRegular():
				if filepath.Ext(path) == ext {
					fs = append(fs, path)
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return fs, nil
}

func findYangFiles(files []string) ([]string, error) {
	yfiles := make([]string, 0, len(files))
	for _, file := range files {
		fi, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		switch mode := fi.Mode(); {
		case mode.IsDir():
			fls, err := walkDir(file, ".yang")
			if err != nil {
				return nil, err
			}
			yfiles = append(yfiles, fls...)
		case mode.IsRegular():
			if filepath.Ext(file) == ".yang" {
				yfiles = append(yfiles, file)
			}
		}
	}
	return yfiles, nil
}

// sanitizeArrayFlagValue trims trailing and leading brackets ([]),
// from each of ls elements only if both are present.
func sanitizeArrayFlagValue(ls []string) []string {
	res := make([]string, 0, len(ls))
	for i := range ls {
		if strings.HasPrefix(ls[i], "[") && strings.HasSuffix(ls[i], "]") {
			ls[i] = strings.Trim(ls[i], "[]")
			res = append(res, strings.Split(ls[i], " ")...)
			continue
		}
		res = append(res, ls[i])
	}
	return res
}

// CollectSchemaEntries returns all entries of the schema tree.
func CollectSchemaEntries(e *SchemaNode, leafOnly bool) []*SchemaNode {
	if e == nil {
		return []*SchemaNode{}
	}
	collected := make([]*SchemaNode, 0, 128)
	for _, child := range e.Children {
		collected = append(collected,
			CollectSchemaEntries(child, leafOnly)...)
	}
	if e.Parent != nil {
		switch {
		case e.Dir == nil && e.ListAttr != nil: // leaf-list
			fallthrough
		case e.Dir == nil: // leaf
			collected = append(collected, e)
		case e.ListAttr != nil: // list
			fallthrough
		default: // container
			if !leafOnly {
				collected = append(collected, e)
			}
		}
	}
	return collected
}

func GeneratePath(schema *SchemaNode, keyPrint, prefixTagging bool) string {
	path := ""
	for e := schema; e != nil && e.Parent != nil; e = e.Parent {
		if e.IsCase() || e.IsChoice() {
			continue
		}
		elementName := e.Name
		if prefixTagging && e.Prefix != nil {
			elementName = e.Prefix.Name + ":" + elementName
		}
		if keyPrint && e.Key != "" {
			keylist := strings.Split(e.Key, " ")
			for _, k := range keylist {
				if prefixTagging && e.Prefix != nil {
					k = e.Prefix.Name + ":" + k
				}
				elementName = elementName + "[" + k + "=*]"
			}
		}
		path = "/" + elementName + path
	}
	return path
}

func IsCreatedWithDefault(schema *SchemaNode) bool {
	return schema.Option.CreatedWithDefault
}

func updatType(schema *SchemaNode, typ *yang.YangType) error {
	if typ == nil {
		return nil
	}
	switch typ.Kind {
	case yang.Ybits:
		if schema.Enum == nil {
			schema.Enum = map[string]int64{}
		}
		newenum := typ.Bit.NameMap()
		for bs, bi := range newenum {
			schema.Enum[bs] = bi
		}
	case yang.Yenum:
		if schema.Enum == nil {
			schema.Enum = map[string]int64{}
		}
		newenum := typ.Enum.NameMap()
		for bs, bi := range newenum {
			schema.Enum[bs] = bi
		}
	case yang.Yidentityref:
		if schema.Identityref == nil {
			schema.Identityref = make(map[string]*yang.Module)
		}
		for i := range typ.IdentityBase.Values {
			name := typ.IdentityBase.Values[i].NName()
			m := yang.RootNode(typ.IdentityBase.Values[i])
			schema.Identityref[name] = m
			// identityref[name] = typ.IdentityBase.Values[i].PrefixedName()
			// identityref[typ.IdentityBase.Values[i].PrefixedName()] = name
		}
	case yang.Yunion:
		for i := range typ.Type {
			if err := updatType(schema, typ.Type[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

var collector *SchemaNode

// buildRootSchema() builds the fake root schema node of the loaded yangtree.
func buildRootSchema(module *yang.Module, option *SchemaOption, ext map[string]*SchemaNode, ms *yang.Modules) *SchemaNode {
	me := yang.ToEntry(module)
	e := me.Dir["root"]
	root, err := buildSchemaNode(e, module, nil, option, ext, ms)
	if err != nil {
		panic(err)
	}
	root.IsRoot = true
	root.Parent = nil
	if collector == nil {
		e = me.Dir["collector"]
		collector, err = buildSchemaNode(e, module, nil, option, ext, ms)
		if err != nil {
			panic(err)
		}
		collector.IsRoot = true
		collector.Parent = nil
	}
	return root
}

// GetRootSchema() returns its root schema node.
func (schema *SchemaNode) GetRootSchema() *SchemaNode {
	for schema != nil {
		if schema.IsRoot {
			return schema
		}
		schema = schema.Parent
	}
	return nil
}

// IsDuplicatable() checks the data nodes can be inserted duplicately several times.
func (schema *SchemaNode) IsDuplicatable() bool {
	if len(schema.Children) > 0 {
		// Is it non-key list node?
		return schema.ListAttr != nil && schema.Key == ""
	}
	// Is it read-only leaf-list node when single leaf-list option is enabled?
	return (schema.IsLeafList() && schema.IsState && !schema.Option.SingleLeafList)
}

// IsDuplicatableList() checks the data nodes is a list node and it can be inserted duplicately.
func (schema *SchemaNode) IsDuplicatableList() bool {
	return schema.IsList() && schema.Key == ""
}

// IsListHasKey() checks the list nodes has keys.
func (schema *SchemaNode) IsListHasKey() bool {
	return schema.IsList() && schema.Key != ""
}

// IsListable() checks if the data node is a list or a leaf-list node.
// If SingleLeafList is set, a single leaf-list node has several values and it is not listable.
func (schema *SchemaNode) IsListable() bool {
	if schema.IsDir() {
		return schema.ListAttr != nil
	}
	if schema.ListAttr != nil {
		return !schema.Option.SingleLeafList
	}
	return false
}

// IsOrderedByUser() is used to check the node is ordered by the user.
func (schema *SchemaNode) IsOrderedByUser() bool {
	return schema.OrderedByUser
}

// IsAnyData() returns true if the schema node is anydata.
func (schema *SchemaNode) IsAnyData() bool {
	return schema.Kind == yang.AnyDataEntry
}

// GetQName() returns the qname (namespace-qualified name e.g. module-name:node-name) of the schema node.
func (schema *SchemaNode) GetQName() (string, bool) {
	return schema.QName, schema.Qboundary
}

// getModule() returns the module strcture of the schema node.
func getModule(e *yang.Entry, base *yang.Module, ms *yang.Modules) *yang.Module {
	var m *yang.Module
	if e.Node != nil {
		nname := strings.SplitN(e.Node.NName(), ":", 2)
		if len(nname) > 1 {
			return yang.FindModuleByPrefix(base, nname[0])
		} else if base != nil {
			return base
		}
	}
	if m == nil {
		if ns := e.Namespace(); ns.Name != "" {
			m, _ = ms.FindModuleByNamespace(ns.Name)
		}
	}
	return m
}

func getNameAndModule(n yang.Node, base *yang.Module) (string, *yang.Module) {
	nname := strings.SplitN(n.NName(), ":", 2)
	if len(nname) > 1 {
		return nname[1], yang.FindModuleByPrefix(base, nname[0])
	} else {
		return nname[0], base
	}
}

func collectExtension(module *yang.Module, option *SchemaOption, ext map[string]*SchemaNode, ms *yang.Modules) error {
	// yang-metadadta
	for _, extstatement := range module.Extensions {
		name, mod := getNameAndModule(extstatement, module)
		if mod == nil {
			mod = module
		}
		extEntry := &yang.Entry{
			Node:   extstatement,
			Name:   name,
			Parent: nil,
		}
		presentExt := false
		for _, substatement := range extstatement.SubStatements() {
			if substatement.Kind() == "type" {
				var typedef *yang.Typedef
				tname, tmod := getNameAndModule(substatement, module)
				if tmod == nil {
					tmod = mod
				}
				typedef = yang.BaseTypedefs[tname]
				for j := range tmod.Typedef {
					if tmod.Typedef[j].Name == tname {
						typedef = tmod.Typedef[j]
						break
					}
				}
				if typedef == nil {
					return fmt.Errorf("type %q not found", substatement.NName())
				}
				extEntry.Kind = yang.LeafEntry
				extEntry.Type = typedef.YangType
				presentExt = true
			}
			if substatement.Kind() == "uses" {
				extEntry.Dir = map[string]*yang.Entry{}
				extEntry.Kind = yang.DirectoryEntry
				usesname, usemod := getNameAndModule(substatement, module)
				if usemod == nil {
					usemod = mod
				}
				for k := range mod.Grouping {
					gname, _ := getNameAndModule(mod.Grouping[k], usemod)
					if usesname == gname {
						e := yang.ToEntry(mod.Grouping[k])
						for _, ce := range e.Dir {
							ce.Parent = extEntry
							extEntry.Dir[e.Name] = ce
						}
					}
				}
				presentExt = true
			}
		}
		if presentExt {
			extNode, err := buildSchemaNode(extEntry, mod, nil, option, ext, ms)
			if err != nil {
				panic(err)
			}
			ext[name] = extNode
		}
	}
	return nil
}

type MultipleError []error

func (me MultipleError) Error() string {
	var errstr strings.Builder
	for i := range me {
		errstr.WriteString(me[i].Error() + "\n")
	}
	return errstr.String()
}

func generateSchemaTree(d, f, e []string, option ...Option) (*SchemaNode, error) {
	if len(f) == 0 {
		return nil, fmt.Errorf("no yang file")
	}

	ms := yang.NewModules()
	var schemaOption SchemaOption
	for i := range option {
		switch o := option[i].(type) {
		case SchemaOption:
			schemaOption = o
		}
	}
	ext := make(map[string]*SchemaNode)

	// built-in data model loading
	if yfile, err := Unzip(builtInYangtreeRoot); err == nil {
		if err := ms.Parse(string(yfile),
			"yangtree.yang"); err != nil {
			return nil, err
		}
	}
	if yfile, err := Unzip(builtInYangMetadata); err == nil {
		if err := ms.Parse(string(yfile),
			"ietf-yang-metadata@2016-08-05.yang"); err != nil {
			return nil, err
		}
	}
	if schemaOption.YANGLibrary2019 {
		userYangLibFile := false
		for i := range f {
			if strings.Contains(f[i], "ietf-yang-library") {
				userYangLibFile = true
			}
		}
		if !userYangLibFile {
			if yfile, err := Unzip(builtInYanglib2019); err == nil {
				if err := ms.Parse(string(yfile),
					"ietf-yang-library@2019-01-04.yang"); err != nil {
					return nil, err
				}
			}
		}
	} else if schemaOption.YANGLibrary2016 {
		userYangLibFile := false
		for i := range f {
			if strings.Contains(f[i], "ietf-yang-library") {
				userYangLibFile = true
			}
		}
		if !userYangLibFile {
			if yfile, err := Unzip(builtInYanglib2016); err == nil {
				if err := ms.Parse(string(yfile),
					"ietf-yang-library@2016-06-21.yang"); err != nil {
					return nil, err
				}
			}
		}
	}

	for _, name := range f {
		if err := ms.Read(name); err != nil {
			return nil, err
		}
	}
	if errors := ms.Process(); len(errors) > 0 {
		err := make(MultipleError, 0, len(errors)+1)
		err = append(err, errors...)
		err = append(err, fmt.Errorf("yang loading failed: %d errors", len(errors)))
		return nil, err
	}

	// Keep track of the top level modules we read in.
	// Those are the only modules we want to print below.
	var modnames []string
	root := buildRootSchema(ms.Modules["yangtree"], &schemaOption, ext, ms)
	for modname := range ms.Modules {
		if strings.HasPrefix(modname, "yangtree") ||
			strings.Contains(modname, "@") {
			continue
		}
		modnames = append(modnames, modname)
	}

	sort.Strings(modnames)
	for _, modname := range modnames {
		skip := false
		for i := range e {
			if strings.HasPrefix(modname, e[i]) {
				skip = true
			}
		}
		if !skip {
			collectExtension(ms.Modules[modname], &schemaOption, ext, ms)
			entry := yang.ToEntry(ms.Modules[modname])
			for _, schema := range entry.Dir {
				if _, ok := root.Entry.Dir[schema.Name]; ok {
					return nil, fmt.Errorf(
						"duplicated schema %q found", entry.Name)
				}
				schema.Parent = root.Entry
				root.Entry.Dir[schema.Name] = schema
				if _, err := buildSchemaNode(schema, ms.Modules[modname], root, &schemaOption, ext, ms); err != nil {
					return nil, err
				}
			}
		}
	}

	err := loadYanglibrary(root, e)
	if err != nil {
		return nil, err
	}
	return root, nil
}

// Load() loads all yang files and then build the schema tree of the files.
// dir is reference directories for imported or included yang files.
// excluded is yang module names to be excluded.
func Load(file, dir, excluded []string, option ...Option) (*SchemaNode, error) {
	dir = sanitizeArrayFlagValue(dir)
	file = sanitizeArrayFlagValue(file)
	excluded = sanitizeArrayFlagValue(excluded)

	var err error
	dir, err = resolveGlobs(dir)
	if err != nil {
		return nil, err
	}
	file, err = resolveGlobs(file)
	if err != nil {
		return nil, err
	}
	for _, dirpath := range dir {
		expanded, err := yang.PathsWithModules(dirpath)
		if err != nil {
			return nil, err
		}

		// for _, fdir := range expanded {
		// 	fmt.Printf("adding %s to yang Paths\n", fdir)
		// }
		yang.AddPath(expanded...)
	}
	yfiles, err := findYangFiles(file)
	if err != nil {
		return nil, err
	}
	file = make([]string, 0, len(yfiles))
	file = append(file, yfiles...)
	// for _, file := range file {
	// 	fmt.Printf("loading %s yang file\n", file)
	// }

	return generateSchemaTree(dir, file, excluded, option...)
}

// GetSchema() returns a child of the schema node. The namespace-qualified name is used for the name.
func (schema *SchemaNode) GetSchema(name string) *SchemaNode {
	// if schema == nil {
	// 	return nil
	// }
	return schema.Directory[name]
}

// FindSchema() returns a descendant schema node in the path.
func (schema *SchemaNode) FindSchema(path string) *SchemaNode {
	var target *SchemaNode
	pathnode, err := ParsePath(&path)
	if err != nil {
		return nil
	}
	if len(pathnode) == 0 {
		return schema
	}
	target = schema
	for i := range pathnode {
		if target == nil {
			break
		}
		switch pathnode[i].Select {
		case NodeSelectSelf:
		case NodeSelectParent:
			target = target.Parent
		case NodeSelectFromRoot:
			target = target.GetRootSchema()
		case NodeSelectAllChildren, NodeSelectAll:
			// not supported
			return nil
		}
		if pathnode[i].Name != "" {
			target = target.Directory[pathnode[i].Name]
		}
	}
	return target
}

// extractSchemaName extracts the schema name from the keystr.
func extractSchemaName(keystr *string) (string, bool, error) {
	i := strings.IndexAny(*keystr, "[=]")
	if i >= 0 {
		switch (*keystr)[i] {
		case '[':
			return (*keystr)[:i], true, nil
		default:
			return "", false, fmt.Errorf("invalid keystr %q inserted", *keystr)
		}
	}
	return *keystr, false, nil
}

// extractKeyValues extracts the list key values from the keystr.
func extractKeyValues(keys []string, keystr *string) ([]string, error) {
	length := len(*keystr)
	if length <= 0 {
		return nil, fmt.Errorf("empty key string inserted")
	}
	index := 0
	begin := 0
	end := 0
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0
	keyval := make([]string, len(keys))

	switch (*keystr)[end] {
	case '/':
		begin = 1
	case '[':
		begin = 1
		insideBrackets++
	case ']', '=':
		return nil, fmt.Errorf("key string %q starts with invalid char: ] or =", *keystr)
	}
	end++
	// fmt.Println(keys, (*keystr))

	for end < length {
		// fmt.Printf("%c, '%s', %d\n", (*keystr)[end], (*keystr)[begin:end], insideBrackets)
		switch (*keystr)[end] {
		case '/':
			if insideBrackets <= 0 {
				begin = end + 1
			}
			end++
		case '[':
			if (*keystr)[end-1] != '\\' {
				if insideBrackets <= 0 {
					begin = end + 1
				}
				insideBrackets++
			}
			end++
		case ']':
			if (*keystr)[end-1] != '\\' {
				insideBrackets--
				if insideBrackets <= 0 {
					// fmt.Println((*keystr)[begin:end])
					keyval[index-1] = (*keystr)[begin:end]
					begin = end + 1
				}
			}
			end++
		case '=':
			if insideBrackets <= 0 {
				return nil, fmt.Errorf("invalid key format %q", (*keystr)[begin:end])
			} else if insideBrackets == 1 {
				if begin < end {
					if keys[index] != (*keystr)[begin:end] {
						return nil, fmt.Errorf("invalid key %q", (*keystr)[begin:end])
					}
					index++
					begin = end + 1
				}
			}
			end++
		default:
			end++
		}
	}
	if len(keys) != index {
		return nil, fmt.Errorf("invalid key %q", (*keystr))
	}
	return keyval, nil
}

// StringToValue() converts a string value to an yangtree value
// It also check the range, length and pattern of the schema.
func StringToValue(schema *SchemaNode, typ *yang.YangType, value string) (interface{}, error) {
	switch typ.Kind {
	case yang.Ystring, yang.Ybinary:
		if len(typ.Range) > 0 {
			length := yang.FromInt(int64(len(value)))
			inrange := false
			for i := range typ.Range {
				if !(typ.Range[i].Max.Less(length) || length.Less(typ.Range[i].Min)) {
					inrange = true
				}
			}
			if inrange {
				return value, nil
			}
			return nil, fmt.Errorf("%q is out of the range, %v", value, typ.Range)
		}

		// Check that the value satisfies any regex patterns.
		patterns, isPOSIX := util.SanitizedPattern(typ)
		for _, p := range patterns {
			var r *regexp.Regexp
			var err error
			if isPOSIX {
				r, err = regexp.CompilePOSIX(p)
			} else {
				r, err = regexp.Compile(p)
			}
			if err != nil {
				return nil, fmt.Errorf("pattern compile error: %v", err)
			}
			if !r.MatchString(value) {
				return nil, fmt.Errorf("invalid pattern %q inserted for %q: %v", value, schema.Name, r)
			}
		}
		return value, nil
	case yang.Ybool:
		v := strings.ToLower(value)
		if v == "true" {
			return true, nil
		} else if v == "false" {
			return false, nil
		} else {
			return false, fmt.Errorf("%q is not boolean", value)
		}
	case yang.Yempty:
		return nil, nil
	case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64, yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64:
		number, err := yang.ParseInt(value)
		if err != nil {
			return nil, err
		}
		if len(typ.Range) > 0 {
			inrange := false
			for i := range typ.Range {
				if !(typ.Range[i].Max.Less(number) || number.Less(typ.Range[i].Min)) {
					inrange = true
				}
			}
			if !inrange {
				return nil, fmt.Errorf("%q is out of the range, %v", value, typ.Range)
			}
		}
		if typ.Kind == yang.Yuint64 {
			return number.Value, nil
		}
		n, err := number.Int()
		if err != nil {
			return nil, err
		}
		switch typ.Kind {
		case yang.Yint8:
			return int8(n), nil
		case yang.Yint16:
			return int16(n), nil
		case yang.Yint32:
			return int32(n), nil
		case yang.Yint64:
			return int64(n), nil
		case yang.Yuint8:
			return uint8(n), nil
		case yang.Yuint16:
			return uint16(n), nil
		case yang.Yuint32:
			return uint32(n), nil
		case yang.Yuint64:
			return uint64(n), nil
		}
		return number, nil
	case yang.Ybits, yang.Yenum:
		if _, ok := schema.Enum[value]; ok {
			return value, nil
		}
	case yang.Yidentityref:
		if i := strings.Index(value, ":"); i >= 0 {
			iref := value[i+1:]
			if _, ok := schema.Identityref[iref]; ok {
				return iref, nil
			}
		} else {
			if _, ok := schema.Identityref[value]; ok {
				return value, nil
			}
		}
		return nil, fmt.Errorf("identityref %q not found", value)
	case yang.Yleafref:
		// [FIXME] Check the schema ? or data ?
		// [FIXME] check the path refered
		return value, nil
	case yang.Ydecimal64:
		number, err := yang.ParseDecimal(value, uint8(typ.FractionDigits))
		if err != nil {
			return nil, err
		}
		if len(typ.Range) > 0 {
			inrange := false
			for i := range typ.Range {
				if !(typ.Range[i].Max.Less(number) || number.Less(typ.Range[i].Min)) {
					inrange = true
				}
			}
			if inrange {
				return number, nil
			}
			return nil, fmt.Errorf("%q is out of the range, %v", value, typ.Range)
		}
	case yang.Yunion:
		for i := range typ.Type {
			v, err := StringToValue(schema, typ.Type[i], value)
			if err == nil {
				return v, nil
			}
		}
	case yang.YinstanceIdentifier:
		return value, nil
	case yang.Ynone:
		break
	}
	return nil, fmt.Errorf("invalid value %q inserted for %q", value, schema.Name)
}

// ValueToString() converts a golang value to the string value.
func ValueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(uint64(v), 10)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case yang.Number:
		return v.String()
	case nil:
		return ""
	}
	return fmt.Sprint(value)
}

// ValueToJSONBytes() marshals a value based on its schema, type and representing format.
func ValueToJSONBytes(schema *SchemaNode, typ *yang.YangType, value interface{}, rfc7951 bool) ([]byte, error) {
	switch typ.Kind {
	case yang.Yunion:
		for i := range typ.Type {
			v, err := ValueToJSONBytes(schema, typ.Type[i], value, rfc7951)
			if err == nil {
				return v, nil
			}
		}
		return nil, fmt.Errorf("unexpected value \"%v\" for %q type", value, typ.Name)
	case yang.YinstanceIdentifier:
		// [FIXME] The leftmost (top-level) data node name is always in the
		//   namespace-qualified form (qname).
	case yang.Ydecimal64:
		switch v := value.(type) {
		case yang.Number:
			return []byte(v.String()), nil
		case string:
			return []byte(v), nil
		}
	}
	if rfc7951 {
		switch typ.Kind {
		// case yang.Ystring, yang.Ybinary:
		// case yang.Ybool:
		// case yang.Yleafref:
		// case yang.Ynone:
		// case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yuint8, yang.Yuint16, yang.Yuint32:
		// case yang.Ybits, yang.Yenum:
		case yang.Yempty:
			return []byte("[null]"), nil
		case yang.Yidentityref:
			if s, ok := value.(string); ok {
				m, ok := schema.Identityref[s]
				if !ok {
					return nil, fmt.Errorf("%q is not a value of %q", s, typ.Name)
				}
				return json.Marshal(m.Name + ":" + s)
			}
		case yang.Yint64:
			if v, ok := value.(int64); ok {
				str := strconv.FormatInt(v, 10)
				return json.Marshal(str)
			}
		case yang.Yuint64:
			if v, ok := value.(uint64); ok {
				str := strconv.FormatUint(v, 10)
				return json.Marshal(str)
			}
		}
	}
	// else {
	// 	switch typ.Kind {
	// 	case yang.Yempty:
	// 		return []byte("null"), nil
	// 	}
	// }
	return json.Marshal(value)
}

func isIntegral(val float64) bool {
	return val == float64(int(val))
}

// JSONValueToString() returns a string value from the json scalar value that is unmarshalled by json.Unmarshal()
func JSONValueToString(jval interface{}) (string, error) {
	switch jdata := jval.(type) {
	case float64:
		if isIntegral(jdata) {
			return fmt.Sprint(int64(jdata)), nil
		}
		return fmt.Sprint(jdata), nil
	case string:
		return jdata, nil
	case nil:
		return "", nil
	case bool:
		if jdata {
			return "true", nil
		}
		return "false", nil
	case []interface{}:
		if len(jdata) == 1 && jdata[0] == nil {
			return "true", nil
		}
	}
	return "", fmt.Errorf("unexpected json-value %v (%T)", jval, jval)
}

// GetMust() returns the "must" statements of the schema node.
func (schema *SchemaNode) GetMust() []*yang.Must {
	switch n := schema.Node.(type) {
	case *yang.Container:
		return n.Must
	case *yang.Leaf:
		return n.Must
	case *yang.LeafList:
		return n.Must
	case *yang.List:
		return n.Must
	// case *yang.Choice:
	// case *yang.Case:
	// case *yang.Augment:
	// case *yang.Action:
	// case *yang.Grouping:
	// case *yang.Argument:
	// case *yang.BelongsTo:
	// case *yang.Deviation:
	// case *yang.Bit:
	// case *yang.Deviate:
	// 	return n.Must
	case *yang.AnyXML:
		return n.Must
	case *yang.AnyData:
		return n.Must
	}
	return nil
}

func SplitQName(qname *string) (string, string) {
	if i := strings.Index(*qname, ":"); i >= 0 {
		return (*qname)[:i], (*qname)[i+1:]
	}
	return "", *qname
}

// Unzip() is used to extracts the builtin schema.
func Unzip(gzj []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(gzj))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	s, err := ioutil.ReadAll(gzr)
	if err != nil {
		return nil, err
	}
	return s, nil
}
