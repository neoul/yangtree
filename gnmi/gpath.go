package gnmi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/neoul/yangtree"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
)

var (
	// RootGNMIPath - '/'
	RootGNMIPath = &gnmipb.Path{}
	// EmptyGNMIPath - ./
	EmptyGNMIPath = RootGNMIPath

	// WildcardGNMIPathDot3 - Wildcard Path '...'
	WildcardGNMIPathDot3 = &gnmipb.Path{
		Elem: []*gnmipb.PathElem{
			&gnmipb.PathElem{
				Name: "...",
			},
		},
	}
	// WildcardGNMIPathAsterisk - Wildcard Path '*'
	WildcardGNMIPathAsterisk = &gnmipb.Path{
		Elem: []*gnmipb.PathElem{
			&gnmipb.PathElem{
				Name: "*",
			},
		},
	}
)

// CloneGNMIPath returns cloned gNMI Path.
func CloneGNMIPath(src *gnmipb.Path) *gnmipb.Path {
	return proto.Clone(src).(*gnmipb.Path)
}

func FindSchema(schema *yang.Entry, gpath *gnmipb.Path) *yang.Entry {
	if gpath == nil {
		return schema
	}
	for i := range gpath.Elem {
		schema = yangtree.GetSchema(schema, gpath.Elem[i].Name)
		if schema == nil {
			return nil
		}
	}
	return schema
}

// ValidateGNMIPath checks the validation of the gnmi path.
func ValidateGNMIPath(schema *yang.Entry, gpath *gnmipb.Path) error {
	if gpath == nil {
		return nil
	}
	if gpath.GetOrigin() != "" {
		module := yangtree.GetAllModules(schema)
		if _, ok := module[gpath.GetOrigin()]; !ok {
			return fmt.Errorf("schema %q not found", gpath.GetOrigin())
		}
	}
	if gpath.GetElem() == nil && gpath.GetElement() != nil {
		return fmt.Errorf("deprecated path element used")
	}
	// if gpath.Target != "" { // 2.2.2.1 Path Target
	// 	return fmt.Errorf("path.target MUST only ever be present on the prefix path")
	// }

	for i := range gpath.Elem {
		switch gpath.Elem[i].Name {
		case "*", "...":
			if isvalid := isValidGNMIPath(schema, gpath.Elem[i:]); !isvalid {
				return fmt.Errorf("schema not found for %v", gpath)
			}
			return nil
		default:
			schema = yangtree.GetSchema(schema, gpath.Elem[i].Name)
			if schema == nil {
				return fmt.Errorf("schema %q not found", gpath.Elem[i].Name)
			}
			if len(gpath.Elem[i].Key) > 0 {
				for k := range gpath.Elem[i].Key {
					if j := strings.Index(schema.Key, k); j < 0 {
						return fmt.Errorf("%q is not a key for schema %q", k, schema.Name)
					}
				}
			}
		}
	}
	return nil
}

// MergeGNMIPath merges input gnmi paths.
func MergeGNMIPath(gpath ...*gnmipb.Path) *gnmipb.Path {
	if len(gpath) == 0 {
		return &gnmipb.Path{}
	}
	num := 0
	for i := range gpath {
		if gpath[i] == nil {
			continue
		}
		num += len(gpath[i].Elem)
	}
	first := true
	p := &gnmipb.Path{
		Elem: make([]*gnmipb.PathElem, 0, num),
	}
	for i := range gpath {
		if gpath[i] == nil {
			continue
		}
		for j := range gpath[i].Elem {
			if gpath[i].Elem[j] != nil {
				elem := &gnmipb.PathElem{Name: gpath[i].Elem[j].Name}
				if len(gpath[i].Elem[j].Key) > 0 {
					elem.Key = make(map[string]string)
					for k, v := range gpath[i].Elem[j].Key {
						elem.Key[k] = v
					}
				}
				p.Elem = append(p.Elem, elem)
			}
		}
		if first {
			if gpath[i].Target != "" {
				p.Target = gpath[i].Target
			}
			if gpath[i].Origin != "" {
				p.Origin = gpath[i].Origin
			}
		}
		first = false
	}
	return p
}

