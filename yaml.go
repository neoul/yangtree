package yangtree

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"gopkg.in/yaml.v2"
)

// unmarshalYAMLList constructs list entries of a data node using YAML values.
func (branch *DataBranch) unmarshalYAMLList(cschema *yang.Entry, kname []string, kval []interface{}, yval interface{}) error {
	jdata, ok := yval.(map[interface{}]interface{})
	if !ok {
		if yval == nil {
			return nil
		}
		return fmt.Errorf("unexpected yaml-val \"%v\" (%T) for %q", yval, yval, cschema.Name)
	}
	if len(kname) != len(kval) {
		for k, v := range jdata {
			kval = append(kval, k)
			err := branch.unmarshalYAMLList(cschema, kname, kval, v)
			if err != nil {
				return err
			}
			kval = kval[:len(kval)-1]
		}
		return nil
	}
	// check existent DataNode
	var err error
	var key strings.Builder
	key.WriteString(cschema.Name)
	for i := range kval {
		key.WriteString("[")
		key.WriteString(kname[i])
		key.WriteString("=")
		key.WriteString(ValueToString(kval[i]))
		key.WriteString("]")
	}
	var child DataNode
	found := branch.Get(key.String())
	if found == nil {
		if child, err = branch.New(key.String()); err != nil {
			return err
		}
	} else {
		child = found
	}
	// Update DataNode
	return unmarshalYAML(child, yval)
}

// unmarshalYAMLListRFC7951 constructs list entries of a data node using YAML values.
func (branch *DataBranch) unmarshalYAMLListRFC7951(cschema *yang.Entry, kname []string, listentry []interface{}) error {
	for i := range listentry {
		entry, ok := listentry[i].(map[interface{}]interface{})
		if !ok {
			return fmt.Errorf("unexpected yaml type '%T' for %s", listentry[i], cschema.Name)
		}
		// check existent DataNode
		var err error
		var key strings.Builder
		key.WriteString(cschema.Name)
		for i := range kname {
			key.WriteString("[")
			key.WriteString(kname[i])
			key.WriteString("=")
			key.WriteString(fmt.Sprint(entry[kname[i]]))
			key.WriteString("]")
			// [FIXME] need to check key validation
			// kchild, err := New(kschema, kval)
			// if err != nil {
			// 	return err
			// }
		}
		var child DataNode
		if IsDuplicatedList(cschema) {
			if child, err = branch.New(key.String()); err != nil {
				return err
			}
		} else {
			found := branch.Get(key.String())
			if found == nil {
				if child, err = branch.New(key.String()); err != nil {
					return err
				}
			} else {
				child = found
			}
		}

		// Update DataNode
		if err := unmarshalYAML(child, entry); err != nil {
			return err
		}
	}
	return nil
}

