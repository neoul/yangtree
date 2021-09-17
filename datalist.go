package yangtree

import (
	"bytes"
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

type DataNodeList []DataNode

// MarshalJSON() encodes the data node list to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format].
//   // usage:
//   var node []DataNode
//   jsonbytes, err := DataNodeList(got).MarshalYAML()
func (list DataNodeList) MarshalJSON(option ...Option) ([]byte, error) {
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
//   yamlbytes, err := DataNodeList(got).MarshalYAML()
func (list DataNodeList) MarshalYAML(option ...Option) ([]byte, error) {
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
