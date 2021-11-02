package yangtree

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestYANGLibrary(t *testing.T) {
	moduleSetNum = 0
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
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	schema, err := Load(file, dir, excluded, YANGTreeOption{YANGLibrary2019: true, SchemaSetName: "mySchema"})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	yanglib := schema.GetYangLibrary()
	if yanglib == nil {
		t.Fatalf("failed to get yang library")
	}

	j, err := MarshalJSON(yanglib)
	if err != nil {
		t.Fatalf("error in marshalling: %v", err)
	}
	f, err := os.Open("testdata/json/yanglib.json")
	if err != nil {
		t.Fatalf("err in file open: %v", err)
	}

	jbyte, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("err in file read: %v", err)
	}
	f.Close()
	expected := strings.ReplaceAll(string(jbyte), " ", "")
	expected = strings.ReplaceAll(expected, "\t", "")
	expected = strings.ReplaceAll(expected, "\n", "")
	if expected != string(j) {
		t.Errorf("unexpected json marshalling:")
		t.Errorf("  expected: %s", expected)
		t.Errorf("       got: %s", string(j))
	}
}
