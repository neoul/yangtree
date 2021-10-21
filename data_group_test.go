package yangtree

import (
	"encoding/json"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestNewDataGroup(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	{
		jleaflist := `["leaf-list-first","leaf-list-fourth","leaf-list-second","leaf-list-third"]`
		leafListSchema := RootSchema.FindSchema("sample/container-val/leaf-list-val")
		jleaflistnodes, err := NewDataNodeGroup(leafListSchema, jleaflist)
		if err != nil {
			t.Fatal(err)
		}
		j, err := json.Marshal(jleaflistnodes)
		if err != nil {
			t.Fatal(err)
		}
		y, err := yaml.Marshal(jleaflistnodes)
		if err != nil {
			t.Fatal(err)
		}
		// y2, err := MarshalYAML(jleaflistnodes)
		// if err != nil {
		// 	t.Fatal(err)
		// }
		// fmt.Println(string(y2))
		var o1, o2 interface{}
		if err := json.Unmarshal(j, &o1); err != nil {
			t.Fatal(err)
		}
		if err := yaml.Unmarshal(y, &o2); err != nil {
			t.Fatal(err)
		}
		// if err := yaml.Unmarshal(y2, &o3); err != nil {
		// 	t.Fatal(err)
		// }
		// pretty.Println("json:", o1)
		// pretty.Println("yaml:", o2)
		// pretty.Println("yaml:", o3)
		if !reflect.DeepEqual(o1, o2) {
			t.Error("not equal leaf-list nodes")
			t.Error("json:", o1)
			t.Error("yaml:", o1)
			return
		}
	}
	{
		jlist := `[
			{
				"country-code": "KR",
				"decimal-range": 1.01,
				"empty-node": [
				 null
				],
				"list-key": "BBB"
			},
			{"list-key":"CCC"},
			{
			 "country-code": "KR",
			 "decimal-range": 1.01,
			 "empty-node": [
			  null
			 ],
			 "list-key": "AAA",
			 "uint32-range": 100,
			 "uint64-node": "1234567890"
			}
			]`

		schema := RootSchema.FindSchema("sample/single-key-list")
		jlistnodes, err := NewDataNodeGroup(schema, jlist)
		if err != nil {
			t.Fatal(err)
		}
		j, err := json.Marshal(jlistnodes)
		if err != nil {
			t.Fatal(err)
		}
		y, err := yaml.Marshal(jlistnodes)
		if err != nil {
			t.Fatal(err)
		}
		// y2, err := MarshalYAML(jlistnodes, RFC7951Format{})
		// if err != nil {
		// 	t.Fatal(err)
		// }
		// fmt.Println(string(y2))
		var o1, o2 interface{}
		if err := json.Unmarshal(j, &o1); err != nil {
			t.Fatal(err)
		}
		if err := yaml.Unmarshal(y, &o2); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(o1, o2) {
			t.Error("not equal list nodes")
			t.Log("json:", o1)
			t.Log("yaml:", o2)
			return
		}
	}
}
