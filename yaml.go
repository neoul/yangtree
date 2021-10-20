package yangtree

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"gopkg.in/yaml.v2"
)

// unmarshalYAMLListNode constructs list nodes of the parent node using YAML values.
func unmarshalYAMLListNode(parent DataNode, cschema *SchemaNode, kname []string, kval []interface{}, yamlHash interface{}) error {
	entry, ok := yamlHash.(map[interface{}]interface{})
	if !ok {
		if yamlHash == nil {
			return nil
		}
		return fmt.Errorf("unexpected yaml value %q (%T) for %q", yamlHash, yamlHash, cschema.Name)
	}

	if len(kname) != len(kval) {
		for k, v := range entry {
			kval = append(kval, k)
			err := unmarshalYAMLListNode(parent, cschema, kname, kval, v)
			if err != nil {
				return err
			}
			kval = kval[:len(kval)-1]
		}
		return nil
	}
	if cschema.IsDuplicatableList() {
		return fmt.Errorf("non-id list %q must have the array format", cschema.Name)
	}
	// check existent DataNode
	var err error
	var idBuilder strings.Builder
	idBuilder.WriteString(cschema.Name)
	for i := range kval {
		idBuilder.WriteString("[")
		idBuilder.WriteString(kname[i])
		idBuilder.WriteString("=")
		idBuilder.WriteString(ValueToString(kval[i]))
		idBuilder.WriteString("]")
	}
	id := idBuilder.String()
	var child, found DataNode
	if found = parent.Get(id); found == nil {
		child, err = NewDataNodeByID(cschema, id)
		if err != nil {
			return err
		}
		if _, err = parent.Insert(child, nil); err != nil {
			return err
		}
	} else {
		child = found
	}
	if err := unmarshalYAML(child, yamlHash); err != nil {
		if found == nil {
			parent.Delete(child)
		}
		return err
	}
	return nil
}

// unmarshalYAMLListableNode constructs listable child nodes of the parent data node using YAML values.
func unmarshalYAMLListableNode(parent DataNode, cschema *SchemaNode, kname []string, sequnce []interface{}) error {
	if cschema.IsLeafList() {
		for i := range sequnce {
			child, err := NewDataNode(cschema)
			if err != nil {
				return err
			}
			if err = unmarshalYAML(child, sequnce[i]); err != nil {
				return err
			}
			if _, err = parent.Insert(child, nil); err != nil {
				return err
			}
		}
		return nil
	}
	for i := range sequnce {
		entry, ok := sequnce[i].(map[interface{}]interface{})
		if !ok {
			return fmt.Errorf("unexpected yaml value %T for %s", sequnce[i], cschema.Name)
		}
		// check existent DataNode
		var err error
		var idBuilder strings.Builder
		idBuilder.WriteString(cschema.Name)
		for i := range kname {
			idBuilder.WriteString("[")
			idBuilder.WriteString(kname[i])
			idBuilder.WriteString("=")
			idBuilder.WriteString(fmt.Sprint(entry[kname[i]]))
			idBuilder.WriteString("]")
			// [FIXME] Does it need to check id validation?
			// kchild, err := NewDataNode(kschema, kval)
			// if err != nil {
			// 	return err
			// }
		}
		id := idBuilder.String()
		var child, found DataNode
		if cschema.IsDuplicatableList() {
			child = nil
		} else if found = parent.Get(id); found != nil {
			child = found
		}
		if child == nil {
			child, err = NewDataNodeByID(cschema, id)
			if err != nil {
				return err
			}
			if _, err = parent.Insert(child, nil); err != nil {
				return err
			}
		}
		if err := unmarshalYAML(child, entry); err != nil {
			if found == nil {
				parent.Delete(child)
			}
			return err
		}
	}
	return nil
}