func ToFullGNMIPath(gprefix *gnmipb.Path, gpaths []*gnmipb.Path) ([]*gnmipb.Path, error) {
	if len(gpaths) == 0 {
		return nil, fmt.Errorf("no path requested")
	}
	gfullpath := make([]*gnmipb.Path, 0, len(gpaths))
	for i := range gpaths {
		if gpaths[i].GetElem() == nil && gpaths[i].GetElement() != nil {
			return nil, fmt.Errorf("deprecated path element is used for %v", gpaths[i])
		}
		gfullpath = append(gfullpath, MergeGNMIPath(gprefix, gpaths[i]))
	}
	return gfullpath, nil
}

// UpdateGNMIPath updates the target and origin fields of the dest path using the src path.
func UpdateGNMIPath(dest, src *gnmipb.Path) {
	if dest == nil || src == nil {
		return
	}
	dest.Target = src.Target
	dest.Origin = src.Origin
}

// ToGNMIPath returns the gnmi path for the path.
func ToGNMIPath(path string) (*gnmipb.Path, error) {
	pathnode, err := yangtree.ParsePath(&path)
	if err != nil {
		return nil, err
	}
	if len(pathnode) == 0 {
		return nil, nil
	}
	gpath := &gnmipb.Path{}
	for i := range pathnode {
		switch pathnode[i].Select {
		case yangtree.NodeSelectParent:
			gpath.Elem = append(gpath.Elem, &gnmipb.PathElem{Name: ".."})
		case yangtree.NodeSelectAllChildren:
			gpath.Elem = append(gpath.Elem, &gnmipb.PathElem{Name: "*"})
		case yangtree.NodeSelectAll:
			gpath.Elem = append(gpath.Elem, &gnmipb.PathElem{Name: "..."})
		case yangtree.NodeSelectChild, yangtree.NodeSelectFromRoot, yangtree.NodeSelectSelf:
			if pathnode[i].Name != "" {
				elem := &gnmipb.PathElem{Name: pathnode[i].Name}
				if len(pathnode[i].Predicates) > 0 {
					elem.Key = make(map[string]string)
				}
				for j := range pathnode[i].Predicates {
					token, _, err := yangtree.TokenizePathExpr(nil, &(pathnode[i].Predicates[j]), 0)
					if err != nil {
						return nil, err
					}
					tokenlen := len(token)
					if tokenlen != 3 || (tokenlen == 3 && token[1] != "=") {
						return nil, fmt.Errorf("wrong predicate %q of the path", pathnode[i].Predicates[j])
					}
					elem.Key[token[0]] = token[2]
				}
				gpath.Elem = append(gpath.Elem, elem)
			}
		}
	}
	return gpath, nil
}

// IsSchemaGNMIPath returns the path is schema path.
func IsSchemaGNMIPath(path *gnmipb.Path) bool {
	isSchemaPath := true
	if path != nil {
		for _, e := range path.Elem {
			if e.Key != nil && len(e.Key) > 0 {
				for _, v := range e.Key {
					if v != "*" {
						isSchemaPath = false
					}
				}
			}
		}
	}
	return isSchemaPath
}

// ToPath returns the path for the gnmi path.
func ToPath(abspath bool, gpath ...*gnmipb.Path) string {
	num := 0
	for _, p := range gpath {
		if p == nil {
			continue
		}
		num = num + len(p.Elem)
	}
	pe := make([]string, 0, num+1)
	if abspath {
		pe = append(pe, "")
	}
	for _, p := range gpath {
		if p == nil {
			continue
		}
		for _, e := range p.GetElem() {
			if len(e.Key) > 0 {
				kname := make([]string, 0, len(e.Key))
				for k := range e.Key {
					kname = append(kname, k)
				}
				sort.Slice(kname, func(i, j int) bool {
					return kname[i] > kname[j]
				})
				ke := make([]string, 0, len(kname)+1)
				ke = append(ke, e.Name)
				for i := range kname {
					ke = append(ke, "["+kname[i]+"="+e.Key[kname[i]]+"]")
				}
				pe = append(pe, strings.Join(ke, ""))
			} else {
				pe = append(pe, e.GetName())
			}
		}
	}
	return strings.Join(pe, "/")
}

