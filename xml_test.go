package yangtree

import (
	"encoding/xml"
	"testing"
)

// type XMLInterface interface {
// 	ToXML(e *xml.Encoder, start xml.StartElement) error
// 	FromXML(d *xml.Decoder, start xml.StartElement) error
// }

func TestXML(t *testing.T) {
	moduleSetNum = 0
	file := []string{
		"../../YangModels/yang/standard/ietf/RFC/ietf-interfaces.yang",
		"../../YangModels/yang/standard/ietf/RFC/iana-if-type@2017-01-19.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{}
	schema, err := Load(file, dir, excluded, SchemaOption{YANGLibrary2019: true})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	yanglib := schema.GetYangLibrary()
	if yanglib == nil {
		t.Fatalf("failed to get yang library")
	}
	xmlstr, _ := xml.MarshalIndent(yanglib, "", " ")
	newyanglib, err := NewDataNode(yanglib.Schema())
	if err != nil {
		t.Fatalf("error in new: %v", err)
	}
	if err := xml.Unmarshal(xmlstr, newyanglib); err != nil {
		t.Fatalf("error in new: %v", err)
	}
	if !Equal(yanglib, newyanglib) {
		t.Error("invalid xml marshalling & unmarshalling")
	}
	// y, err := MarshalYAML(yanglib, RFC7951Format{})
	// if err != nil {
	// 	t.Fatalf("error in marshalling: %v", err)
	// }
	// fmt.Println(string(y))
}
