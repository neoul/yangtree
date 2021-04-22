package yangtree

import "fmt"

type PathSelect int
type PathPredicates int

const (
	PathSelectChild       PathSelect = iota // path will select children by name
	PathSelectSelf                          // if the path starts with `.`
	PathSelectFromRoot                      // if the path starts with `/`
	PathSelectAllMatched                    // if the path starts with `//`
	PathSelectParent                        // if the path starts with `..`
	PathSelectAllChildren                   // Wildcard '*'

	PathPredicateNone    PathPredicates = iota
	PathPredicateNumeric                // p[1] (p[position()=1]), p[last()] (p[position()=last()])
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
)

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
					begin = end + 1
					node = append(node, updatePathSelect(pathnode))
					pathnode = &PathNode{}
				}
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
					pathnode.Predicates = append(pathnode.Predicates, (*path)[begin:end])
					begin = end + 1
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

	if (*path)[end-1] == '/' {
		pathnode.Select = PathSelectAllMatched
	} else {
		if begin < end {
			pathnode.Name = (*path)[begin:end]
		}
	}
	node = append(node, updatePathSelect(pathnode))
	return node, nil
}
