package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

type RFC7951Format struct{}

func (f RFC7951Format) IsOption() {}

// RFC7951S (rfc7951 processing status)
type RFC7951S int

const (
	// rfc7951disabled - rfc7951 encoding disabled
	RFC7951Disabled RFC7951S = iota
	// RFC7951InProgress - rfc7951 encoding is in progress
	RFC7951InProgress
	// RFC7951Enabled - rfc7951 encoding enabled
	RFC7951Enabled
)

func (s RFC7951S) String() string {
	switch s {
	case RFC7951Disabled:
		return "rfc7951.disabled"
	case RFC7951InProgress:
		return "rfc7951.in-progress"
	case RFC7951Enabled:
		return "rfc7951.enabled"
	}
	return "rfc7951.unknown"
}

type jsonNode struct {
	DataNode
	RFC7951S
	ConfigOnly yang.TriState
}

func (jnode *jsonNode) getQname() string {
	switch jnode.RFC7951S {
	case RFC7951InProgress, RFC7951Enabled:
		if qname, boundary := jnode.Schema().GetQName(true); boundary ||
			jnode.RFC7951S == RFC7951Enabled {
			jnode.RFC7951S = RFC7951InProgress
			return qname
		}
		return jnode.Schema().Name
	}
	return jnode.Schema().Name
}

func (jnode *jsonNode) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	err := jnode.marshalJSON(&buffer, false)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (jnode *jsonNode) marshalJSON(buffer *bytes.Buffer, skipRootMarshalling bool) error {
	if jnode == nil || jnode.DataNode == nil {
		buffer.WriteString(`null`)
		return nil
	}
	switch {
	case jnode.IsBranchNode():
		cjnode := *jnode
		comma := false
		children := jnode.Children()
		if !skipRootMarshalling {
			buffer.WriteString(`{`)
		}
		for i := 0; i < len(children); {
			schema := children[i].Schema()
			if schema.IsListable() {
				var err error
				i, comma, err = jnode.marshalJSONListableNode(buffer, children, i, comma, skipRootMarshalling)
				if err != nil {
					return err
				}
				continue
			}
			// container, leaf, single leaf-list node
			if (jnode.ConfigOnly == yang.TSTrue && children[i].IsStateNode()) ||
				(jnode.ConfigOnly == yang.TSFalse && !children[i].IsStateNode() && !children[i].HasStateNode()) {
				// skip the node according to the retrieval option
				i++
				continue
			}
			cjnode.DataNode = children[i]
			cjnode.RFC7951S = jnode.RFC7951S
			if !skipRootMarshalling {
				if comma {
					buffer.WriteString(",")
				}
				comma = true
				buffer.WriteString(`"`)
				buffer.WriteString(cjnode.getQname()) // namespace-qualified name
				buffer.WriteString(`":`)
			}
			if err := cjnode.marshalJSON(buffer, false); err != nil {
				return err
			}
			i++
		}
		if !skipRootMarshalling {
			buffer.WriteString(`}`)
		}
	case jnode.HasMultipleValues(): // single leaf-list schema node
		schema := jnode.Schema()
		value := jnode.Values()
		rfc7951enabled := jnode.RFC7951S != RFC7951Disabled
		comma := false
		buffer.WriteString("[")
		for i := range value {
			if comma {
				buffer.WriteString(",")
			}
			b, err := schema.ValueToJSONBytes(schema.Type, value[i], rfc7951enabled)
			if err != nil {
				return err
			}
			buffer.Write(b)
			comma = true
		}
		buffer.WriteString("]")
		return nil
	case jnode.IsLeafNode(): // leaf, multiple leaf-list schema node
		rfc7951enabled := jnode.RFC7951S != RFC7951Disabled
		schema := jnode.Schema()
		b, err := schema.ValueToJSONBytes(schema.Type, jnode.Value(), rfc7951enabled)
		if err != nil {
			return err
		}
		buffer.Write(b)
	}
	return nil
}

