package yangtree

import (
	"fmt"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"gopkg.in/yaml.v2"
)

// unmarshalYAMLList decode yval to the list that has the keys.
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
			found := GetSchema(cschema, kname[i])
			if found == nil {
				return fmt.Errorf("schema %q not found", kname[i])
			}
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
	case *DataLeafList:
		if vslice, ok := yval.([]interface{}); ok {
			for i := range vslice {
				if err := n.Set(ValueToString(vslice[i])); err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("unexpected yaml value %q for %s", yval, n)
	case *DataLeaf:
		return n.Set(ValueToString(yval))
	default:
		return fmt.Errorf("unknown data node type: %T", node)
	}
}

func (branch *DataBranch) UnmarshalYAML(in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(branch, ydata)
}

func (leaf *DataLeaf) UnmarshalYAML(in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(leaf, ydata)
}

func (leaflist *DataLeafList) UnmarshalYAML(in []byte) error {
	var ydata interface{}
	err := yaml.Unmarshal(in, &ydata)
	if err != nil {
		return err
	}
	return unmarshalYAML(leaflist, ydata)
}

// func (branch *DataBranch) MarshalYAML() (interface{}, error) {

// }

// func (leaf *DataLeaf) MarshalYAML() (interface{}, error) {

// }

// func (leaflist *DataLeafList) MarshalYAML() (interface{}, error) {

// }
