package yangtree

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

type SchemaGlobalOption struct {
}

// SchemaMetadata is used to keep the additional data for a schema entry.
type SchemaMetadata struct {
	Module      *yang.Module           // used to store the module of the schema entry
	Dir         map[string]*yang.Entry // used to store the children of the schema entry with all schema entry's aliases
	Enum        map[string]int64       // used to store all enumeration string
	Identityref map[string]string      // used to store all identity values of the schema entry
	KeyName     []string               // used to store key list
	QName       string                 // namespace-qualified name of RFC 7951
	Qboundary   bool                   // used to indicate the boundary of the namespace-qualified name of RFC 7951
	IsRoot      bool
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

func GeneratePath(entry *yang.Entry, prefixTagging bool) string {
	path := ""
	for e := entry; e != nil && e.Parent != nil; e = e.Parent {
		if e.IsCase() || e.IsChoice() {
			continue
		}
		elementName := e.Name
		if prefixTagging && e.Prefix != nil {
			elementName = e.Prefix.Name + ":" + elementName
		}
		if e.Key != "" {
			keylist := strings.Split(e.Key, " ")
			for _, k := range keylist {
				if prefixTagging && e.Prefix != nil {
					k = e.Prefix.Name + ":" + k
				}
				elementName = fmt.Sprintf("%s[%s=*]", elementName, k)
			}
		}
		path = fmt.Sprintf("/%s%s", elementName, path)
	}
	// if *pathPathType == "gnmi" {
	// 	gnmiPath, err := xpath.ToGNMIPath(path)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "path: %s could not be changed to gnmi: %v\n", path, err)
	// 	}
	// 	path = gnmiPath.String()
	// }
	// if *pathTypes {
	// 	path = fmt.Sprintf("%s (type=%s)", path, entry.Type.Name)
	// }
	return path
}

func getEnum(entry *yang.Entry) map[string]int64 {
	if entry == nil || entry.Annotation == nil {
		return nil
	}
	if data, ok := entry.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			return m.Enum
		}
	}
	return nil
}

func setEnum(entry *yang.Entry, enum map[string]int64) error {
	if entry == nil || entry.Annotation == nil {
		return fmt.Errorf("yangtree: nil entry or annotation for setting enum")
	}
	if data, ok := entry.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			m.Enum = enum
			return nil
		}
	}
	return fmt.Errorf("yangtree: no schema meta data for setting enum")
}

func getIdentityref(entry *yang.Entry) map[string]string {
	if entry == nil || entry.Annotation == nil {
		return nil
	}
	if data, ok := entry.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			return m.Identityref
		}
	}
	return nil
}

func setIdentityref(entry *yang.Entry, identityref map[string]string) error {
	if entry == nil || entry.Annotation == nil {
		return fmt.Errorf("yangtree: nil entry or annotation for setting identityref")
	}
	if data, ok := entry.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			m.Identityref = identityref
			return nil
		}
	}
	return fmt.Errorf("yangtree: no schema meta data for setting identityref")
}

func getModule(entry *yang.Entry) *yang.Module {
	if entry == nil || entry.Annotation == nil {
		return nil
	}
	if data, ok := entry.Annotation["meta"]; ok {
		if m, ok := data.(*SchemaMetadata); ok {
			return m.Module
		}
	}
	return nil
}

