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
	printMeta  bool
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
	_, err := jnode.marshalJSON(&buffer, false, false, false)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (jnode *jsonNode) marshalJSONMetadata(buffer *bytes.Buffer, comma bool) (bool, error) {
	// marshalling metadata
	var err error
	m := jnode.Metadata()
	if len(m) == 0 {
		return comma, nil
	}
	if comma {
		buffer.WriteString(",")
	}
	comma = true

	switch {
	case jnode.IsBranchNode():
		buffer.WriteString(`"@":{`)
	case jnode.HasMultipleValues(): // single leaf-list schema node
		if jnode.RFC7951S == RFC7951Disabled {
			buffer.WriteString(fmt.Sprintf(`"@%s":{`, jnode.getQname()))
			break
		}
		buffer.WriteString(fmt.Sprintf(`"@%s":[`, jnode.getQname()))
		mcomma := false
		length := len(jnode.Values())
		for i := 0; i < length; i++ {
			if mcomma {
				buffer.WriteString(",")
			}
			mcomma = true
			buffer.WriteString(`{`)
			mmcomma := false
			mjnode := *jnode
			for _, mdata := range m {
				mjnode.DataNode = mdata
				mjnode.RFC7951S = jnode.RFC7951S
				if mmcomma, err = mjnode.marshalJSON(buffer, mmcomma, true, false); err != nil {
					return false, err
				}
			}
			buffer.WriteString(`}`)
		}
		buffer.WriteString(`]`)
		return comma, nil
	case jnode.IsLeafNode(): // leaf, multiple leaf-list schema node
		buffer.WriteString(fmt.Sprintf(`"@%s":{`, jnode.getQname()))
	default:
		return false, fmt.Errorf("unknown ynode type %v", jnode.DataNode)
	}
	mcomma := false
	mjnode := *jnode
	for _, mdata := range m {
		mjnode.DataNode = mdata
		mjnode.RFC7951S = jnode.RFC7951S
		if mcomma, err = mjnode.marshalJSON(buffer, mcomma, true, false); err != nil {
			return false, err
		}
	}
	buffer.WriteString(`}`)
	return comma, err
}

