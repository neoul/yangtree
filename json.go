package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

type RFC7951Format struct{}

func (f RFC7951Format) IsOption() {}

// rfc7951s (rfc7951 processing status)
type rfc7951s int

const (
	// rfc7951disabled - rfc7951 encoding disabled
	rfc7951Disabled rfc7951s = iota
	// rfc7951InProgress - rfc7951 encoding is in progress
	rfc7951InProgress
	// rfc7951enabled - rfc7951 encoding enabled
	rfc7951Enabled
)

type jsonDataNode struct {
	DataNode
	rfc7951s
	config yang.TriState
}

func (jnode *jsonDataNode) rfc7951() rfc7951s {
	if jnode == nil {
		return rfc7951Disabled
	}
	return jnode.rfc7951s
}

func isEmptyJSONBytes(s string) bool {
	for _, r := range s {
		switch r {
		case '[', ']', '{', '}', ',', ' ', '\t', '\n', '\r':
		default:
			return false
		}
	}
	return true
}

func (jnode *jsonDataNode) marshalJSON() ([]byte, error) {
	var err error
	var jbytes []byte
	if jnode.config == yang.TSFalse { // state retrieval
		if IsConfig(jnode.Schema()) {
			if !jnode.Schema().IsDir() {
				return nil, nil
			}
			jbytes, err = json.Marshal(jnode)
			if err != nil {
				return nil, err
			}
			if isEmptyJSONBytes(string(jbytes)) {
				return nil, nil
			}
			return jbytes, err
		}
	} else { // config or all node retrieval
		if jnode.config == yang.TSTrue { // config retrieval
			if !IsConfig(jnode.Schema()) {
				return nil, nil
			}
		}
	}
	return json.Marshal(jnode)
}

func (jnode *jsonDataNode) MarshalJSON() ([]byte, error) {
	if jnode == nil || jnode.DataNode == nil {
		return []byte("null"), nil
	}
	switch datanode := jnode.DataNode.(type) {
	case *DataBranch:
		comma := false
		node := datanode.children
		length := len(datanode.children)
		buffer := bytes.NewBufferString("{")
		for i := 0; i < length; {
			if IsList(node[i].Schema()) {
				var err error
				if jnode.rfc7951() != rfc7951Disabled {
					i, comma, err = marshalListRFC7951(buffer, node, i, comma, jnode.rfc7951(), jnode.config)
				} else {
					if IsDuplicatedList(node[i].Schema()) {
						i, comma, err = marshalDuplicatedList(buffer, node, i, comma, jnode.config)
					} else {
						i, comma, err = marshalList(buffer, node, i, comma, jnode.config)
					}
				}
				if err != nil {
					return nil, err
				}
				continue
			}
			// container, leaf or leaflist
			var err error
			var jbytes []byte
			var qname interface{} // namespace-qualified name
			cjnode := &jsonDataNode{DataNode: node[i], config: jnode.config}
			switch jnode.rfc7951() {
			case rfc7951InProgress, rfc7951Enabled:
				cjnode.rfc7951s = rfc7951InProgress
				if qn, boundary := GetQName(cjnode.Schema()); boundary ||
					jnode.rfc7951() == rfc7951Enabled {
					qname = qn
				}
			}
			if jbytes, err = cjnode.marshalJSON(); err != nil {
				return nil, err
			}
			if jbytes == nil {
				i++
				continue
			}
			if comma {
				buffer.WriteString(",")
			}
			if qname != nil {
				buffer.WriteString("\"" + qname.(string) + "\":" + string(jbytes))
			} else {
				buffer.WriteString("\"" + cjnode.Key() + "\":" + string(jbytes))
			}
			comma = true
			i++
		}
		buffer.WriteString("}")
		return buffer.Bytes(), nil
	case *DataLeafList:
		leaflist := datanode
		if leaflist == nil {
			return nil, nil
		}
		rfc7951enabled := false
		if jnode.rfc7951() != rfc7951Disabled {
			rfc7951enabled = true
		}
		var b bytes.Buffer
		b.WriteString("[")
		length := len(leaflist.value)
		for i := 0; i < length; i++ {
			valbyte, err := ValueToJSONBytes(leaflist.schema, leaflist.schema.Type, leaflist.value[i], rfc7951enabled)
			if err != nil {
				return nil, err
			}
			b.Write(valbyte)
			if i < length-1 {
				b.WriteString(",")
			}
		}
		b.WriteString("]")
		return b.Bytes(), nil
	case *DataLeaf:
		leaf := datanode
		if leaf == nil {
			return nil, nil
		}
		rfc7951enabled := false
		if jnode.rfc7951() != rfc7951Disabled {
			rfc7951enabled = true
		}
		return ValueToJSONBytes(leaf.schema, leaf.schema.Type, leaf.value, rfc7951enabled)
	}
	return nil, nil
}

