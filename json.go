package yangtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

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

func (s rfc7951s) String() string {
	switch s {
	case rfc7951Disabled:
		return "rfc7951.disabled"
	case rfc7951InProgress:
		return "rfc7951.in-progress"
	case rfc7951Enabled:
		return "rfc7951.enabled"
	}
	return "rfc7951.unknown"
}

type jDataNode struct {
	DataNode
	rfc7951s
	configOnly yang.TriState
}

func (jnode *jDataNode) getQname() string {
	switch jnode.rfc7951s {
	case rfc7951InProgress, rfc7951Enabled:
		if qname, boundary := GetQName(jnode.Schema()); boundary ||
			jnode.rfc7951s == rfc7951Enabled {
			jnode.rfc7951s = rfc7951InProgress
			return qname
		}
		return jnode.Schema().Name
	}
	return jnode.Schema().Name
}

func (jnode *jDataNode) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (jnode *jDataNode) marshalJSON(buffer *bytes.Buffer) error {
	if jnode == nil || jnode.DataNode == nil {
		buffer.WriteString(`null`)
		return nil
	}
	cjnode := *jnode
	switch datanode := jnode.DataNode.(type) {
	case *DataBranch:
		comma := false
		node := datanode.children
		buffer.WriteString(`{`)
		for i := 0; i < len(datanode.children); {
			schema := node[i].Schema()
			if IsListable(schema) {
				var err error
				i, comma, err = marshalJSONListableNode(buffer, node, i, comma, jnode)
				if err != nil {
					return err
				}
				continue
			}
			// container, leaf
			m := GetSchemaMeta(schema)
			if (jnode.configOnly == yang.TSTrue && m.IsState) ||
				(jnode.configOnly == yang.TSFalse && !m.IsState && !m.HasState) {
				i++
				continue
			}
			cjnode.DataNode = node[i]
			cjnode.rfc7951s = jnode.rfc7951s
			if comma {
				buffer.WriteString(",")
			}
			comma = true
			buffer.WriteString(`"`)
			buffer.WriteString(cjnode.getQname()) // namespace-qualified name
			buffer.WriteString(`":`)
			if err := cjnode.marshalJSON(buffer); err != nil {
				return err
			}
			i++
		}
		buffer.WriteString(`}`)
		return nil
	case *DataLeaf:
		rfc7951enabled := false
		if jnode.rfc7951s != rfc7951Disabled {
			rfc7951enabled = true
		}
		b, err := ValueToJSONBytes(datanode.schema, datanode.schema.Type, datanode.value, rfc7951enabled)
		if err != nil {
			return err
		}
		buffer.Write(b)
	}
	return nil
}

