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
		"count":   funcXPathCount,
		"current": "node.Value",
	}
)

func (pathnode *PathNode) PredicatesToMap() (map[string]interface{}, error) {
	pmap := make(map[string]interface{})
LOOP:
	for i := range pathnode.Predicates {
		token, _, err := TokenizePathExpr(nil, &(pathnode.Predicates[i]), 0)
		if err != nil {
			return nil, err
		}
		switch len(token) {
		case 0:
			continue LOOP
		case 1:
			if index, err := strconv.Atoi(pathnode.Predicates[0]); err == nil {
				if index <= 0 {
					return nil, fmt.Errorf("index path predicate %q must be > 0", pathnode.Predicates[0])
				}
				pmap["@index"] = index - 1
			} else {
				pmap["@evaluate-xpath"] = true
			}
			continue LOOP
		case 2, 3:
			if token[1] != "=" {
				pmap["@evaluate-xpath"] = true
			}
		default:
			pmap["@evaluate-xpath"] = true
		}

		var value string
		if len(token) > 2 {
			value = token[2]
			if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
				value = strings.Trim(value, "'")
			}
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
		}
		switch token[0] {
		case ".":
			pmap["."] = value
		default:
			var name string
			if j := strings.Index(token[0], ":"); j >= 0 {
				name = token[0][j+1:]
			} else {
				name = token[0]
			}
			if v, exist := pmap[name]; exist {
				if v != value {
					return nil, fmt.Errorf("duplicated path predicate %q found", name)
				}
			}
			pmap[name] = value
		}
	}
	return pmap, nil
}

// GenerateKey generates a key of the schema using the key value in the pmap.
// It also returns a boolean value to true if the prefix matching is required for the target node search.
func GenerateKey(schema *yang.Entry, pmap map[string]interface{}) (string, bool) {
	switch {
	case schema.IsList() && schema.Key != "":
		var key bytes.Buffer
		key.WriteString(schema.Name)
		keyname := GetKeynames(schema)
		for i := range keyname {
			v, ok := pmap[keyname[i]]
			if !ok {
				return key.String(), true
			}
			value := v.(string)
			switch value {
			case "*":
				return key.String(), true
			default:
				key.WriteString("[")
				key.WriteString(keyname[i])
				key.WriteString("=")
				key.WriteString(value)
				key.WriteString("]")
			}
		}
		return key.String(), false
	case schema.IsLeafList():
		v, ok := pmap["."]
		if !ok {
			return schema.Name, true
		}
		var key bytes.Buffer
		key.WriteString(schema.Name)
		key.WriteString("[.=")
		key.WriteString(v.(string))
		key.WriteString("]")
		return key.String(), false
	}
	return schema.Name, false // If it is a key for a container or a leaf node
}

func _find(branch *DataBranch, cschema *yang.Entry, key *string, prefixmatch bool, pmap map[string]interface{}, forSet bool) []DataNode {
	i := indexFirst(branch, key)
	if i >= len(branch.children) ||
		(i < len(branch.children) && cschema != branch.children[i].Schema()) {
		return nil
	}
	if index, ok := pmap["@index"]; ok {
		j := i + index.(int)
		if j >= len(branch.children) {
			return nil
		}
		return branch.children[i : i+1]
	}
	max := i
	if IsOrderedByUser(cschema) || IsDuplicatable(cschema) {
		var node []DataNode
		for ; max < len(branch.children); max++ {
			if branch.children[i].Schema() != branch.children[max].Schema() {
				break
			}
			if prefixmatch {
				if strings.HasPrefix(branch.children[max].Key(), *key) {
					node = append(node, branch.children[max])
				}
			} else if branch.children[max].Key() == *key {
				node = append(node, branch.children[max])
			}
		}
		return node
	}
	for ; max < len(branch.children); max++ {
		if branch.children[i].Schema() != branch.children[max].Schema() {
			break
		}
		if prefixmatch {
			if !strings.HasPrefix(branch.children[max].Key(), *key) {
				break
			}
		} else if branch.children[max].Key() != *key {
			break
		}
	}
	if forSet {
		if i < max {
			result := make([]DataNode, max-i)
			copy(result, branch.children[i:max])
			return result
		}
		return nil
	}
	return branch.children[i:max]
}

