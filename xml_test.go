package yangtree

import (
	"encoding/xml"
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
	// fmt.Println(string(xmlstr))
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
	root1, err := New(schema)
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
	if err := UnmarshalYAML(root1, b); err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}
	// xmlstr, _ := MarshalXMLIndent(root1, "", " ", Metadata{})
	// fmt.Println(string(xmlstr))

	root2, err := New(schema)
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
	if err := xml.Unmarshal(b, root2); err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}
	j1, _ := MarshalJSON(root1, Metadata{})
	j2, _ := MarshalJSON(root2, Metadata{})
	if string(j1) != string(j2) {
		t.Errorf("different result: root1 %s\n", string(j1))
		t.Errorf("different result: root2 %s\n", string(j2))
	}
}