// unmarshalYAML updates a data node using YAML values.
func unmarshalYAML(node DataNode, yval interface{}) error {
	if node.IsBranchNode() {
		switch entry := yval.(type) {
		case map[interface{}]interface{}:
			schema := node.Schema()
			for k, v := range entry {
				keystr := ValueToString(k)
				name, haskey, err := extractSchemaName(&keystr)
				if err != nil {
					return Error(EAppTagYAMLParsing, err)
				}
				cschema := schema.GetSchema(name)
				if cschema == nil {
					return Errorf(EAppTagYAMLParsing, "schema %q not found from %q", k, schema.Name)
				}
				if haskey {
					keyname := cschema.Keyname
					keyval, err := extractKeyValues(keyname, &keystr)
					if err != nil {
						return Error(EAppTagYAMLParsing, err)
					}
					keymap := map[interface{}]interface{}{}
					m := keymap
					for x := range keyval {
						if x < len(keyname)-1 {
							if n := m[keyval[x]]; n == nil {
								n := map[interface{}]interface{}{}
								m[keyval[x]] = n
								m = n
							} else {
								m = n.(map[interface{}]interface{})
							}
						} else {
							if v != nil {
								m[keyval[x]] = v
							} else {
								m[keyval[x]] = map[interface{}]interface{}{}
							}
						}
					}
					v = keymap
				}
				if cschema.IsListable() {
					switch vv := v.(type) {
					case []interface{}:
						if err := unmarshalYAMLListableNode(node, cschema, cschema.Keyname, vv); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
					case map[interface{}]interface{}:
						kname := cschema.Keyname
						kval := make([]interface{}, 0, len(kname))
						if err := unmarshalYAMLListNode(node, cschema, kname, kval, v); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
					default:
						return Errorf(EAppTagYAMLParsing, "unexpected yaml value %q for %q", vv, cschema.Name)
					}
				} else {
					if child := node.Get(keystr); child == nil {
						if child, err = NewDataNode(cschema); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
						if err = unmarshalYAML(child, v); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
						if _, err = node.Insert(child, nil); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
					} else {
						if err := unmarshalYAML(child, v); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
					}
				}
			}
			return nil
		case []interface{}:
			for i := range entry {
				if err := unmarshalYAML(node, entry[i]); err != nil {
					return Error(EAppTagYAMLParsing, err)
				}
			}
			return nil
		default:
			return Errorf(EAppTagYAMLParsing, "unexpected yaml value %q inserted for %q", yval, node)
		}
	} else {
		switch entry := yval.(type) {
		case map[interface{}]interface{}:
			return Errorf(EAppTagYAMLParsing, "unexpected yaml value %q inserted for %q", entry, node.ID())
		case []interface{}:
			if !node.HasMultipleValues() {
				return Errorf(EAppTagYAMLParsing, "unexpected yaml value %q inserted for %q", entry, node.ID())
			}
			for i := range entry {
				if err := node.Set(ValueToString(entry[i])); err != nil {
					return Error(EAppTagYAMLParsing, err)
				}
			}
			return nil
		default:
			return Error(EAppTagYAMLParsing, node.Set(ValueToString(yval)))
		}
	}
}

// UnmarshalYAML updates the data node using YAML-encoded data.
func UnmarshalYAML(node DataNode, in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(node, ydata)
}

type yamlNode struct {
	DataNode            // Target data node to encode the data node
	RFC7951S            // Modified RFC7951 format for YAML
	InternalFormat bool // Interval YAML format
	ConfigOnly     yang.TriState
	IndentStr      string
	PrefixStr      string
}

func (ynode *yamlNode) getQname() string {
	switch ynode.RFC7951S {
	case RFC7951InProgress, RFC7951Enabled:
		if qname, boundary := ynode.QName(true); boundary ||
			ynode.RFC7951S == RFC7951Enabled {
			ynode.RFC7951S = RFC7951InProgress
			return qname
		}
		return ynode.Schema().Name
	default:
		if ynode.InternalFormat && ynode.IsBranchNode() {
			return ynode.ID()
		}
		return ynode.Schema().Name
	}
}