// Find() returns children related to the PathNode
func FindByPathNode(parent DataNode, pathnode *PathNode) ([]DataNode, error) {
	branch, ok := parent.(*DataBranch)
	if !ok {
		return nil, fmt.Errorf("unable to find a child from %q", parent)
	}
	cschema := GetSchema(branch.schema, pathnode.Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode.Name, branch.schema.Name)
	}
	pmap, err := pathnode.PredicatesToMap()
	if err != nil {
		return nil, err
	}

	key, prefixmatch := GenerateKey(cschema, pmap)
	if _, ok := pmap["@evaluate-xpath"]; ok {
		first, last := indexRangeBySchema(branch, &key)
		node, err := findByPredicates(branch.children[first:last], pathnode.Predicates)
		if err != nil {
			return nil, err
		}
		return node, nil
	}
	return _find(branch, cschema, &key, prefixmatch, pmap, true), nil
}

func UpdateByPathNode(parent DataNode, pathnode *PathNode, value string) ([]DataNode, error) {
	branch, ok := parent.(*DataBranch)
	if !ok {
		return nil, fmt.Errorf("unable to find children from %q", parent)
	}
	cschema := GetSchema(parent.Schema(), pathnode.Name)
	if cschema == nil {
		return nil, fmt.Errorf("schema %q not found from %q", pathnode.Name, branch.schema.Name)
	}
	pmap, err := pathnode.PredicatesToMap()
	if err != nil {
		return nil, err
	}
	key, prefixmatch := GenerateKey(cschema, pmap)
	if IsUpdatable(cschema) {
		children := _find(branch, cschema, &key, prefixmatch, pmap, true)
		if len(children) == 0 {
			child, err := New(cschema)
			if err != nil {
				return nil, err
			}
			err = UpdateByMap(child, pmap)
			if err != nil {
				return nil, err
			}
			err = branch.Insert(child)
			if err != nil {
				return nil, err
			}
			return []DataNode{child}, nil
		}
		for i := range children {
			err = UpdateByMap(children[i], pmap)
			if err != nil {
				return nil, err
			}
			if err := children[i].Set(value); err != nil {
				return nil, err
			}
			err = branch.Insert(children[i])
			if err != nil {
				return nil, err
			}
		}
		return children, nil
	}
	return nil, fmt.Errorf("%q is not updatable node", key)
}

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
	if length == 0 {
		return nil, nil
	}
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
		return nil, fmt.Errorf("invalid path format %q", *path)
	}

	if begin < end {
		pathnode.Name = (*path)[begin:end]
	}
	node = append(node, updateNodeSelect(pathnode))
	return node, nil
}

