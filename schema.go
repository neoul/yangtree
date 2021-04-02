package yangtree

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

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

// TypeInfoString returns a type information
func TypeInfoString(e *yang.Entry, pathWithPrefix bool) string {
	if e == nil || e.Type == nil {
		return "unknown type"
	}
	t := e.Type
	rstr := fmt.Sprintf("- type: %s", t.Kind)
	switch t.Kind {
	case yang.Ybits:
		data := GetAnnotation(e, "bits")
		if data != nil {
			rstr += fmt.Sprintf(" %v", data)
		}
	case yang.Yenum:
		data := GetAnnotation(e, "enum")
		if data != nil {
			rstr += fmt.Sprintf(" %v", data)
		}
	case yang.Yleafref:
		rstr += fmt.Sprintf(" %q", t.Path)
	case yang.Yidentityref:
		rstr += fmt.Sprintf(" %q", t.IdentityBase.Name)
		if pathWithPrefix {
			data := GetAnnotation(e, "prefix-qualified-identities")
			if data != nil {
				rstr += fmt.Sprintf(" %v", data)
			}
		} else {
			identities := make([]string, 0, 64)
			for i := range t.IdentityBase.Values {
				identities = append(identities, t.IdentityBase.Values[i].Name)
			}
			rstr += fmt.Sprintf(" %v", identities)
		}

	case yang.Yunion:
		unionlist := make([]string, 0, len(t.Type))
		for i := range t.Type {
			unionlist = append(unionlist, t.Type[i].Name)
		}
		rstr += fmt.Sprintf(" %v", unionlist)
	default:
	}
	rstr += "\n"

	if t.Root != nil {
		data := GetAnnotation(e, "root.type")
		if data != nil && t.Kind.String() != data.(string) {
			rstr += fmt.Sprintf("- root.type: %v\n", data)
		}
	}
	if t.Units != "" {
		rstr += fmt.Sprintf("- units: %s\n", t.Units)
	}
	if t.Default != "" {
		rstr += fmt.Sprintf("- default: %q\n", t.Default)
	}
	if t.FractionDigits != 0 {
		rstr += fmt.Sprintf("- fraction-digits: %d\n", t.FractionDigits)
	}
	if len(t.Length) > 0 {
		rstr += fmt.Sprintf("- length: %s\n", t.Length)
	}
	if t.Kind == yang.YinstanceIdentifier && !t.OptionalInstance {
		rstr += "- required\n"
	}

	if len(t.Pattern) > 0 {
		rstr += fmt.Sprintf("- pattern: %s\n", strings.Join(t.Pattern, "|"))
	}
	b := yang.BaseTypedefs[t.Kind.String()].YangType
	if len(t.Range) > 0 && !t.Range.Equal(b.Range) {
		rstr += fmt.Sprintf("- range: %s\n", t.Range)
	}
	return rstr
}

// GetAnnotation finds an annotation from the schema entry
func GetAnnotation(entry *yang.Entry, name string) interface{} {
	if entry.Annotation != nil {
		data, ok := entry.Annotation[name]
		if ok {
			return data
		}
	}
	return nil
}

// updateAnnotation updates the schema info before enconding.
func updateAnnotation(entry *yang.Entry) {
	for _, child := range entry.Dir {
		updateAnnotation(child)
		child.Annotation = map[string]interface{}{}
		t := child.Type
		if t == nil {
			continue
		}

		switch t.Kind {
		case yang.Ybits:
			nameMap := t.Bit.NameMap()
			bits := make([]string, 0, len(nameMap))
			for bitstr := range nameMap {
				bits = append(bits, bitstr)
			}
			child.Annotation["bits"] = bits
		case yang.Yenum:
			nameMap := t.Enum.NameMap()
			enum := make([]string, 0, len(nameMap))
			for enumstr := range nameMap {
				enum = append(enum, enumstr)
			}
			child.Annotation["enum"] = enum
		case yang.Yidentityref:
			identities := make([]string, 0, len(t.IdentityBase.Values))
			for i := range t.IdentityBase.Values {
				identities = append(identities, t.IdentityBase.Values[i].PrefixedName())
			}
			child.Annotation["prefix-qualified-identities"] = identities
		}
		if t.Root != nil {
			child.Annotation["root.type"] = t.Root.Name
		}
	}
}

