package yangtree

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaesslerAG/gval"
	"github.com/openconfig/goyang/pkg/yang"
)

// XPath for yangtree
// Location path of yangtree:
//  - separted by /
//  - not support unabbreviated syntax.
//  - predicate order is significant
//  - not support variable reference

type NodeSelect int

const (
	NodeSelectChild       NodeSelect = iota // select the child nodes using the element name and predicate
	NodeSelectSelf                          // if the path starts with `.`
	NodeSelectFromRoot                      // if the path starts with `/`
	NodeSelectAll                           // if the path starts with `//` or `...`
	NodeSelectParent                        // if the path starts with `..`
	NodeSelectAllChildren                   // Wildcard '*'
)

type PathNode struct {
	Prefix     string // used to save the prefix of the data node
	Name       string // used to save the name of the data node
	Value      string // used to save the value of the data node
	Select     NodeSelect
	Predicates []string // used to filter the selected data node set.
}

var (
	pathNodeKeyword map[string]NodeSelect = map[string]NodeSelect{
		".":   NodeSelectSelf,
		"..":  NodeSelectParent,
		"*":   NodeSelectAllChildren,
		"...": NodeSelectAll, // descendant_or_self

		"descendant-or-self::node()": NodeSelectAll,
		"child::":                    NodeSelectChild,
		// ...
	}

	opToGoExpr map[string]string = map[string]string{
		"or":  "||",
		"and": "&&",
		"mod": "%",
		"div": "/",
		"=":   "==",
		">=":  ">=",
		"<=":  "<=",
		"!=":  "!=",
		"<":   "<",
		">":   ">",
		",":   ",",
	}

	funcXPath map[string]interface{} = map[string]interface{}{
		"count": funcXPathCount,

		// // boolean functions
		// "not":   "not",
		// "true":  "True",
		// "false": "False",
		// "sum":   "sum",

		// "string": "nodestring",
	}
)

func updateNodeSelect(pathnode *PathNode) *PathNode {
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
		pathnode.Select = NodeSelectFromRoot
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
					pathnode.Select = NodeSelectAll
				} else {
					if begin < end {
						pathnode.Name = (*path)[begin:end]
					}
				}
				begin = end + 1
				node = append(node, updateNodeSelect(pathnode))
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
				return append(node, updateNodeSelect(pathnode)), nil
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
	node = append(node, updateNodeSelect(pathnode))
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

func keyGen(schema *yang.Entry, pathnode *PathNode) (string, map[string]interface{}, error) {
	p := map[string]interface{}{}
	numP := 0
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
			if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
				value = strings.Trim(value, "'")
			}
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
			if name == "." {
				p["."] = value
				continue
			}
		}
		cschema, ok := meta.Dir[name]
		if !ok {
			return "", nil, fmt.Errorf("complex or invalid path predicate %q not supported", pathnode.Predicates[i])
		}
		numP++
		p[cschema.Name] = value
	}

	switch {
	case IsUniqueList(schema):
		var key bytes.Buffer
		key.WriteString(schema.Name)
		keyname := strings.Split(schema.Key, " ")
		usedPredicates := 0
	LOOP:
		for i := range keyname {
			v, ok := p[keyname[i]]
			if !ok {
				p["@prefix"] = true
				break LOOP
			}
			usedPredicates++
			// delete(p, keyname[i])
			value := v.(string)
			// if value == "" || value == "*" {
			// 	p["@present"] = true
			// 	break LOOP
			// }
			switch value {
			case "*":
				p["@prefix"] = true
				break LOOP
			default:
				key.WriteString("[")
				key.WriteString(keyname[i])
				key.WriteString("=")
				key.WriteString(value)
				key.WriteString("]")
			}
		}
		if usedPredicates < numP {
			p["@findbypredicates"] = true
		}
		return key.String(), p, nil
	}
	if numP > 0 {
		p["@findbypredicates"] = true
	}
	return pathnode.Name, p, nil
}

func tokenizeXPath(token []string, s *string, pos int) ([]string, int, error) {
	var err error
	length := len((*s))
	if token == nil {
		token = make([]string, 0, 6)
	}
	var w strings.Builder
	var isLiteral rune
	for ; pos < length; pos++ {
		if isLiteral != 0 {
			if isLiteral == rune((*s)[pos]) {
				w.WriteByte('"')
				token = append(token, w.String())
				w.Reset()
				isLiteral = 0
			} else {
				w.WriteByte((*s)[pos])
			}
			continue
		}
		switch (*s)[pos] {
		case '\'', '"': // xpath literal
			isLiteral = rune((*s)[pos])
			w.WriteByte('"')
		case '@':
			return nil, 0, fmt.Errorf("xml attr in %q not supported", *s)
		case ' ', '\t', '\n', '\r':
		case ',':
			if w.Len() > 0 {
				token = append(token, w.String())
				w.Reset()
			}
		case '=':
			if w.Len() > 0 {
				token = append(token, w.String())
				w.Reset()
			}
			token = append(token, "=")
		case '(':
			if w.Len() > 0 {
				token = append(token, w.String())
				w.Reset()
			}
			token = append(token, "(")
			token, pos, err = tokenizeXPath(token, s, pos+1)
			if err != nil {
				return nil, 0, err
			}
			if (*s)[pos] != ')' {
				return nil, 0, fmt.Errorf("parenthesis not terminated in %q", *s)
			}
		case ')':
			if w.Len() > 0 {
				token = append(token, w.String())
				w.Reset()
			}
			token = append(token, ")")
			return token, pos, nil
		case '!', '<', '>':
			if pos+1 == length {
				return nil, 0, fmt.Errorf("invalid syntex in %q", (*s))
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
				return nil, 0, fmt.Errorf("invalid predicate syntax %q", (*s))
			}
		default:
			w.WriteByte((*s)[pos])
		}
	}
	if isLiteral != 0 {
		return nil, 0, fmt.Errorf("missing quotation in %q", *s)
	}
	if w.Len() > 0 {
		token = append(token, w.String())
		w.Reset()
	}
	return token, pos, nil
}

