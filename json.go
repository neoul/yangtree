package yangtree

import (
	"bytes"
	"encoding/json"
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
			i, err = marshalList(buffer, node, i, length, false)
			if err != nil {
				return nil, err
			}
			continue
		}
		jsonValue, err := json.Marshal(node[i])
		if err != nil {
			return nil, err
		}
		if qname := GetAnnotation(node[i].Schema(), "ns-qualified-name"); qname != nil {
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

func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	if leaf == nil {
		return nil, nil
	}
	return encodingToJSON(leaf.schema, leaf.schema.Type, leaf.Value, false)
}

func (leaflist *DataLeafList) MarshalJSON() ([]byte, error) {
	if leaflist == nil {
		return nil, nil
	}
	return json.Marshal(leaflist.Value)
}

// type JSON_IETF_Marshaler interface {
// 	MarshalJSON_IETF() ([]byte, error)
// }

// func (branch *DataBranch) MarshalJSON_IETF() ([]byte, error) {
// 	if branch == nil {
// 		return nil, nil
// 	}
// 	length := len(branch.Children)
// 	if length == 0 {
// 		return nil, nil
// 	}
// 	buffer := bytes.NewBufferString("{")
// 	node := make([]DataNode, 0, length)
// 	for _, c := range branch.Children {
// 		node = append(node, c)
// 	}
// 	sort.Slice(node, func(i, j int) bool {
// 		return node[i].Key() < node[j].Key()
// 	})
// 	for i := 0; i < length; {
// 		if node[i].Schema().IsList() {
// 			var err error
// 			i, err = marshalList(buffer, node, i, length, true)
// 			if err != nil {
// 				return nil, err
// 			}
// 			continue
// 		}
// 		jsonValue, err := json.Marshal(node[i])
// 		if err != nil {
// 			return nil, err
// 		}
// 		mod := node[i].Schema().Modules()
// 		m, _ := mod.FindModuleByPrefix(node[i].Schema().Prefix.Name)
// 		buffer.WriteString("\"" + m.Name + ":" + node[i].Key() + "\":" + string(jsonValue))
// 		if i < length-1 {
// 			buffer.WriteString(",")
// 		}
// 		i++
// 	}
// 	buffer.WriteString("}")
// 	return buffer.Bytes(), nil
// }

// func (this Stuff) UnmarshalJSON(b []byte) error {
// 	var stuff map[string]string
// 	err := json.Unmarshal(b, &stuff)
// 	if err != nil {
// 		return err
// 	}
// 	for key, value := range stuff {
// 		numericKey, err := strconv.Atoi(key)
// 		if err != nil {
// 			return err
// 		}
// 		this[numericKey] = value
// 	}
// 	return nil
// }
