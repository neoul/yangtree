package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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
		keyval, err := ExtractKeys(keyname, node[i].Key())
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
		qname = GetAnnotation(schema, "ns-boundary")
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
			qname = GetAnnotation(node[i].Schema(), "ns-boundary")
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
	return encodingToJSON(leaf.schema, leaf.schema.Type, leaf.Value, prefix)
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