type jsonbytes struct {
	bytes []byte
}

func (mjb *jsonbytes) MarshalJSON() ([]byte, error) {
	return mjb.bytes, nil
}

func marshalList(buffer *bytes.Buffer, node []DataNode, i int, comma bool, config yang.TriState) (int, bool, error) {
	schema := node[i].Schema()
	keyname := GetKeynames(schema)
	keynamelen := len(keyname)
	keymap := map[string]interface{}{}
	length := len(node)
	for ; i < length; i++ {
		jnode := &jsonDataNode{DataNode: node[i], config: config}
		if schema != jnode.Schema() {
			break
		}
		jbytes, err := jnode.marshalJSON()
		if err != nil {
			return i, comma, err
		}
		if jbytes == nil {
			continue
		}
		keyval, err := ExtractKeyValues(keyname, jnode.Key())
		if err != nil {
			return i, comma, err
		}
		m := keymap
		for x := range keyval {
			if x < keynamelen-1 {
				if n := m[keyval[x]]; n == nil {
					n := map[string]interface{}{}
					m[keyval[x]] = n
					m = n
				} else {
					m = n.(map[string]interface{})
				}
			} else {
				m[keyval[x]] = &jsonbytes{bytes: jbytes}
			}
		}
	}
	if config == yang.TSFalse { // state retrieval
		if IsConfig(schema) {
			if len(keymap) == 0 {
				return i, comma, nil
			}
		}
	} else { // config or all node retrieval
		if config == yang.TSTrue { // config retrieval
			if !IsConfig(schema) {
				return i, comma, nil
			}
		}
	}
	jbytes, err := json.Marshal(keymap)
	if err != nil {
		return i, comma, err
	}
	if comma {
		buffer.WriteString(",")
	}
	buffer.WriteString("\"" + schema.Name + "\":")
	buffer.WriteString(string(jbytes))
	comma = true
	return i, comma, nil
}

func marshalDuplicatedList(buffer *bytes.Buffer, node []DataNode, i int, comma bool, config yang.TriState) (int, bool, error) {
	schema := node[i].Schema()

	j := i
	length := len(node)
	for ; j < length; j++ {
		if schema != node[j].Schema() {
			break
		}
	}
	keylist := make([]*jsonbytes, 0, j-i)
	for ; i < j; i++ {
		if schema != node[i].Schema() {
			break
		}
		jnode := &jsonDataNode{DataNode: node[i], config: config}
		jbytes, err := jnode.marshalJSON()
		if err != nil {
			return i, comma, err
		}
		if jbytes == nil {
			continue
		}
		keylist = append(keylist, &jsonbytes{bytes: jbytes})
	}
	if config == yang.TSFalse { // state retrieval
		if IsConfig(schema) {
			if len(keylist) == 0 {
				return j, comma, nil
			}
		}
	} else { // config or all node retrieval
		if config == yang.TSTrue { // config retrieval
			if !IsConfig(schema) {
				return j, comma, nil
			}
		}
	}
	jsonValue, err := json.Marshal(keylist)
	if err != nil {
		return j, comma, err
	}
	if comma {
		buffer.WriteString(",")
	}
	buffer.WriteString("\"" + schema.Name + "\":")
	buffer.WriteString(string(jsonValue))
	comma = true
	return j, comma, nil
}

