package yangtree

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"gopkg.in/yaml.v2"
)

// unmarshalYAMLListNode constructs list nodes of the parent node using YAML values.
func unmarshalYAMLListNode(parent DataNode, cschema *SchemaNode, kname []string, kval []interface{}, yamlHash interface{}) error {
	switch entry := yamlHash.(type) {
	case map[interface{}]interface{}:
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
	case map[string]interface{}:
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
	default:
		if yamlHash == nil {
			return nil
		}
		return fmt.Errorf("unexpected value %q (%T) for %q", yamlHash, yamlHash, cschema.Name)
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
		idBuilder.WriteString(ValueToValueString(kval[i]))
		idBuilder.WriteString("]")
	}
	id := idBuilder.String()
	var child, found DataNode
	if found = parent.Get(id); found == nil {
		child, err = NewWithID(cschema, id)
		if err != nil {
			return err
		}
		if _, err = parent.Insert(child, nil); err != nil {
			return err
		}
	} else {
		child = found
	}
	if err := unmarshalYAML(child, cschema, yamlHash); err != nil {
		if found == nil {
			parent.Delete(child)
		}
		return err
	}
	return nil
}

func getValueFromYAMLHash(yamlHash interface{}, kname *string) interface{} {
	switch entry := yamlHash.(type) {
	case map[interface{}]interface{}:
		return entry[*kname]
	case map[string]interface{}:
		return entry[*kname]
	default:
		return nil
	}
}

