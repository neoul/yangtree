package yangtree

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	file := []string{
		"../../YangModels/yang/standard/ietf/RFC/iana-if-type@2017-01-19.yang",
		"../../openconfig/public/release/models/interfaces/openconfig-interfaces.yang",
		"../../openconfig/public/release/models/system/openconfig-messages.yang",
		"../../openconfig/public/release/models/telemetry/openconfig-telemetry.yang",
		"../../openconfig/public/release/models/openflow/openconfig-openflow.yang",
		"../../openconfig/public/release/models/platform/openconfig-platform.yang",
		"../../openconfig/public/release/models/system/openconfig-system.yang",
		"testdata/modules/openconfig-simple-target.yang",
		"testdata/modules/openconfig-simple-augment.yang",
		"testdata/modules/openconfig-simple-deviation.yang",
		"modules/ietf-yang-library@2016-06-21.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	_, err := Load(file, dir, excluded)
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
}

func TestYANGMetaData(t *testing.T) {
	yangfiles := []string{
		"testdata/sample/sample.yang",
		"testdata/modules/example-last-modified.yang",
		// "modules/ietf-yang-metadata@2016-08-05.yang", // built-in yangtree module
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	RootSchema, err := Load(yangfiles, dir, excluded)
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	RootData, err := NewWithValueString(RootSchema)
	if err != nil {
		t.Fatalf("error in new yangtree: %v", err)
	}
	var file *os.File
	file, err = os.Open("testdata/yaml/sample1.yaml")
	if err != nil {
		t.Errorf("file open err: %v\n", err)
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		t.Errorf("file read error: %v\n", err)
	}
	file.Close()
	if err := UnmarshalYAML(RootData, b); err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}

	// Metadata access using path
	// /sample/@last-modified
	// /sample/container-val/a
	// /sample/@container-val
	// /sample/@container-val
	// /sample/@multiple-key-list[str=first][integer=1]
	// /sample/@non-key-list[0]

	tests := []struct {
		path          string
		value         string
		wantInsertErr bool
		wantDeleteErr bool
	}{
		{wantInsertErr: false, path: "/sample/@last-modified", value: "2015-06-18T17:01:14+02:00"},
	}
	for _, tt := range tests {
		t.Run("SetValueString."+tt.path, func(t *testing.T) {
			err := SetValueString(RootData, tt.path, nil, tt.value)
			if (err != nil) != tt.wantInsertErr {
				t.Errorf("SetValueString() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.path)
			}
		})
	}
	if err := Validate(RootData); err != nil {
		t.Error(err)
	}
}

func TestYANGExtension(t *testing.T) {
	yangfiles := []string{
		"testdata/sample/sample.yang",
		"testdata/modules/example-last-modified.yang",
		"modules/ietf-restconf@2017-01-26.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	schema, err := Load(yangfiles, dir, excluded, YANGTreeOption{YANGLibrary2016: true})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	for key, extschema := range schema.Extension {
		fmt.Println(key, extschema.Name)
	}

	lastmodifiedSchema := schema.Extension["last-modified"]
	node, err := NewWithValueString(lastmodifiedSchema, "2021-11-02T12:56:00Z")
	if err != nil {
		t.Fatalf("error in creating an last-modified extension data node: %v", err)
	}
	if node.ValueString() != "2021-11-02T12:56:00Z" {
		t.Fatalf("error in storing an last-modified extension data node: %v", err)
	}
	yangapiSchema := schema.Extension["yang-api"]
	node, err = NewWithValueString(yangapiSchema)
	if err != nil {
		t.Fatalf("error in creating an last-modified extension data node: %v", err)
	}
	// node.CreateByMap(map[string]interface{}{"restconf": })

	fmt.Println(node.ValueString(), node.Value())

	yanglib := schema.GetYangLibrary()
	if yanglib == nil {
		t.Fatalf("failed to get yang library")
	}
	y, _ := MarshalYAML(yanglib)
	fmt.Println(string(y))
}