func (ynode *yamlNode) marshalYAMChildListableNodes(buffer *bytes.Buffer, node []DataNode, i int, indent int, disableFirstIndent bool) (int, error) {
	schema := node[i].Schema()
	cynode := *ynode          // copy options
	cynode.DataNode = node[i] // update the marshalling target
	switch cynode.ConfigOnly {
	case yang.TSTrue:
		if schema.IsState {
			for ; i < len(node); i++ {
				if schema != node[i].Schema() {
					return i, nil
				}
			}
		}
	case yang.TSFalse: // stateOnly
		if !schema.IsState && !schema.HasState {
			for ; i < len(node); i++ {
				if schema != node[i].Schema() {
					return i, nil
				}
			}
		}
	}
	if cynode.RFC7951S != RFC7951Disabled || schema.IsDuplicatableList() || schema.IsLeafList() {
		cynode.WriteIndent(buffer, indent, disableFirstIndent)
		buffer.WriteString(cynode.getQname())
		buffer.WriteString(":\n")
		indent++
		for ; i < len(node); i++ {
			cynode.DataNode = node[i]
			if schema != cynode.Schema() {
				break
			}
			cynode.WriteIndent(buffer, indent, false)
			buffer.WriteString("-")
			cynode.WriteIndent(buffer, 1, false)
			err := cynode.marshalYAML(buffer, indent+2, true)
			if err != nil {
				return i, err
			}
			if cynode.IsLeafList() {
				buffer.WriteString("\n")
			}
		}
		return i, nil
	}
	var lastKeyval []string
	if !cynode.InternalFormat {
		disableFirstIndent = cynode.WriteIndent(buffer, indent, disableFirstIndent)
		buffer.WriteString(cynode.getQname())
		buffer.WriteString(":\n")
		indent++
	}
	for ; i < len(node); i++ {
		cynode.DataNode = node[i]
		if schema != cynode.Schema() {
			break
		}
		if cynode.InternalFormat {
			disableFirstIndent = cynode.WriteIndent(buffer, indent, disableFirstIndent)
			buffer.WriteString(cynode.getQname())
			buffer.WriteString(":\n")
			err := cynode.marshalYAML(buffer, indent+1, false)
			if err != nil {
				return i, err
			}
		} else {
			keyname, keyval := GetKeyValues(cynode.DataNode)
			if len(keyname) != len(keyval) {
				return i, fmt.Errorf("list %q doesn't have a id value", schema.Name)
			}
			for j := range keyval {
				if len(lastKeyval) > 0 && keyval[j] == lastKeyval[j] {
					continue
				}
				cynode.WriteIndent(buffer, indent+j, false)
				buffer.WriteString(keyval[j])
				buffer.WriteString(":\n")
			}
			err := cynode.marshalYAML(buffer, indent+len(keyval), false)
			if err != nil {
				return i, err
			}
			lastKeyval = keyval
		}
	}
	return i, nil
}