// GNMIPathElemToPATH returns path string converted from gNMI Path
func GNMIPathElemToPATH(abspath, schemapath bool, elem []*gnmipb.PathElem) string {
	if elem == nil {
		return ""
	}
	var pe []string
	pe = make([]string, 0, len(elem)+1)
	if abspath {
		pe = append(pe, "")
	}
	if schemapath {
		for _, e := range elem {
			pe = append(pe, e.GetName())
		}
		return strings.Join(pe, "/")
	}
	for _, e := range elem {
		if e.GetKey() != nil {
			ke := []string{e.GetName()}
			for k, kv := range e.GetKey() {
				ke = append(ke, fmt.Sprintf("[%s=%s]", k, kv))
			}
			pe = append(pe, strings.Join(ke, ""))
		} else {
			pe = append(pe, e.GetName())
		}
	}
	return strings.Join(pe, "/")
}

// FindPaths finds all possible paths. It resolves the gNMI path wildcards.
func FindPaths(schema *yang.Entry, gpath *gnmipb.Path) []string {
	if schema == nil {
		return nil
	}
	if gpath == nil || len(gpath.GetElem()) == 0 {
		return []string{"/"}
	}
	return findPaths(schema, "", gpath.GetElem())
}

func findPaths(schema *yang.Entry, prefix string, elems []*gnmipb.PathElem) []string {
	if len(elems) == 0 {
		return []string{prefix}
	}
	if schema.Dir == nil || len(schema.Dir) == 0 {
		return nil
	}
	e := elems[0]
	if e.Name == "*" {
		founds := make([]string, 0, 8)
		for cname, centry := range schema.Dir {
			if centry.IsCase() || centry.IsChoice() {
				founds = append(founds,
					findPaths(centry, prefix, elems)...)
			} else {
				pp := strings.Join([]string{prefix, cname}, "/")
				founds = append(founds,
					findPaths(centry, pp, elems[1:])...)
			}
		}
		return founds
	} else if e.Name == "..." {
		founds := make([]string, 0, 16)
		for cname, centry := range schema.Dir {
			if centry.IsCase() || centry.IsChoice() {
				founds = append(founds,
					findPaths(centry, prefix, elems)...)
			} else {
				pp := strings.Join([]string{prefix, cname}, "/")
				founds = append(founds,
					findPaths(centry, pp, elems[1:])...)
				founds = append(founds,
					findPaths(centry, pp, elems[0:])...)
			}
		}
		return founds
	}
	name := e.Name
	if i := strings.Index(name, ":"); i >= 0 {
		name = name[i+1:]
	}
	schema = schema.Dir[name]
	if schema == nil {
		return nil
	}
	if e.Key != nil {
		for kname := range e.Key {
			if !strings.Contains(schema.Key, kname) {
				return nil
			}
		}
		knames := yangtree.GetKeynames(schema)
		for _, kname := range knames {
			if kval, ok := e.Key[kname]; ok {
				if kval == "*" {
					break
				}
				name = name + "[" + kname + "=" + kval + "]"
			} else {
				break
			}
		}
	}
	return findPaths(schema, strings.Join([]string{prefix, name}, "/"), elems[1:])
}