func (jnode *jsonNode) marshalJSON(buffer *bytes.Buffer, comma, printName, skipRootMarshalling bool) (bool, error) {
	var err error
	if !skipRootMarshalling {
		if printName {
			if comma {
				buffer.WriteString(",")
			}
			comma = true
			buffer.WriteString(`"`)
			buffer.WriteString(jnode.getQname()) // namespace-qualified name
			buffer.WriteString(`":`)
		}
	}

	if jnode == nil || jnode.DataNode == nil {
		buffer.WriteString(`null`)
		return comma, nil
	}
	switch {
	case jnode.IsBranchNode():
		cjnode := *jnode
		childcomma := false
		children := jnode.Children()
		if !skipRootMarshalling {
			buffer.WriteString(`{`)
		}
		for i := 0; i < len(children); {
			schema := children[i].Schema()
			if schema.IsListable() {
				var err error
				i, childcomma, err = jnode.marshalJSONListableNode(buffer, children, i, childcomma, skipRootMarshalling)
				if err != nil {
					return childcomma, err
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
			if childcomma, err = cjnode.marshalJSON(buffer, childcomma, true, false); err != nil {
				return comma, err
			}
			i++
		}

		if !skipRootMarshalling {
			// marshalling metadata
			if jnode.printMeta {
				_, err = jnode.marshalJSONMetadata(buffer, childcomma)
				if err != nil {
					return false, err
				}
			}
			buffer.WriteString(`}`)
		}
	case jnode.HasMultipleValues(): // single leaf-list schema node
		schema := jnode.Schema()
		value := jnode.Values()
		valcomma := false
		buffer.WriteString("[")
		for i := range value {
			if valcomma {
				buffer.WriteString(",")
			}
			b, err := schema.ValueToJSONBytes(schema.Type, value[i], jnode.RFC7951S != RFC7951Disabled)
			if err != nil {
				return false, err
			}
			buffer.Write(b)
			valcomma = true
		}
		buffer.WriteString("]")
		// marshalling metadata
		if jnode.printMeta && !skipRootMarshalling {
			comma, err = jnode.marshalJSONMetadata(buffer, comma)
			if err != nil {
				return comma, err
			}
		}
	case jnode.IsLeafNode(): // leaf, multiple leaf-list schema node
		schema := jnode.Schema()
		b, err := schema.ValueToJSONBytes(schema.Type, jnode.Value(), jnode.RFC7951S != RFC7951Disabled)
		if err != nil {
			return comma, err
		}
		buffer.Write(b)
		// marshalling metadata
		if jnode.printMeta && !skipRootMarshalling {
			comma, err = jnode.marshalJSONMetadata(buffer, comma)
			if err != nil {
				return comma, err
			}
		}
	}
	return comma, nil
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
		printMeta := first.printMeta
		if schema.IsLeafList() {
			printMeta = false
		}
		nodelist := make([]interface{}, 0, i-ii)
		for ; ii < i; ii++ {
			jnode := &jsonNode{DataNode: node[ii], ConfigOnly: first.ConfigOnly,
				RFC7951S: first.RFC7951S, printMeta: printMeta}
			nodelist = append(nodelist, jnode)
		}
		err := marshalJNodeTree(buffer, nodelist)
		if err == nil {
			// marshalling metadata of a leaf-list
			if first.printMeta && schema.IsLeafList() {
				if !skipRootMarshalling {
					if comma {
						buffer.WriteString(",")
					}
					comma = true
					buffer.WriteString(`"`)
					buffer.WriteString(`@`)
					buffer.WriteString(first.getQname())
					buffer.WriteString(`":[`)
				}
				mcomma := false
				for j := 0; j < len(nodelist); j++ {
					m := nodelist[j].(*jsonNode).Metadata()
					if mcomma {
						buffer.WriteString(`,`)
					}
					mcomma = true
					if len(m) == 0 {
						buffer.WriteString("null")
						continue
					}
					mmcomma := false
					mjnode := first
					buffer.WriteString(`{`)
					for _, mdata := range m {
						mjnode.DataNode = mdata
						mjnode.RFC7951S = first.RFC7951S
						if mmcomma, err = mjnode.marshalJSON(buffer, mmcomma, true, false); err != nil {
							return i, comma, err
						}
					}
					buffer.WriteString(`}`)
				}
				if !skipRootMarshalling {
					buffer.WriteString(`]`)
				}
			}
		}
		return i, comma, err
	}

	// object format
	nodemap := map[string]interface{}{}
	for ; i < len(node); i++ {
		jnode := &jsonNode{DataNode: node[i], ConfigOnly: first.ConfigOnly,
			RFC7951S: first.RFC7951S, printMeta: first.printMeta}
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
		if _, err := jj.marshalJSON(buffer, false, false, false); err != nil {
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
		idBuilder.WriteString(ValueToValueString(kval[i]))
		idBuilder.WriteString("]")
	}
	var child, found DataNode
	id := idBuilder.String()
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
	if err := unmarshalJSON(child, cschema, object); err != nil {
		if found == nil {
			parent.Delete(child)
		}
		return err
	}
	return nil
}

func unmarshalJSONListableNode(parent DataNode, cschema *SchemaNode, kname []string, arrary, metadata []interface{}) error {
	if cschema.IsLeafList() {
		for i := range arrary {
			child, err := New(cschema)
			if err != nil {
				return err
			}
			if err = unmarshalJSON(child, cschema, arrary[i]); err != nil {
				return err
			}
			// proccessing metadata
			if i < len(metadata) {
				msrc, ok := metadata[i].(map[string]interface{})
				if !ok {
					return Errorf(EAppTagJSONParsing, "invalid metadata format inserted for %q", cschema)
				}
				for k, v := range msrc {
					vstr, err := JSONValueToString(v)
					if err != nil {
						return Error(EAppTagJSONParsing, err)
					}
					if err := child.SetMetadataString(k, vstr); err != nil {
						return Error(EAppTagJSONParsing, err)
					}
				}
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
			child, err = NewWithID(cschema, id)
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

func unmarshalJSONMeta(node DataNode, metakey string, jval interface{}) error {
	msrc, ok := jval.(map[string]interface{})
	if !ok {
		if msrc, ok := jval.([]interface{}); ok {
			if len(msrc) > 0 {
				return unmarshalJSONMeta(node, metakey, msrc[len(msrc)-1])
			}
		}
		return Errorf(EAppTagJSONParsing, "invalid metadata format for %q", node)
	}
	for k, v := range msrc {
		vstr, err := JSONValueToString(v)
		if err != nil {
			return Error(EAppTagJSONParsing, err)
		}
		if metakey == "@" {
			if err := node.SetMetadataString(k, vstr); err != nil {
				return Error(EAppTagJSONParsing, err)
			}
		} else {
			nodes := node.GetAll(metakey[1:])
			for j := range nodes {
				if err := nodes[j].SetMetadataString(k, vstr); err != nil {
					return Error(EAppTagJSONParsing, err)
				}
			}
		}
	}
	return nil
}

func unmarshalJSON(node DataNode, schema *SchemaNode, jval interface{}) error {
	if node.IsBranchNode() {
		switch entry := jval.(type) {
		case map[string]interface{}:
			var metakey []string
			for k, v := range entry {
				if strings.HasPrefix(k, "@") {
					metakey = append(metakey, k)
					continue
				}
				cschema := schema.GetSchema(k)
				if cschema == nil {
					return fmt.Errorf("schema %q not found from %q", k, schema.Name)
				}
				switch {
				case cschema.IsListable():
					switch vv := v.(type) {
					case []interface{}:
						// processing metadata
						var metadata []interface{}
						if cschema.IsLeafList() {
							if _meta, ok := entry["@"+k]; ok {
								if _meta, ok := _meta.([]interface{}); ok {
									metadata = _meta
								}
							}
						}
						if err := unmarshalJSONListableNode(node, cschema, cschema.Keyname, vv, metadata); err != nil {
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
						if child, err = New(cschema); err != nil {
							return Error(EAppTagJSONParsing, err)
						}
						if err = unmarshalJSON(child, cschema, v); err != nil {
							return Error(EAppTagJSONParsing, err)
						}
						if _, err = node.Insert(child, nil); err != nil {
							return Error(EAppTagJSONParsing, err)
						}
					} else {
						if err = unmarshalJSON(child, cschema, v); err != nil {
							return Error(EAppTagJSONParsing, err)
						}
					}
				}
			}
			// unmarshalling metadata
			for i := range metakey {
				unmarshalJSONMeta(node, metakey[i], entry[metakey[i]])
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
				return Error(EAppTagJSONParsing, node.UnsetValueString(""))
			}
			if !node.HasMultipleValues() {
				return Errorf(EAppTagJSONParsing, "*unexpected json value %q inserted for %q", entry, node.ID())
			}
			for i := range entry {
				valstr, err := JSONValueToString(entry[i])
				if err != nil {
					return err
				}
				if err := node.SetValueString(valstr); err != nil {
					return Error(EAppTagJSONParsing, err)
				}
			}
			return nil
		default:
			valstr, err := JSONValueToString(jval)
			if err != nil {
				return err
			}
			return Error(EAppTagJSONParsing, node.SetValueString(valstr))
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
		case Metadata:
			jnode.printMeta = true
		}
	}
	skipRootMarshalling := false
	if _, ok := node.(*DataNodeGroup); ok {
		skipRootMarshalling = true
	}
	_, err := jnode.marshalJSON(&buffer, false, false, skipRootMarshalling)
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
		case Metadata:
			jnode.printMeta = true
		}
	}
	skipRootMarshalling := false
	if _, ok := node.(*DataNodeGroup); ok {
		skipRootMarshalling = true
	}
	_, err := jnode.marshalJSON(&buffer, false, false, skipRootMarshalling)
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