func marshalJSONListableNode(buffer *bytes.Buffer, node []DataNode, i int, comma bool, parent *jDataNode) (int, bool, error) {
	first := *parent
	first.DataNode = node[i]
	schema := first.Schema()
	m := GetSchemaMeta(schema)
	switch first.configOnly {
	case yang.TSTrue:
		if m.IsState {
			for ; i < len(node); i++ {
				if schema != node[i].Schema() {
					return i, comma, nil
				}
			}
		}
	case yang.TSFalse: // stateOnly
		if !m.IsState && !m.HasState {
			for ; i < len(node); i++ {
				if schema != node[i].Schema() {
					return i, comma, nil
				}
			}
		}
	}
	if comma {
		buffer.WriteString(",")
	}
	comma = true
	buffer.WriteString(`"`)
	buffer.WriteString(first.getQname())
	buffer.WriteString(`":`)
	if first.rfc7951s != rfc7951Disabled || IsDuplicatableList(schema) || schema.IsLeafList() {
		ii := i
		for ; i < len(node); i++ {
			if schema != node[i].Schema() {
				break
			}
		}
		nodelist := make([]interface{}, 0, i-ii)
		for ; ii < i; ii++ {
			jnode := &jDataNode{DataNode: node[ii],
				configOnly: first.configOnly, rfc7951s: first.rfc7951s}
			nodelist = append(nodelist, jnode)
		}
		err := marshalJNodeTree(buffer, nodelist)
		return i, comma, err
	}

	nodemap := map[string]interface{}{}
	for ; i < len(node); i++ {
		jnode := &jDataNode{DataNode: node[i],
			configOnly: first.configOnly, rfc7951s: first.rfc7951s}
		if schema != jnode.Schema() {
			break
		}
		keyname, keyval := GetKeyValues(jnode.DataNode)
		if len(keyname) != len(keyval) {
			return i, comma, fmt.Errorf("list %q doesn't have key value pairs", schema.Name)
		}
		m := nodemap
		for x := range keyval {
			if x < len(keyname)-1 {
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
	err := marshalJNodeTree(buffer, nodemap)
	return i, comma, err
}

func marshalJNodeTree(buffer *bytes.Buffer, jnodeTree interface{}) error {
	comma := false
	switch jj := jnodeTree.(type) {
	case map[string]interface{}:
		buffer.WriteString(`{`)
		k := make([]string, 0, len(jj))
		for key := range jj {
			k = append(k, key)
		}
		sort.Slice(k, func(i, j int) bool { return k[i] < k[j] })
		for i := range k {
			if comma {
				buffer.WriteString(",")
			}
			comma = true
			buffer.WriteString(`"`)
			buffer.WriteString(k[i])
			buffer.WriteString(`":`)
			if err := marshalJNodeTree(buffer, jj[k[i]]); err != nil {
				return err
			}
		}
		buffer.WriteString(`}`)
	case []interface{}:
		buffer.WriteString(`[`)
		for i := range jj {
			if comma {
				buffer.WriteString(",")
			}
			comma = true
			if err := marshalJNodeTree(buffer, jj[i]); err != nil {
				return err
			}
		}
		buffer.WriteString(`]`)
	case *jDataNode:
		if err := jj.marshalJSON(buffer); err != nil {
			return err
		}
	}
	return nil
}

func (branch *DataBranch) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: branch}
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: leaf}
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (branch *DataBranch) MarshalJSON_IETF() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: branch}
	jnode.rfc7951s = rfc7951Enabled
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (leaf *DataLeaf) MarshalJSON_IETF() ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: leaf}
	jnode.rfc7951s = rfc7951Enabled
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// unmarshalJSONList decode jval to the list that has the keys.
func (branch *DataBranch) unmarshalJSONList(cschema *yang.Entry, kname []string, kval []string, jval interface{}) error {
	jdata, ok := jval.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected json-val \"%v\" (%T) for %q", jval, jval, cschema.Name)
	}
	if len(kname) != len(kval) {
		for k, v := range jdata {
			kval = append(kval, k)
			err := branch.unmarshalJSONList(cschema, kname, kval, v)
			if err != nil {
				return err
			}
			kval = kval[:len(kval)-1]
		}
		return nil
	}

	var err error
	pmap := make(map[string]interface{})
	for i := range kname {
		pmap[kname[i]] = kval[i]
	}

	id, groupSearch := GenerateID(cschema, pmap)
	children := branch.find(cschema, &id, groupSearch, nil)
	if IsDuplicatable(cschema) {
		children = nil // clear found nodes
	}
	var created bool
	var child DataNode
	if len(children) > 0 {
		child = children[0]
	} else {
		child, err = NewDataNode(cschema)
		if err != nil {
			return err
		}
		created = true
	}
	if err = UpdateByMap(child, pmap); err != nil {
		return err
	}

	if _, err := branch.insert(child, EditMerge, nil); err != nil {
		return err
	}

	// Update DataNode
	err = unmarshalJSON(child, jval)
	if err != nil {
		if created {
			child.Remove()
		}
		return err
	}
	return nil
}