func marshalListRFC7951(buffer *bytes.Buffer, node []DataNode, i int, comma bool, rfc7951 rfc7951s, config yang.TriState) (int, bool, error) {
	schema := node[i].Schema()
	j := i
	length := len(node)
	for ; j < length; j++ {
		if schema != node[j].Schema() {
			break
		}
	}
	keylist := make([]*jsonbytes, 0, j-i)
	for ; i < j; i++ {
		if schema != node[i].Schema() {
			break
		}
		jnode := &jsonDataNode{
			DataNode: node[i],
			config:   config,
			rfc7951s: rfc7951InProgress,
		}
		jbytes, err := jnode.marshalJSON()
		if err != nil {
			return i, comma, err
		}
		if jbytes == nil {
			continue
		}
		keylist = append(keylist, &jsonbytes{bytes: jbytes})
	}
	if config == yang.TSFalse { // state retrieval
		if IsConfig(schema) {
			if len(keylist) == 0 {
				return j, comma, nil
			}
		}
	} else { // config or all node retrieval
		if config == yang.TSTrue { // config retrieval
			if !IsConfig(schema) {
				return j, comma, nil
			}
		}
	}
	jbytes, err := json.Marshal(keylist)
	if err != nil {
		return j, comma, err
	}
	if comma {
		buffer.WriteString(",")
	}
	if qname, boundary := GetQName(schema); boundary || rfc7951 == rfc7951Enabled {
		buffer.WriteString("\"" + qname + "\":")
	} else {
		buffer.WriteString("\"" + schema.Name + "\":")
	}
	buffer.WriteString(string(jbytes))
	comma = true
	return j, comma, nil
}

func (branch *DataBranch) MarshalJSON() ([]byte, error) {
	jnode := &jsonDataNode{DataNode: branch, rfc7951s: rfc7951Disabled}
	return jnode.MarshalJSON()
}

func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	jnode := &jsonDataNode{DataNode: leaf, rfc7951s: rfc7951Disabled}
	return jnode.MarshalJSON()
}

func (leaflist *DataLeafList) MarshalJSON() ([]byte, error) {
	jnode := &jsonDataNode{DataNode: leaflist, rfc7951s: rfc7951Disabled}
	return jnode.MarshalJSON()
}

func (branch *DataBranch) MarshalJSON_IETF() ([]byte, error) {
	jnode := &jsonDataNode{DataNode: branch, rfc7951s: rfc7951Enabled}
	return jnode.MarshalJSON()
}

func (leaf *DataLeaf) MarshalJSON_IETF() ([]byte, error) {
	jnode := &jsonDataNode{DataNode: leaf, rfc7951s: rfc7951Enabled}
	return jnode.MarshalJSON()
}

func (leaflist *DataLeafList) MarshalJSON_IETF() ([]byte, error) {
	jnode := &jsonDataNode{DataNode: leaflist, rfc7951s: rfc7951Enabled}
	return jnode.MarshalJSON()
}

// unmarshalList decode jval to the list that has the keys.
func (branch *DataBranch) unmarshalList(cschema *yang.Entry, kname []string, kval []string, jval interface{}) error {
	jdata, ok := jval.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected json type '%T' for %s", jval, cschema.Name)
	}
	if len(kname) != len(kval) {
		for k, v := range jdata {
			kval = append(kval, k)
			err := branch.unmarshalList(cschema, kname, kval, v)
			if err != nil {
				return err
			}
			kval = kval[:len(kval)-1]
		}
		return nil
	}
	// check existent DataNode
	var err error
	var key string
	for i := range kval {
		key = key + "[" + kname[i] + "=" + kval[i] + "]"
	}
	key = cschema.Name + key
	var child DataNode
	found := branch.Get(key)
	if len(found) == 0 {
		if child, err = branch.New(key); err != nil {
			return err
		}
	} else {
		child = found[0]
	}
	// Update DataNode
	return unmarshalJSON(child, jval)
}