// unmarshalYAMLListableNode constructs listable child nodes of the parent data node using YAML values.
func unmarshalYAMLListableNode(parent DataNode, cschema *SchemaNode, kname []string, sequnce []interface{}, meta interface{}) error {
	if cschema.IsLeafList() {
		_meta, isSlice := meta.([]interface{})
		for i := range sequnce {
			child, err := New(cschema)
			if err != nil {
				return err
			}
			if err = unmarshalYAML(child, cschema, sequnce[i]); err != nil {
				return err
			}
			if _, err = parent.Insert(child, nil); err != nil {
				return err
			}
			if isSlice && i < len(_meta) {
				if err := unmarshalYAMLUpdateMetadata(child, cschema, _meta[i]); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for i := range sequnce {
		switch sequnce[i].(type) {
		case map[interface{}]interface{}, map[string]interface{}:
		default:
			return fmt.Errorf("unexpected value %T for %s", sequnce[i], cschema.Name)
		}
		// check existent DataNode
		var err error
		var idBuilder strings.Builder
		idBuilder.WriteString(cschema.Name)
		for j := range kname {
			kvalue := getValueFromYAMLHash(sequnce[i], &(kname[j]))
			if kvalue == nil {
				kcschema := cschema.GetSchema(kname[j])
				qname, _ := kcschema.GetQName(true)
				if kvalue = getValueFromYAMLHash(sequnce[i], &qname); kvalue == nil {
					qname, _ = kcschema.GetQName(false)
					if kvalue = getValueFromYAMLHash(sequnce[i], &qname); kvalue == nil {
						return fmt.Errorf("not found key data node %q", kname[j])
					}
				}
			}
			idBuilder.WriteString("[")
			idBuilder.WriteString(kname[j])
			idBuilder.WriteString("=")
			idBuilder.WriteString(fmt.Sprint(kvalue))
			idBuilder.WriteString("]")
		}
		id := idBuilder.String()
		var child, found DataNode
		if cschema.IsDuplicatableList() {
			child = nil
		} else if found = parent.Get(id); found != nil {
			child = found
		}
		if child == nil {
			child, err = NewWithID(cschema, id)
			if err != nil {
				return err
			}
			if _, err = parent.Insert(child, nil); err != nil {
				return err
			}
		}
		if err := unmarshalYAML(child, cschema, sequnce[i]); err != nil {
			if found == nil {
				parent.Delete(child)
			}
			return err
		}
	}
	return nil
}

func unmarshalYAMLUpdateMetadata(node DataNode, schema *SchemaNode, meta interface{}) error {
	switch _meta := meta.(type) {
	case map[interface{}]interface{}:
		for k, v := range _meta {
			if kv, ok := k.(string); ok {
				if err := node.SetMetadata(kv, v); err != nil {
					return err
				}
			}
		}
	case map[string]interface{}:
		for k, v := range _meta {
			if err := node.SetMetadata(k, v); err != nil {
				return err
			}
		}
	case []interface{}:
		for i := range _meta {
			if err := unmarshalYAMLUpdateMetadata(node, schema, _meta[i]); err != nil {
				return err
			}
		}
	case nil:
		return nil
	default:
		return Errorf(EAppTagYAMLParsing, "invalid metadata format for %q", node)
	}
	return nil
}

func unmarshalYAMLkeyval(parent DataNode, cschema *SchemaNode, haskey bool, keystr *string, v interface{}, meta interface{}) error {
	if haskey {
		keyname := cschema.Keyname
		keyval, err := extractKeyValues(keyname, keystr)
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
	if cschema.IsListable() {
		switch vv := v.(type) {
		case []interface{}:
			if err := unmarshalYAMLListableNode(parent, cschema, cschema.Keyname, vv, meta); err != nil {
				return err
			}
		case map[interface{}]interface{}, map[string]interface{}:
			kname := cschema.Keyname
			kval := make([]interface{}, 0, len(kname))
			if err := unmarshalYAMLListNode(parent, cschema, kname, kval, v); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected value %q for %q", vv, cschema.Name)
		}
	} else {
		var err error
		var child DataNode
		if child = parent.Get(*keystr); child == nil {
			if child, err = New(cschema); err != nil {
				return err
			}
			if err = unmarshalYAML(child, cschema, v); err != nil {
				return err
			}
			if _, err = parent.Insert(child, nil); err != nil {
				return err
			}
		} else {
			if err := unmarshalYAML(child, cschema, v); err != nil {
				return err
			}
		}
		if meta != nil {
			if err := unmarshalYAMLUpdateMetadata(child, cschema, meta); err != nil {
				return err
			}
		}
	}
	return nil
}

// unmarshalYAML updates a data node using YAML values.
func unmarshalYAML(node DataNode, schema *SchemaNode, yval interface{}) error {
	if schema.IsDir() {
		switch entry := yval.(type) {
		case map[interface{}]interface{}:
			for k, v := range entry {
				kstr := ValueToValueString(k)
				name, haskey, err := extractSchemaName(&kstr)
				if err != nil {
					return err
				}
				if name == "@" {
					if err := unmarshalYAMLUpdateMetadata(node, schema, v); err != nil {
						return err
					}
				}
				if strings.HasPrefix(name, "@") {
					continue
				}
				cschema := schema.GetSchema(name)
				if cschema == nil {
					return fmt.Errorf("schema %q not found from %q", kstr, schema.Name)
				}
				mname := "@" + cschema.Name
				if err := unmarshalYAMLkeyval(node, cschema, haskey, &kstr, v, getValueFromYAMLHash(entry, &mname)); err != nil {
					return Error(EAppTagYAMLParsing, err)
				}
			}
			return nil
		case map[string]interface{}:
			for k, v := range entry {
				name, haskey, err := extractSchemaName(&k)
				if err != nil {
					return err
				}
				if name == "@" {
					if err := unmarshalYAMLUpdateMetadata(node, schema, v); err != nil {
						return err
					}
				}
				if strings.HasPrefix(name, "@") {
					continue
				}
				cschema := schema.GetSchema(name)
				if cschema == nil {
					return fmt.Errorf("schema %q not found from %q", k, schema.Name)
				}
				mname := "@" + cschema.Name
				if err := unmarshalYAMLkeyval(node, cschema, haskey, &k, v, getValueFromYAMLHash(entry, &mname)); err != nil {
					return Error(EAppTagYAMLParsing, err)
				}
			}
			return nil
		case []interface{}:
			for i := range entry {
				if err := unmarshalYAML(node, schema, entry[i]); err != nil {
					return Error(EAppTagYAMLParsing, err)
				}
			}
			return nil
		case nil:
			return nil
		default:
			return Errorf(EAppTagYAMLParsing, "unexpected value %q inserted for %q", yval, node)
		}
	} else {
		switch entry := yval.(type) {
		case map[interface{}]interface{}, map[string]interface{}:
			return Errorf(EAppTagYAMLParsing, "unexpected value %q inserted for %q", entry, node.ID())
		case []interface{}:
			if !schema.IsSingleLeafList() {
				return Errorf(EAppTagYAMLParsing, "unexpected value %q inserted for %q", entry, node.ID())
			}
			for i := range entry {
				if err := node.SetValue(entry[i]); err != nil {
					return Error(EAppTagYAMLParsing, err)
				}
			}
			return nil
		default:
			if err := node.SetValue(yval); err != nil {
				return Error(EAppTagYAMLParsing, err)
			}
			return nil
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
	// pretty.Print(ydata)
	return unmarshalYAML(node, node.Schema(), ydata)
}

type yamlNode struct {
	DataNode            // Target data node to encode the data node
	RFC7951S            // Modified RFC7951 format for YAML
	InternalFormat bool // Interval YAML format
	printMeta      bool // Print all metadata
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

func (ynode *yamlNode) marshalYAMLMetadata(buffer *bytes.Buffer, indent int, unindent, printName bool) error {
	// marshalling metadata
	var err error
	var indentoffset int
	if indent < 0 {
		return nil
	}
	m := ynode.Metadata()
	if len(m) == 0 {
		return nil
	}
	switch {
	case ynode.IsBranchNode():
		if printName {
			unindent = ynode.WriteIndent(buffer, indent, unindent)
			buffer.WriteString("\"@\":\n")
			indentoffset++
		}
		mynode := *ynode
		for _, mdata := range m {
			mynode.DataNode = mdata
			mynode.RFC7951S = ynode.RFC7951S
			if err = mynode.marshalYAML(buffer, indent+indentoffset, unindent, true); err != nil {
				return err
			}
			if mynode.IsLeafNode() {
				buffer.WriteString("\n")
			}
		}
	default:
		if printName {
			unindent = ynode.WriteIndent(buffer, indent, unindent)
			buffer.WriteString("\"@")
			buffer.WriteString(ynode.getQname())
			buffer.WriteString("\":\n")
			indentoffset++
		}
		mynode := *ynode
		if ynode.HasMultipleValues() && ynode.RFC7951S != RFC7951Disabled { // single leaf-list schema node
			length := len(ynode.Values())
			for i := 0; i < length; i++ {
				buffer.WriteString("- ")
				for _, mdata := range m {
					mynode.DataNode = mdata
					mynode.RFC7951S = ynode.RFC7951S
					if err = mynode.marshalYAML(buffer, indent+indentoffset+2, unindent, true); err != nil {
						return err
					}
					if mynode.IsLeafNode() {
						buffer.WriteString("\n")
					}
				}
			}
		} else {
			for _, mdata := range m {
				mynode.DataNode = mdata
				mynode.RFC7951S = ynode.RFC7951S
				if err = mynode.marshalYAML(buffer, indent+1, unindent, true); err != nil {
					return err
				}
				if mynode.IsLeafNode() {
					buffer.WriteString("\n")
				}
			}
		}
	}

	return err
}

func (ynode *yamlNode) marshalYAMChildListableNodes(
	buffer *bytes.Buffer, node []DataNode, i int, indent int, unindent bool, printName bool) (int, error) {
	indentoffset := 0
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
		if printName {
			if indent >= 0 {
				cynode.WriteIndent(buffer, indent, unindent)
				buffer.WriteString(cynode.getQname())
				buffer.WriteString(":\n")
			}
			indentoffset++
		}
		j := i
		cindent := indent + indentoffset
		for ; i < len(node); i++ {
			cynode.DataNode = node[i]
			cynode.RFC7951S = ynode.RFC7951S
			if schema != cynode.Schema() {
				break
			}
			if cindent >= 0 {
				cynode.WriteIndent(buffer, cindent, false)
				buffer.WriteString("- ")
			}
			err := cynode.marshalYAML(buffer, cindent+2, true, false)
			if err != nil {
				return i, err
			}
			if cindent >= 0 {
				if cynode.IsLeafList() {
					buffer.WriteString("\n")
				}
			}
		}
		if ynode.printMeta {
			// processing the metadata of multiple leaf-list node schema
			if schema.IsLeafList() && len(node[j].Metadata()) > 0 {
				cynode.DataNode = node[j]
				cynode.RFC7951S = ynode.RFC7951S
				cynode.WriteIndent(buffer, indent, false)
				buffer.WriteString("\"@")
				buffer.WriteString(cynode.getQname())
				buffer.WriteString("\":\n")
				for ; j < i; j++ {
					cynode.DataNode = node[j]
					cynode.RFC7951S = ynode.RFC7951S
					if cindent >= 0 {
						cynode.WriteIndent(buffer, cindent, false)
						buffer.WriteString("- ")
					}
					if err := cynode.marshalYAMLMetadata(buffer, cindent+2, true, false); err != nil {
						return i, err
					}
				}
			}
		}
		return i, nil
	}
	var lastKeyval []string
	if !cynode.InternalFormat {
		if printName {
			if indent >= 0 {
				unindent = cynode.WriteIndent(buffer, indent, unindent)
				buffer.WriteString(cynode.getQname())
				buffer.WriteString(":\n")
			}
			indentoffset++
		}
	}
	for ; i < len(node); i++ {
		cynode.DataNode = node[i]
		cynode.RFC7951S = ynode.RFC7951S
		if schema != cynode.Schema() {
			break
		}
		if !cynode.InternalFormat {
			keyname, keyval := GetKeyValues(cynode.DataNode)
			if len(keyname) != len(keyval) {
				return i, fmt.Errorf("list %q doesn't have a id value", schema.Name)
			}
			cindent := indent + indentoffset
			for j := range keyval {
				if len(lastKeyval) > 0 && keyval[j] == lastKeyval[j] {
					continue
				}
				if cindent+j >= 0 {
					cynode.WriteIndent(buffer, cindent+j, false)
					buffer.WriteString(keyval[j])
					buffer.WriteString(":\n")
				}
			}
			err := cynode.marshalYAML(buffer, cindent+len(keyval), false, false)
			if err != nil {
				return i, err
			}
			lastKeyval = keyval
		} else { // InternalFormat
			err := cynode.marshalYAML(buffer, indent+indentoffset, unindent, true)
			if err != nil {
				return i, err
			}
		}
	}
	return i, nil
}

func (ynode *yamlNode) marshalYAML(buffer *bytes.Buffer, indent int, unindent, printName bool) error {
	var indentoffset int
	if printName {
		if indent >= 0 {
			unindent = ynode.WriteIndent(buffer, indent, unindent)
			buffer.WriteString(ynode.getQname())
			if ynode.IsLeafNode() {
				if ynode.HasMultipleValues() && ynode.Len() > 8 {
					buffer.WriteString(":\n")
				} else {
					buffer.WriteString(": ")
				}
			} else {
				buffer.WriteString(":\n")
			}
		}
		indentoffset++
	}

	cynode := *ynode
	switch {
	case ynode.IsBranchNode():
		children := ynode.Children()
		for i := 0; i < len(children); {
			if children[i].IsListableNode() { // for list and multiple leaf-list nodes
				var err error
				i, err = ynode.marshalYAMChildListableNodes(buffer, children, i, indent+indentoffset, unindent, true)
				if err != nil {
					return Error(EAppTagYAMLEmitting, err)
				}
				unindent = false
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
			if err := cynode.marshalYAML(buffer, indent+indentoffset, unindent, true); err != nil {
				return Error(EAppTagYAMLEmitting, err)
			}
			if unindent {
				unindent = false
			}
			if cynode.IsLeafNode() {
				buffer.WriteString("\n")
				if cynode.printMeta {
					if err := cynode.marshalYAMLMetadata(buffer, indent+indentoffset, false, true); err != nil {
						return err
					}
				}
			}
			i++
		}
		if ynode.printMeta {
			if err := ynode.marshalYAMLMetadata(buffer, indent+indentoffset, false, true); err != nil {
				return err
			}
		}
	case ynode.HasMultipleValues():
		schema := ynode.Schema()
		value := ynode.Values()
		if len(value) > 8 {
			for i := range value {
				ynode.WriteIndent(buffer, indent, false)
				buffer.WriteString("- ")
				valbyte, err := schema.ValueToYAMLBytes(schema.Type, value[i], ynode.RFC7951S != RFC7951Disabled)
				if err != nil {
					return Error(EAppTagYAMLEmitting, err)
				}
				buffer.Write(valbyte)
				buffer.WriteString("\n")
			}
		} else {
			comma := false
			buffer.WriteString("[")
			for i := range value {
				if comma {
					buffer.WriteString(",")
				}
				valbyte, err := schema.ValueToYAMLBytes(schema.Type, value[i], ynode.RFC7951S != RFC7951Disabled)
				if err != nil {
					return Error(EAppTagYAMLEmitting, err)
				}
				buffer.Write(valbyte)
				comma = true
			}
			buffer.WriteString("]")
		}
	case ynode.IsLeafNode():
		schema := ynode.Schema()
		valbyte, err := schema.ValueToYAMLBytes(schema.Type, ynode.Value(), ynode.RFC7951S != RFC7951Disabled)
		if err != nil {
			return Error(EAppTagYAMLEmitting, err)
		}
		buffer.Write(valbyte)
	}
	return nil
}

func (yamlnode *yamlNode) WriteIndent(buffer *bytes.Buffer, indent int, unindent bool) bool {
	if unindent {
		return false
	}
	buffer.WriteString(yamlnode.PrefixStr)
	for i := 0; i < indent; i++ {
		buffer.WriteString(yamlnode.IndentStr)
	}
	return unindent
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
			kschema := keynode.Schema()
			v, err := kschema.ValueToQValue(kschema.Type, keynode.Value(), true)
			if err != nil {
				return nil, err
			}
			keyvals = append(keyvals, v)
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
			values = node.Values()
			schema := node.Schema()
			for i := range values {
				v, err := schema.ValueToQValue(schema.Type, values[i], true)
				if err != nil {
					return "", err
				}
				values[i] = v
			}
		}
		parent[key] = append(rvalues, values...)
		return key, nil
	}
	if ynode.RFC7951S == RFC7951Disabled {
		parent[key] = node.Value()
	} else {
		kschema := node.Schema()
		v, err := kschema.ValueToQValue(kschema.Type, node.Value(), true)
		if err != nil {
			return "", err
		}
		parent[key] = v
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

// toMap() encodes the data node using golang yaml marshaler interface.
func (ynode *yamlNode) toMap(skipRoot bool) (interface{}, error) {
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
		if skipRoot {
			if len(top) == 1 {
				for _, v := range top {
					return v, nil
				}
			}
			return nil, Errorf(EAppTagYAMLEmitting, "unable to skip top node marshalling because there are a lot of children")
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

func (ynode *yamlNode) MarshalYAML() (interface{}, error) {
	return ynode.toMap(false)
}

// InternalFormat is an option to marshal a data node to an internal YAML format.
type InternalFormat struct{}

func (o InternalFormat) IsOption() {}

type YAMLIndent string

func (o YAMLIndent) IsOption() {}

// MarshalYAML encodes the data node to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format, InternalFormat].
func MarshalYAML(node DataNode, option ...Option) ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yamlNode{DataNode: node, IndentStr: " "}
	for i := range option {
		switch o := option[i].(type) {
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
		case Metadata:
			ynode.printMeta = true
		case YAMLIndent:
			ynode.IndentStr = string(o)
		}
	}
	if _, ok := node.(*DataNodeGroup); ok {
		if err := ynode.marshalYAML(buffer, -2, false, false); err != nil {
			return nil, err
		}
	} else {
		if err := ynode.marshalYAML(buffer, 0, false, false); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}