func updateSchemaMetaForType(entry *yang.Entry, typ *yang.YangType) error {
	if typ == nil {
		return nil
	}
	switch typ.Kind {
	case yang.Ybits:
		var enum map[string]int64
		if enum = getEnum(entry); enum == nil {
			enum = map[string]int64{}
		}
		newenum := typ.Bit.NameMap()
		for bs, bi := range newenum {
			enum[bs] = bi
		}
		if err := setEnum(entry, enum); err != nil {
			return err
		}
	case yang.Yenum:
		var enum map[string]int64
		if enum = getEnum(entry); enum == nil {
			enum = map[string]int64{}
		}
		newenum := typ.Enum.NameMap()
		for bs, bi := range newenum {
			enum[bs] = bi
		}
		if err := setEnum(entry, enum); err != nil {
			return err
		}
	case yang.Yidentityref:
		var identityref map[string]string
		if identityref = getIdentityref(entry); identityref == nil {
			identityref = map[string]string{}
		}
		for i := range typ.IdentityBase.Values {
			identityref[typ.IdentityBase.Values[i].NName()] = typ.IdentityBase.Values[i].PrefixedName()
			identityref[typ.IdentityBase.Values[i].PrefixedName()] = typ.IdentityBase.Values[i].NName()
		}
		if err := setIdentityref(entry, identityref); err != nil {
			return err
		}
	case yang.Yunion:
		for i := range typ.Type {
			if err := updateSchemaMetaForType(entry, typ.Type[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildRootEntry() *yang.Entry {
	rootEntry := &yang.Entry{
		Dir: map[string]*yang.Entry{},
		Annotation: map[string]interface{}{
			"meta": &SchemaMetadata{
				IsRoot: true,
				Dir:    map[string]*yang.Entry{},
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

func GetSchemaMeta(entry *yang.Entry) *SchemaMetadata {
	if m, ok := entry.Annotation["meta"]; ok {
		return m.(*SchemaMetadata)
	}
	return nil
}

func GetQName(entry *yang.Entry) (string, bool) {
	if m, ok := entry.Annotation["meta"]; ok {
		meta := m.(*SchemaMetadata)
		return meta.QName, meta.Qboundary
	}
	return "", false
}

func KeyGen(entry *yang.Entry, attr map[string]string) (string, bool, error) {
	for k := range attr {
		if entry.Dir[k] == nil {
			return "", false, fmt.Errorf("yangtree: unknown attribute %v", k)
		}
	}
	m, ok := entry.Annotation["meta"]
	if !ok {
		return "", false, fmt.Errorf("no metadata found")
	}
	if !entry.IsList() {
		return entry.Name, false, nil
	}
	key := entry.Name
	testAll := false
	meta := m.(*SchemaMetadata)
	if len(meta.KeyName) == 0 {
		testAll = true
	}
	for i := range meta.KeyName {
		if v, ok := attr[meta.KeyName[i]]; ok {
			key = key + "[" + meta.KeyName[i] + "=" + v + "]"
		} else {
			testAll = true
			break
		}
	}
	return key, testAll, nil
}

func updateSchemaEntry(parent, entry *yang.Entry, current *yang.Module, modules *yang.Modules) error {
	if entry.Annotation == nil {
		entry.Annotation = map[string]interface{}{}
	}
	meta := &SchemaMetadata{
		Enum: map[string]int64{},
	}
	entry.Annotation["meta"] = meta

	module, err := modules.FindModuleByPrefix(entry.Prefix.Name)
	if err != nil {
		return err
	}
	meta.Module = module

	// namespace-qualified name of RFC 7951
	nsname := fmt.Sprintf("%s:%s", module.Name, entry.Name)
	meta.QName = nsname
	if current != module {
		meta.Qboundary = true
	}

	// set keyname
	if entry.Key != "" {
		meta.KeyName = strings.Split(entry.Key, " ")
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
				return fmt.Errorf("yangtree: updating schema tree failed")
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
		if pmeta.Dir == nil {
			pmeta.Dir = map[string]*yang.Entry{}
		}
		pmeta.Dir[entry.Prefix.Name+":"+entry.Name] = entry
		pmeta.Dir[module.Name+":"+entry.Name] = entry
		pmeta.Dir[entry.Name] = entry
	}
	if err := updateSchemaMetaForType(entry, entry.Type); err != nil {
		return err
	}

	for _, child := range entry.Dir {
		if err := updateSchemaEntry(entry, child, module, modules); err != nil {
			return err
		}
	}
	return nil
}

func generateSchemaTree(d, f, e []string) (*yang.Entry, error) {
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
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "yangtree: yang loading error: %v\n", e)
		}
		return nil, fmt.Errorf("yangtree: yang loading failed with %d errors", len(errors))
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

	root := buildRootEntry()
	for _, mentry := range entries {
		skip := false
		for i := range e {
			if mentry.Name == e[i] {
				skip = true
			}
		}
		if !skip {
			for _, entry := range mentry.Dir {
				if same, ok := root.Dir[entry.Name]; ok {
					mo := getModule(same)
					return nil, fmt.Errorf(
						"yangtree: multiple top-level nodes are defined in %s and %s",
						mentry.Name, mo.Name)
				}
				entry.Parent = root
				root.Dir[entry.Name] = entry
				if err := updateSchemaEntry(root, entry, nil, ms); err != nil {
					return nil, err
				}
			}
		}
	}
	return root, nil
}

// Load loads all yang files and build the schema tree
func Load(_file, _dir, _excluded []string) (*yang.Entry, error) {
	_dir = sanitizeArrayFlagValue(_dir)
	_file = sanitizeArrayFlagValue(_file)
	_excluded = sanitizeArrayFlagValue(_excluded)

	var err error
	_dir, err = resolveGlobs(_dir)
	if err != nil {
		return nil, err
	}
	_file, err = resolveGlobs(_file)
	if err != nil {
		return nil, err
	}
	for _, dirpath := range _dir {
		expanded, err := yang.PathsWithModules(dirpath)
		if err != nil {
			return nil, err
		}

		// for _, fdir := range expanded {
		// 	fmt.Printf("adding %s to yang Paths\n", fdir)
		// }
		yang.AddPath(expanded...)
	}
	yfiles, err := findYangFiles(_file)
	if err != nil {
		return nil, err
	}
	_file = make([]string, 0, len(yfiles))
	_file = append(_file, yfiles...)
	// for _, file := range _file {
	// 	fmt.Printf("loading %s yang file\n", file)
	// }
	return generateSchemaTree(_dir, _file, _excluded)
}

func checkKey(entry *yang.Entry, key string) error {
	if !entry.IsList() {
		return fmt.Errorf("yangtree: schema '%s' is not list", entry.Name)
	}
	idx := strings.Index(key, "=")
	if idx < 0 {
		return fmt.Errorf("yangtree: invalid key format '%s' for '%s'", key, entry.Name)
	}
	name := key[:idx]
	if entry.Dir[name] == nil {
		return fmt.Errorf("yangtree: key '%s' not found from '%s'", name, entry.Name)
	}
	if !strings.Contains(entry.Key, name) {
		return fmt.Errorf("yangtree: '%s' is not a key for '%s'", name, entry.Name)
	}
	return nil
}

func GetSchema(entry *yang.Entry, name string) *yang.Entry {
	if entry == nil {
		return nil
	}
	var child *yang.Entry
	switch name {
	case "", ".":
		return entry
	case "..":
		child = entry.Parent
	default:
		// child = entry.Dir[name]
		if meta := GetSchemaMeta(entry); meta != nil {
			child = meta.Dir[name]
		}
	}
	return child
}

// GetPresentParentSchema is used to get the non-choice and non-case parent schema entry.
func GetPresentParentSchema(entry *yang.Entry) *yang.Entry {
	for p := entry.Parent; p != nil; p = p.Parent {
		if !p.IsCase() && !p.IsChoice() {
			return p
		}
	}
	return nil
}

// FindSchema finds *yang.Entry from the schema entry
func FindSchema(entry *yang.Entry, path string) (*yang.Entry, error) {
	if entry == nil {
		return nil, fmt.Errorf("yangtree: nil schema")
	}
	length := len(path)
	if length <= 0 {
		return entry, nil
	}
	begin := 0
	end := 0
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0

	switch path[end] {
	case '/':
		begin = 1
	case '[', ']', '=':
		return nil, fmt.Errorf("yangtree: path '%s' starts with bracket", path)
	}
	end++

	for end < length {
		// fmt.Printf("%c, '%s', %d\n", path[end], path[begin:end], insideBrackets)
		switch path[end] {
		case '/':
			if insideBrackets <= 0 {
				if entry = GetSchema(entry, path[begin:end]); entry == nil {
					return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
				}
				begin = end + 1
			}
			end++
		case '[':
			if path[end-1] != '\\' {
				if insideBrackets <= 0 {
					if entry = GetSchema(entry, path[begin:end]); entry == nil {
						return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
					}
					begin = end + 1
				}
				insideBrackets++
			}
			end++
		case ']':
			if path[end-1] != '\\' {
				insideBrackets--
				if insideBrackets <= 0 {
					if err := checkKey(entry, path[begin:end]); err != nil {
						return nil, err
					}
				}
				begin = end + 1
			}
			end++
		default:
			end++
		}
	}
	if entry = GetSchema(entry, path[begin:end]); entry == nil {
		return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
	}
	return entry, nil
}

func lookupSchema(entry *yang.Entry, name string) (centry *yang.Entry, reachToEnd bool) {
	switch {
	case entry.Dir == nil: // leaf, leaf-list
		centry = entry
		reachToEnd = true
	default: // container, case, etc.
		centry = GetSchema(entry, name)
	}
	return
}

// // SplitPath splits the path and check the validation of the schema in the path
// func SplitPath(entry *yang.Entry, path string) ([]string, error) {
// 	if entry == nil {
// 		return nil, fmt.Errorf("yangtree: nil schema")
// 	}
// 	if !entry.IsDir() {
// 		return nil, nil
// 	}
// 	length := len(path)
// 	if length <= 0 {
// 		return nil, nil
// 	}
// 	begin := 0
// 	end := 0
// 	// insideBrackets is counted up when at least one '[' has been found.
// 	// It is counted down when a closing ']' has been found.
// 	insideBrackets := 0

// 	switch path[end] {
// 	case '/':
// 		begin = 1
// 	case '[', ']', '=':
// 		return nil, fmt.Errorf("yangtree: path '%s' starts with bracket", path)
// 	}
// 	end++

// 	pathbegin := begin
// 	pathelem := make([]string, 0, 8)
// 	reachToEnd := false

// 	for end < length {
// 		// fmt.Printf("%c, '%s', %d\n", path[end], path[begin:end], insideBrackets)
// 		switch path[end] {
// 		case '/':
// 			if insideBrackets <= 0 {
// 				entry, reachToEnd = lookupSchema(entry, path[begin:end])
// 				if entry == nil {
// 					return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
// 				}
// 				begin = end + 1
// 				if pathbegin < end {
// 					pathelem = append(pathelem, path[pathbegin:end])
// 				}
// 				pathbegin = begin
// 			}
// 			end++
// 		case '[':
// 			if path[end-1] != '\\' {
// 				if insideBrackets <= 0 {
// 					entry, reachToEnd = lookupSchema(entry, path[begin:end])
// 					if entry == nil {
// 						return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
// 					}
// 					begin = end + 1
// 				}
// 				insideBrackets++
// 			}
// 			end++
// 		case ']':
// 			if path[end-1] != '\\' {
// 				insideBrackets--
// 				begin = end + 1
// 			}
// 			end++
// 		case '=':
// 			if insideBrackets <= 0 {
// 				entry, _ = lookupSchema(entry, path[begin:end])
// 				if entry == nil {
// 					return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
// 				}
// 				begin = end + 1
// 				reachToEnd = true
// 			} else {
// 				end++
// 			}
// 		default:
// 			end++
// 		}
// 		if reachToEnd {
// 			end = length
// 		}
// 	}
// 	entry, _ = lookupSchema(entry, path[begin:end])
// 	if entry == nil {
// 		return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
// 	}
// 	if pathbegin < end {
// 		pathelem = append(pathelem, path[pathbegin:end])
// 	}
// 	return pathelem, nil
// }

// ExtractKeyValues extracts the list key values from keystr
func ExtractKeyValues(keys []string, keystr string) ([]string, error) {
	length := len(keystr)
	if length <= 0 {
		return nil, fmt.Errorf("yangtree: extractkeys from empty keystr")
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
		return nil, fmt.Errorf("yangtree: extractkeys keystr '%s' starts with invalid char", keystr)
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
				return nil, fmt.Errorf("yangtree: invalid key format '%s'", keystr[begin:end])
			} else if insideBrackets == 1 {
				if begin < end {
					if keys[index] != keystr[begin:end] {
						return nil, fmt.Errorf("yangtree: invalid key '%s'", keystr[begin:end])
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
		return nil, fmt.Errorf("yangtree: invalid key '%s'", keystr)
	}
	return keyval, nil
}

func StringToValue(entry *yang.Entry, typ *yang.YangType, value string) (interface{}, error) {
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
				return length.String(), nil
			}
			return nil, fmt.Errorf("out-of-range %v", typ.Range)
		}
		return value, nil
	case yang.Ybool:
		v := strings.ToLower(value)
		if v == "true" {
			return true, nil
		} else if v == "false" {
			return false, nil
		} else {
			return false, nil
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
				return nil, fmt.Errorf("out-of-range %v", typ.Range)
			}
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
		emap := getEnum(entry)
		if _, ok := emap[value]; ok {
			return value, nil
		}
	case yang.Yidentityref:
		imap := getIdentityref(entry)
		if _, ok := imap[value]; ok {
			return value, nil
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
			return nil, fmt.Errorf("out-of-range %v", typ.Range)
		}
	case yang.Yunion:
		for i := range typ.Type {
			v, err := StringToValue(entry, typ.Type[i], value)
			if err == nil {
				return v, nil
			}
		}
	case yang.Ynone:
		break
	}
	return nil, fmt.Errorf("invalid type '%v' for '%s' (%v)", value, entry.Name, typ.Kind)
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

func ValueToJSONValue(entry *yang.Entry, typ *yang.YangType, value interface{}, rfc7951 bool) ([]byte, error) {
	switch typ.Kind {
	case yang.Yunion:
		for i := range typ.Type {
			v, err := ValueToJSONValue(entry, typ.Type[i], value, rfc7951)
			if err == nil {
				return v, nil
			}
		}
		return nil, fmt.Errorf("unexpected type found for value type '%s'", typ.Name)
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
				m := getIdentityref(entry)
				return json.Marshal(m[s])
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
	return "", fmt.Errorf("unexpected jsonval '%v (%T)'", jval, jval)
}