// convertToGoExpr converts the xpath expression to runnable go expression
func convertToGoExpr(goExpr *strings.Builder, env map[string]interface{}, token []string, i int) (int, error) {
	var err error
	length := len(token)
	for ; i < length; i++ {
		switch token[i] {
		case "(":
			goExpr.WriteString("(")
			i, err = convertToGoExpr(goExpr, env, token, i+1)
			if err != nil {
				return i, err
			}
			if token[i] != ")" {
				return i, fmt.Errorf("not terminated path expr, %q", strings.Join(token, ""))
			}
			goExpr.WriteString(")")
		case ")":
			return i, nil
		default:
			if o := opToGoExpr[token[i]]; o != "" {
				goExpr.WriteString(o)
				break
			} else if i < length-1 {
				if token[i+1] == "(" {
					if f, ok := funcXPath[token[i]]; ok {
						env[token[i]] = f
					}
					goExpr.WriteString(token[i])
					break
				}
			}
			if strings.HasPrefix(token[i], "\"") &&
				strings.HasSuffix(token[i], "\"") {
				goExpr.WriteString(token[i])
			} else if _, err := strconv.ParseBool(token[i]); err == nil {
				goExpr.WriteString(token[i])
			} else if _, err := strconv.ParseFloat(token[i], 64); err == nil {
				goExpr.WriteString(token[i])
			} else if _, err := strconv.ParseInt(token[i], 10, 64); err == nil {
				goExpr.WriteString(token[i])
			} else if _, err := strconv.ParseUint(token[i], 10, 64); err == nil {
				goExpr.WriteString(token[i])
			} else {
				goExpr.WriteString("findvalue(node,")
				goExpr.WriteString("\"" + token[i] + "\"")
				goExpr.WriteString(")")
			}
		}
	}
	return i, nil
}

func funcXPathCount(n interface{}) int {
	if n == nil {
		return 0
	}
	switch cn := n.(type) {
	case []interface{}:
		return len(cn)
	case interface{}:
		return 1
	}
	return 0
}

func funcXPathFindValue(node DataNode, path string) interface{} {
	r, err := FindValue(node, path)
	if err != nil {
		return nil
	}
	switch len(r) {
	case 0:
		return nil
	case 1:
		return r[0]
	default:
		return r
	}
}

func funcXPathResult(value interface{}) bool {
	switch v := value.(type) {
	case string:
		if v != "" {
			return true
		}
		return true
	case bool:
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint32, uint64:
		if v == 0 {
			return false
		}
		return true
	case float32, float64:
		if v == 0.0 {
			return false
		}
		return true
	}
	return false
}

func findByPredicates(current []DataNode, predicates []string) ([]DataNode, error) {
	var first, last, pos int
	var node DataNode
	var e strings.Builder
	env := map[string]interface{}{
		"result":    funcXPathResult,
		"findvalue": funcXPathFindValue,
		"node":      node,
		"current":   func() interface{} { return node.Value() },
		"position":  func() int { return pos + 1 },
		"first":     func() int { return first + 1 },
		"last":      func() int { return last },
	}

	for i := range predicates {
		token, _, err := tokenizeXPath(nil, &(predicates[i]), 0)
		if err != nil {
			return nil, err
		}
		first, last = 0, len(current)
		if len(token) == 1 {
			if pos, err = strconv.Atoi(token[0]); err == nil {
				pos = pos - 1
				if pos >= last {
					return nil, nil
				}
				current = []DataNode{current[pos]}
				continue
			}
		}
		e.WriteString("result(")
		if _, err := convertToGoExpr(&e, env, token, 0); err != nil {
			return nil, err
		}
		e.WriteString(")")
		newchildren := make([]DataNode, 0, last)
		for pos = first; pos < last; pos++ {
			node = current[pos]
			ok, err := gval.Evaluate(e.String(), env)
			if err != nil {
				return nil, fmt.Errorf("%q expr running error: %v", e.String(), err)
			}
			if ok.(bool) {
				newchildren = append(newchildren, node)
			}
		}
		current = newchildren
		e.Reset()
	}
	return current, nil
}

func evaluatePathExpr(node DataNode, exprstr string) (bool, error) {
	token, _, err := tokenizeXPath(nil, &exprstr, 0)
	if err != nil {
		return false, err
	}
	var e strings.Builder
	env := map[string]interface{}{
		"result":    funcXPathResult,
		"findvalue": funcXPathFindValue,
		"node":      node,
		"current":   func() interface{} { return node.Value() },
	}
	_, err = convertToGoExpr(&e, env, token, 0)
	if err != nil {
		return false, nil
	}
	value, err := gval.Evaluate(e.String(), env)
	if err != nil {
		return false, nil
	}
	return value.(bool), nil
}