func (parent *jsonNode) marshalJSONListableNode(buffer *bytes.Buffer, node []DataNode, i int, comma bool, skipRootMarshalling bool) (int, bool, error) {
	first := *parent
	first.DataNode = node[i]
	schema := first.Schema()
	switch first.ConfigOnly {
	case yang.TSTrue:
		if schema.IsState {
			for ; i < len(node); i++ {
				if schema != node[i].Schema() {
					return i, comma, nil
				}
			}
		}
	case yang.TSFalse: // stateOnly
		if !schema.IsState && !schema.HasState {
			for ; i < len(node); i++ {
				if schema != node[i].Schema() {
					return i, comma, nil
				}
			}
		}
	}
	if !skipRootMarshalling {
		if comma {
			buffer.WriteString(",")
		}
		comma = true
		buffer.WriteString(`"`)
		buffer.WriteString(first.getQname())
		buffer.WriteString(`":`)
	}
	// arrary format
	if first.RFC7951S != RFC7951Disabled || schema.IsDuplicatableList() || schema.IsLeafList() {
		ii := i
		for ; i < len(node); i++ {
			if schema != node[i].Schema() {
				break
			}
		}
		nodelist := make([]interface{}, 0, i-ii)
		for ; ii < i; ii++ {
			jnode := &jsonNode{DataNode: node[ii],
				ConfigOnly: first.ConfigOnly, RFC7951S: first.RFC7951S}
			nodelist = append(nodelist, jnode)
		}
		err := marshalJNodeTree(buffer, nodelist)
		return i, comma, err
	}

	// object format
	nodemap := map[string]interface{}{}
	for ; i < len(node); i++ {
		jnode := &jsonNode{DataNode: node[i],
			ConfigOnly: first.ConfigOnly, RFC7951S: first.RFC7951S}
		if schema != jnode.Schema() {
			break
		}
		keyname, keyval := GetKeyValues(jnode.DataNode)
		if len(keyname) != len(keyval) {
			return i, comma, fmt.Errorf("list %q doesn't have key value pairs", schema.Name)
		}
		m := nodemap
		for x := range keyval {
			if x < len(keyname)-1 {
				if n := m[keyval[x]]; n == nil {
					n := map[string]interface{}{}
					m[keyval[x]] = n
					m = n
				} else {
					m = n.(map[string]interface{})
				}
			} else {
				m[keyval[x]] = jnode
			}
		}
	}
	err := marshalJNodeTree(buffer, nodemap)
	return i, comma, err
}

func marshalJNodeTree(buffer *bytes.Buffer, jnodeTree interface{}) error {
	comma := false
	switch jj := jnodeTree.(type) {
	case map[string]interface{}:
		buffer.WriteString(`{`)
		k := make([]string, 0, len(jj))
		for key := range jj {
			k = append(k, key)
		}
		sort.Slice(k, func(i, j int) bool { return k[i] < k[j] })
		for i := range k {
			if comma {
				buffer.WriteString(",")
			}
			comma = true
			buffer.WriteString(`"`)
			buffer.WriteString(k[i])
			buffer.WriteString(`":`)
			if err := marshalJNodeTree(buffer, jj[k[i]]); err != nil {
				return err
			}
		}
		buffer.WriteString(`}`)
	case []interface{}:
		buffer.WriteString(`[`)
		for i := range jj {
			if comma {
				buffer.WriteString(",")
			}
			comma = true
			if err := marshalJNodeTree(buffer, jj[i]); err != nil {
				return err
			}
		}
		buffer.WriteString(`]`)
	case *jsonNode:
		if err := jj.marshalJSON(buffer, false); err != nil {
			return err
		}
	}
	return nil
}

// unmarshalJSONListNode decode jval to the list that has the keys.
func unmarshalJSONListNode(parent DataNode, cschema *SchemaNode, kname []string, kval []string, object interface{}) error {
	jobj, ok := object.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected json-val \"%v\" (%T) for %q", object, object, cschema.Name)
	}
	if len(kname) != len(kval) {
		for k, v := range jobj {
			kval = append(kval, k)
			err := unmarshalJSONListNode(parent, cschema, kname, kval, v)
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
	var child, found DataNode
	id := idBuilder.String()
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
	if err := unmarshalJSON(child, cschema, object); err != nil {
		if found == nil {
			parent.Delete(child)
		}
		return err
	}
	return nil
}

func unmarshalJSONListableNode(parent DataNode, cschema *SchemaNode, kname []string, arrary []interface{}) error {
	if cschema.IsLeafList() {
		for i := range arrary {
			child, err := NewDataNode(cschema)
			if err != nil {
				return err
			}
			if err = unmarshalJSON(child, cschema, arrary[i]); err != nil {
				return err
			}
			if _, err = parent.Insert(child, nil); err != nil {
				return err
			}
		}
		return nil
	}
	for i := range arrary {
		entry, ok := arrary[i].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected yaml value %T for %s", arrary[i], cschema.Name)
		}

		var err error
		var idBuilder strings.Builder
		idBuilder.WriteString(cschema.Name)
		for i := range kname {
			kvalue := entry[kname[i]]
			if kvalue == nil {
				kcschema := cschema.GetSchema(kname[i])
				qname, _ := kcschema.GetQName(true)
				if kvalue = entry[qname]; kvalue == nil {
					return fmt.Errorf("not found key data node %q from %v", kname[i], entry)
				}
			}
			idBuilder.WriteString("[")
			idBuilder.WriteString(kname[i])
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
			child, err = NewDataNodeByID(cschema, id)
			if err != nil {
				return err
			}
			if _, err = parent.Insert(child, nil); err != nil {
				return err
			}
		}
		if err := unmarshalJSON(child, cschema, arrary[i]); err != nil {
			if found == nil {
				parent.Delete(child)
			}
			return err
		}
	}
	return nil
}

