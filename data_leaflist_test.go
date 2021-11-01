package yangtree

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSingleDataLeafList(t *testing.T) {
	schema, err := Load([]string{"testdata/sample"}, nil, nil, SchemaOption{SingleLeafList: true})
	if err != nil {
		t.Fatal(err)
	}

	testItem1 := []struct {
		path     string
		input    string
		json     string
		yaml     string
		expected []interface{}
	}{
		{
			path:     "single-leaf-list-rw-system",
			input:    `["first","second","third","fourth"]`,
			json:     `["first","fourth","second","third"]`,
			yaml:     "[first,fourth,second,third]",
			expected: []interface{}{"first", "fourth", "second", "third"},
		},
		{
			path:     "single-leaf-list-rw-user",
			input:    `["first","second","third","fourth"]`,
			json:     `["first","second","third","fourth"]`,
			yaml:     "[first,second,third,fourth]",
			expected: []interface{}{"first", "second", "third", "fourth"},
		},
		{
			path:     "single-leaf-list-ro",
			input:    `["first","second","third","fourth"]`,
			json:     `["first","second","third","fourth"]`,
			yaml:     "[first,second,third,fourth]",
			expected: []interface{}{"first", "second", "third", "fourth"},
		},
		{
			path:     "single-leaf-list-ro-int",
			input:    `[1,2,3,4]`,
			json:     `[1,2,3,4]`,
			yaml:     "[1,2,3,4]",
			expected: []interface{}{int32(1), int32(2), int32(3), int32(4)},
		},
	}
	for _, tt := range testItem1 {
		t.Run("Set."+tt.path, func(t *testing.T) {
			singleLeafListSchema := schema.FindSchema(tt.path)
			singleLeafList, err := NewDataNode(singleLeafListSchema, tt.input)
			if err != nil {
				t.Errorf("NewDataNode() error = %v, path = %s", err, tt.path)
				return
			}
			// check the values of the single leaf-list (ordered-by system)
			values := singleLeafList.Values()
			if !reflect.DeepEqual(values, tt.expected) {
				t.Errorf("invalid single leaf-list values %q", singleLeafList.Values())
				return
			}

			y, err := MarshalYAML(singleLeafList)
			if err != nil {
				t.Errorf("leaflist marshalling to YAML: %v", err)
				return
			}
			if string(y) != tt.yaml {
				t.Errorf("leaflist yaml marshalling failed: %s", string(y))
				return
			}
			j, err := MarshalJSON(singleLeafList)
			if err != nil {
				t.Errorf("leaflist marshalling to JSON: %v", err)
				return
			}
			if string(j) != tt.json {
				t.Errorf("leaflist json marshalling failed: %s", string(j))
				return
			}

		})
	}

	root, err := NewDataNode(schema)
	if err != nil {
		t.Fatalf("root creation failed: %v", err)
	}

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
	if err := Set(root, "sample", jcontainer); err != nil {
		t.Fatalf("sample set failed: %v", err)
	}
	if j, err := MarshalJSON(root, RFC7951Format{}); err != nil {
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
	y, err := MarshalYAML(found[0])
	if err != nil {
		t.Fatalf("leaf-list-val yaml marshalling failed: %v", err)
	}
	if string(y) != `[first,fourth,second,third]` {
		t.Fatalf("leaflist json marshalling failed: %s", string(y))
	}
	j, err := MarshalJSON(found[0], RFC7951Format{})
	if err != nil {
		t.Fatalf("leaf-list-val yaml marshalling failed: %v", err)
	}
	if string(j) != `["first","fourth","second","third"]` {
		t.Fatalf("leaflist json marshalling failed: %s", string(j))
	}

	testItem2 := []struct {
		values   []string
		expected string
	}{
		{values: []string{"fifth", "sixth"}, expected: `["fifth","first","fourth","second","sixth","third"]`},
		{values: []string{`["seventh","eighth"]`}, expected: `["eighth","fifth","first","fourth","second","seventh","sixth","third"]`},
	}
	for _, tt := range testItem2 {
		t.Run(fmt.Sprint("single-leaf-list test", tt.values), func(t *testing.T) {
			// Set each values
			if err := Set(root, "sample/container-val/leaf-list-val", tt.values...); err != nil {
				t.Fatalf("sample set failed: %v", err)
			}
			j, err = MarshalJSON(found[0], RFC7951Format{})
			if err != nil {
				t.Fatalf("leaf-list-val yaml marshalling failed: %v", err)
			}
			if string(j) != tt.expected {
				t.Fatalf("leaflist json marshalling failed: %s", string(j))
			}
		})
	}
}
