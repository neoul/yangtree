package yangtree

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/openconfig/goyang/pkg/yang"
)

// value2XMLString() marshals a value based on its schema, type and representing format.
func value2XMLString(schema *SchemaNode, typ *yang.YangType, value interface{}) (string, error) {
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
	// case yang.Ydecimal64:
	// case yang.Ybits, yang.Yenum:
	case yang.Yunion:
		for i := range typ.Type {
			v, err := value2XMLString(schema, typ.Type[i], value)
			if err == nil {
				return v, nil
			}
		}
		return "", fmt.Errorf("unexpected value \"%v\" for %s type", value, typ.Name)
	case yang.Yempty:
		return "", nil
	case yang.Yidentityref:
		if s, ok := value.(string); ok {
			m, ok := schema.Identityref[s]
			if !ok {
				return "", fmt.Errorf("%s is not a value of %s", s, typ.Name)
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

type xmlNode struct {
	DataNode
	ConfigOnly yang.TriState
	printMeta  bool
	metaNS     map[string]string
}

func (xnode *xmlNode) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	schema := xnode.Schema()
	if node, ok := xnode.DataNode.(*DataNodeGroup); ok {
		for _, child := range node.Nodes {
			cxnode := *xnode
			cxnode.DataNode = child
			if err := e.EncodeElement(&cxnode, xml.StartElement{Name: xml.Name{Local: cxnode.Name() + "?"}}); err != nil {
				return err
			}
		}
		return nil
	}
	boundary := false
	if start.Name.Local != schema.Name {
		boundary = true
	} else if schema.Qboundary {
		boundary = true
	}
	// xmlns
	if boundary {
		ns := schema.Module.Namespace
		if ns != nil {
			start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: ns.Name})
			start.Name.Local = schema.Name
		}
	} else {
		start = xml.StartElement{Name: xml.Name{Local: schema.Name}}
	}

	// metadata
	if xnode.printMeta {
		if xnode.metaNS == nil {
			xnode.metaNS = make(map[string]string)
		}
		meta := xnode.DataNode.Metadata()
		for _, v := range meta {
			mschema := v.Schema()
			ns, prefix := mschema.GetNamespaceAndPrefix()
			if _prefix, ok := xnode.metaNS[ns]; !ok {
				nsattr := xml.Attr{Name: xml.Name{Local: "xmlns:" + prefix}, Value: ns}
				start.Attr = append(start.Attr, nsattr)
				xnode.metaNS[ns] = prefix
				start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: prefix + ":" + v.Name()}, Value: v.ValueString()})
			} else {
				start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: _prefix + ":" + v.Name()}, Value: v.ValueString()})
			}
		}
	}

	// if err := e.EncodeToken(xml.Comment(leaflist.ID())); err != nil {
	// 	return err
	// }
	switch node := xnode.DataNode.(type) {
	case *DataBranch:
		if err := e.EncodeToken(xml.Token(start)); err != nil {
			return err
		}
		for _, child := range node.children {
			cxnode := *xnode
			cxnode.DataNode = child
			if err := e.EncodeElement(&cxnode, xml.StartElement{Name: xml.Name{Local: cxnode.Name()}}); err != nil {
				return err
			}
		}
		return e.EncodeToken(xml.Token(xml.EndElement{Name: xml.Name{Local: schema.Name}}))
	case *DataLeafList:
		for i := range node.value {
			if err := e.EncodeElement(ValueToValueString(node.value[i]), start); err != nil {
				return err
			}
		}
		return nil
	case *DataLeaf:
		vstr, err := value2XMLString(schema, schema.Type, node.value)
		if err != nil {
			return err
		}
		return e.EncodeElement(vstr, start)
	case *DataNodeGroup:
		return fmt.Errorf("unexpected data node type %T", node)
	}
	return nil
}

// MarshalXML returns the XML bytes of a data node.
func MarshalXML(node DataNode, option ...Option) ([]byte, error) {
	xnode := &xmlNode{DataNode: node}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v is not allowed for marshalling", option[i])
		case ConfigOnly:
			xnode.ConfigOnly = yang.TSTrue
		case StateOnly:
			xnode.ConfigOnly = yang.TSFalse
		case RFC7951Format:
			return nil, fmt.Errorf("%v is not allowed for marshalling", option[i])
		case Metadata:
			xnode.printMeta = true
		}
	}
	return xml.Marshal(xnode)
}

// MarshalXMLIndent returns the XML bytes of a data node.
func MarshalXMLIndent(node DataNode, prefix, indent string, option ...Option) ([]byte, error) {
	xnode := &xmlNode{DataNode: node}
	for i := range option {
		switch option[i].(type) {
		case HasState:
			return nil, fmt.Errorf("%v is not allowed for marshalling", option[i])
		case ConfigOnly:
			xnode.ConfigOnly = yang.TSTrue
		case StateOnly:
			xnode.ConfigOnly = yang.TSFalse
		case RFC7951Format:
			return nil, fmt.Errorf("%v is not allowed for marshalling", option[i])
		case Metadata:
			xnode.printMeta = true
		}
	}
	return xml.MarshalIndent(xnode, prefix, indent)
}

// UnmarshalXML updates the data node using an XML document.
func UnmarshalXML(node DataNode, data []byte, option ...Option) error {
	for i := range option {
		switch option[i].(type) {
		case RepresentItself:
			// xml node already represents itself.
		default:
			return fmt.Errorf("%s option not supported", option[i])
		}
	}
	return xml.Unmarshal(data, node)
}
