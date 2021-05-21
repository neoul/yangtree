package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

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

func (jnode *jsonDataNode) MarshalJSON() ([]byte, error) {
	if jnode == nil || jnode.DataNode == nil {
		return []byte("null"), nil
	}
	switch datanode := jnode.DataNode.(type) {
	case *DataBranch:
		length := len(datanode.children)
		if length == 0 {
			return []byte("{}"), nil
		}
		node := datanode.children
		buffer := bytes.NewBufferString("{")
		for i := 0; i < length; {
			if IsList(node[i].Schema()) {
				var err error
				if jnode.rfc7951() != rfc7951Disabled {
					i, err = marshalListRFC7951(buffer, node, i, jnode.rfc7951(), jnode.config)
				} else {
					if IsDuplicatedList(node[i].Schema()) {
						i, err = marshalDuplicatedList(buffer, node, i, jnode.config)
					} else {
						i, err = marshalList(buffer, node, i, jnode.config)
					}
				}
				if err != nil {
					return nil, err
				}
				continue
			}
			var err error
			var jsonValue []byte
			var qname interface{} // namespace-qualified name
			cjnode := &jsonDataNode{DataNode: node[i], config: jnode.config}
			if cjnode.config == yang.TSTrue {
				if !IsConfig(cjnode.Schema()) {
					continue
				}
			} else if cjnode.config == yang.TSFalse {
				// if IsConfig(cjnode.Schema()) {
				// 	continue
				// }
			}
			// if !cjnode.Schema().IsDir() { // is leaf or leaflist
			// 	if IsConfig(cjnode.Schema()) {
			// 		continue
			// 	}
			// }
			switch jnode.rfc7951() {
			case rfc7951InProgress, rfc7951Enabled:
				cjnode.rfc7951s = rfc7951InProgress
				if jsonValue, err = json.Marshal(cjnode); err != nil {
					return nil, err
				}
				if qn, boundary := GetQName(cjnode.Schema()); boundary ||
					jnode.rfc7951() == rfc7951Enabled {
					qname = qn
				}
			default:
				if jsonValue, err = json.Marshal(cjnode); err != nil {
					return nil, err
				}
			}
			if qname != nil {
				buffer.WriteString("\"" + qname.(string) + "\":" + string(jsonValue))
			} else {
				buffer.WriteString("\"" + cjnode.Key() + "\":" + string(jsonValue))
			}
			if i < length-1 {
				buffer.WriteString(",")
			}
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
			valbyte, err := ValueToJSONValue(leaflist.schema, leaflist.schema.Type, leaflist.value[i], rfc7951enabled)
			if err != nil {
				return nil, err
			}
			b.Write(valbyte)
			if i < length-1 {
				b.WriteString(",")
			}
		}
		b.WriteString("]")
		return json.Marshal(leaflist.value)
	case *DataLeaf:
		leaf := datanode
		if leaf == nil {
			return nil, nil
		}
		rfc7951enabled := false
		if jnode.rfc7951() != rfc7951Disabled {
			rfc7951enabled = true
		}
		return ValueToJSONValue(leaf.schema, leaf.schema.Type, leaf.value, rfc7951enabled)
	}
	return nil, nil
}

func marshalList(buffer *bytes.Buffer, node []DataNode, i int, config yang.TriState) (int, error) {
	schema := node[i].Schema()
	keyname := strings.Split(schema.Key, " ")
	keynamelen := len(keyname)
	buffer.WriteString("\"" + schema.Name + "\":")
	keymap := map[string]interface{}{}
	length := len(node)
	for ; i < length; i++ {
		jnode := &jsonDataNode{DataNode: node[i], config: config}
		if schema != jnode.Schema() {
			break
		}
		keyval, err := ExtractKeyValues(keyname, jnode.Key())
		if err != nil {
			return 0, err
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
				m[keyval[x]] = jnode
			}
		}
	}
	jsonValue, err := json.Marshal(keymap)
	if err != nil {
		return i, err
	}
	buffer.WriteString(string(jsonValue))
	if i < length {
		buffer.WriteString(",")
	}
	return i, nil
}

