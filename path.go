package yangtree

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/openconfig/goyang/pkg/yang"
)

type PathSelect int

const (
	PathSelectChild       PathSelect = iota // path will select children by name
	PathSelectSelf                          // if the path starts with `.`
	PathSelectFromRoot                      // if the path starts with `/`
	PathSelectAllMatched                    // if the path starts with `//` or `...`
	PathSelectParent                        // if the path starts with `..`
	PathSelectAllChildren                   // Wildcard '*'
)

// Predicate order is significant

type PathNode struct {
	Prefix     string // The namespace prefix of the path
	Name       string // the nodename of the path
	Value      string
	Select     PathSelect
	Predicates []string
}

type PathPredicateEnv struct {
	Expression string
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
	predicateFunc map[string]*PathPredicateEnv = map[string]*PathPredicateEnv{
		"position": &PathPredicateEnv{Expression: "position()"},
		"last":     &PathPredicateEnv{Expression: "last()"},
		"count":    &PathPredicateEnv{Expression: "count()"},
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
		return nil, fmt.Errorf("path '%s' starts with bracket", *path)
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
		return nil, fmt.Errorf("invalid path format '%s'", *path)
	}

	if begin < end {
		pathnode.Name = (*path)[begin:end]
	}
	node = append(node, updatePathSelect(pathnode))
	return node, nil
}

func PredicatesToValues(schema *yang.Entry, pathnode *PathNode) (map[string]string, error) {
	p := map[string]string{}
	// if len(pathnode.Predicates) == 1 {
	// 	if _, err := strconv.Atoi(pathnode.Predicates[0]); err == nil {
	// 		return nil, nil
	// 	}
	// }

	for j := range pathnode.Predicates {
		d := strings.IndexAny(pathnode.Predicates[j], "=")
		if d < 0 {
			return nil, fmt.Errorf("predicate %q is not value predicate", pathnode.Predicates[j])
		}
		name := pathnode.Predicates[j][:d]
		cschema := GetSchema(schema, name)
		if cschema == nil {
			return nil, fmt.Errorf("schema.%s not found from schema.%s", name, schema.Name)
		}
		p[cschema.Name] = pathnode.Predicates[j][d+1:]
	}
	return p, nil
}

func keyGen(schema *yang.Entry, pathnode *PathNode) (string, int) {
	numP := len(pathnode.Predicates)
	switch {
	case IsUniqueList(schema) && numP > 0:
		var key bytes.Buffer
		key.WriteString(schema.Name)
		keyname := strings.Split(schema.Key, " ")
		keylen := len(keyname)
		remainedP := numP
		if numP < keylen {
			keylen = numP
		}

		for i := 0; i < keylen; i++ {
			d := strings.IndexAny(pathnode.Predicates[i], "=")
			if d < 0 {
				return key.String(), remainedP
			}
			name := pathnode.Predicates[i][:d]
			if strings.HasSuffix(name, keyname[i]) {
				remainedP--
				if cschema := GetSchema(schema, name); cschema == nil {
					return key.String(), remainedP
				}
				rvalue := pathnode.Predicates[i][d+1:]
				switch rvalue {
				case "*":
					return key.String(), remainedP
				default:
					key.WriteString("[")
					key.WriteString(keyname[i])
					key.WriteString("=")
					key.WriteString(rvalue)
					key.WriteString("]")
				}
			} else {
				return key.String(), remainedP
			}
		}
		return key.String(), remainedP
	default:
		return schema.Name, numP
	}
}

func keyGenForSet(schema *yang.Entry, pathnode *PathNode) (string, map[string]interface{}, error) {
	p := map[string]interface{}{}
	if len(pathnode.Predicates) == 1 {
		if index, err := strconv.Atoi(pathnode.Predicates[0]); err == nil {
			if index <= 0 {
				return "", nil, fmt.Errorf("index path predicate %q must be > 0", pathnode.Predicates[0])
			}
			p["@index"] = index - 1
			return pathnode.Name, p, nil
		}
	}
	meta := GetSchemaMeta(schema)
	for i := range pathnode.Predicates {
		var name, value string
		d := strings.IndexAny(pathnode.Predicates[i], "=")
		if d < 0 {
			name = pathnode.Predicates[i]
			value = ""
		} else {
			name = pathnode.Predicates[i][:d]
			value = pathnode.Predicates[i][d+1:]
			if name == "." {
				p["."] = value
				continue
			}
		}
		cschema, ok := meta.Dir[name]
		if !ok {
			return "", nil, fmt.Errorf("complex or invalid path predicate %q not supported", pathnode.Predicates[i])
		}
		p[cschema.Name] = value
	}

	switch {
	case IsUniqueList(schema):
		var key bytes.Buffer
		key.WriteString(schema.Name)
		keyname := strings.Split(schema.Key, " ")
	LOOP:
		for i := range keyname {
			v, ok := p[keyname[i]]
			if !ok {
				p["@prefix"] = true
				break LOOP
			}
			// delete(p, keyname[i])
			value := v.(string)
			// if value == "" || value == "*" {
			// 	p["@present"] = true
			// 	break LOOP
			// }
			key.WriteString("[")
			key.WriteString(keyname[i])
			key.WriteString("=")
			key.WriteString(value)
			key.WriteString("]")
		}
		return key.String(), p, nil
	}
	return pathnode.Name, p, nil
}

