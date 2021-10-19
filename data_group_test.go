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
	jcontainer := `
	{
		"container-val": {
			"a": "A",
			"enum-val": "enum2",
			"leaf-list-val": [
				"leaf-list-first",
				"leaf-list-fourth",
				"leaf-list-second",
				"leaf-list-third"
			],
			"test-default": 11
		},
		"empty-val": null,
		"multiple-key-list": {
			"first": {
				"1": {
					"integer": 1,
					"ok": true,
					"str": "first"
				},
				"2": {
					"integer": 2,
					"str": "first"
				}
			}
		},
		"non-key-list": [
			{
				"strval": "XYZ",
				"uintval": 10
			}
		],
		"single-key-list": {
			"AAA": {
				"country-code": "KR",
				"decimal-range": 1.01,
				"empty-node": null,
				"list-key": "AAA",
				"uint32-range": 100,
				"uint64-node": 1234567890
			}
		},
		"str-val": "abc"
	}
	`
	schema := RootSchema.FindSchema("sample")
	jcontainernodes, err := NewDataNodeGroup(schema, jcontainer)
	if err != nil {
		t.Fatal(err)
	}
	if y, err := MarshalYAML(jcontainernodes); err == nil {
		t.Log("\n", string(y))
	}

	jleaflist := `["leaf-list-first","leaf-list-fourth","leaf-list-second","leaf-list-third"]`
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

	schema = RootSchema.FindSchema("sample/container-val/leaf-list-val")
	jleaflistnodes, err := NewDataNodeGroup(schema, jleaflist)
	if err != nil {
		t.Fatal(err)
	}
	{
		j, err := jleaflistnodes.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		y, err := yaml.Marshal(jleaflistnodes)
		if err != nil {
			t.Fatal(err)
		}
		var o1, o2 interface{}
		if err := json.Unmarshal(j, &o1); err != nil {
			t.Fatal(err)
		}
		if err := yaml.Unmarshal(y, &o2); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(o1, o2) {
			t.Error("not equal leaf-list nodes")
			t.Error("json:", o1)
			t.Error("yaml:", o1)
			return
		}
	}

	schema = RootSchema.FindSchema("sample/single-key-list")
	jlistnodes, err := NewDataNodeGroup(schema, jlist)
	if err != nil {
		t.Fatal(err)
	}
	if j, err := jlistnodes.MarshalJSON_RFC7951(); err == nil {
		t.Log(string(j))
	}
	if y, err := MarshalYAML(jlistnodes, RFC7951Format{}); err == nil {
		t.Log(string(y))
	}
}