func (ynode *yamlNode) marshalYAML(buffer *bytes.Buffer, indent int, disableFirstIndent bool) error {
	cynode := *ynode
	switch {
	case ynode.IsBranchNode():
		children := ynode.Children()
		for i := 0; i < len(children); {
			if children[i].IsListableNode() { // for list and multiple leaf-list nodes
				var err error
				i, err = ynode.marshalYAMChildListableNodes(buffer, children, i, indent, disableFirstIndent)
				if err != nil {
					return Error(EAppTagYAMLEmitting, err)
				}
				disableFirstIndent = false
				continue
			}
			// container, leaf, single leaf-list node
			if (ynode.ConfigOnly == yang.TSTrue && children[i].IsStateNode()) ||
				(ynode.ConfigOnly == yang.TSFalse && !children[i].IsStateNode() && !children[i].HasStateNode()) {
				// skip the node according to the retrieval option
				i++
				continue
			}
			cynode.DataNode = children[i]
			cynode.RFC7951S = ynode.RFC7951S
			disableFirstIndent = cynode.WriteIndent(buffer, indent, disableFirstIndent)
			buffer.WriteString(cynode.getQname())
			buffer.WriteString(":")
			if cynode.IsLeaf() {
				newline := ynode.HasMultipleValues() && ynode.Len() > 8
				if newline {
					buffer.WriteString("\n")
				} else {
					buffer.WriteString(" ")
				}
				if err := cynode.marshalYAML(buffer, indent+1, false); err != nil {
					return Error(EAppTagYAMLEmitting, err)
				}
				if !newline {
					buffer.WriteString("\n")
				}
			} else {
				buffer.WriteString("\n")
				if err := cynode.marshalYAML(buffer, indent+1, false); err != nil {
					return Error(EAppTagYAMLEmitting, err)
				}
			}
			i++
		}
	case ynode.IsLeafNode():
		// write the data to the buffer without the data node name
		if ynode.HasMultipleValues() {
			schema := ynode.Schema()
			value := ynode.Values()
			rfc7951enabled := ynode.RFC7951S != RFC7951Disabled
			if len(value) <= 8 {
				comma := false
				buffer.WriteString("[")
				for i := range value {
					if comma {
						buffer.WriteString(",")
					}
					valbyte, err := ValueToYAMLBytes(schema, schema.Type, value[i], rfc7951enabled)
					if err != nil {
						return Error(EAppTagYAMLEmitting, err)
					}
					buffer.Write(valbyte)
					comma = true
				}
				buffer.WriteString("]")
			} else {
				for i := range value {
					ynode.WriteIndent(buffer, indent, false)
					buffer.WriteString("- ")
					valbyte, err := ValueToYAMLBytes(schema, schema.Type, value[i], rfc7951enabled)
					if err != nil {
						return Error(EAppTagYAMLEmitting, err)
					}
					buffer.Write(valbyte)
					buffer.WriteString("\n")
				}
			}
			return nil
		}
		schema := ynode.Schema()
		rfc7951enabled := ynode.RFC7951S != RFC7951Disabled
		valbyte, err := ValueToYAMLBytes(schema, schema.Type, ynode.Value(), rfc7951enabled)
		if err != nil {
			return Error(EAppTagYAMLEmitting, err)
		}
		buffer.Write(valbyte)
	}
	return nil
}

func (yamlnode *yamlNode) WriteIndent(buffer *bytes.Buffer, indent int, disableIndent bool) bool {
	if disableIndent {
		return false
	}
	buffer.WriteString(yamlnode.PrefixStr)
	for i := 0; i < indent; i++ {
		buffer.WriteString(yamlnode.IndentStr)
	}
	return disableIndent
}

// ValueToYAMLBytes encodes the value to a YAML-encoded data. the schema and the type of the value must be set.
func ValueToYAMLBytes(schema *SchemaNode, typ *yang.YangType, value interface{}, rfc7951 bool) ([]byte, error) {
	switch typ.Kind {
	case yang.Yunion:
		for i := range typ.Type {
			v, err := ValueToYAMLBytes(schema, typ.Type[i], value, rfc7951)
			if err == nil {
				return v, nil
			}
		}
		return nil, fmt.Errorf("unexpected value \"%v\" for %q type", value, typ.Name)
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
	case yang.Yempty:
		return []byte(""), nil
	}
	if rfc7951 {
		switch typ.Kind {
		// case yang.Ystring, yang.Ybinary:
		// case yang.Ybool:
		// case yang.Yleafref:
		// case yang.Ynone:
		// case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yuint8, yang.Yuint16, yang.Yuint32:
		// case yang.Ybits, yang.Yenum:
		// case yang.Yempty:
		// 	return []byte(""), nil
		case yang.Yidentityref:
			if s, ok := value.(string); ok {
				m, ok := schema.Identityref[s]
				if !ok {
					return nil, fmt.Errorf("%q is not a value of %q", s, typ.Name)
				}
				value = m.Name + ":" + s
			}
		case yang.Yint64:
			if v, ok := value.(int64); ok {
				str := strconv.FormatInt(v, 10)
				return []byte(str), nil
			}
		case yang.Yuint64:
			if v, ok := value.(uint64); ok {
				str := strconv.FormatUint(v, 10)
				return []byte(str), nil
			}
		}
	}
	// else {
	// 	switch typ.Kind {
	// 	case yang.Yempty:
	// 		return []byte("null"), nil
	// 	}
	// }
	out, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(string(out), "\n") {
		return out[:len(out)-1], nil
	}
	return out, err
}

