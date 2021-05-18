package gnmi

import (
	"fmt"

	"github.com/neoul/yangtree"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/value"
	"github.com/openconfig/goyang/pkg/yang"
)

// ValueToTypedValue encodes val into a gNMI TypedValue message, using the specified encoding
// type if the value is a struct.
func ValueToTypedValue(val interface{}, enc gnmipb.Encoding) (*gnmipb.TypedValue, error) {
	var err error
	var tv *gnmipb.TypedValue
	if node, ok := val.(yangtree.DataNode); ok {
		switch {
		case node.IsBranch():
			switch enc {
			case gnmipb.Encoding_JSON:
				jbyte, err := node.MarshalJSON()
				if err != nil {
					return nil, err
				}
				return &gnmipb.TypedValue{Value: &gnmipb.TypedValue_JsonVal{JsonVal: jbyte}}, nil
			case gnmipb.Encoding_JSON_IETF:
				jbyte, err := node.MarshalJSON_IETF()
				if err != nil {
					return nil, err
				}
				return &gnmipb.TypedValue{Value: &gnmipb.TypedValue_JsonIetfVal{JsonIetfVal: jbyte}}, nil
			default:
				return nil, fmt.Errorf("typed value encoding %q not supported", enc)
			}
		case node.IsLeaf():
			return value.FromScalar(val)
		}
	}
	tv, err = value.FromScalar(val)
	if err == nil {
		return tv, err
	}

	switch v := val.(type) {
	case []int:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{int64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []int8:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{int64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []int16:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{int64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []int32:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{int64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []int64:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{int64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []uint:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{uint64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []uint8:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{uint64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []uint16:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{uint64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []uint32:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{uint64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []uint64:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{uint64(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []float32:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_FloatVal{float32(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []float64:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_FloatVal{float32(s)}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	case []bool:
		sa := &gnmipb.ScalarArray{Element: make([]*gnmipb.TypedValue, len(v))}
		for x, s := range v {
			sa.Element[x] = &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BoolVal{s}}
		}
		tv.Value = &gnmipb.TypedValue_LeaflistVal{sa}
	}
	if tv == nil {
		return nil, fmt.Errorf("unable to convert to typed value for %v", val)
	}
	return tv, nil
}

func DataNodeToTypedValue(node yangtree.DataNode, enc gnmipb.Encoding) (*gnmipb.TypedValue, error) {
	return ValueToTypedValue(node, enc)
}

func TypedValueToDataNode(schema *yang.Entry, tv *gnmipb.TypedValue) (yangtree.DataNode, error) {
	return nil, nil
}