// unmarshalYAML updates a data node using YAML values.
func unmarshalYAML(node DataNode, yval interface{}) error {
	switch n := node.(type) {
	case *DataBranch:
		switch data := yval.(type) {
		case map[interface{}]interface{}:
			for k, v := range data {
				keystr := ValueToString(k)
				name, haskey, err := ExtractSchemaName(&keystr)
				if err != nil {
					return err
				}
				cschema := GetSchema(n.schema, name)
				if cschema == nil {
					return fmt.Errorf("schema %q not found from %q", k, n.schema.Name)
				}

				switch {
				case IsList(cschema):
					if haskey {
						keyname := GetKeynames(cschema)
						keyval, err := ExtractKeyValues(keyname, &keystr)
						if err != nil {
							return err
						}
						keymap := map[interface{}]interface{}{}
						m := keymap
						for x := range keyval {
							if x < len(keyname)-1 {
								if n := m[keyval[x]]; n == nil {
									n := map[interface{}]interface{}{}
									m[keyval[x]] = n
									m = n
								} else {
									m = n.(map[interface{}]interface{})
								}
							} else {
								if v != nil {
									m[keyval[x]] = v
								} else {
									m[keyval[x]] = map[interface{}]interface{}{}
								}
							}
						}
						v = keymap
					}
					if rfc7951StyleList, ok := v.([]interface{}); ok {
						if err := n.unmarshalYAMLListRFC7951(cschema, GetKeynames(cschema), rfc7951StyleList); err != nil {
							return err
						}
					} else {
						if IsDuplicatedList(cschema) {
							return fmt.Errorf("non-key list %q must have the array format", cschema.Name)
						}
						kname := GetKeynames(cschema)
						kval := make([]interface{}, 0, len(kname))
						if err := n.unmarshalYAMLList(cschema, kname, kval, v); err != nil {
							return err
						}
					}
				case cschema.IsLeafList():
					vv, ok := v.([]interface{})
					if !ok {
						return fmt.Errorf("unexpected type inserted for %q", cschema.Name)
					}
					for i := range vv {
						child, err := New(cschema)
						if err != nil {
							return err
						}
						if err := unmarshalYAML(child, vv[i]); err != nil {
							return err
						}
						if err := n.Insert(child); err != nil {
							return err
						}
					}
				default:
					var child DataNode
					i, _ := n.Index(keystr)
					if i < len(n.children) && n.children[i].Key() == k {
						child = n.children[i]
						if err := unmarshalYAML(child, v); err != nil {
							return err
						}
					} else {
						child, err := New(cschema)
						if err != nil {
							return err
						}
						if err := unmarshalYAML(child, v); err != nil {
							return err
						}
						if err := n.Insert(child); err != nil {
							return err
						}
					}

				}
			}
			return nil
		case []interface{}:
			for i := range data {
				if err := unmarshalYAML(node, data[i]); err != nil {
					return err
				}
			}
			return nil
		default:
			return fmt.Errorf("unexpected yaml value \"%v\" (%T) inserted for %q", yval, yval, n)
		}
	case *DataLeaf:
		return n.Set(ValueToString(yval))
	default:
		return fmt.Errorf("unknown data node type: %T", node)
	}
}

// UnmarshalYAML updates the branch data node using YAML-encoded data.
func (branch *DataBranch) UnmarshalYAML(in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(branch, ydata)
}

// UnmarshalYAML updates the leaf data node using YAML-encoded data.
func (leaf *DataLeaf) UnmarshalYAML(in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(leaf, ydata)
}

// UnmarshalYAML updates the data node using YAML-encoded data.
func UnmarshalYAML(node DataNode, in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(node, ydata)
}

type yDataNode struct {
	DataNode        // Target data node to encode the data node
	rfc7951s        // Modified RFC7951 format for YAML
	iformat    bool // Interval YAML format
	configOnly yang.TriState
	indentStr  string // indent string used at YAML encoding
}

func (ynode *yDataNode) rfc7951() rfc7951s {
	if ynode == nil {
		return rfc7951Disabled
	}
	return ynode.rfc7951s
}

func (ynode *yDataNode) getQname() string {
	switch ynode.rfc7951() {
	case rfc7951InProgress, rfc7951Enabled:
		ynode.rfc7951s = rfc7951InProgress
		if qname, boundary := GetQName(ynode.Schema()); boundary ||
			ynode.rfc7951() == rfc7951Enabled {
			return qname
		}
		return ynode.Schema().Name
	}
	if ynode.IsDataBranch() {
		if ynode.iformat {
			return ynode.Key()
		}
	}
	return ynode.Schema().Name
}

