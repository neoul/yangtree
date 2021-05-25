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

// UpdateGNMIPath returns the updated gNMI Path.
func UpdateGNMIPath(target, src *gnmipb.Path) *gnmipb.Path {
	if target == nil {
		return CloneGNMIPath(src)
	}
	if src == nil {
		return target
	}
	l := len(src.GetElem())
	pathElems := []*gnmipb.PathElem{}
	if l > 0 {
		pathElems = make([]*gnmipb.PathElem, l)
		copy(pathElems, src.GetElem())
	}
	target.Elem = pathElems
	return target
}

// NewGNMIAliasPath returns Alias gNMI Path.
func NewGNMIAliasPath(name, target, origin string) *gnmipb.Path {
	return &gnmipb.Path{
		Target: target,
		Origin: origin,
		Elem: []*gnmipb.PathElem{
			&gnmipb.PathElem{
				Name: name,
			},
		},
	}
}

// ValidateGNMIPath checks the validation of the gnmi path and merge all the input gnmi paths.
func ValidateGNMIPath(schema *yang.Entry, gpath ...*gnmipb.Path) error {
	if len(gpath) == 0 {
		return fmt.Errorf("no path")
	}
	p := proto.Clone(gpath[0]).(*gnmipb.Path)
	for i := 1; i < len(gpath); i++ {
		if gpath[i].GetElem() == nil && gpath[i].GetElement() != nil {
			return fmt.Errorf("deprecated path.element used")
		}
		if gpath[i].Target != "" { // 2.2.2.1 Path Target
			return fmt.Errorf("path.target MUST only ever be present on the prefix path")
		}
		p.Elem = append(p.Elem, gpath[i].Elem...)
	}
	origin := p.GetOrigin()
	module := yangtree.GetAllModules(schema)
	if origin != "" {
		if _, ok := module[origin]; !ok {
			return fmt.Errorf("schema %q not found", origin)
		}
	}
	for i := range p.Elem {
		schema = yangtree.GetSchema(schema, p.Elem[i].Name)
		if schema == nil {
			return fmt.Errorf("schema %q not found", p.Elem[i].Name)
		}
	}
	return nil
}

// MergeGNMIPath checks the validation of the gnmi path and merge all the input gnmi paths.
func MergeGNMIPath(gpath ...*gnmipb.Path) *gnmipb.Path {
	if len(gpath) == 0 {
		return &gnmipb.Path{}
	}
	p := proto.Clone(gpath[0]).(*gnmipb.Path)
	for i := 1; i < len(gpath); i++ {
		p.Elem = append(p.Elem, gpath[i].Elem...)
	}
	return p
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
		case yangtree.NodeSelectSelf:
		case yangtree.NodeSelectFromRoot:
		case yangtree.NodeSelectChild:
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
	entry := schema.Dir[e.Name]
	if entry == nil {
		return nil
	}
	if e.Key != nil {
		for kname := range e.Key {
			if !strings.Contains(entry.Key, kname) {
				return nil
			}
		}
		knames := strings.Split(entry.Key, " ")
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
	return findPaths(entry, strings.Join([]string{prefix, name}, "/"), elems[1:])
}