func marshalDuplicatedList(buffer *bytes.Buffer, node []DataNode, i int, config yang.TriState) (int, error) {
	schema := node[i].Schema()
	buffer.WriteString("\"" + schema.Name + "\":")

	j := i
	length := len(node)
	for ; j < length; j++ {
		if schema != node[j].Schema() {
			break
		}
	}
	keylist := make([]*jsonDataNode, 0, j-i)
	for ; i < j; i++ {
		if schema != node[i].Schema() {
			break
		}
		jnode := &jsonDataNode{DataNode: node[i], config: config}
		keylist = append(keylist, jnode)
	}
	jsonValue, err := json.Marshal(keylist)
	if err != nil {
		return i, err
	}
	buffer.WriteString(string(jsonValue))
	if i < length {
		buffer.WriteString(",")
	}
	return j, nil
}

func marshalListRFC7951(buffer *bytes.Buffer, node []DataNode, i int, rfc7951 rfc7951s, config yang.TriState) (int, error) {
	schema := node[i].Schema()
	if qname, boundary := GetQName(schema); boundary || rfc7951 == rfc7951Enabled {
		buffer.WriteString("\"" + qname + "\":")
	} else {
		buffer.WriteString("\"" + schema.Name + "\":")
	}

	j := i
	length := len(node)
	for ; j < length; j++ {
		if schema != node[j].Schema() {
			break
		}
	}
	keylist := make([]*jsonDataNode, 0, j-i)
	for ; i < j; i++ {
		if schema != node[i].Schema() {
			break
		}
		jnode := &jsonDataNode{
			DataNode: node[i],
			config:   config,
			rfc7951s: rfc7951InProgress,
		}
		keylist = append(keylist, jnode)
	}
	jsonValue, err := json.Marshal(keylist)
	if err != nil {
		return i, err
	}
	buffer.WriteString(string(jsonValue))
	if i < length {
		buffer.WriteString(",")
	}
	return j, nil
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
						if err := n.unmarshalListRFC7951(cschema, ListKeyname(cschema), rfc7951StyleList); err != nil {
							return err
						}
					} else {
						if IsDuplicatedList(cschema) {
							return fmt.Errorf("non-key list '%s' must have the array format of RFC7951", cschema.Name)
						}
						kname := ListKeyname(cschema)
						kval := make([]string, 0, len(kname))
						if err := n.unmarshalList(cschema, kname, kval, v); err != nil {
							return err
						}
					}
				default:
					var err error
					var child DataNode
					i, _ := n.Index(k)
					if i < len(n.children) && n.children[i].Key() == k {
						child = n.children[i]
					} else {
						if child, err = n.New(k); err != nil {
							return err
						}
					}
					if err := unmarshalJSON(child, v); err != nil {
						return err
					}
				}
			}
			return nil
		default:
			return fmt.Errorf("unexpected json '%v' inserted for %s", jdata, n)
		}
	case *DataLeafList:
		if islice, ok := jval.([]interface{}); ok {
			for i := range islice {
				valstr, err := JSONValueToString(islice[i])
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

func (branch *DataBranch) UnmarshalJSON(jsonbyte []byte) error {
	var jval interface{}
	err := json.Unmarshal(jsonbyte, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(branch, jval)
}

func (leaf *DataLeaf) UnmarshalJSON(jsonbyte []byte) error {
	var jval interface{}
	err := json.Unmarshal(jsonbyte, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(leaf, jval)
}

func (leaflist *DataLeafList) UnmarshalJSON(jsonbyte []byte) error {
	var jval interface{}
	err := json.Unmarshal(jsonbyte, &jval)
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
		case FindConfig:
			jnode.config = yang.TSTrue
		case FindState:
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
		case FindConfig:
			jnode.config = yang.TSTrue
		case FindState:
			jnode.config = yang.TSFalse
		case RFC7951Format:
			jnode.rfc7951s = rfc7951Enabled
		}
	}
	return json.MarshalIndent(jnode, prefix, indent)
}

func UnmarshalJSON(node DataNode, jsonbyte []byte) error {
	var jval interface{}
	err := json.Unmarshal(jsonbyte, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(node, jval)
}
