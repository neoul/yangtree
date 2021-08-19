package yangtree

import (
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
	RootData, err := New(RootSchema)
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
	if err := RootData.UnmarshalYAML(b); err != nil {
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
		value         []string
		wantInsertErr bool
		wantDeleteErr bool
	}{
		{wantInsertErr: false, path: "/sample/@last-modified", value: []string{"2015-06-18T17:01:14+02:00"}},
		// {wantInsertErr: true, path: "/sample/@last-modifiedx"}, // invalid
		// {wantInsertErr: false, path: "/sample/str-val", value: []string{"abc"}},
		// {wantInsertErr: false, path: "/sample/empty-val", value: []string{"true"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/", value: nil},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/country-code", value: []string{"KR"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/uint32-range", value: []string{"100"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/decimal-range", value: []string{"1.01"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/empty-node", value: nil},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/uint64-node[.=1234567890]", value: nil},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=BBB]/uint64-node[.=1234567890]", value: nil},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=BBB]/uint32-range", value: []string{"200"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=CCC]/uint32-range", value: []string{"300"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=DDD]/uint32-range", value: []string{"400"}},
		// {wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=1]/ok", value: []string{"true"}},
		// {wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=2]/str", value: []string{"first"}},
		// {wantInsertErr: false, path: "/sample/multiple-key-list[str=second][integer=1]/str", value: []string{"second"}},
		// {wantInsertErr: false, path: "/sample/multiple-key-list[sample:str=second][integer=2]/str", value: []string{"second"}},
		// {wantInsertErr: false, path: "/sample:sample/container-val", value: nil},
		// {wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: nil},
		// {wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: []string{"leaf-list-first", "leaf-list-second"}},
		// {wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: []string{"leaf-list-third"}},
		// {wantInsertErr: false, path: "/sample/container-val/leaf-list-val/leaf-list-fourth", value: nil},
		// {wantInsertErr: false, path: "/sample/container-val/leaf-list-val[.=leaf-list-fifth]", value: nil},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/sample:enum-val", value: []string{"enum2"}},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/sample:test-default", value: []string{"11"}},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/a", value: []string{"A"}},
		// {wantInsertErr: false, path: "/sample:sample/non-key-list", value: []string{`{"uintval": "11", "strval": "XYZ"}`}},
		// {wantInsertErr: false, path: "/sample:sample/non-key-list", value: []string{`{"uintval": "12", "strval": "XYZ"}`}},
		// {wantInsertErr: false, path: "/sample:sample/non-key-list[uintval=13][strval=ABC]", value: nil},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/test-instance-identifier", value: []string{"/sample:sample/sample:container-val/a"}},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/test-must", value: []string{"5"}},

		// {wantInsertErr: true, path: "/sample/single-key-list[list-ke=first]", value: []string{"true"}},
		// {wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/uint32-range", value: []string{"493"}},
		// {wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/int8-range", value: []string{"500"}},
	}
	for _, tt := range tests {
		t.Run("Set."+tt.path, func(t *testing.T) {
			err := Set(RootData, tt.path, tt.value...)
			if (err != nil) != tt.wantInsertErr {
				t.Errorf("Set() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.path)
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
	_, err := Load(yangfiles, dir, excluded)
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
}
