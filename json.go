package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// func marshalListKeyGroup(buffer *bytes.Buffer, ) (int, error) {
// }

func marshalList(buffer *bytes.Buffer, node []DataNode, i int, length int) (int, error) {
	schema := node[i].Schema()
	buffer.WriteString("\"" + schema.Name + "\":{")
	j := i
	for j < length {
		if schema != node[j].Schema() {
			j--
			break
		}
		j++
	}
	for ; i <= j; i++ {
		jsonValue, err := json.Marshal(node[i])
		if err != nil {
			return i, err
		}
		keyval, err := ExtractKeys(strings.Split(schema.Key, " "), node[i].Key())
		if err != nil {
			return 0, err
		}
		buffer.WriteString(fmt.Sprintf("\"%s\":%s", strings.Join(keyval, " "), string(jsonValue)))
		if i < j {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("}")
	if i < length {
		buffer.WriteString(",")
	}
	return i, nil
}

func (branch *DataBranch) MarshalJSON() ([]byte, error) {
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
			i, err = marshalList(buffer, node, i, length)
			if err != nil {
				return nil, err
			}
			continue
		}
		jsonValue, err := json.Marshal(node[i])
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("\"%s\":%s", node[i].Key(), string(jsonValue)))
		if i < length-1 {
			buffer.WriteString(",")
		}
		i++
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	if leaf == nil {
		return nil, nil
	}
	if v, ok := leaf.Value.(yang.Number); ok {
		return []byte(v.String()), nil
	}
	return json.Marshal(leaf.Value)
}

func (leaflist *DataLeafList) MarshalJSON() ([]byte, error) {
	if leaflist == nil {
		return nil, nil
	}
	return json.Marshal(leaflist.Value)
}
