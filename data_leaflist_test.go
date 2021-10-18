package yangtree

import "testing"

func TestDataLeafList(t *testing.T) {
	schema, err := Load([]string{"testdata/sample"}, nil, nil, SchemaOption{SingleLeafList: true})
	if err != nil {
		t.Fatal(err)
	}
	// jcontainer := `
	// {
	// 	"container-val": {
	// 		"a": "A",
	// 		"enum-val": "enum2",
	// 		"leaf-list-val": [
	// 			"first",
	// 			"fourth",
	// 			"second",
	// 			"third"
	// 		],
	// 		"test-default": 11
	// 	},
	// 	"empty-val": null,
	// 	"multiple-key-list": {
	// 		"first": {
	// 			"1": {
	// 				"integer": 1,
	// 				"ok": true,
	// 				"str": "first"
	// 			},
	// 			"2": {
	// 				"integer": 2,
	// 				"str": "first"
	// 			}
	// 		}
	// 	},
	// 	"non-key-list": [
	// 		{
	// 			"strval": "XYZ",
	// 			"uintval": 10
	// 		}
	// 	],
	// 	"single-key-list": {
	// 		"AAA": {
	// 			"country-code": "KR",
	// 			"decimal-range": 1.01,
	// 			"empty-node": null,
	// 			"list-key": "AAA",
	// 			"uint32-range": 100,
	// 			"uint64-node": 1234567890
	// 		}
	// 	},
	// 	"str-val": "abc"
	// }
	// `
	jcontainer := `
	{
		"container-val": {
			"leaf-list-val": [
				"first",
				"second",
				"third",
				"fourth"
			]
		}
	}
	`
	singleLeafList, err := NewDataNode(schema.FindSchema("/sample/container-val/leaf-list-val"), `["first","fourth","second","third"]`)
	if err != nil {
		t.Fatal(err)
	}
	y, err := singleLeafList.MMarshalYAML()
	if err != nil {
		t.Fatalf("leaflist marshalling to YAML: %v", err)
	}
	if string(y) != "[first,fourth,second,third]" {
		t.Fatalf("leaflist yaml marshalling failed: %s", string(y))
	}
	j, err := singleLeafList.MarshalJSON()
	if err != nil {
		t.Fatalf("leaflist marshalling to JSON: %v", err)
	}
	if string(j) != `["first","fourth","second","third"]` {
		t.Fatalf("leaflist json marshalling failed: %s", string(j))
	}

	root, err := NewDataNode(schema)
	if err != nil {
		t.Fatalf("root creation failed: %v", err)
	}
	if err := Set(root, "sample", jcontainer); err != nil {
		t.Fatalf("sample set failed: %v", err)
	}
	if j, err = root.MarshalJSON_RFC7951(); err != nil {
		t.Fatalf("sample json marshalling failed: %v", err)
	} else if string(j) != `{"sample:sample":{"container-val":{"leaf-list-val":["first","fourth","second","third"]}}}` {
		t.Fatalf("json marshalling failed: %s", string(j))
	}
	found, err := Find(root, "/sample/container-val/leaf-list-val")
	if err != nil {
		t.Fatalf("leaf-list-val finding failed: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("leaf-list-val finding failed: single leaf-list node")
	}
	y, err = found[0].MMarshalYAML()
	if err != nil {
		t.Fatalf("leaf-list-val yaml marshalling failed: %v", err)
	}
	if string(y) != `[first,fourth,second,third]` {
		t.Fatalf("leaflist json marshalling failed: %s", string(y))
	}
	j, err = found[0].MarshalJSON_RFC7951()
	if err != nil {
		t.Fatalf("leaf-list-val yaml marshalling failed: %v", err)
	}
	if string(j) != `["first","fourth","second","third"]` {
		t.Fatalf("leaflist json marshalling failed: %s", string(j))
	}
}