func keyGen(schema *yang.Entry, pathnode *PathNode) (string, map[string]interface{}, error) {
	numP := 0
	pmap := make(map[string]interface{})
	meta := GetSchemaMeta(schema)
	for i := range pathnode.Predicates {
		token, _, err := TokenizePathExpr(nil, &(pathnode.Predicates[i]), 0)
		if err != nil {
			return "", nil, err
		}
		switch len(token) {
		case 0:
			continue
		case 1:
			if index, err := strconv.Atoi(pathnode.Predicates[0]); err == nil {
				if index <= 0 {
					return "", nil, fmt.Errorf("index path predicate %q must be > 0", pathnode.Predicates[0])
				}
				pmap["@index"] = index - 1
				return pathnode.Name, pmap, nil
			}
			pmap["@evaluate-xpath"] = true
			return pathnode.Name, pmap, nil
		case 2, 3:
			if token[1] != "=" {
				pmap["@evaluate-xpath"] = true
				return pathnode.Name, pmap, nil
			}
		default:
			pmap["@evaluate-xpath"] = true
			return pathnode.Name, pmap, nil
		}
		var value string
		if len(token) > 2 {
			value = token[2]
		}
		if token[0] == "." {
			pmap["."] = value
			continue
		}

		cschema, ok := meta.Dir[token[0]]
		if !ok {
			pmap["@evaluate-xpath"] = true
			return pathnode.Name, pmap, nil
		}
		if !IsKeyNode(cschema) {
			pmap["@need-to-update"] = true
		}
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = strings.Trim(value, "'")
		}
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}
		if v, exist := pmap[cschema.Name]; exist {
			if v != value {
				return "", nil, fmt.Errorf("duplicated path predicate %q found", cschema.Name)
			}
		}
		numP++
		pmap[cschema.Name] = value
	}

	switch {
	case IsListHasKey(schema):
		var key bytes.Buffer
		key.WriteString(schema.Name)
		keyname := GetKeynames(schema)
		usedPredicates := 0
	LOOP:
		for i := range keyname {
			v, ok := pmap[keyname[i]]
			if !ok {
				pmap["@nokey"] = true
				pmap["@prefix"] = true
				break LOOP
			}
			usedPredicates++
			// delete(pmap, keyname[i])
			value := v.(string)
			// if value == "" || value == "*" {
			// 	pmap["@present"] = true
			// 	break LOOP
			// }
			switch value {
			case "*":
				pmap["@prefix"] = true
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
			pmap["@evaluate-xpath"] = true
		}
		return key.String(), pmap, nil
	}
	if numP > 0 {
		pmap["@evaluate-xpath"] = true
	}
	return pathnode.Name, pmap, nil
}

func TokenizePathExpr(token []string, s *string, pos int) ([]string, int, error) {
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
			e := w.String()
			if _, ok := opToGoExpr[e]; ok {
				token = append(token, w.String())
				w.Reset()
			}
		case ',':
			if w.Len() > 0 {
				token = append(token, w.String())
				w.Reset()
			}
		case '=':
			if len(token) > 0 {
				prev := token[len(token)-1]
				if prev == "=" || prev == ">=" || prev == "<=" || prev == "!=" {
					w.WriteByte((*s)[pos])
					continue
				}
			}
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
			token, pos, err = TokenizePathExpr(token, s, pos+1)
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
				if len(token) > 0 {
					prev := token[len(token)-1]
					if prev == "=" || prev == ">=" || prev == "<=" || prev == "!=" {
						w.WriteString((*s)[pos : pos+2])
						pos++
						continue
					}
				}
				if w.Len() > 0 {
					token = append(token, w.String())
					w.Reset()
				}
				token = append(token, (*s)[pos:pos+2])
				pos++
			default:
				return nil, 0, fmt.Errorf("invalid syntax %q", (*s))
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
				return i, fmt.Errorf("not terminated path expr: %q", strings.Join(token, ""))
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
						if fs, ok := f.(string); ok {
							goExpr.WriteString(fs)
							break
						}
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
	var e strings.Builder
	env := map[string]interface{}{
		"result":    funcXPathResult,
		"findvalue": funcXPathFindValue,
		"position":  func() int { return pos + 1 },
		"first":     func() int { return first + 1 },
		"last":      func() int { return last },
	}

	for i := range predicates {
		token, _, err := TokenizePathExpr(nil, &(predicates[i]), 0)
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
			env["node"] = current[pos]
			ok, err := gval.Evaluate(e.String(), env)
			if err != nil {
				return nil, fmt.Errorf("%q expr running error: %v", e.String(), err)
			}
			if ok.(bool) {
				newchildren = append(newchildren, current[pos])
			}
		}
		current = newchildren
		e.Reset()
	}
	return current, nil
}