// yamlkeys returns the listed key values.
func yamlkeys(node DataNode, rfc7951s RFC7951S) ([]interface{}, error) {
	keynames := node.Schema().Keyname
	keyvals := make([]interface{}, 0, len(keynames)+1)
	keyvals = append(keyvals, node.Name())
	for i := range keynames {
		keynode := node.Get(keynames[i])
		if keynode == nil {
			return nil, fmt.Errorf("%q doesn't have a key node", node.ID())
		}
		if rfc7951s == RFC7951Disabled {
			keyvals = append(keyvals, keynode.Value())
		} else {
			keyvals = append(keyvals, keynode.QValue(true))
		}
	}
	return keyvals, nil
}

func (ynode *yamlNode) marshalYAMLValue(node DataNode, parent map[interface{}]interface{}) (string, error) {
	key := node.Name()
	if ynode.RFC7951S != RFC7951Disabled {
		qname, boundary := node.QName(true)
		if boundary || ynode.DataNode == node {
			key = qname
		}
	}
	if node.IsListableNode() {
		var values, rvalues []interface{}
		if values := parent[key]; values != nil {
			rvalues = parent[key].([]interface{})
		}
		if ynode.RFC7951S == RFC7951Disabled {
			values = node.Values()
		} else {
			values = node.QValues(true)
		}
		parent[key] = append(rvalues, values...)
		return key, nil
	}
	if ynode.RFC7951S == RFC7951Disabled {
		parent[key] = node.Value()
	} else {
		parent[key] = node.QValue(true)
	}
	return key, nil
}

func (ynode *yamlNode) skip(node DataNode) bool {
	schema := node.Schema()
	if (ynode.ConfigOnly == yang.TSTrue && schema.IsState) ||
		(ynode.ConfigOnly == yang.TSFalse && !schema.IsState && !schema.HasState) {
		// skip the node according to the retrieval option
		return true
	}
	return false
}