func (branch *DataBranch) unmarshalJSONListable(cschema *yang.Entry, kname []string, listentry []interface{}) error {
	for i := range listentry {
		var err error
		pmap := make(map[string]interface{})
		switch jentry := listentry[i].(type) {
		case map[string]interface{}:
			for i := range kname {
				valstr, err := JSONValueToString(jentry[kname[i]])
				if err != nil {
					return err
				}
				pmap[kname[i]] = valstr
			}
		// case []interface{}:
		// 	return fmt.Errorf("unexpected json type '%T' for %s", listentry[i], cschema.Name)
		default:
			valstr, err := JSONValueToString(jentry)
			if err != nil {
				return err
			}
			pmap["."] = valstr
		}

		id, groupSearch := GenerateID(cschema, pmap)
		children := branch.find(cschema, &id, groupSearch, nil)
		if IsDuplicatable(cschema) {
			children = nil // clear found nodes
		}
		var created bool
		var child DataNode
		if len(children) > 0 {
			child = children[0]
		} else {
			child, err = NewDataNode(cschema)
			if err != nil {
				return err
			}
			created = true
		}
		if err = UpdateByMap(child, pmap); err != nil {
			return err
		}
		if _, err = branch.insert(child, EditMerge, nil); err != nil {
			return err
		}

		// Update DataNode if it is a list node.
		if IsList(cschema) {
			if err := unmarshalJSON(child, listentry[i]); err != nil {
				if created {
					branch.Delete(child)
				}
				return err
			}
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
					return fmt.Errorf("schema %q not found from %q", k, n.schema.Name)
				}
				switch {
				case IsListable(cschema):
					if list, ok := v.([]interface{}); ok {
						if err := n.unmarshalJSONListable(cschema, GetKeynames(cschema), list); err != nil {
							return err
						}
					} else {
						if IsDuplicatableList(cschema) {
							return fmt.Errorf("non-key list %q must have the array format of RFC7951", cschema.Name)
						}
						kname := GetKeynames(cschema)
						kval := make([]string, 0, len(kname))
						if err := n.unmarshalJSONList(cschema, kname, kval, v); err != nil {
							return err
						}
					}
				default:
					var child DataNode
					i := n.Index(k)
					if i < len(n.children) && n.children[i].ID() == k {
						child = n.children[i]
						if err := unmarshalJSON(child, v); err != nil {
							return err
						}
					} else {
						child, err := NewDataNode(cschema)
						if err != nil {
							return err
						}
						if err := unmarshalJSON(child, v); err != nil {
							return err
						}
						if _, err := n.Insert(child); err != nil {
							return err
						}
					}
				}
			}
			return nil
		case []interface{}:
			for i := range jdata {
				if err := unmarshalJSON(node, jdata[i]); err != nil {
					return err
				}
			}
			return nil
		default:
			return fmt.Errorf("unexpected json value \"%v\" (%T) inserted for %q", jval, jval, n)
		}
	case *DataLeaf:
		valstr, err := JSONValueToString(jval)
		if err != nil {
			return err
		}
		return n.Set(valstr)
	default:
		return fmt.Errorf("unknown data node type: %T", node)
	}
}

func (branch *DataBranch) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(branch, jval) // merge
}

func (leaf *DataLeaf) UnmarshalJSON(jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(leaf, jval) // merge
}

// MarshalJSON returns the JSON encoding of DataNode.
//
// Marshal traverses the value v recursively.
func MarshalJSON(node DataNode, option ...Option) ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: node}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v is not allowed for marshaling", option[i])
		case ConfigOnly:
			jnode.configOnly = yang.TSTrue
		case StateOnly:
			jnode.configOnly = yang.TSFalse
		case RFC7951Format:
			jnode.rfc7951s = rfc7951Enabled
		}
	}
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// MarshalJSONIndent is like Marshal but applies an indent and a prefix to format the output.
func MarshalJSONIndent(node DataNode, prefix, indent string, option ...Option) ([]byte, error) {
	var buffer bytes.Buffer
	jnode := &jDataNode{DataNode: node}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v is not allowed for marshaling", option[i])
		case ConfigOnly:
			jnode.configOnly = yang.TSTrue
		case StateOnly:
			jnode.configOnly = yang.TSFalse
		case RFC7951Format:
			jnode.rfc7951s = rfc7951Enabled
		}
	}
	err := jnode.marshalJSON(&buffer)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, buffer.Bytes(), prefix, indent)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the data node.
func UnmarshalJSON(node DataNode, jbytes []byte) error {
	var jval interface{}
	err := json.Unmarshal(jbytes, &jval)
	if err != nil {
		return err
	}
	return unmarshalJSON(node, jval)
}
