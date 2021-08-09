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

// SchemaOption is the widely used option for the creation/deletion of the data tree.
type SchemaOption struct {
	CreatedWithDefault bool   // DataNode (data node) is created with the default value of the schema if set.
	SchemaName         string // The name of the schema tree
	Metadata           map[string]*yang.Entry
}

func (schemaoption SchemaOption) IsOption() {}

// SchemaMetadata is used to keep the additional data for each schema entry.
type SchemaMetadata struct {
	Module      *yang.Module // used to store the module of the schema entry
	Child       []*yang.Entry
	Dir         map[string]*yang.Entry // used to store the children of the schema entry with all schema entry's aliases
	Enum        map[string]int64       // used to store all enumeration string
	Identityref map[string]string      // used to store all identity values of the schema entry
	Keyname     []string               // used to store key list
	QName       string                 // namespace-qualified name of RFC 7951
	Qboundary   bool                   // used to indicate the boundary of the namespace-qualified name of RFC 7951
	IsRoot      bool                   // used to indicate the schema is the root of the schema tree.
	IsKey       bool                   // used to indicate the schema entry is a key node of a list.
	IsState     bool                   // used to indicate the schema entry is state node.
	HasState    bool                   // used to indicate the schema entry has a state node at least.
	Option      *SchemaOption
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
func CollectSchemaEntries(e *yang.Entry, leafOnly bool) []*yang.Entry {
	if e == nil {
		return []*yang.Entry{}
	}
	collected := make([]*yang.Entry, 0, 128)
	for _, child := range e.Dir {
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

func GeneratePath(schema *yang.Entry, keyPrint, prefixTagging bool) string {
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

func getEnum(schema *yang.Entry) map[string]int64 {
	if schema == nil || schema.Annotation == nil {
		return nil
	}
	if data, ok := schema.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			return m.Enum
		}
	}
	return nil
}

func setEnum(schema *yang.Entry, enum map[string]int64) error {
	if schema == nil || schema.Annotation == nil {
		return fmt.Errorf("nil schema or annotation for setting enum")
	}
	if data, ok := schema.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			m.Enum = enum
			return nil
		}
	}
	return fmt.Errorf("no schema meta data for setting enum")
}

func getIdentityref(schema *yang.Entry) map[string]string {
	if schema == nil || schema.Annotation == nil {
		return nil
	}
	if data, ok := schema.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			return m.Identityref
		}
	}
	return nil
}

func setIdentityref(schema *yang.Entry, identityref map[string]string) error {
	if schema == nil || schema.Annotation == nil {
		return fmt.Errorf("nil schema or annotation for setting identityref")
	}
	if data, ok := schema.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			m.Identityref = identityref
			return nil
		}
	}
	return fmt.Errorf("no schema meta data for setting identityref")
}

func GetModule(schema *yang.Entry) *yang.Module {
	if data, ok := schema.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			return m.Module
		}
	}
	return nil
}

func IsCreatedWithDefault(schema *yang.Entry) bool {
	if data, ok := schema.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			return m.Option.CreatedWithDefault
		}
	}
	return false
}

