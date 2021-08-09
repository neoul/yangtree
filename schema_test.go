package yangtree

import (
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

func TestYANGLibrary(t *testing.T) {
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
		"modules/ietf-yang-library@2019-01-04.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	_, err := Load(file, dir, excluded)
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
}
func TestYANGMetaData(t *testing.T) {
	file := []string{
		"testdata/modules/example-last-modified.yang",
		// "testdata/modules/openconfig-simple-target.yang",
		// "testdata/modules/openconfig-simple-augment.yang",
		// "testdata/modules/openconfig-simple-deviation.yang",
		// "modules/ietf-yang-metadata@2016-08-05.yang",
		// "modules/ietf-yang-metadata@2016-08-05.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	_, err := Load(file, dir, excluded)
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
}