func evaluatePathExpr(node DataNode, exprstr string) (bool, error) {
	token, _, err := TokenizePathExpr(nil, &exprstr, 0)
	if err != nil {
		return false, err
	}
	var e strings.Builder
	env := map[string]interface{}{
		"result":    funcXPathResult,
		"findvalue": funcXPathFindValue,
		"node":      node,
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

func RemovePredicates(path *string) (string, bool) {
	pathnode, err := ParsePath(path)
	if err != nil {
		return "", false
	}
	var removed bool
	var b strings.Builder
	for i := range pathnode {
		switch pathnode[i].Select {
		case NodeSelectFromRoot:
			b.WriteString("/")
		}
		if pathnode[i].Prefix != "" {
			b.WriteString(pathnode[i].Prefix)
		}
		b.WriteString(pathnode[i].Name)
		if len(pathnode[i].Predicates) > 0 {
			removed = true
		}
	}
	if removed {
		return b.String(), true
	}
	return "", false
}

// KeyGen() generates the child key and child map from the PathNode.
func KeyGen(pschema *yang.Entry, pathnode *PathNode) (string, map[string]interface{}, error) {
	cschema := GetSchema(pschema, pathnode.Name)
	if cschema == nil {
		return "", nil, fmt.Errorf("schema %q not found from %q", pathnode.Name, pschema.Name)
	}
	return keyGen(cschema, pathnode)
}

// FindAllPossiblePath finds all possible paths. It resolves the gNMI path wildcards.
func FindAllPossiblePath(schema *yang.Entry, spath string) []string {
	if schema == nil {
		return nil
	}
	pathnode, err := ParsePath(&spath)
	if err != nil {
		return nil
	}
	prefix := make([]string, 0, 12)
	if IsRootSchema(schema) {
		prefix = append(prefix, "")
	}
	return findAllPossiblePath(schema, prefix, pathnode)
}

func findAllPossiblePath(schema *yang.Entry, prefix []string, pathnode []*PathNode) []string {
	if len(pathnode) == 0 {
		return []string{strings.Join(prefix, "/")}
	}
	switch pathnode[0].Select {
	case NodeSelectSelf:
		return findAllPossiblePath(schema, prefix, pathnode[1:])
	case NodeSelectParent:
		if schema.Parent == nil {
			return nil
		}
		if IsRootSchema(schema.Parent) {
			return findAllPossiblePath(schema.Parent, append(prefix[:0], ""), pathnode[1:])
		} else if len(prefix) > 0 {
			return findAllPossiblePath(schema.Parent, prefix[:len(prefix)-1], pathnode[1:])
		}
		return findAllPossiblePath(schema.Parent, []string{".."}, pathnode[1:])
	case NodeSelectFromRoot:
		schema = GetRootSchema(schema)
	case NodeSelectAllChildren:
		cschema := GetAllChildSchema(schema)
		if len(cschema) == 0 {
			return nil
		}
		founds := make([]string, 0, len(cschema))
		for i := range cschema {
			founds = append(founds,
				findAllPossiblePath(cschema[i], append(prefix, cschema[i].Name), pathnode[1:])...)
		}
		return founds
	case NodeSelectAll:
		cschema := GetAllChildSchema(schema)
		if len(cschema) == 0 {
			return nil
		}
		founds := make([]string, 0, 16)
		for i := range cschema {
			cprefix := append(prefix, cschema[i].Name)
			founds = append(founds,
				findAllPossiblePath(cschema[i], cprefix, pathnode[1:])...)
			founds = append(founds,
				findAllPossiblePath(cschema[i], cprefix, pathnode[0:])...)
		}
		return founds
	}

	if pathnode[0].Name == "" {
		return []string{strings.Join(prefix, "/")}
	}
	schema = GetSchema(schema, pathnode[0].Name)
	if schema == nil {
		return nil
	}
	if len(pathnode[0].Predicates) > 0 {
		// predicates are not validated. It is just copied.
		var elem strings.Builder
		elem.WriteString(pathnode[0].Name)
		for i := range pathnode[0].Predicates {
			elem.WriteString("[")
			elem.WriteString(pathnode[0].Predicates[i])
			elem.WriteString("]")
		}
		return findAllPossiblePath(schema, append(prefix, elem.String()), pathnode[1:])
	}
	return findAllPossiblePath(schema, append(prefix, pathnode[0].Name), pathnode[1:])
}
