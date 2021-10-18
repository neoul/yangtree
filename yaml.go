package yangtree

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"gopkg.in/yaml.v2"
)

// unmarshalYAMLListableNode constructs list entries of a data node using YAML values.
func (branch *DataBranch) unmarshalYAMLListableNode(cschema *SchemaNode, kname []string, kval []interface{}, yval interface{}) error {
	jdata, ok := yval.(map[interface{}]interface{})
	if !ok {
		if yval == nil {
			return nil
		}
		return fmt.Errorf("unexpected yaml-val \"%v\" (%T) for %q", yval, yval, cschema.Name)
	}
	if len(kname) != len(kval) {
		for k, v := range jdata {
			kval = append(kval, k)
			err := branch.unmarshalYAMLListableNode(cschema, kname, kval, v)
			if err != nil {
				return err
			}
			kval = kval[:len(kval)-1]
		}
		return nil
	}
	// check existent DataNode
	var err error
	var id strings.Builder
	id.WriteString(cschema.Name)
	for i := range kval {
		id.WriteString("[")
		id.WriteString(kname[i])
		id.WriteString("=")
		id.WriteString(ValueToString(kval[i]))
		id.WriteString("]")
	}
	var child DataNode
	found := branch.Get(id.String())
	if found == nil {
		if child, err = branch.NewDataNode(id.String()); err != nil {
			return err
		}
	} else {
		child = found
	}
	// Update DataNode
	return unmarshalYAML(child, yval)
}

// unmarshalYAMLListableNodeForRFC7951 constructs list entries of a data node using YAML values.
func (branch *DataBranch) unmarshalYAMLListableNodeForRFC7951(cschema *SchemaNode, kname []string, listentry []interface{}) error {
	for i := range listentry {
		entry, ok := listentry[i].(map[interface{}]interface{})
		if !ok {
			return fmt.Errorf("unexpected yaml type '%T' for %s", listentry[i], cschema.Name)
		}
		// check existent DataNode
		var err error
		var id strings.Builder
		id.WriteString(cschema.Name)
		for i := range kname {
			id.WriteString("[")
			id.WriteString(kname[i])
			id.WriteString("=")
			id.WriteString(fmt.Sprint(entry[kname[i]]))
			id.WriteString("]")
			// [FIXME] need to check id validation
			// kchild, err := NewDataNode(kschema, kval)
			// if err != nil {
			// 	return err
			// }
		}
		var child DataNode
		if cschema.IsDuplicatableList() {
			if child, err = branch.NewDataNode(id.String()); err != nil {
				return err
			}
		} else {
			found := branch.Get(id.String())
			if found == nil {
				if child, err = branch.NewDataNode(id.String()); err != nil {
					return err
				}
			} else {
				child = found
			}
		}

		// Update DataNode
		if err := unmarshalYAML(child, entry); err != nil {
			return err
		}
	}
	return nil
}

