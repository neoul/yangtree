package yangtree

import (
	"fmt"
	"io/ioutil"
	"log"
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

func TestType(t *testing.T) {
	yangfiles := []string{
		"testdata/sample/sample.yang",
	}
	// dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	schema, err := Load(yangfiles, nil, nil)
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	bitsSchema := schema.FindSchema("/sample/bits-val")
	if bitsSchema == nil {
		t.Fatal("error in finding a bits schema")
	}
	node, err := NewWithValueString(bitsSchema, "zero two one")
	if err != nil {
		t.Fatalf("error in creating a bits node: %v", err)
	}
	if err := node.SetValue("zero three"); err == nil {
		t.Fatalf(`it must be failed due to the bits %s`, "three")
	}
	if node.ValueString() != "zero one two" {
		t.Fatal("unexpected bits set")
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

	// configure /restconf/data schema to insert any child node into the in the node.
	yangapiSchema.FindSchema("/restconf/data").ContainAny = true
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
	bval, err := NewWithValueString(schema.FindSchema("/sample/bits-val"), "one zero")
	if err != nil {
		t.Fatalf("error in creating bval: %v", err)
	}
	data, _ := Find(node, "/restconf/data")
	if _, err := data[0].Insert(bval, nil); err != nil {
		t.Fatalf("error in updating bits-val to restconf/data: %v", err)
	}

	j, err := MarshalJSON(node)
	if err != nil {
		t.Fatalf("error in marshalling an extension data node: %v", err)
	}
	if string(j) != `{"restconf":{"data":{"bits-val":"zero one"},"operations":{},"yang-library-version":"2016-06-21"}}` {
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

		if y[i], err = MarshalYAML(root[i], Metadata{}, RFC7951Format{}); err != nil {
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

func TestRESTCONF(t *testing.T) {
	moduleSetNum = 0
	yangfiles := []string{
		"../open-restconf/modules/ietf-restconf@2017-01-26.yang",
		"../open-restconf/modules/example/example-jukebox.yang",
		"../open-restconf/modules/example/example-mod.yang",
		"../open-restconf/modules/example/example-ops.yang",
		"../open-restconf/modules/example/example-actions.yang",
	}
	dir := []string{"modules"}
	excluded := []string{}
	schema, err := Load(yangfiles, dir, excluded, YANGTreeOption{YANGLibrary2016: true})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	schemaData := schema.ExtSchema["yang-api"].GetSchema("restconf").GetSchema("data")
	if schemaData == nil {
		log.Fatalf("restconf: unable to load restconf schema")
	}
	for i := range schema.Children {
		if schema.Children[i].RPC == nil {
			schemaData.Append(true, schema.Children[i])
		}
	}

	root := make([]DataNode, 3)
	filesuffix := []string{"xml", "json", "yaml"}
	for i := range filesuffix {
		root[i], err = New(schemaData)
		if err != nil {
			t.Fatalf("error in new yangtree: %v", err)
		}
		var file *os.File
		file, err = os.Open("../open-restconf/testdata/jukebox." + filesuffix[i])
		if err != nil {
			t.Fatalf("restconf: %v", err)
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			t.Fatalf("restconf: %v", err)
		}
		file.Close()
		switch filesuffix[i] {
		case "xml":
			if err := UnmarshalXML(root[i], b, RepresentItself{}); err != nil {
				t.Fatalf("restconf: %v", err)
			}
		case "json":
			if err := UnmarshalJSON(root[i], b, RepresentItself{}); err != nil {
				t.Fatalf("restconf: %v", err)
			}
		case "yaml":
			if err := UnmarshalYAML(root[i], b, RepresentItself{}); err != nil {
				t.Fatalf("restconf: %v", err)
			}
		}

		if i > 1 {
			if !Equal(root[i-1], root[i]) {
				t.Errorf("unmarshalled restconf data is not equal (%s %s)", filesuffix[i-1], filesuffix[i])
			}
		}
	}

	nodes, _ := Find(root[0], "jukebox/library/artist")
	group, _ := ConvertToGroup(nodes[0].Schema(), nodes)
	if b, err := MarshalXMLIndent(group, " ", "  "); err == nil {
		fmt.Printf(string(b))
	}
	if b, err := MarshalJSONIndent(group, "", " "); err == nil {
		fmt.Printf(string(b))
	}
	if b, err := MarshalYAMLIndent(group, "", " "); err == nil {
		fmt.Printf(string(b))
	}
	// if b, err := xml.MarshalIndent(group, "", " "); err == nil {
	// 	fmt.Printf(string(b))
	// }
	// if b, err := xml.MarshalIndent(nodes[0], " ", " "); err == nil {
	// 	fmt.Printf(string(b))
	// }

	// for _, n := range nodes {
	// 	if b, err := MarshalXMLIndent(n, "", " ", RepresentItself{}); err == nil {
	// 		fmt.Printf(string(b))
	// 	}
	// }
}
