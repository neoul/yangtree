package yangtree

import (
	"fmt"
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
		// "modules/ietf-yang-library@2019-01-04.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	schema, err := Load(file, dir, excluded, SchemaOption{YANGLibrary2019: true})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	yanglib := schema.GetYangLibrary()
	if yanglib == nil {
		t.Fatalf("failed to get yang library")
	}
	y, err := MarshalYAML(yanglib, RFC7951Format{})
	if err != nil {
		t.Fatalf("error in marshalling: %v", err)
	}
	fmt.Println(string(y))
	// y, err := MarshalJSONIndent(yanglib, "", " ")
	// if err != nil {
	// 	t.Fatalf("error in marshalling: %v", err)
	// }
	// fmt.Println(string(y))
}