func (branch *DataBranch) unmarshalListRFC7951(cschema *yang.Entry, kname []string, listentry []interface{}) error {
	for i := range listentry {
		jentry, ok := listentry[i].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected json type '%T' for %s", listentry[i], cschema.Name)
		}
		// check existent DataNode
		var err error
		var key string
		for i := range kname {
			found := GetSchema(cschema, kname[i])
			if found == nil {
				return fmt.Errorf("schema '%s' not found", kname[i])
			}
			kval := fmt.Sprint(jentry[kname[i]])
			// [FIXME] need to check key validation
			// kchild, err := New(kschema, kval)
			// if err != nil {
			// 	return err
			// }
			key = key + "[" + kname[i] + "=" + kval + "]"
		}
		key = cschema.Name + key
		var child DataNode
		if IsDuplicatedList(cschema) {
			if child, err = branch.New(key); err != nil {
				return err
			}
		} else {
			found := branch.Get(key)
			if len(found) == 0 {
				if child, err = branch.New(key); err != nil {
					return err
				}
			} else {
				child = found[0]
			}
		}

		// Update DataNode
		if err := unmarshalJSON(child, jentry); err != nil {
			return err
		}
	}
	return nil
}

func unmarshalJSON(node DataNode, jval interface{}) error {
	switch n := node.(type) {
	case *DataBranch:
		switch jdata := jval.(type) {
		case map[string]interface{}:
			for k, v := range jdata {
				cschema := GetSchema(n.schema, k)
				if cschema == nil {
					return fmt.Errorf("schema.%s not found from schema.%s", k, n.schema.Name)
				}
				switch {
				case IsList(cschema):
					if rfc7951StyleList, ok := v.([]interface{}); ok {
						if err := n.unmarshalListRFC7951(cschema, GetKeynames(cschema), rfc7951StyleList); err != nil {
							return err
						}
					} else {
						if IsDuplicatedList(cschema) {
							return fmt.Errorf("non-key list '%s' must have the array format of RFC7951", cschema.Name)
						}
						kname := GetKeynames(cschema)
						kval := make([]string, 0, len(kname))
						if err := n.unmarshalList(cschema, kname, kval, v); err != nil {
							return err
						}
					}
				default:
					var child DataNode
					i, _ := n.Index(k)
					if i < len(n.children) && n.children[i].Key() == k {
						child = n.children[i]
						if err := unmarshalJSON(child, v); err != nil {
							return err
						}
					} else {
						child, err := New(cschema)
						if err != nil {
							return err
						}
						if err := unmarshalJSON(child, v); err != nil {
							return err
						}
						if err := n.Insert(child); err != nil {
							return err
						}
					}

				}
			}
			return nil
		}
		return fmt.Errorf("unexpected json '%v' inserted for %s", jval, n)
	case *DataLeafList:
		if vslice, ok := jval.([]interface{}); ok {
			for i := range vslice {
				valstr, err := JSONValueToString(vslice[i])
				if err != nil {
					return err
				}
				if err = n.Set(valstr); err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("unexpected json value %q for %s", jval, n)
	case *DataLeaf:
		valstr, err := JSONValueToString(jval)
		if err != nil {
			return err
		}
		return n.Set(valstr)
	default:
		return fmt.Errorf("unknown data node type '%T'", node)
	}
}

func (branch *DataBranch) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(branch, jval)
}

func (leaf *DataLeaf) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(leaf, jval)
}

func (leaflist *DataLeafList) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(leaflist, jval)
}

// MarshalJSON returns the JSON encoding of DataNode.
//
// Marshal traverses the value v recursively.
func MarshalJSON(node DataNode, option ...Option) ([]byte, error) {
	jnode := &jsonDataNode{DataNode: node}
	for i := range option {
		switch option[i].(type) {
		case ConfigOnly:
			jnode.config = yang.TSTrue
		case StateOnly:
			jnode.config = yang.TSFalse
		case RFC7951Format:
			jnode.rfc7951s = rfc7951Enabled
		}
	}
	return jnode.MarshalJSON()
}

// MarshalJSON_IETF is like Marshal but applies Indent to format the output.
func MarshalJSONIndent(node DataNode, prefix, indent string, option ...Option) ([]byte, error) {
	jnode := &jsonDataNode{DataNode: node}
	for i := range option {
		switch option[i].(type) {
		case ConfigOnly:
			jnode.config = yang.TSTrue
		case StateOnly:
			jnode.config = yang.TSFalse
		case RFC7951Format:
			jnode.rfc7951s = rfc7951Enabled
		}
	}
	return json.MarshalIndent(jnode, prefix, indent)
}

func UnmarshalJSON(node DataNode, jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(node, jval)
}
