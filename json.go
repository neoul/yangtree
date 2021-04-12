package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

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

func marshalList(buffer *bytes.Buffer, node []DataNode, i int, length int) (int, error) {
	schema := node[i].Schema()
	keyname := strings.Split(schema.Key, " ")
	keynamelen := len(keyname)
	buffer.WriteString("\"" + schema.Name + "\":")
	keymap := map[string]interface{}{}
	for ; i < length; i++ {
		if schema != node[i].Schema() {
			break
		}
		keyval, err := ExtractKeyValues(keyname, node[i].Key())
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
				m[keyval[x]] = node[i]
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

func marshalListRFC7951(buffer *bytes.Buffer, node []DataNode, i int, length int, rfc7951 rfc7951s) (int, error) {
	schema := node[i].Schema()
	var qname interface{} // namespace-qualified name required
	switch rfc7951 {
	case rfc7951InProgress:
		qname = GetAnnotation(schema, "qname-boundary")
	case rfc7951Enabled:
		qname = GetAnnotation(schema, "qname")
	default:
	}
	if qname != nil {
		buffer.WriteString("\"" + qname.(string) + "\":")
	} else {
		buffer.WriteString("\"" + schema.Name + "\":")
	}

	j := i
	for ; j < length; j++ {
		if schema != node[j].Schema() {
			break
		}
	}
	keylist := make([]rfc7951DataNode, 0, j-i)
	for ; i < j; i++ {
		if schema != node[i].Schema() {
			break
		}
		keylist = append(keylist, rfc7951DataNode{node[i]})
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

func (branch *DataBranch) marshalJSON(rfc7951 rfc7951s) ([]byte, error) {
	if branch == nil {
		return []byte("null"), nil
	}
	length := len(branch.Children)
	if length == 0 {
		// FIXME - Which should be returned? nil or empty object?
		return []byte("{}"), nil
	}
	buffer := bytes.NewBufferString("{")
	node := make([]DataNode, 0, length)
	for _, c := range branch.Children {
		node = append(node, c)
	}
	sort.Slice(node, func(i, j int) bool {
		return node[i].Key() < node[j].Key()
	})
	for i := 0; i < length; {
		if node[i].Schema().IsList() {
			var err error
			if rfc7951 != rfc7951Disabled {
				i, err = marshalListRFC7951(buffer, node, i, length, rfc7951)
			} else {
				i, err = marshalList(buffer, node, i, length)
			}
			if err != nil {
				return nil, err
			}
			continue
		}
		var err error
		var jsonValue []byte
		var qname interface{} // namespace-qualified name required
		switch rfc7951 {
		case rfc7951InProgress:
			if jsonValue, err = json.Marshal(rfc7951DataNode{node[i]}); err != nil {
				return nil, err
			}
			qname = GetAnnotation(node[i].Schema(), "qname-boundary")
		case rfc7951Enabled:
			if jsonValue, err = json.Marshal(rfc7951DataNode{node[i]}); err != nil {
				return nil, err
			}
			qname = GetAnnotation(node[i].Schema(), "qname")
		default:
			if jsonValue, err = json.Marshal(node[i]); err != nil {
				return nil, err
			}
		}
		if qname != nil {
			buffer.WriteString("\"" + qname.(string) + "\":" + string(jsonValue))
		} else {
			buffer.WriteString("\"" + node[i].Key() + "\":" + string(jsonValue))
		}
		if i < length-1 {
			buffer.WriteString(",")
		}
		i++
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func (leaf *DataLeaf) marshalJSON(rfc7951 rfc7951s) ([]byte, error) {
	if leaf == nil {
		return nil, nil
	}
	prefix := false
	if rfc7951 != rfc7951Disabled {
		prefix = true
	}
	return encodeToJSONValue(leaf.schema, leaf.schema.Type, leaf.Value, prefix)
}

func (leaflist *DataLeafList) marshalJSON(rfc7951 rfc7951s) ([]byte, error) {
	if leaflist == nil {
		return nil, nil
	}
	// [FIXME] - need json encoding for each entry of leaflist
	return json.Marshal(leaflist.Value)
}

func (branch *DataBranch) MarshalJSON() ([]byte, error) {
	return branch.marshalJSON(rfc7951Disabled)
}

func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	return leaf.marshalJSON(rfc7951Disabled)
}

func (leaflist *DataLeafList) MarshalJSON() ([]byte, error) {
	return leaflist.marshalJSON(rfc7951Disabled)
}

func (branch *DataBranch) unmarshalList(cschema *yang.Entry, kname []string, kval []string, jval interface{}) error {
	jdata, ok := jval.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected json type %T", jval)
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
	child := branch.Get(key)
	if child == nil {
		if child, err = New(cschema); err != nil {
			return err
		}
		if err = branch.Insert(key, child); err != nil {
			return err
		}
	}
	// Update DataNode
	return child.unmarshalJSON(jval)
}

func (branch *DataBranch) unmarshalListRFC7951(cschema *yang.Entry, kname []string, listentry []interface{}) error {
	for i := range listentry {
		jentry, ok := listentry[i].(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected json type %T", listentry[i])
		}
		// check existent DataNode
		var err error
		var key string
		for i := range kname {
			_, err := GetSchema(cschema, kname[i])
			if err != nil {
				return err
			}
			kval := fmt.Sprint(jentry[kname[i]])
			// kchild, err := New(kschema, kval)
			// if err != nil {
			// 	return err
			// }
			key = key + "[" + kname[i] + "=" + kval + "]"
		}
		key = cschema.Name + key
		child := branch.Get(key)
		if child == nil {
			if child, err = New(cschema); err != nil {
				return err
			}
			if err = branch.Insert(key, child); err != nil {
				return err
			}
		}
		// Update DataNode
		if err := child.unmarshalJSON(jentry); err != nil {
			return err
		}
	}
	return nil
}

func (branch *DataBranch) unmarshalJSON(jval interface{}) error {
	switch jdata := jval.(type) {
	case map[string]interface{}:
		for k, v := range jdata {
			cschema, err := FindSchema(branch.schema, k)
			if err != nil {
				return err
			}
			switch {
			case cschema.IsList():
				if rfc7951List, ok := v.([]interface{}); ok {
					kname := strings.Split(cschema.Key, " ")
					branch.unmarshalListRFC7951(cschema, kname, rfc7951List)
				} else {
					kname := strings.Split(cschema.Key, " ")
					kval := make([]string, 0, len(kname))
					if err := branch.unmarshalList(cschema, kname, kval, v); err != nil {
						return err
					}
				}
			default:
				child := branch.Children[k]
				if child == nil {
					if child, err = New(cschema); err != nil {
						return err
					}
					if err = branch.Insert(k, child); err != nil {
						return err
					}
				}
				if err := child.unmarshalJSON(v); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unexpected json '%v' inserted for %s", jdata, branch)
	}
	return nil
}

func (leaf *DataLeaf) unmarshalJSON(jval interface{}) error {
	valstr, err := JSONValueToString(jval)
	if err != nil {
		return err
	}
	return leaf.Set(valstr)
}

func (leaflist *DataLeafList) unmarshalJSON(jval interface{}) error {
	if islice, ok := jval.([]interface{}); ok {
		for i := range islice {
			valstr, err := JSONValueToString(islice[i])
			if err != nil {
				return err
			}
			if err = leaflist.Set(valstr); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("unexpected json type %T", jval)
}

func (branch *DataBranch) UnmarshalJSON(jsonbyte []byte) error {
	var jval interface{}
	err := json.Unmarshal(jsonbyte, &jval)
	if err != nil {
		return err
	}
	return branch.unmarshalJSON(jval)
}

func (leaf *DataLeaf) UnmarshalJSON(jsonbyte []byte) error {
	var jval interface{}
	err := json.Unmarshal(jsonbyte, &jval)
	if err != nil {
		return err
	}
	return leaf.unmarshalJSON(jval)
}

func (leaflist *DataLeafList) UnmarshalJSON(jsonbyte []byte) error {
	var jval interface{}
	err := json.Unmarshal(jsonbyte, &jval)
	if err != nil {
		return err
	}
	return leaflist.unmarshalJSON(jval)
}

// rfc7951DataNode is used to print the qname of the namespace boundary nodes for rfc7951.
type rfc7951DataNode struct {
	DataNode
}

func (node rfc7951DataNode) MarshalJSON() ([]byte, error) {
	switch n := node.DataNode.(type) {
	case *DataBranch:
		return n.marshalJSON(rfc7951InProgress)
	case *DataLeaf:
		return n.marshalJSON(rfc7951InProgress)
	case *DataLeafList:
		return n.marshalJSON(rfc7951InProgress)
	}
	return nil, fmt.Errorf("unknown type '%T'", node.DataNode)
}

// rfc7951TopDataNode is used to print the qname of the top nodes for rfc7951.
type rfc7951TopDataNode struct {
	DataNode
}

func (node rfc7951TopDataNode) MarshalJSON() ([]byte, error) {
	switch n := node.DataNode.(type) {
	case *DataBranch:
		return n.marshalJSON(rfc7951Enabled)
	case *DataLeaf:
		return n.marshalJSON(rfc7951Enabled)
	case *DataLeafList:
		return n.marshalJSON(rfc7951Enabled)
	}
	return nil, fmt.Errorf("unknown type '%T'", node.DataNode)
}

func (branch *DataBranch) MarshalJSON_IETF() ([]byte, error) {
	n := rfc7951TopDataNode{
		DataNode: branch,
	}
	return n.MarshalJSON()
}

func (leaf *DataLeaf) MarshalJSON_IETF() ([]byte, error) {
	n := rfc7951TopDataNode{
		DataNode: leaf,
	}
	return n.MarshalJSON()
}

func (leaflist *DataLeafList) MarshalJSON_IETF() ([]byte, error) {
	n := rfc7951TopDataNode{
		DataNode: leaflist,
	}
	return n.MarshalJSON()
}

// MarshalJSON returns the JSON encoding of DataNode.
//
// Marshal traverses the value v recursively.
func MarshalJSON(node DataNode, rfc7951 bool) ([]byte, error) {
	if rfc7951 {
		return node.MarshalJSON_IETF()
	} else {
		return node.MarshalJSON()
	}
}

// MarshalJSON_IETF is like Marshal but applies Indent to format the output.
func MarshalJSONIndent(node DataNode, prefix, indent string, rfc7951 bool) ([]byte, error) {
	if rfc7951 {
		n := rfc7951TopDataNode{node}
		return json.MarshalIndent(n, prefix, indent)
	} else {
		return json.MarshalIndent(node, prefix, indent)
	}
}
