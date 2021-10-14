package yangtree

import (
	"fmt"
	"strconv"

	"github.com/openconfig/goyang/pkg/yang"
)

// value2String() marshals a value based on its schema, type and representing format.
func value2String(schema *SchemaNode, typ *yang.YangType, value interface{}) (string, error) {
	switch typ.Kind {
	// case yang.YinstanceIdentifier:
	// [FIXME] The leftmost (top-level) data node name is always in the
	//   namespace-qualified form (qname).
	// case yang.Ystring, yang.Ybinary:
	// case yang.Ybool:
	// case yang.Yleafref:
	// case yang.Ynone:
	// case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yuint8, yang.Yuint16, yang.Yuint32:
	// case yang.Yint64:
	// case yang.Yuint64:
	// case yang.Ybits, yang.Yenum:
	// case yang.Ydecimal64:
	case yang.Yunion:
		for i := range typ.Type {
			v, err := value2String(schema, typ.Type[i], value)
			if err == nil {
				return v, nil
			}
		}
		return "", fmt.Errorf("unexpected value \"%v\" for %q type", value, typ.Name)
	case yang.Yempty:
		return "", nil
	case yang.Yidentityref:
		if s, ok := value.(string); ok {
			m, ok := schema.Identityref[s]
			if !ok {
				return "", fmt.Errorf("%q is not a value of %q", s, typ.Name)
			}
			if m.Prefix == nil {
				return m.Name + ":" + s, nil
			}
			return m.Prefix.Name + ":" + s, nil
		}
	}
	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(int64(v), 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(uint64(v), 10), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case yang.Number:
		return v.String(), nil
	case nil:
		return "", nil
	}
	return fmt.Sprint(value), nil
}
