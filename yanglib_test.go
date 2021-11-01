package yangtree

import (
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
	schema, err := Load(file, dir, excluded, SchemaOption{YANGLibrary2019: true, SchemaSetName: "mySchema"})
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
	expected := `{"content-id":"umjPQjRCZnGvXt1qZ6YO+5gIJYc=","module-set":{"mySchema":{"import-only-module":{"iana-if-type":{"2017-01-19":{"name":"iana-if-type","namespace":"urn:ietf:params:xml:ns:yang:iana-if-type","revision":"2017-01-19"}},"ietf-datastores":{"2018-01-11":{"name":"ietf-datastores","namespace":"urn:ietf:params:xml:ns:yang:ietf-datastores","revision":"2018-01-11"}},"ietf-inet-types":{"2013-07-15":{"name":"ietf-inet-types","namespace":"urn:ietf:params:xml:ns:yang:ietf-inet-types","revision":"2013-07-15"}},"ietf-interfaces":{"2018-02-20":{"name":"ietf-interfaces","namespace":"urn:ietf:params:xml:ns:yang:ietf-interfaces","revision":"2018-02-20"}},"ietf-yang-metadata":{"2016-08-05":{"name":"ietf-yang-metadata","namespace":"urn:ietf:params:xml:ns:yang:ietf-yang-metadata","revision":"2016-08-05"}},"ietf-yang-types":{"2013-07-15":{"name":"ietf-yang-types","namespace":"urn:ietf:params:xml:ns:yang:ietf-yang-types","revision":"2013-07-15"}},"openconfig-aaa":{"2020-07-30":{"name":"openconfig-aaa","namespace":"http://openconfig.net/yang/aaa","revision":"2020-07-30","submodule":{"openconfig-aaa-radius":{"name":"openconfig-aaa-radius","revision":"2020-07-30"},"openconfig-aaa-tacacs":{"name":"openconfig-aaa-tacacs","revision":"2020-07-30"}}}},"openconfig-aaa-types":{"2018-11-21":{"name":"openconfig-aaa-types","namespace":"http://openconfig.net/yang/aaa/types","revision":"2018-11-21"}},"openconfig-alarm-types":{"2018-11-21":{"name":"openconfig-alarm-types","namespace":"http://openconfig.net/yang/alarms/types","revision":"2018-11-21"}},"openconfig-extensions":{"2020-06-16":{"name":"openconfig-extensions","namespace":"http://openconfig.net/yang/openconfig-ext","revision":"2020-06-16"}},"openconfig-inet-types":{"2020-10-12":{"name":"openconfig-inet-types","namespace":"http://openconfig.net/yang/types/inet","revision":"2020-10-12"}},"openconfig-license":{"2020-04-22":{"name":"openconfig-license","namespace":"http://openconfig.net/yang/license","revision":"2020-04-22"}},"openconfig-openflow-types":{"2020-06-30":{"name":"openconfig-openflow-types","namespace":"http://openconfig.net/yang/openflow/types","revision":"2020-06-30"}},"openconfig-platform-types":{"2019-06-03":{"name":"openconfig-platform-types","namespace":"http://openconfig.net/yang/platform-types","revision":"2019-06-03"}},"openconfig-procmon":{"2019-03-15":{"name":"openconfig-procmon","namespace":"http://openconfig.net/yang/system/procmon","revision":"2019-03-15"}},"openconfig-system-logging":{"2018-11-21":{"name":"openconfig-system-logging","namespace":"http://openconfig.net/yang/system/logging","revision":"2018-11-21"}},"openconfig-system-management":{"2020-01-14":{"name":"openconfig-system-management","namespace":"http://openconfig.net/yang/system/management","revision":"2020-01-14"}},"openconfig-system-terminal":{"2018-11-21":{"name":"openconfig-system-terminal","namespace":"http://openconfig.net/yang/system/terminal","revision":"2018-11-21"}},"openconfig-telemetry-types":{"2018-11-21":{"name":"openconfig-telemetry-types","namespace":"http://openconfig.net/yang/telemetry-types","revision":"2018-11-21"}},"openconfig-types":{"2019-04-16":{"name":"openconfig-types","namespace":"http://openconfig.net/yang/openconfig-types","revision":"2019-04-16"}},"openconfig-yang-types":{"2020-06-30":{"name":"openconfig-yang-types","namespace":"http://openconfig.net/yang/types/yang","revision":"2020-06-30"}}},"module":{"ietf-yang-library":{"name":"ietf-yang-library","namespace":"urn:ietf:params:xml:ns:yang:ietf-yang-library","revision":"2019-01-04"},"openconfig-alarms":{"name":"openconfig-alarms","namespace":"http://openconfig.net/yang/alarms","revision":"2019-07-09"},"openconfig-interfaces":{"name":"openconfig-interfaces","namespace":"http://openconfig.net/yang/interfaces","revision":"2019-11-19"},"openconfig-messages":{"name":"openconfig-messages","namespace":"http://openconfig.net/yang/messages","revision":"2018-08-13"},"openconfig-openflow":{"name":"openconfig-openflow","namespace":"http://openconfig.net/yang/openflow","revision":"2018-11-21"},"openconfig-platform":{"name":"openconfig-platform","namespace":"http://openconfig.net/yang/platform","revision":"2019-04-16"},"openconfig-simple-augment":{"name":"openconfig-simple-augment","namespace":"urn:a","revision":"2021-08-05"},"openconfig-simple-deviation":{"name":"openconfig-simple-deviation","namespace":"urn:oc-simple-deviation","revision":"2021-08-05"},"openconfig-simple-target":{"deviation":["openconfig-simple-deviation"],"name":"openconfig-simple-target","namespace":"urn:t","revision":"2021-08-05"},"openconfig-system":{"name":"openconfig-system","namespace":"http://openconfig.net/yang/system","revision":"2020-03-25"},"openconfig-telemetry":{"name":"openconfig-telemetry","namespace":"http://openconfig.net/yang/telemetry","revision":"2018-11-21"},"yangtree":{"name":"yangtree","namespace":"https://github.com/neoul/yangtree","revision":"2020-08-18"}},"name":"mySchema"}}}`
	if expected != string(j) {
		t.Errorf("unexpected json marshalling:")
		t.Errorf("  expected: %s", expected)
		t.Errorf("       got: %s", string(j))
	}
}