func unmarshalJSON(node DataNode, schema *SchemaNode, jval interface{}) error {
	if node.IsBranchNode() {
		switch entry := jval.(type) {
		case map[string]interface{}:
			for k, v := range entry {
				cschema := schema.GetSchema(k)
				if cschema == nil {
					return fmt.Errorf("schema %q not found from %q", k, schema.Name)
				}
				switch {
				case cschema.IsListable():
					switch vv := v.(type) {
					case []interface{}:
						if err := unmarshalJSONListableNode(node, cschema, cschema.Keyname, vv); err != nil {
							return Error(EAppTagJSONParsing, err)
						}
					case map[string]interface{}:
						kname := cschema.Keyname
						kval := make([]string, 0, len(kname))
						if err := unmarshalJSONListNode(node, cschema, kname, kval, v); err != nil {
							return Error(EAppTagJSONParsing, err)
						}
					default:
						return Errorf(EAppTagJSONParsing, "unexpected json value %q for %q", vv, cschema.Name)
					}
				default:
					var err error
					if child := node.Get(k); child == nil {
						if child, err = NewDataNode(cschema); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
						if err = unmarshalJSON(child, cschema, v); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
						if _, err = node.Insert(child, nil); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
					} else {
						if err = unmarshalJSON(child, cschema, v); err != nil {
							return Error(EAppTagYAMLParsing, err)
						}
					}
				}
			}
			return nil
		case []interface{}:
			for i := range entry {
				if err := unmarshalJSON(node, schema, entry[i]); err != nil {
					return err
				}
			}
			return nil
		default:
			return fmt.Errorf("unexpected json value \"%v\" (%T) inserted for %q", jval, jval, node)
		}
	} else {
		switch entry := jval.(type) {
		case map[string]interface{}:
			return Errorf(EAppTagJSONParsing, "unexpected json value %q inserted for %q", entry, node.ID())
		case []interface{}:
			if len(entry) == 1 && entry[0] == nil { // empty type
				return Error(EAppTagJSONParsing, node.UnsetString(""))
			}
			if !node.HasMultipleValues() {
				return Errorf(EAppTagJSONParsing, "*unexpected json value %q inserted for %q", entry, node.ID())
			}
			for i := range entry {
				valstr, err := JSONValueToString(entry[i])
				if err != nil {
					return err
				}
				if err := node.SetString(valstr); err != nil {
					return Error(EAppTagJSONParsing, err)
				}
			}
			return nil
		default:
			valstr, err := JSONValueToString(jval)
			if err != nil {
				return err
			}
			return Error(EAppTagJSONParsing, node.SetString(valstr))
		}
	}
}

// MarshalJSON returns the JSON encoding of DataNode.
//
// Marshal traverses the value v recursively.
func MarshalJSON(node DataNode, option ...Option) ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jsonNode{DataNode: node}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v is not allowed for marshaling", option[i])
		case ConfigOnly:
			jnode.ConfigOnly = yang.TSTrue
		case StateOnly:
			jnode.ConfigOnly = yang.TSFalse
		case RFC7951Format:
			jnode.RFC7951S = RFC7951Enabled
		}
	}
	skipRootMarshalling := false
	if _, ok := node.(*DataNodeGroup); ok {
		skipRootMarshalling = true
	}
	err := jnode.marshalJSON(&buffer, skipRootMarshalling)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalJSONIndent is like Marshal but applies an indent and a prefix to format the output.
func MarshalJSONIndent(node DataNode, prefix, indent string, option ...Option) ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jsonNode{DataNode: node}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v is not allowed for marshaling", option[i])
		case ConfigOnly:
			jnode.ConfigOnly = yang.TSTrue
		case StateOnly:
			jnode.ConfigOnly = yang.TSFalse
		case RFC7951Format:
			jnode.RFC7951S = RFC7951Enabled
		}
	}
	skipRootMarshalling := false
	if _, ok := node.(*DataNodeGroup); ok {
		skipRootMarshalling = true
	}
	err := jnode.marshalJSON(&buffer, skipRootMarshalling)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, buffer.Bytes(), prefix, indent)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the data node.
func UnmarshalJSON(node DataNode, jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(node, node.Schema(), jval)
}

func isIntegral(val float64) bool {
	return val == float64(int(val))
}

// JSONValueToString() returns a string value from the json scalar value that is unmarshalled by json.Unmarshal()
func JSONValueToString(jval interface{}) (string, error) {
	switch jdata := jval.(type) {
	case float64:
		if isIntegral(jdata) {
			return fmt.Sprint(int64(jdata)), nil
		}
		return fmt.Sprint(jdata), nil
	case string:
		return jdata, nil
	case nil:
		return "", nil
	case bool:
		if jdata {
			return "true", nil
		}
		return "false", nil
	case []interface{}:
		if len(jdata) == 1 && jdata[0] == nil {
			return "true", nil
		}
	}
	return "", fmt.Errorf("unexpected json-value %v (%T)", jval, jval)
}