func marshalYAMLListableNode(buffer *bytes.Buffer, node []DataNode, i int, indent int, parent *yDataNode) (int, error) {
	schema := node[i].Schema()
	ynode := *parent
	ynode.DataNode = node[i]
	m := GetSchemaMeta(schema)
	switch ynode.configOnly {
	case yang.TSTrue:
		if m.IsState {
			for ; i < len(node); i++ {
				if schema != node[i].Schema() {
					return i, nil
				}
			}
		}
	case yang.TSFalse: // stateOnly
		if !m.IsState && !m.HasState {
			for ; i < len(node); i++ {
				if schema != node[i].Schema() {
					return i, nil
				}
			}
		}
	}
	if ynode.rfc7951() != rfc7951Disabled || IsDuplicatedList(schema) || schema.IsLeafList() {
		writeIndent(buffer, indent, ynode.indentStr)
		buffer.WriteString(ynode.getQname())
		buffer.WriteString(":\n")
		indent++
		for ; i < len(node); i++ {
			ynode.DataNode = node[i]
			if schema != ynode.Schema() {
				break
			}
			writeIndent(buffer, indent, ynode.indentStr)
			buffer.WriteString("-")
			writeIndent(buffer, 1, ynode.indentStr)
			err := ynode.marshalYAML(buffer, indent+2, true)
			if err != nil {
				return i, err
			}
			if ynode.IsLeafList() {
				buffer.WriteString("\n")
			}
		}
		return i, nil
	}
	var lastKeyval []string
	if !ynode.iformat {
		writeIndent(buffer, indent, ynode.indentStr)
		buffer.WriteString(ynode.getQname())
		buffer.WriteString(":\n")
		indent++
	}
	for ; i < len(node); i++ {
		ynode.DataNode = node[i]
		if schema != ynode.Schema() {
			break
		}
		if ynode.iformat {
			writeIndent(buffer, indent, ynode.indentStr)
			buffer.WriteString(ynode.getQname())
			buffer.WriteString(":\n")
			err := ynode.marshalYAML(buffer, indent+1, false)
			if err != nil {
				return i, err
			}
		} else {
			keyname, keyval := GetKeyValues(ynode.DataNode)
			if len(keyname) != len(keyval) {
				return i, fmt.Errorf("list %q doesn't have a key value", schema.Name)
			}
			for j := range keyval {
				if len(lastKeyval) > 0 && keyval[j] == lastKeyval[j] {
					continue
				}
				writeIndent(buffer, indent+j, ynode.indentStr)
				buffer.WriteString(keyval[j])
				buffer.WriteString(":\n")
			}
			err := ynode.marshalYAML(buffer, indent+len(keyval), false)
			if err != nil {
				return i, err
			}
			lastKeyval = keyval
		}
	}
	return i, nil
}

func (ynode *yDataNode) marshalYAML(buffer *bytes.Buffer, indent int, disableFirstIndent bool) error {
	if ynode == nil || ynode.DataNode == nil {
		return nil
	}
	cynode := *ynode
	switch datanode := ynode.DataNode.(type) {
	case *DataBranch:
		node := datanode.children
		for i := 0; i < len(datanode.children); {
			schema := node[i].Schema()
			if IsListable(schema) {
				var err error
				i, err = marshalYAMLListableNode(buffer, node, i, indent, ynode)
				if err != nil {
					return err
				}
				continue
			}
			// container, leaf
			m := GetSchemaMeta(schema)
			if (ynode.configOnly == yang.TSTrue && m.IsState) ||
				(ynode.configOnly == yang.TSFalse && !m.IsState && !m.HasState) {
				i++
				continue
			}
			cynode.DataNode = node[i]
			if disableFirstIndent {
				disableFirstIndent = false
			} else {
				writeIndent(buffer, indent, cynode.indentStr)
			}
			buffer.WriteString(cynode.getQname())
			buffer.WriteString(":")
			if cynode.IsLeaf() {
				buffer.WriteString(" ")
			} else {
				buffer.WriteString("\n")
			}
			if err := cynode.marshalYAML(buffer, indent+1, false); err != nil {
				return err
			}
			if cynode.IsLeaf() {
				buffer.WriteString("\n")
			}
			i++
		}
	case *DataLeaf:
		rfc7951enabled := false
		if ynode.rfc7951() != rfc7951Disabled {
			rfc7951enabled = true
		}
		valbyte, err := ValueToYAMLBytes(datanode.schema, datanode.schema.Type, datanode.value, rfc7951enabled)
		if err != nil {
			return err
		}
		buffer.Write(valbyte)
	}
	return nil
}

// MarshalYAML encodes the branch data node to a YAML document.
func (branch *DataBranch) MarshalYAML() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: branch, indentStr: " ", iformat: true}
	// ynode := &yDataNode{DataNode: branch, indentStr: " "}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalYAML encodes the leaf data node to a YAML document.
