package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

type DataNodeGroup []DataNode

// MarshalJSON() encodes the data node list to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format].
//   // usage:
//   var node []DataNode
//   jsonbytes, err := DataNodeGroup(got).MarshalYAML()
func (list DataNodeGroup) MarshalJSON(option ...Option) ([]byte, error) {
	var comma bool
	var buffer bytes.Buffer
	buffer.WriteString("[")
	configOnly := yang.TSUnset
	rfc7951s := rfc7951Enabled
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v is not allowed for marshaling", option[i])
		case ConfigOnly:
			configOnly = yang.TSTrue
		case StateOnly:
			configOnly = yang.TSFalse
		case RFC7951Format:
			rfc7951s = rfc7951Enabled
		}
	}
	for _, n := range list {
		if comma {
			buffer.WriteString(",")
		}
		jnode := &jDataNode{DataNode: n, configOnly: configOnly, rfc7951s: rfc7951s}
		err := jnode.marshalJSON(&buffer)
		if err != nil {
			return nil, err
		}
		comma = true
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

// MarshalYAML() encodes the data node list to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format, InternalFormat].
//   // usage:
//   var node []DataNode
//   yamlbytes, err := DataNodeGroup(got).MarshalYAML()
func (list DataNodeGroup) MarshalYAML(option ...Option) ([]byte, error) {
	var buffer bytes.Buffer
	configOnly := yang.TSUnset
	rfc7951s := rfc7951Disabled
	iformat := false
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v option can be used to find nodes", option[i])
		case ConfigOnly:
			configOnly = yang.TSTrue
		case StateOnly:
			configOnly = yang.TSFalse
		case RFC7951Format:
			rfc7951s = rfc7951Enabled
		case InternalFormat:
			iformat = true
		}
	}
	comma := false
	for _, n := range list {
		if comma {
			buffer.WriteString(", ")
		}
		if n.IsDataBranch() {
			buffer.WriteString("- ")
		} else {
			if !comma {
				buffer.WriteString("[")
			}
		}
		ynode := &yDataNode{DataNode: n, indentStr: " ",
			configOnly: configOnly, rfc7951s: rfc7951s, iformat: iformat}
		if err := ynode.marshalYAML(&buffer, 2, true); err != nil {
			return nil, err
		}
		if n.IsDataLeaf() {
			comma = true
		}
	}
	if comma {
		buffer.WriteString("]")
	}
	return buffer.Bytes(), nil
}

// NewDataGroup() creates a set of new DataNodes having the same schema.
// To create a set of data nodes, the value must be encoded to a JSON object or a JSON array of the data.
// It is useful to create multiple list nodes or leaf-list nodes.
// The returned collector data node is a data node containing any data node.
//    // e.g.
//    collector, err := NewDataGroup(schema, `["leaf-list-value1", "leaf-list-value2"]`)
//    for _, node := range collector.Children {
//         // Process the created nodes ("leaf-list-value1" and "leaf-list-value2") here.
//         // The collector is an yang anydata node to keep various data nodes.
//    }
func NewDataGroup(schema *yang.Entry, nodes []DataNode, value ...string) (DataNodeGroup, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	c := NewDataNodeCollector().(*DataBranch)
	for i := range nodes {
		c.insert(Clone(nodes[i]), EditMerge, nil)
	}
	for i := range value {
		var jval interface{}
		if err := json.Unmarshal([]byte(value[i]), &jval); err != nil {
			return nil, err
		}
		switch jdata := jval.(type) {
		case map[string]interface{}:
			if IsDuplicatableList(schema) {
				return nil, fmt.Errorf("non-key list %q must have the array format of RFC7951", schema.Name)
			}
			kname := GetKeynames(schema)
			kval := make([]string, 0, len(kname))
			if err := c.unmarshalJSONList(schema, kname, kval, jdata, nil); err != nil {
				return nil, err
			}
		case []interface{}:
			if err := c.unmarshalJSONListable(schema, GetKeynames(schema), jdata, nil); err != nil {
				return nil, err
			}
		default:
			n, err := NewDataNode(schema, value...)
			if err != nil {
				return nil, err
			}
			if _, err := c.insert(n, EditMerge, nil); err != nil {
				return nil, err
			}
		}
	}
	return c.children, nil
}
