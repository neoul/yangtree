package yangtree

import (
	"encoding/json"
	"fmt"
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
	CreatedWithDefault bool // DataNode (data node) is created with the default value of the schema if set.
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
	IsRoot      bool
	IsKey       bool
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
	// if *pathPathType == "gnmi" {
	// 	gnmiPath, err := xpath.ToGNMIPath(path)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "path: %s could not be changed to gnmi: %v\n", path, err)
	// 	}
	// 	path = gnmiPath.String()
	// }
	// if *pathTypes {
	// 	path = fmt.Sprintf("%s (type=%s)", path, schema.Type.Name)
	// }
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

func getModule(schema *yang.Entry) *yang.Module {
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

func buildRootEntry(mods map[string]*yang.Module, option SchemaOption) *yang.Entry {
	rootEntry := &yang.Entry{
		Dir: map[string]*yang.Entry{},
		Annotation: map[string]interface{}{
			"modules": mods,
			"meta": &SchemaMetadata{
				IsRoot: true,
				Dir:    map[string]*yang.Entry{},
				Option: &option,
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

func IsRoot(schema *yang.Entry) bool {
	if m, ok := schema.Annotation["meta"]; ok {
		meta := m.(*SchemaMetadata)
		return meta.IsRoot
	}
	return false
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
	nsname := strings.Join([]string{module.Name, ":", schema.Name}, "")
	meta.QName = nsname
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

type MultipleError []error

func (me MultipleError) Error() string {
	var errstr strings.Builder
	for i := range me {
		errstr.WriteString(me[i].Error() + "\n")
	}
	return errstr.String()
}

func generateSchemaTree(d, f, e []string, option SchemaOption) (*yang.Entry, error) {
	if len(f) == 0 {
		return nil, fmt.Errorf("no yang file")
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
	entries := make([]*yang.Entry, len(names))
	for x, n := range names {
		entries[x] = yang.ToEntry(mods[n])
	}
	root := buildRootEntry(mods, option)
	for _, mentry := range entries {
		skip := false
		for i := range e {
			if mentry.Name == e[i] {
				skip = true
			}
		}
		if !skip {
			for _, schema := range mentry.Dir {
				if same, ok := root.Dir[schema.Name]; ok {
					mo := getModule(same)
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
	var op SchemaOption
	for i := range option {
		if o, ok := option[i].(SchemaOption); ok {
			op = o
		}
	}
	return generateSchemaTree(dir, file, excluded, op)
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
	if schema == nil {
		return false
	}
	for ; schema.Parent != nil; schema = schema.Parent {
		switch schema.Config {
		case yang.TSTrue:
			return true
		case yang.TSFalse:
			return false
		}
	}

	// Reached the last element in the tree without explicit configuration
	// being set.
	return schema.Config != yang.TSFalse
}

// ExtractKeyValues extracts the list key values from keystr
func ExtractKeyValues(keys []string, keystr string) ([]string, error) {
	length := len(keystr)
	if length <= 0 {
		return nil, fmt.Errorf("empty key string found")
	}
	index := 0
	begin := 0
	end := 0
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0
	keyval := make([]string, len(keys))

	switch keystr[end] {
	case '/':
		begin = 1
	case '[':
		begin = 1
		insideBrackets++
	case ']', '=':
		return nil, fmt.Errorf("key string %q starts with invalid char: ] or =", keystr)
	}
	end++
	// fmt.Println(keys, keystr)

	for end < length {
		// fmt.Printf("%c, '%s', %d\n", keystr[end], keystr[begin:end], insideBrackets)
		switch keystr[end] {
		case '/':
			if insideBrackets <= 0 {
				begin = end + 1
			}
			end++
		case '[':
			if keystr[end-1] != '\\' {
				if insideBrackets <= 0 {
					begin = end + 1
				}
				insideBrackets++
			}
			end++
		case ']':
			if keystr[end-1] != '\\' {
				insideBrackets--
				if insideBrackets <= 0 {
					// fmt.Println(keystr[begin:end])
					keyval[index-1] = keystr[begin:end]
					begin = end + 1
				}
			}
			end++
		case '=':
			if insideBrackets <= 0 {
				return nil, fmt.Errorf("invalid key format %q", keystr[begin:end])
			} else if insideBrackets == 1 {
				if begin < end {
					if keys[index] != keystr[begin:end] {
						return nil, fmt.Errorf("invalid key %q", keystr[begin:end])
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
		return nil, fmt.Errorf("invalid key %q", keystr)
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
