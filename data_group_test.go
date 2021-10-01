package yangtree

import "testing"

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
	schema := FindSchema(RootSchema, "sample")
	jcontainernodes, err := NewDataNodeGroup(schema, jcontainer)
	if err != nil {
		t.Fatal(err)
	}
	if y, err := jcontainernodes.MarshalYAML(); err == nil {
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

	schema = FindSchema(RootSchema, "sample/container-val/leaf-list-val")
	jleaflistnodes, err := NewDataNodeGroup(schema, jleaflist)
	if err != nil {
		t.Fatal(err)
	}
	if j, err := jleaflistnodes.MarshalJSON(); err == nil {
		t.Log(string(j))
	}
	if y, err := jleaflistnodes.MarshalYAML(); err == nil {
		t.Log(string(y))
	}

	schema = FindSchema(RootSchema, "sample/single-key-list")
	jlistnodes, err := NewDataNodeGroup(schema, jlist)
	if err != nil {
		t.Fatal(err)
	}
	if j, err := jlistnodes.MarshalJSON(RFC7951Format{}); err == nil {
		t.Log(string(j))
	}
	if y, err := jlistnodes.MarshalYAML(RFC7951Format{}); err == nil {
		t.Log(string(y))
	}
}
