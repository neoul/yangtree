package yangtree

import (
	"fmt"
	"testing"
)

func TestDiffUpdate(t *testing.T) {
	files := []string{
		"../../YangModels/yang/standard/ietf/RFC/iana-if-type@2017-01-19.yang",
		"../../openconfig/public/release/models/interfaces/openconfig-interfaces.yang",
		"../../openconfig/public/release/models/system/openconfig-messages.yang",
		"../../openconfig/public/release/models/telemetry/openconfig-telemetry.yang",
		"../../openconfig/public/release/models/openflow/openconfig-openflow.yang",
		"../../openconfig/public/release/models/platform/openconfig-platform.yang",
		"../../openconfig/public/release/models/system/openconfig-system.yang",
	}
	dir := []string{"../../openconfig/public/", "../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	rootschema, err := Load(files, dir, excluded)
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewWithValueString(rootschema)
	if err != nil {
		t.Fatal(err)
	}
	schema := rootschema.FindSchema("interfaces/interface")
	for i := 1; i < 5; i++ {
		v := fmt.Sprintf(`{"name":"e%d", "config": {"enabled":"true"}}`, i)
		new, err := NewWithValueString(schema, v)
		if err != nil {
			t.Error(err)
		}
		err = Replace(root, "/interfaces/interface", new)
		if err != nil {
			t.Error(err)
		}
	}
	prev := Clone(root)
	for i := 3; i < 7; i++ {
		v := `{ "config": {"enabled":"false"}, "state": {"admin-status":"DOWN"}}`
		new, err := NewWithValueString(schema, v)
		if err != nil {
			t.Error(err)
		}
		err = Replace(root, fmt.Sprintf("interfaces/interface[name=e%v]", i), new)
		if err != nil {
			t.Error(err)
		}
	}
	ifnodes, err := Find(root, "interfaces/interface")
	if err != nil {
		t.Error(err)
	}
	if len(ifnodes) != 6 {
		t.Errorf("expected num: %d, got: %d", 6, len(ifnodes))
	}
	created, replaced := DiffUpdated(prev, root, false)
	// created
	t.Log("[created]", len(created))
	for i := range created {
		b, _ := MarshalJSON(created[i])
		t.Log(created[i].Path(), string(b))
	}
	if len(created) != 16 {
		t.Errorf("expected num: %d, got: %d", 16, len(created))
	}
	t.Log("[replaced]", len(replaced))
	for i := range replaced {
		b, _ := MarshalJSON(replaced[i])
		t.Log(replaced[i].Path(), string(b))
	}
	if len(replaced) != 2 {
		t.Errorf("expected num: %d, got: %d", 2, len(replaced))
	}

	b, _ := MarshalJSON(root)
	t.Log(string(b))
}
