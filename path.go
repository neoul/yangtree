package yangtree

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

type PathSelect int
type PathPredicates int

const (
	PathSelectChild       PathSelect = iota // path will select children by name
	PathSelectSelf                          // if the path starts with `.`
	PathSelectFromRoot                      // if the path starts with `/`
	PathSelectAllMatched                    // if the path starts with `//` or `...`
	PathSelectParent                        // if the path starts with `..`
	PathSelectAllChildren                   // Wildcard '*'

	PathPredicateNone         PathPredicates = iota
	PathPredicatePosition                    // position()
	PathPredicatePositionLast                // position()
	PathPredicateNumeric                     // p[1] (p[position()=1]), p[last()] (p[position()=last()])
	PathPredicateCondition
	PathPredicateEl
)

// Predicate order is significant

type PathNode struct {
	Prefix     string // The namespace prefix of the path
	Name       string // the nodename of the path
	Value      string
	Select     PathSelect
	Predicates []string
}

var (
	pathNodeKeyword map[string]PathSelect = map[string]PathSelect{
		".":                          PathSelectSelf,
		"self::node()":               PathSelectAllChildren,
		"..":                         PathSelectParent,
		"parent::node()":             PathSelectParent,
		"*":                          PathSelectAllChildren,
		"...":                        PathSelectAllMatched,
		"descendant-or-self::node()": PathSelectAllMatched,
		"child::node()":              PathSelectChild,
	}
	// predicatesKeyword *gtrie.Trie
)

func init() {
	// predicatesKeyword = gtrie.New()
	// keywords := []struct {
	// 	keyword   string
	// 	predicate PathPredicates
	// }{
	// 	{keyword: "position()", predicate: PathPredicatePosition},
	// 	{keyword: "last()", predicate: PathPredicatePositionLast},
	// 	{keyword: "count(", predicate: PathPredicatePositionLast},
	// }
}

func updatePathSelect(pathnode *PathNode) *PathNode {
	if s, ok := pathNodeKeyword[pathnode.Name]; ok {
		pathnode.Select = s
	}
	return pathnode
}

// ParsePath parses the input xpath and return a single element with its attrs.
func ParsePath(path *string) ([]*PathNode, error) {
	node := make([]*PathNode, 0, 8)
	pathnode := &PathNode{}
	length := len(*path)
	begin := 0
	end := begin
	// insideBrackets is counted up when at least one '[' has been found.
	// It is counted down when a closing ']' has been found.
	insideBrackets := 0
	switch (*path)[end] {
	case '/':
		pathnode.Select = PathSelectFromRoot
		begin++
	case '=': // ignore data string in path
		pathnode.Value = (*path)[end+1:]
		return append(node, pathnode), nil
	case '[', ']':
		return nil, fmt.Errorf("yangtree: path '%s' starts with bracket", *path)
	}
	end++
	for end < length {
		switch (*path)[end] {
		case '/':
			if insideBrackets <= 0 {
				if (*path)[end-1] == '/' {
					pathnode.Select = PathSelectAllMatched
				} else {
					if begin < end {
						pathnode.Name = (*path)[begin:end]
					}
				}
				begin = end + 1
				node = append(node, updatePathSelect(pathnode))
				pathnode = &PathNode{}
			}
		case '[':
			if (*path)[end-1] != '\\' {
				if insideBrackets <= 0 {
					if begin < end {
						pathnode.Name = (*path)[begin:end]
					}
					begin = end + 1
				}
				insideBrackets++
			}
		case ']':
			if (*path)[end-1] != '\\' {
				insideBrackets--
				if insideBrackets <= 0 {
					// if end < 2 || (*path)[end-2:end+1] != "=*]" { // * wildcard inside predicates
					pathnode.Predicates = append(pathnode.Predicates, (*path)[begin:end])
					begin = end + 1
					// }
				}
			}
		case '=':
			if insideBrackets <= 0 {
				if begin < end {
					pathnode.Name = (*path)[begin:end]
					begin = end + 1
				}
				pathnode.Value = (*path)[begin:]
				return append(node, updatePathSelect(pathnode)), nil
			}
		case ':':
			if insideBrackets <= 0 {
				pathnode.Prefix = (*path)[begin:end]
				begin = end + 1
			}
		}
		end++
	}
	if insideBrackets > 0 {
		return nil, fmt.Errorf("yangtree: invalid path format '%s'", *path)
	}

	if begin < end {
		pathnode.Name = (*path)[begin:end]
	}
	node = append(node, updatePathSelect(pathnode))
	return node, nil
}

func PredicatesMap(predicates []string) (map[string]string, error) {
	p := map[string]string{}
	for j := range predicates {
		pathnode, err := ParsePath(&predicates[j])
		if err != nil {
			return nil, err
		}
		for _, n := range pathnode {
			p[n.Name] = n.Value
		}
	}
	return p, nil
}

func KeyGen(schema *yang.Entry, predicates []string) (string, error) {
	switch {
	case IsUniqueList(schema):
		keyname := strings.Split(schema.Key, " ")
		p := map[string]*PathNode{}
		for j := range predicates {
			pathnode, err := ParsePath(&predicates[j])
			if err != nil {
				return "", err
			}
			for _, n := range pathnode {
				p[n.Name] = n
			}
		}
		var key bytes.Buffer
		key.WriteString(schema.Name)
	Loop:
		for i := range keyname {
			pathnode, ok := p[keyname[i]]
			if !ok {
				break Loop
			}
			switch pathnode.Value {
			case "*":
				break Loop
			default:
				key.WriteString("[")
				key.WriteString(keyname[i])
				key.WriteString("=")
				key.WriteString(pathnode.Value)
				key.WriteString("]")
			}
		}
		return key.String(), nil
	default:
		return schema.Name, nil
	}
}