func tokenizePredicate(token []string, s *string, pos int) ([]string, int, error) {
	var err error
	length := len((*s))
	if token == nil {
		token = make([]string, 0, 6)
	}
	var w bytes.Buffer
	isConst := false
	for ; pos < length; pos++ {
		if (*s)[pos] == '\'' {
			if isConst {
				token = append(token, w.String())
				w.Reset()
				isConst = false
			} else {
				isConst = true
			}
			continue
		}
		if isConst {
			w.WriteByte((*s)[pos])
			continue
		}
		switch (*s)[pos] {
		case '@':
			return nil, 0, fmt.Errorf("xml attribute '%s' not supported", *s)
		case ' ':
		case '=':
			if w.Len() > 0 {
				token = append(token, w.String())
				w.Reset()
			}
			token = append(token, (*s)[pos:pos+1])
		case '(':
			if w.Len() > 0 {
				token = append(token, w.String())
				w.Reset()
			}
			token = append(token, (*s)[pos:pos+1])
			token, pos, err = tokenizePredicate(token, s, pos+1)
			if err != nil {
				return nil, 0, err
			}
			if (*s)[pos] != ')' {
				return nil, 0, fmt.Errorf("not terminated parenthesis in '%s'", *s)
			}
		case ')':
			if w.Len() > 0 {
				token = append(token, w.String())
				w.Reset()
			}
			token = append(token, (*s)[pos:pos+1])
			return token, pos, nil
		case '!', '<', '>':
			if pos+1 == length {
				return nil, 0, fmt.Errorf("invalid predicate syntax '%s'", (*s))
			}
			switch (*s)[pos : pos+2] {
			case "<=", ">=", "!=":
				if w.Len() > 0 {
					token = append(token, w.String())
					w.Reset()
				}
				token = append(token, (*s)[pos:pos+2])
				pos++
			default:
				return nil, 0, fmt.Errorf("invalid predicate syntax '%s'", (*s))
			}
		default:
			w.WriteByte((*s)[pos])
		}
	}
	if isConst {
		return nil, 0, fmt.Errorf("missing quotation in %s", *s)
	}
	if w.Len() > 0 {
		token = append(token, w.String())
		w.Reset()
	}
	if len(token) == 0 {
		return nil, 0, fmt.Errorf("empty predicate'")
	}
	return token, pos, nil
}

func getExpression(expression *bytes.Buffer, token []string, pos int) (int, error) {
	var err error
	lvalue := true
	length := len(token)
	if length == 1 {
		if _, err := strconv.Atoi(token[0]); err == nil {
			expression.WriteString("position()==")
			expression.WriteString(token[0])
			return 0, nil
		}
	}
	for ; pos < length; pos++ {
		switch token[pos] {
		case "(":
			expression.WriteString("(")
			pos, err = getExpression(expression, token, pos+1)
			if err != nil {
				return pos, err
			}
			if token[pos] != ")" {
				return pos, fmt.Errorf("not terminated parenthesis in '%s'", strings.Join(token, ""))
			}
			expression.WriteString(")")
		case "or":
			expression.WriteString("||")
		case "and":
			expression.WriteString("&&")
		case ")":
			return pos, nil
		case "=":
			expression.WriteString("==")
		case ">=", "<=", "!=":
			expression.WriteString(token[pos])
		default:
			if _, ok := predicateFunc[token[pos]]; ok {
				if pos+1 < length {
					if token[pos+1] == "(" {
						expression.WriteString(token[pos])
						break
					}
				}
			}
			if lvalue {
				expression.WriteString(`value(node(), "`)
				expression.WriteString(token[pos])
				expression.WriteString(`")`)
				lvalue = false
			} else {
				expression.WriteString(`"`)
				expression.WriteString(token[pos])
				expression.WriteString(`"`)
				lvalue = true
			}
		}
	}
	return pos, nil
}

func predicateFuncValue(parent DataNode, name string) string {
	nodes := parent.Get(name)
	if len(nodes) > 0 {
		return nodes[0].ValueString()
	}
	return ""
}

func predicateFuncResult(value interface{}) bool {
	switch v := value.(type) {
	case string:
		if v != "" {
			return true
		} else {
			return false
		}
	case bool:
		return v
	case int:
		if v == 0 {
			return false
		} else {
			return true
		}
	}
	return false
}

func findByPredicates(branch *DataBranch, pathnode *PathNode, first, last int) ([]DataNode, error) {
	var pos int
	var node DataNode
	var expression bytes.Buffer
	children := branch.children[first:last]
	env := map[string]interface{}{
		"node":     func() DataNode { return node },
		"position": func() int { return pos + 1 },
		"first":    func() int { return first + 1 },
		"last":     func() int { return last },
		"count":    func() int { return last - first },
		"result":   predicateFuncResult,
		"value":    predicateFuncValue,
	}
	for i := range pathnode.Predicates {
		token, _, err := tokenizePredicate(nil, &(pathnode.Predicates[i]), 0)
		if err != nil {
			return nil, err
		}
		expression.WriteString("result(")
		if _, err := getExpression(&expression, token, 0); err != nil {
			return nil, err
		}
		expression.WriteString(")")
		program, err := expr.Compile(expression.String(), expr.Env(env))
		if err != nil {
			return nil, fmt.Errorf("predicate expression %q compile error  %v", expression.String(), err)
		}
		first, last = 0, len(children)
		newchildren := make([]DataNode, 0, last)
		for pos = 0; pos < last; pos++ {
			node = children[pos]
			ok, err := expr.Run(program, env)
			if err != nil {
				return nil, fmt.Errorf("predicate expression %q running error  %v", expression.String(), err)
			}
			if ok.(bool) {
				newchildren = append(newchildren, node)
			}
		}
		children = newchildren
		expression.Reset()
	}
	return children, nil
}
