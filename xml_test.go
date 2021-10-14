package yangtree

import (
	"encoding/xml"
	"fmt"
	"testing"
)

// type XMLInterface interface {
// 	ToXML(e *xml.Encoder, start xml.StartElement) error
// 	FromXML(d *xml.Decoder, start xml.StartElement) error
// }

func TestXML(t *testing.T) {
	moduleSetNum = 0
	file := []string{
		"../../YangModels/yang/standard/ietf/RFC/ietf-interfaces.yang",
		"../../YangModels/yang/standard/ietf/RFC/iana-if-type@2017-01-19.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{}
	schema, err := Load(file, dir, excluded, SchemaOption{YANGLibrary2019: true, SingleLeafList: true})
	if err != nil {
		t.Fatalf("error in loading: %v", err)
	}
	yanglib := schema.GetYangLibrary()
	if yanglib == nil {
		t.Fatalf("failed to get yang library")
	}
	xmlstr, _ := xml.MarshalIndent(yanglib, "", " ")
	fmt.Println(string(xmlstr))

	// expected := `<yang-library xmlns="urn:ietf:params:xml:ns:yang:ietf-yang-library"><content-id>PTl8fxNoLYAg/XagzNZqjJoce5g=</content-id><module-set><import-only-module><name>iana-if-type</name><namespace>urn:ietf:params:xml:ns:yang:iana-if-type</namespace><revision>2017-01-19</revision></import-only-module><import-only-module><name>ietf-datastores</name><namespace>urn:ietf:params:xml:ns:yang:ietf-datastores</namespace><revision>2018-01-11</revision></import-only-module><import-only-module><name>ietf-inet-types</name><namespace>urn:ietf:params:xml:ns:yang:ietf-inet-types</namespace><revision>2013-07-15</revision></import-only-module><import-only-module><name>ietf-interfaces</name><namespace>urn:ietf:params:xml:ns:yang:ietf-interfaces</namespace><revision>2018-02-20</revision></import-only-module><import-only-module><name>ietf-yang-metadata</name><namespace>urn:ietf:params:xml:ns:yang:ietf-yang-metadata</namespace><revision>2016-08-05</revision></import-only-module><import-only-module><name>ietf-yang-types</name><namespace>urn:ietf:params:xml:ns:yang:ietf-yang-types</namespace><revision>2013-07-15</revision></import-only-module><import-only-module><name>openconfig-aaa-types</name><namespace>http://openconfig.net/yang/aaa/types</namespace><revision>2018-11-21</revision></import-only-module><import-only-module><name>openconfig-aaa</name><namespace>http://openconfig.net/yang/aaa</namespace><revision>2020-07-30</revision><submodule><name>openconfig-aaa-radius</name><revision>2020-07-30</revision></submodule><submodule><name>openconfig-aaa-tacacs</name><revision>2020-07-30</revision></submodule></import-only-module><import-only-module><name>openconfig-alarm-types</name><namespace>http://openconfig.net/yang/alarms/types</namespace><revision>2018-11-21</revision></import-only-module><import-only-module><name>openconfig-extensions</name><namespace>http://openconfig.net/yang/openconfig-ext</namespace><revision>2020-06-16</revision></import-only-module><import-only-module><name>openconfig-inet-types</name><namespace>http://openconfig.net/yang/types/inet</namespace><revision>2020-10-12</revision></import-only-module><import-only-module><name>openconfig-license</name><namespace>http://openconfig.net/yang/license</namespace><revision>2020-04-22</revision></import-only-module><import-only-module><name>openconfig-openflow-types</name><namespace>http://openconfig.net/yang/openflow/types</namespace><revision>2020-06-30</revision></import-only-module><import-only-module><name>openconfig-platform-types</name><namespace>http://openconfig.net/yang/platform-types</namespace><revision>2019-06-03</revision></import-only-module><import-only-module><name>openconfig-procmon</name><namespace>http://openconfig.net/yang/system/procmon</namespace><revision>2019-03-15</revision></import-only-module><import-only-module><name>openconfig-system-logging</name><namespace>http://openconfig.net/yang/system/logging</namespace><revision>2018-11-21</revision></import-only-module><import-only-module><name>openconfig-system-management</name><namespace>http://openconfig.net/yang/system/management</namespace><revision>2020-01-14</revision></import-only-module><import-only-module><name>openconfig-system-terminal</name><namespace>http://openconfig.net/yang/system/terminal</namespace><revision>2018-11-21</revision></import-only-module><import-only-module><name>openconfig-telemetry-types</name><namespace>http://openconfig.net/yang/telemetry-types</namespace><revision>2018-11-21</revision></import-only-module><import-only-module><name>openconfig-types</name><namespace>http://openconfig.net/yang/openconfig-types</namespace><revision>2019-04-16</revision></import-only-module><import-only-module><name>openconfig-yang-types</name><namespace>http://openconfig.net/yang/types/yang</namespace><revision>2020-06-30</revision></import-only-module><module><name>ietf-yang-library</name><namespace>urn:ietf:params:xml:ns:yang:ietf-yang-library</namespace><revision>2019-01-04</revision></module><module><name>openconfig-alarms</name><namespace>http://openconfig.net/yang/alarms</namespace><revision>2019-07-09</revision></module><module><name>openconfig-interfaces</name><namespace>http://openconfig.net/yang/interfaces</namespace><revision>2019-11-19</revision></module><module><name>openconfig-messages</name><namespace>http://openconfig.net/yang/messages</namespace><revision>2018-08-13</revision></module><module><name>openconfig-openflow</name><namespace>http://openconfig.net/yang/openflow</namespace><revision>2018-11-21</revision></module><module><name>openconfig-platform</name><namespace>http://openconfig.net/yang/platform</namespace><revision>2019-04-16</revision></module><module><name>openconfig-simple-augment</name><namespace>urn:a</namespace><revision>2021-08-05</revision></module><module><name>openconfig-simple-deviation</name><namespace>urn:oc-simple-deviation</namespace><revision>2021-08-05</revision></module><module><deviation>openconfig-simple-deviation</deviation><name>openconfig-simple-target</name><namespace>urn:t</namespace><revision>2021-08-05</revision></module><module><name>openconfig-system</name><namespace>http://openconfig.net/yang/system</namespace><revision>2020-03-25</revision></module><module><name>openconfig-telemetry</name><namespace>http://openconfig.net/yang/telemetry</namespace><revision>2018-11-21</revision></module><module><name>yangtree</name><namespace>https://github.com/neoul/yangtree</namespace><revision>2020-08-18</revision></module><name>set-1</name></module-set></yang-library>`
	// // c := NewDataNodeCollector()
	// // c.Insert(yanglib, nil)
	// if xmlstr, err := xml.Marshal(yanglib); err != nil {
	// 	t.Fatalf("failed to get error: %v", err)
	// } else if string(xmlstr) != expected {
	// 	t.Errorf("expect: %s", expected)
	// 	t.Errorf("output: %s", xmlstr)
	// }

}
