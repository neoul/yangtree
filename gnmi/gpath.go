package gnmi

import (
	"fmt"
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
		return fmt.Errorf("deprecated path.element used")
	}
	// if gpath.Target != "" { // 2.2.2.1 Path Target
	// 	return fmt.Errorf("path.target MUST only ever be present on the prefix path")
	// }

	for i := range gpath.Elem {
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
	return nil
}

// MergeGNMIPath merges input gnmi paths.
func MergeGNMIPath(gpath ...*gnmipb.Path) *gnmipb.Path {
	if len(gpath) == 0 {
		return &gnmipb.Path{}
	}
	p := &gnmipb.Path{}
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
	}
	return p
}

// UpdateGNMIPath updates the target and origin field of the dest path using the src path.
func UpdateGNMIPath(dest, src *gnmipb.Path) {
	if dest == nil || src != nil {
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
	gpath := &gnmipb.Path{}
	if len(pathnode) == 0 {
		return gpath, nil
	}
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
func ToPath(gpath ...*gnmipb.Path) string {
	pe := []string{""}
	for _, p := range gpath {
		if p == nil {
			continue
		}
		for _, e := range p.GetElem() {
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
	}
	return strings.Join(pe, "/")
}

// PathElemToXPATH eturns XPath string converted from gNMI Path
func GNMIPathElemToXPATH(elem []*gnmipb.PathElem, schemaPath bool) string {
	if elem == nil {
		return ""
	}
	if schemaPath {
		pe := []string{""}
		for _, e := range elem {
			pe = append(pe, e.GetName())
		}
		return strings.Join(pe, "/")
	}

	pe := []string{""}
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

// FindPaths - validates all schema of the gNMI Path.
func FindPaths(schema *yang.Entry, gpath *gnmipb.Path) []string {
	if schema == nil {
		return nil
	}
	elems := gpath.GetElem()
	if len(elems) == 0 {
		return []string{"/"}
	}
	return findPaths(schema, "", elems)
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
		knames := strings.Split(schema.Key, " ")
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
