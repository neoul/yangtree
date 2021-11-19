package yangtree

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

// type XMLInterface interface {
// 	ToXML(e *xml.Encoder, start xml.StartElement) error
// 	FromXML(d *xml.Decoder, start xml.StartElement) error
// }

func TestXML(t *testing.T) {
	moduleSetNum = 0
	yangfiles := []string{
		"../../YangModels/yang/standard/ietf/RFC/ietf-interfaces.yang",
		"../../YangModels/yang/standard/ietf/RFC/iana-if-type@2017-01-19.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{}
	schema, err := Load(yangfiles, dir, excluded, YANGTreeOption{YANGLibrary2019: true})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	yanglib := schema.GetYangLibrary()
	if yanglib == nil {
		t.Fatalf("failed to get yang library")
	}
	// y, err := MarshalYAML(yanglib, RFC7951Format{})
	// if err != nil {
	// 	t.Fatalf("error in marshalling: %v", err)
	// }
	// fmt.Println(string(y))

	// j, err := MarshalJSON(yanglib, RFC7951Format{})
	// if err != nil {
	// 	t.Fatalf("error in marshalling: %v", err)
	// }
	// fmt.Println(string(j))

	xmlstr, _ := xml.MarshalIndent(yanglib, "", " ")
	newyanglib, err := NewWithValueString(yanglib.Schema())
	if err != nil {
		t.Fatalf("error in new: %v", err)
	}
	if err := xml.Unmarshal(xmlstr, newyanglib); err != nil {
		t.Fatalf("error in new: %v", err)
	}
	if !Equal(yanglib, newyanglib) {
		t.Error("invalid xml marshalling & unmarshalling")
	}
}

func TestXML2(t *testing.T) {
	moduleSetNum = 0
	yangfiles := []string{
		"testdata/sample/sample.yang",
		"testdata/modules/example-last-modified.yang",
		"../../YangModels/yang/standard/ietf/RFC/ietf-interfaces.yang",
		"../../YangModels/yang/standard/ietf/RFC/iana-if-type@2017-01-19.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{}
	schema, err := Load(yangfiles, dir, excluded, YANGTreeOption{SingleLeafList: true})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	root, err := New(schema)
	if err != nil {
		t.Fatalf("error in new yangtree: %v", err)
	}
	var file *os.File
	file, err = os.Open("testdata/yaml/sample-metadata.yaml")
	if err != nil {
		t.Errorf("file open err: %v\n", err)
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		t.Errorf("file read error: %v\n", err)
	}
	file.Close()
	if err := UnmarshalYAML(root, b); err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}
	xmlstr, _ := MarshalXMLIndent(root, "", " ", Metadata{})
	fmt.Println(string(xmlstr))

	r, err := New(schema)
	if err != nil {
		t.Fatalf("error in new yangtree: %v", err)
	}

	file, err = os.Open("testdata/xml/sample.xml")
	if err != nil {
		t.Errorf("file open err: %v\n", err)
	}
	b, err = ioutil.ReadAll(file)
	if err != nil {
		t.Errorf("file read error: %v\n", err)
	}
	file.Close()
	if err := xml.Unmarshal(b, r); err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}
	// fmt.Println(r.Value())

	// xmlstr, _ = xml.MarshalIndent(root, "", " ")
	// fmt.Println(string(xmlstr))
}