func (leaf *DataLeaf) MarshalYAML() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: leaf, indentStr: " "}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalYAML_RFC7951 encodes the branch data node to a YAML document using RFC7951 namespace-qualified name.
// RFC7951 is the encoding specification for JSON. So, MarshalYAML_RFC7951 only utilizes the RFC7951 namespace-qualified name for YAML encoding.
func (branch *DataBranch) MarshalYAML_RFC7951() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: branch, indentStr: " ", rfc7951s: rfc7951Enabled}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalYAML_RFC7951 encodes the leaf data node to a YAML document using RFC7951 namespace-qualified name.
// RFC7951 is the encoding specification for JSON. So, MarshalYAML_RFC7951 only utilizes the RFC7951 namespace-qualified name for YAML encoding.
func (leaf *DataLeaf) MarshalYAML_RFC7951() ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: leaf, indentStr: " ", rfc7951s: rfc7951Enabled}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// InternalFormat is an option to marshal a data node to an internal YAML format.
type InternalFormat struct{}

func (o InternalFormat) IsOption() {}

// MarshalYAML encodes the data node to a YAML document with a number of options.
// The options available are [ConfigOnly, StateOnly, RFC7951Format, InternalFormat].
func MarshalYAML(node DataNode, option ...Option) ([]byte, error) {
	buffer := bytes.NewBufferString("")
	ynode := &yDataNode{DataNode: node, indentStr: " "}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v option is is available to find nodes", option[i])
		case ConfigOnly:
			ynode.configOnly = yang.TSTrue
		case StateOnly:
			ynode.configOnly = yang.TSFalse
		case RFC7951Format:
			ynode.rfc7951s = rfc7951Enabled
		case InternalFormat:
			ynode.iformat = true
		}
	}
	if err := ynode.marshalYAML(buffer, 0, false); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeIndent(buffer *bytes.Buffer, indent int, indentStr string) {
	for i := 0; i < indent; i++ {
		buffer.WriteString(indentStr)
	}
}

// ValueToYAMLBytes encodes the value to a YAML-encoded data. the schema and the type of the value must be set.
func ValueToYAMLBytes(schema *yang.Entry, typ *yang.YangType, value interface{}, rfc7951 bool) ([]byte, error) {
	switch typ.Kind {
	case yang.Yunion:
		for i := range typ.Type {
			v, err := ValueToYAMLBytes(schema, typ.Type[i], value, rfc7951)
			if err == nil {
				return v, nil
			}
		}
		return nil, fmt.Errorf("unexpected value \"%v\" for %q type", value, typ.Name)
	case yang.YinstanceIdentifier:
		// [FIXME] The leftmost (top-level) data node name is always in the
		//   namespace-qualified form (qname).
	case yang.Ydecimal64:
		switch v := value.(type) {
		case yang.Number:
			return []byte(v.String()), nil
		case string:
			return []byte(v), nil
		}
	case yang.Yempty:
		return []byte(""), nil
	}
	if rfc7951 {
		switch typ.Kind {
		// case yang.Ystring, yang.Ybinary:
		// case yang.Ybool:
		// case yang.Yleafref:
		// case yang.Ynone:
		// case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yuint8, yang.Yuint16, yang.Yuint32:
		// case yang.Ybits, yang.Yenum:
		// case yang.Yempty:
		// 	return []byte(""), nil
		case yang.Yidentityref:
			if s, ok := value.(string); ok {
				imap := getIdentityref(schema)
				qvalue, ok := imap[s]
				if !ok {
					return nil, fmt.Errorf("%q is not a value of %q", s, typ.Name)
				}
				value = qvalue
			}
		case yang.Yint64:
			if v, ok := value.(int64); ok {
				str := strconv.FormatInt(v, 10)
				return []byte(str), nil
			}
		case yang.Yuint64:
			if v, ok := value.(uint64); ok {
				str := strconv.FormatUint(v, 10)
				return []byte(str), nil
			}
		}
	}
	// else {
	// 	switch typ.Kind {
	// 	case yang.Yempty:
	// 		return []byte("null"), nil
	// 	}
	// }
	out, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(string(out), "\n") {
		return out[:len(out)-1], nil
	}
	return out, err
}