func updateSchemaMetaForType(schema *yang.Entry, typ *yang.YangType) error {
	if typ == nil {
		return nil
	}
	switch typ.Kind {
	case yang.Ybits:
		var enum map[string]int64
		if enum = getEnum(schema); enum == nil {
			enum = map[string]int64{}
		}
		newenum := typ.Bit.NameMap()
		for bs, bi := range newenum {
			enum[bs] = bi
		}
		if err := setEnum(schema, enum); err != nil {
			return err
		}
	case yang.Yenum:
		var enum map[string]int64
		if enum = getEnum(schema); enum == nil {
			enum = map[string]int64{}
		}
		newenum := typ.Enum.NameMap()
		for bs, bi := range newenum {
			enum[bs] = bi
		}
		if err := setEnum(schema, enum); err != nil {
			return err
		}
	case yang.Yidentityref:
		var identityref map[string]string
		if identityref = getIdentityref(schema); identityref == nil {
			identityref = map[string]string{}
		}
		for i := range typ.IdentityBase.Values {
			QValue := fmt.Sprintf("%s:%s",
				yang.RootNode(typ.IdentityBase.Values[i]).Name, typ.IdentityBase.Values[i].NName())
			identityref[typ.IdentityBase.Values[i].NName()] = QValue
			// identityref[typ.IdentityBase.Values[i].NName()] = typ.IdentityBase.Values[i].PrefixedName()
			// identityref[typ.IdentityBase.Values[i].PrefixedName()] = typ.IdentityBase.Values[i].NName()
		}
		if err := setIdentityref(schema, identityref); err != nil {
			return err
		}
	case yang.Yunion:
		for i := range typ.Type {
			if err := updateSchemaMetaForType(schema, typ.Type[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildRootEntry(mods map[string]*yang.Module, option *SchemaOption) *yang.Entry {
	rootEntry := &yang.Entry{
		Dir: map[string]*yang.Entry{},
		Annotation: map[string]interface{}{
			"modules": mods,
			"meta": &SchemaMetadata{
				IsRoot: true,
				Dir:    map[string]*yang.Entry{},
				Option: option,
			},
		},
		Kind: yang.DirectoryEntry, // root is container
	}
	rootEntry.Name = "root"
	rootEntry.Kind = yang.DirectoryEntry
	// Always annotate the root as a fake root,
	// so that it is not treated as a path element.
	rootEntry.Annotation["root"] = true
	return rootEntry
}

func IsRootSchema(schema *yang.Entry) bool {
	if m, ok := schema.Annotation["meta"]; ok {
		meta := m.(*SchemaMetadata)
		return meta.IsRoot
	}
	return false
}

func GetRootSchema(schema *yang.Entry) *yang.Entry {
	for schema != nil {
		if IsRootSchema(schema) {
			return schema
		}
		schema = schema.Parent
	}
	return nil
}

// IsDuplicatedList() checks the data nodes can be duplicated.
func IsDuplicatedList(schema *yang.Entry) bool {
	return schema.IsList() && schema.Key == ""
}

// IsUniqueList() checks the data nodes can be duplicated.
func IsUniqueList(schema *yang.Entry) bool {
	return schema.IsList() && schema.Key != ""
}

func IsList(schema *yang.Entry) bool {
	return schema.IsList()
}

func GetAllModules(schema *yang.Entry) map[string]*yang.Module {
	if schema == nil {
		return nil
	}
	for schema.Parent != nil {
		schema = schema.Parent
	}
	if m, ok := schema.Annotation["modules"]; ok {
		return m.(map[string]*yang.Module)
	}
	return nil
}

func GetSchemaMeta(schema *yang.Entry) *SchemaMetadata {
	if m, ok := schema.Annotation["meta"]; ok {
		return m.(*SchemaMetadata)
	}
	return nil
}

// Return qname (namespace-qualified name e.g. module-name:node-name)
func GetQName(schema *yang.Entry) (string, bool) {
	if m, ok := schema.Annotation["meta"]; ok {
		meta := m.(*SchemaMetadata)
		return meta.QName, meta.Qboundary
	}
	return "", false
}

func updateSchemaEntry(parent, schema *yang.Entry, current *yang.Module, modules *yang.Modules) error {
	if schema.Annotation == nil {
		schema.Annotation = map[string]interface{}{}
	}
	meta := &SchemaMetadata{
		Enum: map[string]int64{},
	}
	schema.Annotation["meta"] = meta

	module, err := modules.FindModuleByPrefix(schema.Prefix.Name)
	if err != nil {
		return err
	}
	meta.Module = module

	// namespace-qualified name of RFC 7951
	qname := strings.Join([]string{module.Name, ":", schema.Name}, "")
	meta.QName = qname
	if current != module {
		meta.Qboundary = true
	}

	// set keyname
	if schema.Key != "" {
		meta.Keyname = strings.Split(schema.Key, " ")
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
				return fmt.Errorf("no parent found ... updating schema tree failed")
			}
		}
		// Using the Annotation, update addtional information for the schema.
		if parent.Annotation == nil {
			parent.Annotation = map[string]interface{}{}
		}
		pmeta := GetSchemaMeta(parent)
		if pmeta == nil {
			pmeta = &SchemaMetadata{}
			parent.Annotation["meta"] = pmeta
		}
		meta.Option = pmeta.Option
		if pmeta.Dir == nil {
			pmeta.Dir = map[string]*yang.Entry{}
		}
		pmeta.Dir[schema.Prefix.Name+":"+schema.Name] = schema
		pmeta.Dir[module.Name+":"+schema.Name] = schema
		pmeta.Dir[schema.Name] = schema
		pmeta.Dir["."] = schema
		pmeta.Dir[".."] = GetPresentParentSchema(schema)
		pmeta.Child = append(pmeta.Child, schema)

		for i := range pmeta.Keyname {
			if pmeta.Keyname[i] == schema.Name {
				meta.IsKey = true
			}
		}
		var isconfig yang.TriState
		for s := schema; s != nil; s = s.Parent {
			isconfig = s.Config
			if isconfig != yang.TSUnset {
				break
			}
		}
		if isconfig == yang.TSFalse {
			meta.IsState = true
		}
		if schema.Config == yang.TSFalse {
			for p := parent; p != nil; p = p.Parent {
				if m := GetSchemaMeta(p); m != nil {
					m.HasState = true
				}
			}
		}
	}
	if err := updateSchemaMetaForType(schema, schema.Type); err != nil {
		return err
	}

	for _, child := range schema.Dir {
		if err := updateSchemaEntry(schema, child, module, modules); err != nil {
			return err
		}
	}
	return nil
}

func buildMetaEntry(meta *yang.Statement, module *yang.Module, typ *yang.YangType, option *SchemaOption) {
	// 'description', 'if-feature', 'reference', 'status', and 'units'
	metaEntry := &yang.Entry{
		Name: meta.NName(),
		Node: meta,
		Dir:  map[string]*yang.Entry{},
		Annotation: map[string]interface{}{
			"metadata": true,
			"meta": &SchemaMetadata{
				Dir:    map[string]*yang.Entry{},
				Option: option,
			},
		},
		Kind: yang.LeafEntry,
		Type: typ,
		// namespace: module.Namespace,
	}
	if len(option.Metadata) == 0 {
		option.Metadata = make(map[string]*yang.Entry)
	}
	option.Metadata[meta.NName()] = metaEntry
}

func collectMetadata(module *yang.Module, option *SchemaOption) error {
	// yang-metadadta
	for i := range module.Extensions {
		ext := module.Extensions[i]
		name := strings.SplitN(ext.Keyword, ":", 2)
		var m *yang.Module
		if len(name) > 1 {
			m = yang.FindModuleByPrefix(module, name[0])
		} else {
			m = module
		}
		if m.Name == "ietf-yang-metadata" && name[1] == "annotation" {
			// fmt.Println(name, ext.NName())
			typname := ""
			substatements := ext.SubStatements()
			for k := range substatements {
				if substatements[k].Kind() == "type" {
					typname = substatements[k].NName()
					break
				}
			}
			var typedef *yang.Typedef
			tname := strings.SplitN(typname, ":", 2)
			if len(tname) > 1 {
				m = yang.FindModuleByPrefix(module, tname[0])
				if m != nil {
					for j := range m.Typedef {
						if m.Typedef[j].Name == tname[1] {
							typedef = m.Typedef[j]
							break
						}
					}
				}
			} else {
				typedef = yang.BaseTypedefs[tname[0]]
			}
			buildMetaEntry(ext, module, typedef.YangType, option)
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

func generateSchemaTree(d, f, e []string, option ...Option) (*yang.Entry, error) {
	if len(f) == 0 {
		return nil, fmt.Errorf("no yang file")
	}

	var schemaOption SchemaOption
	for i := range option {
		switch o := option[i].(type) {
		case SchemaOption:
			schemaOption = o
		}
	}

	ms := yang.NewModules()
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

	if _metayang, err := Unzip(metayang); err == nil {
		// built-in data model loading
		if err := ms.Parse(string(_metayang),
			"ietf-yang-metadata@2016-08-05.yang"); err != nil {
			return nil, err
		}
	}

	// Keep track of the top level modules we read in.
	// Those are the only modules we want to print below.
	mods := map[string]*yang.Module{}
	var names []string

	for _, m := range ms.Modules {
		if mods[m.Name] == nil {
			mods[m.Name] = m
			names = append(names, m.Name)
		}
	}
	sort.Strings(names)
	entries := make([]*yang.Entry, 0, len(names))
	for _, n := range names {
		collectMetadata(mods[n], &schemaOption)
		e := yang.ToEntry(mods[n])
		entries = append(entries, e)
	}
	root := buildRootEntry(mods, &schemaOption)
	for _, mentry := range entries {
		// fmt.Println(mentry.Extra)
		skip := false
		for i := range e {
			if mentry.Name == e[i] {
				skip = true
			}
		}
		if !skip {
			for _, schema := range mentry.Dir {
				if same, ok := root.Dir[schema.Name]; ok {
					mo := GetModule(same)
					return nil, fmt.Errorf(
						"duplicated schema found: %q, %q",
						mentry.Name, mo.Name)
				}
				schema.Parent = root
				root.Dir[schema.Name] = schema
				if err := updateSchemaEntry(root, schema, nil, ms); err != nil {
					return nil, err
				}
			}
		}
	}
	// fmt.Println(GetSchemaMeta(root).Option.Metadata)
	if _, ok := mods["ietf-yang-library"]; ok {
		err := loadYanglibrary(root, mods, e)
		if err != nil {
			return nil, err
		}
	}
	return root, nil
}

// Load loads all yang files (file) from dir directories and build the schema tree.
func Load(file, dir, excluded []string, option ...Option) (*yang.Entry, error) {
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

// GetAllChildSchema() returns a child schema node. It provides the child name tagged its prefix or module name.
func GetAllChildSchema(schema *yang.Entry) []*yang.Entry {
	if schema == nil {
		return nil
	}
	if meta := GetSchemaMeta(schema); meta != nil {
		return meta.Child
	}
	return nil
}

// GetSchema() returns a child schema node. It provides the child name tagged its prefix or module name.
func GetSchema(schema *yang.Entry, name string) *yang.Entry {
	var child *yang.Entry
	if schema == nil {
		return nil
	}
	if meta := GetSchemaMeta(schema); meta != nil {
		child = meta.Dir[name]
	}
	return child
}

// GetPresentParentSchema() is used to get the non-choice and non-case parent schema entry.
func GetPresentParentSchema(schema *yang.Entry) *yang.Entry {
	for p := schema.Parent; p != nil; p = p.Parent {
		if !p.IsCase() && !p.IsChoice() {
			return p
		}
	}
	return nil
}

// IsEqualSchema() checks if they have the same schema.
func IsEqualSchema(a, b DataNode) bool {
	return a.Schema() == b.Schema()
}

// FindSchema() returns a descendant schema node. It provides the child name tagged its prefix or module name.
func FindSchema(schema *yang.Entry, path string) *yang.Entry {
	var target *yang.Entry
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
			target = GetPresentParentSchema(target)
		case NodeSelectFromRoot:
			for target.Parent != nil {
				target = target.Parent
			}
		case NodeSelectAllChildren, NodeSelectAll:
			// not supported
			return nil
		}
		if pathnode[i].Name != "" {
			meta := GetSchemaMeta(target)
			if meta == nil {
				return nil
			}
			target = meta.Dir[pathnode[i].Name]
		}
	}
	return target
}

func FindModule(schema *yang.Entry, path string) *yang.Module {
	e := FindSchema(schema, path)
	if e == nil {
		return nil
	}
	return GetModule(e)
}

func HasUniqueListParent(schema *yang.Entry) bool {
	for n := schema; n != nil; n = n.Parent {
		if IsUniqueList(n) {
			return true
		}
	}
	return false
}

func GetKeynames(schema *yang.Entry) []string {
	meta := GetSchemaMeta(schema)
	return meta.Keyname
}

func IsKeyNode(schema *yang.Entry) bool {
	meta := GetSchemaMeta(schema)
	return meta.IsKey
}

func IsConfig(schema *yang.Entry) bool {
	meta := GetSchemaMeta(schema)
	return !meta.IsState
}

func IsState(schema *yang.Entry) bool {
	meta := GetSchemaMeta(schema)
	return meta.IsState
}

// ExtractSchemaName extracts the schema name from the keystr.
func ExtractSchemaName(keystr *string) (string, bool, error) {
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

// ExtractKeyValues extracts the list key values from keystr
func ExtractKeyValues(keys []string, keystr *string) ([]string, error) {
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

// StringToValue converts string to the yangtree value
// It also check the range, length and pattern of the schema.
func StringToValue(schema *yang.Entry, typ *yang.YangType, value string) (interface{}, error) {
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
		emap := getEnum(schema)
		if _, ok := emap[value]; ok {
			return value, nil
		}
	case yang.Yidentityref:
		imap := getIdentityref(schema)
		if i := strings.Index(value, ":"); i >= 0 {
			iref := value[i+1:]
			if _, ok := imap[iref]; ok {
				return iref, nil
			}
		} else {
			if _, ok := imap[value]; ok {
				return value, nil
			}
		}
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

func ValueToJSONBytes(schema *yang.Entry, typ *yang.YangType, value interface{}, rfc7951 bool) ([]byte, error) {
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
				imap := getIdentityref(schema)
				qvalue, ok := imap[s]
				if !ok {
					return nil, fmt.Errorf("%q is not a value of %q", s, typ.Name)
				}
				return json.Marshal(qvalue)
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

// GetMust returns the when XPath statement of e if able.
func GetMust(schema *yang.Entry) []*yang.Must {
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

// zip bytes for "ietf-yang-metadata@2016-08-05.yang"
var metayang = []byte{
	0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x84, 0x56, 0xdb, 0x6e, 0x1b, 0xb1, 0x11, 0x7d, 0xf7, 0x57,
	0xc, 0xfc, 0x22, 0x1b, 0x90, 0x56, 0x4e, 0x90, 0xa4, 0xed, 0x26, 0x8, 0xec, 0xf8, 0x86, 0xb6, 0xb2, 0x53, 0xc4, 0x2e,
	0xdc, 0x3e, 0x8e, 0xc8, 0x59, 0x2d, 0x63, 0x2e, 0xb9, 0x20, 0x67, 0x25, 0x2b, 0x41, 0xfe, 0xbd, 0x18, 0xee, 0xd5, 0x4e,
	0x8c, 0xfa, 0xc9, 0x5c, 0xce, 0x9c, 0xb9, 0x9d, 0x33, 0x54, 0xe5, 0x75, 0x63, 0x9, 0xc, 0x71, 0xb1, 0xd8, 0xa3, 0xdb,
	0x2c, 0x2a, 0x62, 0xd4, 0xc8, 0x8, 0x3f, 0xf, 0xe, 0x0, 0x1c, 0x56, 0x14, 0x6b, 0x54, 0x4, 0x87, 0x4d, 0x70, 0xb9,
	0x98, 0xe5, 0x35, 0x6, 0xac, 0x62, 0xfe, 0x54, 0xd9, 0xdc, 0xc5, 0x5c, 0x9c, 0xf2, 0xdf, 0xdd, 0xf, 0x3f, 0x8a, 0x7b,
	0x1d, 0xa8, 0x30, 0x4f, 0x70, 0x58, 0xe9, 0xf6, 0xec, 0xc3, 0x6, 0x9d, 0xf9, 0x81, 0x6c, 0xbc, 0x3b, 0x0, 0x0, 0x38,
	0xfc, 0xfb, 0xe5, 0xfd, 0x15, 0xdc, 0x5e, 0xde, 0xdf, 0x7c, 0xbd, 0x80, 0xa3, 0xdb, 0xcb, 0xfb, 0xf3, 0xaf, 0xb7, 0x57,
	0x70, 0x21, 0xf1, 0x6f, 0xbc, 0x26, 0x6b, 0xdc, 0x6, 0x56, 0xe8, 0x36, 0xd, 0x6e, 0xe8, 0x18, 0x1e, 0x7c, 0x78, 0x94,
	0x2f, 0xd7, 0xc1, 0x37, 0x75, 0x8b, 0xa8, 0xbc, 0x63, 0x54, 0xdc, 0x82, 0x3d, 0x5c, 0xc3, 0x3, 0xad, 0x73, 0x0, 0xf8,
	0x54, 0x32, 0xd7, 0x31, 0x5f, 0x2e, 0x25, 0x17, 0xe, 0xa8, 0x1e, 0x29, 0x64, 0x92, 0x65, 0xe6, 0xc3, 0x66, 0xb9, 0xdb,
	0x2c, 0x1d, 0x71, 0xe5, 0xf5, 0xf2, 0xf3, 0x41, 0xf2, 0x84, 0x87, 0x6b, 0x58, 0x99, 0xc8, 0x39, 0xc0, 0xa7, 0xa, 0x8d,
	0x65, 0x9f, 0xb7, 0x6, 0xa7, 0xbd, 0xcf, 0xc4, 0xf0, 0xbc, 0x44, 0x13, 0x72, 0x58, 0xf9, 0x6, 0xbe, 0x50, 0xd8, 0x50,
	0x68, 0x6f, 0xc6, 0xbf, 0x1e, 0xc2, 0xae, 0xd3, 0xf5, 0xa9, 0xc5, 0xb5, 0xcb, 0x1c, 0xf1, 0xef, 0x18, 0xff, 0x24, 0xc7,
	0xf0, 0x80, 0x1c, 0xc9, 0xbd, 0x6, 0xf2, 0xb8, 0x4b, 0xd7, 0xa7, 0xdf, 0x1b, 0x67, 0x6a, 0xa, 0x53, 0x9c, 0x4b, 0x6d,
	0xd8, 0x7, 0x29, 0x77, 0x85, 0xda, 0x44, 0x8b, 0x5b, 0x58, 0x95, 0x9e, 0x1f, 0xf1, 0xd5, 0x84, 0xd2, 0xed, 0xa9, 0x33,
	0x2a, 0x53, 0x3f, 0x3e, 0xb7, 0xd, 0xd4, 0x14, 0x55, 0x30, 0xf5, 0x38, 0x91, 0xfb, 0xd2, 0x44, 0xf8, 0xef, 0xd9, 0xed,
	0x35, 0x74, 0xdc, 0xd0, 0x54, 0x18, 0x47, 0x11, 0xd0, 0xc1, 0x8c, 0x9e, 0x98, 0x5c, 0x34, 0xde, 0xcd, 0x20, 0x32, 0x32,
	0x55, 0x52, 0x0, 0x97, 0xc8, 0x80, 0xd6, 0xfa, 0x5d, 0x6c, 0x23, 0x17, 0x3e, 0xb4, 0x5e, 0x32, 0xac, 0x81, 0x50, 0xe8,
	0x9c, 0xe7, 0x34, 0xfa, 0x98, 0x75, 0x15, 0x9c, 0xfb, 0x7a, 0x1f, 0xcc, 0xa6, 0x64, 0x38, 0x52, 0xc7, 0xf0, 0xf6, 0xe4,
	0xcd, 0x7, 0x48, 0x84, 0xb8, 0xf, 0x4d, 0x64, 0x40, 0xa7, 0x81, 0x4b, 0x82, 0x9a, 0x42, 0xf4, 0x2e, 0x82, 0xd1, 0xe4,
	0xd8, 0x14, 0x86, 0x34, 0x60, 0x17, 0x9, 0x1b, 0x2e, 0x7d, 0x88, 0xe0, 0x8b, 0x64, 0xa9, 0xbc, 0xa6, 0xc, 0xe0, 0xcc,
	0x5a, 0x48, 0xb0, 0x11, 0x2, 0x45, 0xa, 0x5b, 0xd2, 0x7d, 0xc4, 0x6f, 0xa4, 0x4d, 0xe4, 0x60, 0xd6, 0x8d, 0x24, 0x92,
	0x42, 0x34, 0x91, 0xc0, 0x38, 0x88, 0xbe, 0x9, 0x8a, 0xd2, 0x97, 0xb5, 0x71, 0x18, 0xf6, 0x52, 0x46, 0x15, 0xe7, 0xb0,
	0x33, 0x5c, 0x82, 0xef, 0xa6, 0x2c, 0x7, 0xdf, 0xb0, 0xf4, 0xc6, 0x14, 0x46, 0xa5, 0x72, 0xe6, 0x60, 0xa2, 0x24, 0x59,
	0x19, 0x66, 0xd2, 0x50, 0x37, 0x21, 0x36, 0x28, 0x7d, 0xf1, 0xf3, 0x4, 0x17, 0x9b, 0xf5, 0x77, 0x52, 0x72, 0x6e, 0x31,
	0x24, 0x53, 0x6b, 0x14, 0xb9, 0x48, 0xc0, 0x14, 0xaa, 0xd8, 0xb2, 0xd8, 0x38, 0xd2, 0x60, 0xdc, 0x3c, 0xdd, 0xdf, 0x99,
	0xaa, 0xb6, 0x6d, 0xad, 0x5f, 0xee, 0x2e, 0x60, 0xd5, 0x99, 0x47, 0xe2, 0xa1, 0xc5, 0x5c, 0x4a, 0xda, 0x77, 0xa4, 0x52,
	0x25, 0xef, 0x32, 0xd5, 0x77, 0x61, 0x6c, 0xe1, 0x2c, 0xc2, 0x8a, 0x36, 0x68, 0xe1, 0x5f, 0xc1, 0x6f, 0x8d, 0xcc, 0x2d,
	0xf6, 0x6d, 0xb0, 0xc8, 0x32, 0x1d, 0xf6, 0xad, 0xf9, 0x85, 0x57, 0x8d, 0xc, 0xb3, 0xbb, 0x3f, 0x12, 0xfd, 0xe4, 0xcb,
	0x25, 0xb, 0xa, 0xd1, 0x28, 0x9d, 0x2e, 0xef, 0x85, 0x71, 0x85, 0x3f, 0xee, 0x9b, 0x9a, 0x28, 0xb3, 0xa5, 0x20, 0x1,
	0xda, 0x24, 0x5e, 0x50, 0x48, 0xfa, 0x83, 0x81, 0xe5, 0xee, 0xdb, 0xd5, 0x39, 0xfc, 0xe5, 0x6f, 0xef, 0xdf, 0x3e, 0x8f,
	0xb3, 0xdb, 0xed, 0xb2, 0x50, 0xa8, 0x5, 0x25, 0x4a, 0xa7, 0x48, 0x12, 0x61, 0x19, 0xa, 0x25, 0xc6, 0xc7, 0x1f, 0x21,
	0x12, 0xa5, 0xe2, 0xc4, 0xdf, 0x70, 0x24, 0x5b, 0x8c, 0x5c, 0x2b, 0x1a, 0x6b, 0xc1, 0xa6, 0x42, 0x9d, 0x67, 0xa3, 0x28,
	0x66, 0x2d, 0xb9, 0x3, 0xb5, 0x55, 0x27, 0x6a, 0x2d, 0x4e, 0xfe, 0xba, 0x38, 0x79, 0xf, 0x3f, 0x93, 0xdf, 0x4b, 0xda,
	0xcb, 0x2a, 0x72, 0x86, 0xd, 0xda, 0xc1, 0x49, 0x30, 0xe4, 0x22, 0x50, 0x41, 0x81, 0x9c, 0xa2, 0xde, 0xb0, 0x2f, 0x21,
	0x87, 0x8b, 0x9e, 0xe4, 0x32, 0xe6, 0x7f, 0x47, 0xf9, 0xef, 0xa6, 0xa7, 0x7b, 0xa2, 0x8d, 0x34, 0x21, 0xc1, 0xfc, 0x92,
	0x74, 0x6, 0xf5, 0x4c, 0xb4, 0xd0, 0xe5, 0x83, 0x61, 0x93, 0xfa, 0x9f, 0x36, 0xee, 0xc7, 0xd7, 0x52, 0x4c, 0x8d, 0x9e,
	0xa0, 0x24, 0xcd, 0xfd, 0x7f, 0xb9, 0x81, 0x19, 0xd6, 0xcb, 0x64, 0x2a, 0x31, 0x93, 0xc9, 0x11, 0xcc, 0x2a, 0x9d, 0x8f,
	0xc6, 0x53, 0x61, 0x2b, 0x74, 0x80, 0x75, 0x4d, 0x18, 0xc0, 0x3b, 0xbb, 0xef, 0x31, 0x90, 0xd3, 0x24, 0xd8, 0xd7, 0x60,
	0x69, 0x4b, 0x56, 0xc6, 0x8a, 0xcf, 0xe6, 0xed, 0x83, 0x90, 0xbe, 0x3d, 0xcc, 0xc1, 0x64, 0x94, 0xcd, 0xc1, 0x70, 0xef,
	0xbf, 0x26, 0xe5, 0x2b, 0x59, 0x28, 0xe0, 0x68, 0x7, 0x68, 0x99, 0x82, 0x43, 0x36, 0xdb, 0x24, 0x43, 0x41, 0x3e, 0xfb,
	0x72, 0x7b, 0x5, 0x75, 0xf0, 0xba, 0x69, 0xb9, 0x1d, 0x4, 0xb3, 0xf0, 0xc3, 0xa2, 0x9d, 0xad, 0xbd, 0xde, 0x2f, 0x22,
	0x57, 0x1c, 0x67, 0x70, 0xd4, 0x2b, 0xe0, 0xcd, 0x3b, 0xf1, 0xef, 0x86, 0x73, 0x32, 0x90, 0xb3, 0x2d, 0x72, 0xe8, 0x6f,
	0xa7, 0x91, 0xd7, 0x8b, 0xee, 0xd7, 0x9d, 0x58, 0xc9, 0x30, 0x7a, 0x94, 0xce, 0x71, 0x74, 0xca, 0x0, 0xee, 0xf6, 0xe9,
	0xf5, 0x31, 0xa, 0xad, 0xdd, 0x4b, 0x89, 0xc2, 0xf4, 0xae, 0x15, 0xc3, 0xb6, 0xa, 0xc3, 0xb6, 0x82, 0xe, 0x5c, 0x4f,
	0x85, 0xfb, 0x21, 0x7b, 0x3b, 0xd1, 0xc5, 0xc9, 0x98, 0xf6, 0xd9, 0x33, 0x9a, 0xf4, 0x9e, 0x89, 0x56, 0x49, 0x60, 0x7f,
	0x5e, 0xc6, 0xc6, 0x95, 0x14, 0xc, 0xf, 0x11, 0xfb, 0x32, 0xda, 0x57, 0x5c, 0x78, 0xea, 0xb9, 0xa4, 0x90, 0x56, 0xe,
	0x3d, 0x31, 0x14, 0xc1, 0x57, 0xc9, 0xe8, 0x99, 0x5e, 0x1d, 0xec, 0x4a, 0xa3, 0xca, 0x1e, 0xa4, 0x2d, 0xac, 0x4b, 0xe1,
	0x79, 0x63, 0x13, 0xd7, 0x78, 0x5f, 0xd3, 0xef, 0xd, 0x82, 0x2d, 0xda, 0x26, 0x89, 0x3f, 0xd6, 0xa4, 0xda, 0x65, 0xd6,
	0x4d, 0x38, 0x4e, 0x1a, 0xbb, 0xc3, 0x3d, 0x60, 0xcb, 0x62, 0x4, 0x4b, 0x58, 0xb4, 0x98, 0xce, 0x6b, 0x82, 0x26, 0x69,
	0x2a, 0xd, 0x4c, 0x62, 0x4c, 0xa, 0x7d, 0x9e, 0x45, 0xa4, 0xa, 0x1d, 0x1b, 0x15, 0xff, 0x90, 0xc5, 0x58, 0xb3, 0xee,
	0x96, 0x5c, 0xfb, 0x5d, 0xd8, 0xbd, 0x1e, 0xb2, 0x18, 0x53, 0x1c, 0x63, 0x16, 0x5e, 0x4, 0x26, 0xa7, 0xc8, 0xe8, 0x34,
	0x6, 0xdd, 0x76, 0x29, 0x36, 0xeb, 0x21, 0x8f, 0x8, 0x47, 0x68, 0xed, 0x20, 0x8d, 0x40, 0xe0, 0x93, 0x64, 0xd1, 0x1e,
	0xe7, 0x30, 0x9b, 0x68, 0x78, 0x36, 0x87, 0x99, 0x29, 0x16, 0x5, 0x21, 0x37, 0x81, 0xe4, 0x34, 0x2c, 0x96, 0xd9, 0x7c,
	0x60, 0xb6, 0xe0, 0x36, 0x71, 0xd6, 0x3e, 0x1c, 0xb3, 0xc6, 0x19, 0x8e, 0xb3, 0x9, 0x25, 0x20, 0xbd, 0x66, 0x21, 0x95,
	0xd7, 0x38, 0x45, 0x11, 0x62, 0x53, 0xd7, 0x3e, 0x70, 0xd7, 0x3e, 0xd9, 0xb2, 0x46, 0x35, 0x16, 0xc3, 0xb4, 0x3, 0xeb,
	0x41, 0xba, 0xc6, 0x29, 0xdb, 0xe8, 0xbe, 0xbe, 0x17, 0xd3, 0x7e, 0xd9, 0xb8, 0x71, 0xe4, 0x80, 0x95, 0x77, 0x9b, 0x29,
	0xa5, 0x50, 0x6f, 0x29, 0xb0, 0x89, 0xa4, 0x9f, 0xad, 0x94, 0x39, 0x50, 0xb6, 0x11, 0xb1, 0x3b, 0x40, 0xe8, 0x7f, 0xce,
	0x7d, 0x2a, 0xc9, 0x5a, 0xff, 0xb9, 0x77, 0xaf, 0x28, 0x46, 0xdc, 0xa4, 0x25, 0xd1, 0xb1, 0x21, 0x21, 0x58, 0xb3, 0xe,
	0xf2, 0xea, 0x1e, 0x8d, 0xa, 0xee, 0xa4, 0x3b, 0x66, 0xa4, 0xd0, 0x4d, 0x92, 0x90, 0xf1, 0x1, 0x32, 0xa3, 0x2a, 0x49,
	0xcb, 0x4b, 0x86, 0x6e, 0xf, 0xc6, 0xc9, 0xac, 0x14, 0xb5, 0x4b, 0x69, 0x64, 0xd2, 0x44, 0x79, 0xe8, 0xf6, 0x7f, 0x58,
	0x87, 0xed, 0xcf, 0x18, 0xd1, 0xef, 0x58, 0xda, 0x7a, 0xdf, 0xb2, 0x35, 0x35, 0x7d, 0x1c, 0xc3, 0x7f, 0x6e, 0x56, 0x40,
	0x4e, 0x79, 0xdd, 0xaf, 0xfe, 0x7f, 0xdc, 0x7d, 0xbd, 0x1d, 0xbf, 0x48, 0xe4, 0xc9, 0xf2, 0x15, 0x4e, 0x8c, 0xd1, 0x7b,
	0x8c, 0xfe, 0x11, 0xc9, 0xba, 0xe7, 0xe1, 0xd7, 0xc1, 0xff, 0x2, 0x0, 0x0, 0xff, 0xff, 0x2f, 0x5a, 0x79, 0xe6, 0x8b,
	0xb, 0x0, 0x0}