// ToValidDataPath checks and returns the valid data path (absolute path) for a gNMI path.
func ToValidDataPath(schema *yang.Entry, gpath *gnmipb.Path) (*yang.Entry, string, error) {
	if gpath == nil {
		return schema, "", nil
	}
	var path strings.Builder
	for i := range gpath.Elem {
		switch gpath.Elem[i].Name {
		case "*", "...":
			return schema, "", fmt.Errorf("wildcard %q is unable to use for the data path", gpath.Elem[i].Name)
		default:
			schema = yangtree.GetSchema(schema, gpath.Elem[i].Name)
			if schema == nil {
				return schema, "", fmt.Errorf("schema %q not found", gpath.Elem[i].Name)
			}
			path.WriteString("/" + gpath.Elem[i].Name)
			keyname := yangtree.GetKeynames(schema)
			if len(keyname) > 0 {
				if len(gpath.Elem[i].Key) > len(keyname) {
					return schema, "", fmt.Errorf("more keys inserted for path gpath.elem %q", gpath.Elem[i].Name)
				}
				if len(gpath.Elem[i].Key) < len(keyname) {
					return schema, "", fmt.Errorf("less keys inserted for path gpath.elem %q", gpath.Elem[i].Name)
				}
				for j := range keyname {
					if keyval, ok := gpath.Elem[i].Key[keyname[j]]; ok {
						path.WriteString("[" + keyname[j] + "=" + keyval + "]")
					} else {
						return schema, "", fmt.Errorf("key %q not inserted for path gpath.elem %q", keyname[j], gpath.Elem[i].Name)
					}
				}
			}
		}
	}
	return schema, path.String(), nil
}

// HasGNMIPathWildcards checks the validation of the gnmi path.
func HasGNMIPathWildcards(gpath *gnmipb.Path) bool {
	if gpath == nil {
		return false
	}
	for i := range gpath.Elem {
		switch gpath.Elem[i].Name {
		case "*", "...":
			return true
		}
	}
	return false
}

// ValidateAndConvertGNMIPath checks the validation of the gnmi path and returns string paths.
func ValidateAndConvertGNMIPath(schema *yang.Entry, gprefix *gnmipb.Path, gpath []*gnmipb.Path) (string, []string, error) {
	if len(gpath) == 0 {
		return "", nil, fmt.Errorf("no path inserted")
	}
	var err error
	var sprefix string
	if gprefix != nil {
		if gprefix.GetOrigin() != "" {
			module := yangtree.GetAllModules(schema)
			if _, ok := module[gprefix.GetOrigin()]; !ok {
				return "", nil, fmt.Errorf("invalid prefix: schema %q not found", gprefix.GetOrigin())
			}
		}
		if gprefix.GetElem() == nil && gprefix.GetElement() != nil {
			return "", nil, fmt.Errorf("deprecated path element used for the prefix")
		}
		schema, sprefix, err = ToValidDataPath(schema, gprefix)
		if err != nil {
			return "", nil, fmt.Errorf("invalid prefix: %v", err)
		}
	}

	spath := make([]string, 0, len(gpath))
	for i := range gpath {
		err := ValidateGNMIPath(schema, gpath[i])
		if err != nil {
			return "", nil, fmt.Errorf("invalid path: %v", err)
		}
		spath = append(spath, ToPath(gprefix == nil, gpath[i]))
	}
	return sprefix, spath, nil
}

func isValidGNMIPath(schema *yang.Entry, elem []*gnmipb.PathElem) bool {
	for i := range elem {
		switch elem[i].Name {
		case "*":
			chidren := yangtree.GetAllChildSchema(schema)
			for j := range chidren {
				ok := isValidGNMIPath(chidren[j], elem[i+1:])
				if ok {
					return true
				}
			}
		case "...":
			chidren := yangtree.GetAllChildSchema(schema)
			for j := range chidren {
				if ok := isValidGNMIPath(chidren[j], elem[i+1:]); ok {
					return true
				}
				if ok := isValidGNMIPath(chidren[j], elem[i:]); ok {
					return true
				}
			}
		default:
			schema = yangtree.GetSchema(schema, elem[i].Name)
			if schema == nil {
				return false
			}
			if len(elem[i].Key) > 0 {
				for k := range elem[i].Key {
					if j := strings.Index(schema.Key, k); j < 0 {
						return false
					}
				}
			}
		}
	}
	return schema != nil
}