func buildRootEntry() *yang.Entry {
	rootEntry := &yang.Entry{
		Dir:        map[string]*yang.Entry{},
		Annotation: map[string]interface{}{},
	}
	rootEntry.Name = "root"
	rootEntry.Annotation["schemapath"] = "/"
	rootEntry.Kind = yang.DirectoryEntry
	// Always annotate the root as a fake root, so that it is not treated
	// as a path element in ytypes.
	rootEntry.Annotation["root"] = true
	return rootEntry
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
			fmt.Fprintf(os.Stderr, "yang processing error: %v\n", e)
		}
		return nil, fmt.Errorf("yang processing failed with %d errors", len(errors))
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
		for _, entry := range mentry.Dir {
			skip := false
			for i := range e {
				if entry.Name == e[i] {
					skip = true
				}
			}
			if !skip {
				updateAnnotation(entry)
				root.Dir[entry.Name] = entry
				entry.Parent = root
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
	case '[', ']':
		return nil, fmt.Errorf("yangtree: path '%s' starts with bracket", path)
	}
	end++

	for end < length {
		// fmt.Printf("%c, '%s', %d\n", path[end], path[begin:end], insideBrackets)
		switch path[end] {
		case '/':
			if insideBrackets <= 0 {
				if begin < end {
					entry = entry.Dir[path[begin:end]]
					if entry == nil {
						return nil, fmt.Errorf("yangtree: '%s' schema not found", path[begin:end])
					}
					begin = end + 1
				} else {
					begin++
				}
			}
			end++
		case '[':
			if path[end-1] != '\\' {
				if insideBrackets <= 0 {
					if begin < end {
						entry = entry.Dir[path[begin:end]]
						if entry == nil {
							return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
						}
						begin = end + 1
					} else {
						begin++
					}
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
	if begin < end {
		entry = entry.Dir[path[begin:end]]
		if entry == nil {
			return nil, fmt.Errorf("yangtree: '%s' schema not found", path[begin:end])
		}
	}
	return entry, nil
}

// SplitPath splits the path and check the validation of its schema
func SplitPath(entry *yang.Entry, path string) ([]string, error) {
	if entry == nil {
		return nil, fmt.Errorf("yangtree: nil schema")
	}
	length := len(path)
	if length <= 0 {
		return nil, nil
	}
	begin := 0
	end := 0
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0

	switch path[end] {
	case '/':
		begin = 1
	case '[', ']':
		return nil, fmt.Errorf("yangtree: path '%s' starts with bracket", path)
	}
	end++

	pathbegin := begin
	pathelem := make([]string, 0, 8)

	for end < length {
		fmt.Printf("%c, '%s', %d\n", path[end], path[begin:end], insideBrackets)
		switch path[end] {
		case '/':
			if insideBrackets <= 0 {
				if begin < end {
					entry = entry.Dir[path[begin:end]]
					if entry == nil {
						return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
					}
					begin = end + 1
				} else {
					begin++
				}
				if pathbegin < end {
					pathelem = append(pathelem, path[pathbegin:end])
				}
				pathbegin = begin
			}
			end++
		case '[':
			if path[end-1] != '\\' {
				if insideBrackets <= 0 {
					if begin < end {
						entry = entry.Dir[path[begin:end]]
						if entry == nil {
							return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
						}
						begin = end + 1
					} else {
						begin++
					}
				}
				insideBrackets++
			}
			end++
		case ']':
			if path[end-1] != '\\' {
				insideBrackets--
				begin = end + 1
			}
			end++
		default:
			end++
		}
	}
	if begin < end {
		entry = entry.Dir[path[begin:end]]
		if entry == nil {
			return nil, fmt.Errorf("yangtree: schema '%s' not found", path[begin:end])
		}
	}
	if pathbegin < end {
		fmt.Println("last extracting", path[pathbegin:end])
		pathelem = append(pathelem, path[pathbegin:end])
	}
	return pathelem, nil
}