func (ynode *yamlNode) MarshalYAML() (interface{}, error) {
	top := make(map[interface{}]interface{})
	curkeys := make([]interface{}, 0, 8)
	parent := top
	var traverser func(n DataNode, at TrvsCallOption) error
	switch {
	case ynode.InternalFormat:
		traverser = func(n DataNode, at TrvsCallOption) error {
			if ynode.skip(n) {
				return nil
			}
			switch at {
			case TrvsCalledAtEnter:
				key := n.ID()
				if n.IsDuplicatableNode() {
					dir, ok := parent[key]
					if !ok {
						dir = []interface{}{}
					}
					list := dir.([]interface{})
					dir = make(map[interface{}]interface{})
					parent[key] = append(list, dir)
					curkeys = append(curkeys, key, len(list))
					parent = dir.(map[interface{}]interface{})
				} else {
					dir := make(map[interface{}]interface{})
					parent[key] = dir
					curkeys = append(curkeys, key)
					parent = dir
				}
			case TrvsCalledAtEnterAndExit:
				_, err := ynode.marshalYAMLValue(n, parent)
				return err
			case TrvsCalledAtExit:
				if n.IsDuplicatableNode() {
					curkeys = curkeys[:len(curkeys)-2]
				} else {
					curkeys = curkeys[:len(curkeys)-1]
				}
				var p interface{}
				p = top
				for i := range curkeys {
					switch c := p.(type) {
					case map[interface{}]interface{}:
						p = c[curkeys[i]]
					case []interface{}:
						p = c[curkeys[i].(int)]
					}
				}
				parent = p.(map[interface{}]interface{})
			}
			return nil
		}
	case ynode.RFC7951S != RFC7951Disabled:
		traverser = func(n DataNode, at TrvsCallOption) error {
			if ynode.skip(n) {
				return nil
			}
			switch at {
			case TrvsCalledAtEnter:
				key := n.Name()
				qname, boundary := n.QName(true)
				if boundary || ynode.DataNode == n {
					key = qname
				}
				if n.IsList() {
					dir, ok := parent[key]
					if !ok {
						dir = []interface{}{}
					}
					list := dir.([]interface{})
					dir = make(map[interface{}]interface{})
					parent[key] = append(list, dir)
					curkeys = append(curkeys, key, len(list))
					parent = dir.(map[interface{}]interface{})
				} else {
					dir := make(map[interface{}]interface{})
					parent[key] = dir
					curkeys = append(curkeys, key)
					parent = dir
				}
			case TrvsCalledAtEnterAndExit:
				_, err := ynode.marshalYAMLValue(n, parent)
				return err
			case TrvsCalledAtExit:
				if n.IsList() {
					curkeys = curkeys[:len(curkeys)-2]
				} else {
					curkeys = curkeys[:len(curkeys)-1]
				}
				var p interface{}
				p = top
				for i := range curkeys {
					switch c := p.(type) {
					case map[interface{}]interface{}:
						p = c[curkeys[i]]
					case []interface{}:
						p = c[curkeys[i].(int)]
					}
				}
				parent = p.(map[interface{}]interface{})
			}
			return nil
		}
	default:
		traverser = func(n DataNode, at TrvsCallOption) error {
			if ynode.skip(n) {
				return nil
			}
			switch at {
			case TrvsCalledAtEnter:
				if n.IsDuplicatableNode() {
					key := n.ID()
					dir, ok := parent[key]
					if !ok {
						dir = []interface{}{}
					}
					list := dir.([]interface{})
					dir = make(map[interface{}]interface{})
					parent[key] = append(list, dir)
					curkeys = append(curkeys, key, len(list))
					parent = dir.(map[interface{}]interface{})
				} else {
					keys, err := yamlkeys(n, RFC7951Disabled)
					if err != nil {
						return err
					}
					for i := range keys {
						dir, ok := parent[keys[i]]
						if !ok {
							dir = make(map[interface{}]interface{})
							parent[keys[i]] = dir
						}
						curkeys = append(curkeys, keys[i])
						parent = dir.(map[interface{}]interface{})
					}
				}
			case TrvsCalledAtEnterAndExit:
				_, err := ynode.marshalYAMLValue(n, parent)
				return err
			case TrvsCalledAtExit:
				if n.IsDuplicatableNode() {
					curkeys = curkeys[:len(curkeys)-2]
				} else {
					popcount := len(n.Schema().Keyname) + 1
					curkeys = curkeys[:len(curkeys)-popcount]
				}
				var p interface{}
				p = top
				for i := range curkeys {
					switch c := p.(type) {
					case map[interface{}]interface{}:
						p = c[curkeys[i]]
					case []interface{}:
						p = c[curkeys[i].(int)]
					}
				}
				parent = p.(map[interface{}]interface{})
			}
			return nil
		}
	}

	if ynode.IsBranchNode() {
		children := ynode.Children()
		for i := range children {
			if err := Traverse(children[i], traverser, TrvsCalledAtEnterAndExit, -1, false); err != nil {
				return nil, Error(EAppTagYAMLEmitting, err)
			}
		}
		return top, nil
	} else {
		key, err := ynode.marshalYAMLValue(ynode.DataNode, parent)
		if err != nil {
			return nil, Error(EAppTagYAMLEmitting, err)
		}
		return parent[key], nil
	}
}

// InternalFormat is an option to marshal a data node to an internal YAML format.
type InternalFormat struct{}

func (o InternalFormat) IsOption() {}

// MarshalYAML encodes the data node to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format, InternalFormat].
func MarshalYAML(node DataNode, option ...Option) ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yamlNode{DataNode: node, IndentStr: " "}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, Errorf(EAppTagYAMLEmitting, "%v option can be used to find nodes", option[i])
		case ConfigOnly:
			ynode.ConfigOnly = yang.TSTrue
		case StateOnly:
			ynode.ConfigOnly = yang.TSFalse
		case RFC7951Format:
			ynode.RFC7951S = RFC7951Enabled
		case InternalFormat:
			ynode.InternalFormat = true
		}
	}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
