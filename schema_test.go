package yangtree

import (
	"io/ioutil"
	"os"
	"strings"
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

// Test for DataNodes defined using a YANG extension
func TestYANGExtDataNode(t *testing.T) {
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
	yanglib := schema.GetYangLibrary()
	if yanglib == nil {
		t.Fatalf("failed to get yang library")
	}
	yanglibrev, err := Find(yanglib, "module[name=ietf-yang-library]/revision")
	if err != nil {
		t.Fatalf("failed to get yang library revision")
	}
	// last-modified
	lastmodifiedSchema := schema.ExtSchema["last-modified"]
	node, err := NewWithValueString(lastmodifiedSchema, "2021-11-02T12:56:00Z")
	if err != nil {
		t.Fatalf("error in creating an last-modified extension data node: %v", err)
	}
	if node.ValueString() != "2021-11-02T12:56:00Z" {
		t.Fatalf("error in storing an last-modified extension data node: %v", err)
	}
	// yang-api
	yangapiSchema := schema.ExtSchema["yang-api"]
	node, err = NewWithValueString(yangapiSchema)
	if err != nil {
		t.Fatalf("error in creating an last-modified extension data node: %v", err)
	}
	if err := node.SetValue(map[interface{}]interface{}{
		"restconf": map[interface{}]interface{}{
			"data":                 map[interface{}]interface{}{},
			"operations":           nil,
			"yang-library-version": yanglibrev[0].ValueString(),
		},
	}); err != nil {
		t.Fatalf("error in creating an yang-lib extension data node: %v", err)
	}

	j, err := MarshalJSON(node)
	if err != nil {
		t.Fatalf("error in marshalling an extension data node: %v", err)
	}
	if string(j) != `{"restconf":{"data":{},"operations":{},"yang-library-version":"2016-06-21"}}` {
		t.Fatalf("error in storing an yang-api extension data node: %v", string(j))
	}
}

func TestYANGMetaData(t *testing.T) {
	yangfiles := []string{
		"testdata/sample/sample.yang",
		"testdata/modules/example-last-modified.yang",
		// This yang metadata schema is loaded by default.
		// "modules/ietf-yang-metadata@2016-08-05.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}

	var err error
	schema := make([]*SchemaNode, 2)
	root := make([]DataNode, 2)
	j := make([][]byte, 2)
	y := make([][]byte, 2)
	schema[0], err = Load(yangfiles, dir, excluded)
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}

	schema[1], err = Load(yangfiles, dir, excluded, YANGTreeOption{SingleLeafList: true})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}

	root[0], err = New(schema[0])
	if err != nil {
		t.Fatalf("error in new yangtree: %v", err)
	}
	root[1], err = New(schema[1])
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
	if err := UnmarshalYAML(root[0], b); err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}
	if err := UnmarshalYAML(root[1], b); err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}
	// root[0]: multiple leaf-list schema is enabled. When it is enabled, leaf-list values are separated to multiple leaf-list nodes as a leaf node.
	// root[1]: single leaf-list schema is enabled. When it is enabled, leaf-list values are gathered together to a single leaf-list node.

	// Metadata access using path
	// /sample/@last-modified
	// /sample/container-val/a/@last-modified
	// /sample/container-val/@last-modified
	// /sample/multiple-key-list[str=first][integer=1]/@last-modified
	// /sample/non-key-list[0]/@last-modified

	tests := []struct {
		path          string
		value         string
		wantInsertErr bool
		wantDeleteErr bool
	}{
		{wantInsertErr: false, path: "/sample/@last-modified", value: "2015-06-18T17:01:14+02:01"},
		{wantInsertErr: false, path: "/sample/container-val/a/@last-modified", value: "2015-06-18T17:01:14+02:02"},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val/@last-modified", value: "2015-06-18T17:01:14+02:04"},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val[.=leaf-list-second]/@last-modified", value: "2015-06-18T17:01:14+02:03"},
		{wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=2]/@last-modified", value: "2015-06-18T17:01:14+02:05"},
		{wantInsertErr: false, path: "/sample/str-val/@last-modified", value: "2015-06-18T17:01:14+02:06"},
		{wantInsertErr: false, path: "/sample/non-key-list[1]/@last-modified", value: "2015-06-18T17:01:14+02:07"},
	}
	for i := 0; i < 2; i++ {
		for _, tt := range tests {
			t.Run("set-metadata."+tt.path, func(t *testing.T) {
				err := SetValueString(root[i], tt.path, nil, tt.value)
				if (err != nil) != tt.wantInsertErr {
					t.Errorf("SetValueString() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.path)
				}
			})
		}
		if err := Validate(root[i]); err != nil {
			t.Error(err)
		}

		if j[i], err = MarshalJSON(root[i], Metadata{}, RFC7951Format{}); err != nil {
			t.Errorf("error in marshalling metadata: %v", err)
		}
		// fmt.Println(string(j[i]))

		if y[i], err = MarshalYAML(root[i], Metadata{}); err != nil {
			t.Errorf("error in marshalling metadata: %v", err)
		}
	}

	unmarshallingMetaTests := []struct {
		root   DataNode
		expect []byte
		file   string
	}{
		{root: root[0], expect: j[0], file: "testdata/json/sample-metadata-rfc7951.json"},
		{root: root[1], expect: j[1], file: "testdata/json/sample-metadata.json"},
		{root: root[0], expect: y[0], file: "testdata/yaml/sample-metadata-rfc7951.yaml"},
		{root: root[1], expect: y[1], file: "testdata/yaml/sample-metadata.yaml"},
	}
	for _, tt := range unmarshallingMetaTests {
		t.Run("unmarshal-metadata."+tt.file, func(t *testing.T) {
			r, err := New(tt.root.Schema())
			if err != nil {
				t.Errorf("error in creating a node: %v", err)
				return
			}
			var file *os.File
			file, err = os.Open(tt.file)
			if err != nil {
				t.Errorf("file open err: %v\n", err)
				return
			}
			b, err := ioutil.ReadAll(file)
			if err != nil {
				t.Errorf("file read error: %v\n", err)
				return
			}
			file.Close()
			if strings.HasSuffix(tt.file, ".json") {
				if err := UnmarshalJSON(r, b); err != nil {
					t.Errorf("error in unmarshalling metadata: %v", err)
					return
				}
				jj, err := MarshalJSON(r, Metadata{}, RFC7951Format{})
				if err != nil {
					t.Errorf("error in marshalling metadata: %v", err)
					return
				}
				if string(tt.expect) != string(jj) {
					t.Errorf("different unmarshalled data:")
					t.Errorf(" - A: %s", string(tt.expect))
					t.Errorf(" - B: %s", string(jj))
					return
				}
			} else if strings.HasSuffix(tt.file, ".yaml") {
				if err := UnmarshalYAML(r, b); err != nil {
					t.Errorf("error in unmarshalling metadata: %v", err)
					return
				}
				jj, err := MarshalYAML(r, Metadata{}, RFC7951Format{})
				if err != nil {
					t.Errorf("error in marshalling metadata: %v", err)
					return
				}
				if string(tt.expect) != string(jj) {
					t.Errorf("different unmarshalled data:")
					t.Errorf(" - A: %s", string(tt.expect))
					t.Errorf(" - B: %s", string(jj))
					return
				}
			}
		})
	}
}
