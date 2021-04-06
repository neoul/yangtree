package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func marshalList(buffer *bytes.Buffer, node []DataNode, i int, length int, rfc7951 bool) (int, error) {
	schema := node[i].Schema()
	keyname := strings.Split(schema.Key, " ")
	keynamelen := len(keyname)
	buffer.WriteString("\"" + schema.Name + "\":")
	keymetric := map[string]interface{}{}
	for ; i < length; i++ {
		if schema != node[i].Schema() {
			break
		}
		keyval, err := ExtractKeys(keyname, node[i].Key())
		if err != nil {
			return 0, err
		}
		m := keymetric
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
	jsonValue, err := json.Marshal(keymetric)
	if err != nil {
		return i, err
	}
	buffer.WriteString(string(jsonValue))
	if i < length {
		buffer.WriteString(",")
	}
	return i, nil
}

func (branch *DataBranch) marshalJSON(rfc7951 bool) ([]byte, error) {
	if branch == nil {
		return nil, nil
	}
	length := len(branch.Children)
	if length == 0 {
		return nil, nil
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
			i, err = marshalList(buffer, node, i, length, rfc7951)
			if err != nil {
				return nil, err
			}
			continue
		}
		var err error
		var jsonValue []byte
		if rfc7951 {
			jsonValue, err = json.Marshal(rfc7951DataNode{node[i]})
		} else {
			jsonValue, err = json.Marshal(node[i])
		}
		if err != nil {
			return nil, err
		}
		if rfc7951 {
			if qname := GetAnnotation(node[i].Schema(), "ns-qualified-name"); qname != nil {
				buffer.WriteString("\"" + qname.(string) + "\":" + string(jsonValue))
			} else {
				buffer.WriteString("\"" + node[i].Key() + "\":" + string(jsonValue))
			}
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

func (leaf *DataLeaf) marshalJSON(rfc7951 bool) ([]byte, error) {
	if leaf == nil {
		return nil, nil
	}
	return encodingToJSON(leaf.schema, leaf.schema.Type, leaf.Value, rfc7951)
}

func (leaflist *DataLeafList) marshalJSON(rfc7951 bool) ([]byte, error) {
	if leaflist == nil {
		return nil, nil
	}
	return json.Marshal(leaflist.Value)
}

func (branch *DataBranch) MarshalJSON() ([]byte, error) {
	return branch.marshalJSON(false)
}

func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	return leaf.marshalJSON(false)
}

func (leaflist *DataLeafList) MarshalJSON() ([]byte, error) {
	return leaflist.marshalJSON(false)
}

type rfc7951DataNode struct {
	DataNode
}

func (node rfc7951DataNode) MarshalJSON() ([]byte, error) {
	switch n := node.DataNode.(type) {
	case *DataBranch:
		return n.marshalJSON(true)
	case *DataLeaf:
		return n.marshalJSON(true)
	case *DataLeafList:
		return n.marshalJSON(true)
	}
	return nil, fmt.Errorf("unknown type node %T", node.DataNode)
}

func (branch *DataBranch) MarshalJSON_IETF() ([]byte, error) {
	n := rfc7951DataNode{
		DataNode: branch,
	}
	return n.MarshalJSON()
}

func (leaf *DataLeaf) MarshalJSON_IETF() ([]byte, error) {
	n := rfc7951DataNode{
		DataNode: leaf,
	}
	return n.MarshalJSON()
}

func (leaflist *DataLeafList) MarshalJSON_IETF() ([]byte, error) {
	n := rfc7951DataNode{
		DataNode: leaflist,
	}
	return n.MarshalJSON()
}

// MarshalJSON returns the JSON encoding of DataNode.
//
// Marshal traverses the value v recursively.
func MarshalJSON(node DataNode, rfc7159 bool) ([]byte, error) {
	if rfc7159 {
		return node.MarshalJSON_IETF()
	} else {
		return node.MarshalJSON()
	}
}

// MarshalJSON_IETF is like Marshal but applies Indent to format the output.
func MarshalJSONIndent(node DataNode, prefix, indent string, rfc7159 bool) ([]byte, error) {
	if rfc7159 {
		n := rfc7951DataNode{node}
		return json.MarshalIndent(n, prefix, indent)
	} else {
		return json.MarshalIndent(node, prefix, indent)
	}
}