// unmarshalYAML updates a data node using YAML values.
func unmarshalYAML(node DataNode, yval interface{}) error {
	switch n := node.(type) {
	case *DataBranch:
		switch data := yval.(type) {
		case map[interface{}]interface{}:
			for k, v := range data {
				keystr := ValueToString(k)
				name, haskey, err := extractSchemaName(&keystr)
				if err != nil {
					return err
				}
				cschema := n.schema.GetSchema(name)
				if cschema == nil {
					return fmt.Errorf("schema %q not found from %q", k, n.schema.Name)
				}

				switch {
				case cschema.IsList():
					if haskey {
						keyname := cschema.Keyname
						keyval, err := extractKeyValues(keyname, &keystr)
						if err != nil {
							return err
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
					if rfc7951StyleList, ok := v.([]interface{}); ok {
						if err := n.unmarshalYAMLListableNodeForRFC7951(cschema, cschema.Keyname, rfc7951StyleList); err != nil {
							return err
						}
					} else {
						if cschema.IsDuplicatableList() {
							return fmt.Errorf("non-id list %q must have the array format", cschema.Name)
						}
						kname := cschema.Keyname
						kval := make([]interface{}, 0, len(kname))
						if err := n.unmarshalYAMLListableNode(cschema, kname, kval, v); err != nil {
							return err
						}
					}
				case cschema.IsLeafList():
					vv, ok := v.([]interface{})
					if !ok {
						return fmt.Errorf("unexpected type inserted for %q", cschema.Name)
					}
					for i := range vv {
						child, err := NewDataNode(cschema)
						if err != nil {
							return err
						}
						if err := unmarshalYAML(child, vv[i]); err != nil {
							return err
						}
						if _, err := n.insert(child, EditMerge, nil); err != nil {
							return err
						}
					}
				default:
					var child DataNode
					i := n.Index(keystr)
					if i < len(n.children) && n.children[i].ID() == k {
						child = n.children[i]
						if err := unmarshalYAML(child, v); err != nil {
							return err
						}
					} else {
						child, err := NewDataNode(cschema)
						if err != nil {
							return err
						}
						if err := unmarshalYAML(child, v); err != nil {
							return err
						}
						if _, err := n.insert(child, EditMerge, nil); err != nil {
							return err
						}
					}

				}
			}
			return nil
		case []interface{}:
			for i := range data {
				if err := unmarshalYAML(node, data[i]); err != nil {
					return err
				}
			}
			return nil
		default:
			return fmt.Errorf("unexpected yaml value \"%v\" (%T) inserted for %q", yval, yval, n)
		}
	case *DataLeaf:
		return n.Set(ValueToString(yval))
	default:
		return fmt.Errorf("unknown data node type: %T", node)
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

type yDataNode struct {
	DataNode        // Target data node to encode the data node
	rfc7951s        // Modified RFC7951 format for YAML
	iformat    bool // Interval YAML format
	configOnly yang.TriState
	indentStr  string // indent string used at YAML encoding
}

func (ynode *yDataNode) getQname() string {
	switch ynode.rfc7951s {
	case rfc7951InProgress, rfc7951Enabled:
		if qname, boundary := ynode.Schema().GetQName(true); boundary ||
			ynode.rfc7951s == rfc7951Enabled {
			ynode.rfc7951s = rfc7951InProgress
			return qname
		}
		return ynode.Schema().Name
	}
	if ynode.iformat && ynode.IsDataBranch() {
		return ynode.ID()
	}
	return ynode.Schema().Name
}

func marshalYAMLListableNode(buffer *bytes.Buffer, node []DataNode, i int, indent int, parent *yDataNode, disableFirstIndent bool) (int, error) {
	schema := node[i].Schema()
	ynode := *parent         // copy options
	ynode.DataNode = node[i] // update the marshalling target
	switch ynode.configOnly {
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
	if ynode.rfc7951s != rfc7951Disabled || schema.IsDuplicatableList() || schema.IsLeafList() {
		writeIndent(buffer, indent, ynode.indentStr, disableFirstIndent)
		buffer.WriteString(ynode.getQname())
		buffer.WriteString(":\n")
		indent++
		for ; i < len(node); i++ {
			ynode.DataNode = node[i]
			if schema != ynode.Schema() {
				break
			}
			writeIndent(buffer, indent, ynode.indentStr, false)
			buffer.WriteString("-")
			writeIndent(buffer, 1, ynode.indentStr, false)
			err := ynode.marshalYAML(buffer, indent+2, true)
			if err != nil {
				return i, err
			}
			if ynode.IsLeafList() {
				buffer.WriteString("\n")
			}
		}
		return i, nil
	}
	var lastKeyval []string
	if !ynode.iformat {
		disableFirstIndent = writeIndent(buffer, indent, ynode.indentStr, disableFirstIndent)
		buffer.WriteString(ynode.getQname())
		buffer.WriteString(":\n")
		indent++
	}
	for ; i < len(node); i++ {
		ynode.DataNode = node[i]
		if schema != ynode.Schema() {
			break
		}
		if ynode.iformat {
			disableFirstIndent = writeIndent(buffer, indent, ynode.indentStr, disableFirstIndent)
			buffer.WriteString(ynode.getQname())
			buffer.WriteString(":\n")
			err := ynode.marshalYAML(buffer, indent+1, false)
			if err != nil {
				return i, err
			}
		} else {
			keyname, keyval := GetKeyValues(ynode.DataNode)
			if len(keyname) != len(keyval) {
				return i, fmt.Errorf("list %q doesn't have a id value", schema.Name)
			}
			for j := range keyval {
				if len(lastKeyval) > 0 && keyval[j] == lastKeyval[j] {
					continue
				}
				writeIndent(buffer, indent+j, ynode.indentStr, false)
				buffer.WriteString(keyval[j])
				buffer.WriteString(":\n")
			}
			err := ynode.marshalYAML(buffer, indent+len(keyval), false)
			if err != nil {
				return i, err
			}
			lastKeyval = keyval
		}
	}
	return i, nil
}

func (ynode *yDataNode) marshalYAML(buffer *bytes.Buffer, indent int, disableFirstIndent bool) error {
	if ynode == nil || ynode.DataNode == nil {
		return nil
	}
	cynode := *ynode
	switch datanode := ynode.DataNode.(type) {
	case *DataBranch:
		node := datanode.children
		for i := 0; i < len(datanode.children); {
			schema := node[i].Schema()
			if schema.IsListable() { // for list and multiple leaf-list
				var err error
				i, err = marshalYAMLListableNode(buffer, node, i, indent, ynode, disableFirstIndent)
				if err != nil {
					return err
				}
				disableFirstIndent = false
				continue
			}
			// container, leaf, single leaf-list node
			if (ynode.configOnly == yang.TSTrue && schema.IsState) ||
				(ynode.configOnly == yang.TSFalse && !schema.IsState && !schema.HasState) {
				// skip the node according to the retrieval option
				i++
				continue
			}
			cynode.DataNode = node[i]
			cynode.rfc7951s = ynode.rfc7951s
			disableFirstIndent = writeIndent(buffer, indent, cynode.indentStr, disableFirstIndent)
			buffer.WriteString(cynode.getQname())
			buffer.WriteString(":")
			if cynode.IsLeaf() {
				buffer.WriteString(" ")
				if err := cynode.marshalYAML(buffer, indent+1, false); err != nil {
					return err
				}
				buffer.WriteString("\n")
			} else if leaflist, ok := cynode.DataNode.(*DataLeafList); ok {
				if len(leaflist.value) <= 8 {
					buffer.WriteString(" ")
				} else {
					buffer.WriteString("\n")
				}
				if err := cynode.marshalYAML(buffer, indent+1, false); err != nil {
					return err
				}
				if len(leaflist.value) <= 8 {
					buffer.WriteString("\n")
				}
			} else {
				buffer.WriteString("\n")
				if err := cynode.marshalYAML(buffer, indent+1, false); err != nil {
					return err
				}
			}
			i++
		}
	case *DataLeaf:
		rfc7951enabled := ynode.rfc7951s != rfc7951Disabled
		valbyte, err := ValueToYAMLBytes(datanode.schema, datanode.schema.Type, datanode.value, rfc7951enabled)
		if err != nil {
			return err
		}
		buffer.Write(valbyte)
	case *DataLeafList:
		rfc7951enabled := ynode.rfc7951s != rfc7951Disabled
		if len(datanode.value) <= 8 {
			comma := false
			buffer.WriteString("[")
			for i := range datanode.value {
				if comma {
					buffer.WriteString(",")
				}
				valbyte, err := ValueToYAMLBytes(datanode.schema, datanode.schema.Type, datanode.value[i], rfc7951enabled)
				if err != nil {
					return err
				}
				buffer.Write(valbyte)
				comma = true
			}
			buffer.WriteString("]")
		} else {
			for i := range datanode.value {
				writeIndent(buffer, indent, ynode.indentStr, false)
				buffer.WriteString("- ")
				valbyte, err := ValueToYAMLBytes(datanode.schema, datanode.schema.Type, datanode.value[i], rfc7951enabled)
				if err != nil {
					return err
				}
				buffer.Write(valbyte)
				buffer.WriteString("\n")
			}
		}
	}
	return nil
}

// InternalFormat is an option to marshal a data node to an internal YAML format.
type InternalFormat struct{}

func (o InternalFormat) IsOption() {}

// MarshalYAML encodes the data node to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format, InternalFormat].
func MarshalYAML(node DataNode, option ...Option) ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: node, indentStr: " "}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v option can be used to find nodes", option[i])
		case ConfigOnly:
			ynode.configOnly = yang.TSTrue
		case StateOnly:
			ynode.configOnly = yang.TSFalse
		case RFC7951Format:
			ynode.rfc7951s = rfc7951Enabled
		case InternalFormat:
			ynode.iformat = true
		}
	}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeIndent(buffer *bytes.Buffer, indent int, indentStr string, disableIndent bool) bool {
	if disableIndent {
		return false
	}
	for i := 0; i < indent; i++ {
		buffer.WriteString(indentStr)
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

type YAMLNode struct {
	DataNode            // Target data node to encode the data node
	RFC7951Format  bool // Modified RFC7951 format for YAML
	InternalFormat bool // Interval YAML format
	ConfigOnly     yang.TriState
}

// ValueToYAMLValue encodes the value to a YAML-encoded data. the schema and the type of the value must be set.
func ValueToYAMLValue(schema *SchemaNode, typ *yang.YangType, value interface{}, rfc7951 bool) (interface{}, error) {
	switch typ.Kind {
	case yang.Yunion:
		for i := range typ.Type {
			v, err := ValueToYAMLValue(schema, typ.Type[i], value, rfc7951)
			if err == nil {
				return v, nil
			}
		}
		return nil, fmt.Errorf("unexpected value \"%v\" for %q type", value, typ.Name)
	case yang.YinstanceIdentifier:
		// [FIXME] The leftmost (top-level) data node name is always in the
		//   namespace-qualified form (qname).
	case yang.Yidentityref:
		if rfc7951 {
			v := value
			if m, ok := schema.Identityref[v.(string)]; ok {
				return m.Name + ":" + v.(string), nil
			} else {
				return v, nil
			}
		}
	case yang.Yenum:
	case yang.Ydecimal64:
		if vv, ok := value.(yang.Number); ok {
			return strconv.ParseFloat(vv.String(), 64)
		}
	}
	v := value
	if vv, ok := v.(yang.Number); ok {
		return vv.String(), nil
	}
	return v, nil
}

// yamlkeys returns the listed key values.
func yamlkeys(node DataNode, rfc7951 bool) ([]interface{}, error) {
	keynames := node.Schema().Keyname
	keyvals := make([]interface{}, 0, len(keynames)+1)
	keyvals = append(keyvals, node.Name())
	for i := range keynames {
		keynode := node.Get(keynames[i])
		if keynode == nil {
			return nil, fmt.Errorf("%q doesn't have a key node", node.ID())
		}
		cschema := keynode.Schema()
		v, err := ValueToYAMLValue(cschema, cschema.Type, keynode.Value(), rfc7951)
		if err != nil {
			return nil, err
		}
		keyvals = append(keyvals, v)
	}
	return keyvals, nil
}

func (ynode *YAMLNode) marshalYAMLValue(node DataNode, parent map[interface{}]interface{}) error {
	schema := node.Schema()
	key := node.Name()
	if ynode.RFC7951Format {
		qname, boundary := node.QName(true)
		if boundary || ynode.DataNode == node {
			key = qname
		}
	}
	if node.IsLeafList() {
		var rvalues []interface{}
		if values := parent[key]; values != nil {
			rvalues = parent[key].([]interface{})
		}
		values := node.Values()
		for i := range values {
			v, err := ValueToYAMLValue(schema, schema.Type, values[i], ynode.RFC7951Format)
			if err != nil {
				return err
			}
			rvalues = append(rvalues, v)
		}
		parent[key] = rvalues
		return nil
	}
	v, err := ValueToYAMLValue(schema, schema.Type, node.Value(), ynode.RFC7951Format)
	if err != nil {
		return err
	}
	parent[key] = v
	return nil
}

func (ynode *YAMLNode) skip(node DataNode) bool {
	schema := node.Schema()
	if (ynode.ConfigOnly == yang.TSTrue && schema.IsState) ||
		(ynode.ConfigOnly == yang.TSFalse && !schema.IsState && !schema.HasState) {
		// skip the node according to the retrieval option
		return true
	}
	return false
}

func (ynode *YAMLNode) MarshalYAML() (interface{}, error) {
	top := make(map[interface{}]interface{})
	curkeys := make([]interface{}, 0, 8)
	var targetkeys []interface{}
	parent := top
	var traverser func(n DataNode, at TrvsCallOption) error
	switch {
	case ynode.RFC7951Format:
		traverser = func(n DataNode, at TrvsCallOption) error {
			if ynode.skip(n) {
				return nil
			}
			switch at {
			case TrvsCalledAtEnter:
				key := n.ID()
				if n.IsDuplicatable() {
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
				return ynode.marshalYAMLValue(n, parent)
			case TrvsCalledAtExit:
				if n == ynode.DataNode {
					targetkeys = curkeys
				}
				if n.IsDuplicatable() {
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
	case ynode.RFC7951Format:
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
				return ynode.marshalYAMLValue(n, parent)
			case TrvsCalledAtExit:
				if n == ynode.DataNode {
					targetkeys = curkeys
				}
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
				if n.IsDuplicatable() {
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
					keys, err := yamlkeys(n, false)
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
				return ynode.marshalYAMLValue(n, parent)
			case TrvsCalledAtExit:
				if n == ynode.DataNode {
					targetkeys = curkeys
				}
				if n.IsDuplicatable() {
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

	if err := Traverse(ynode.DataNode, traverser, TrvsCalledAtEnterAndExit, -1, false); err != nil {
		return nil, err
	}

	var _top interface{}
	_top = top
	for i := range targetkeys {
		switch c := _top.(type) {
		case map[interface{}]interface{}:
			_top = c[targetkeys[i]]
		case []interface{}:
			_top = c[targetkeys[i].(int)]
		}
	}
	return _top, nil
}
