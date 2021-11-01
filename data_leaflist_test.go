package yangtree

import (
	"fmt"
	"reflect"
	"testing"
)

// Test single leaf-list node operation
func TestSingleLeafList(t *testing.T) {
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

// Test multiple leaf-list node operation
func TestMultipleLeafList(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil, SchemaOption{LeafListValueAsKey: true})
	if err != nil {
		t.Fatal(err)
	}
	RootData, err := NewDataNode(RootSchema)
	if err != nil {
		t.Fatal(err)
	}

	rwLeafListTest := []struct {
		path          string
		value         string
		wantInsertErr bool
		wantDeleteErr bool
		numOfNodes    int
	}{
		// Read-write leaf-list
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-rw", value: "[]", numOfNodes: 0},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-rw", value: `["leaf-list-1", "leaf-list-2"]`, numOfNodes: 2},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-rw", value: `["leaf-list-2"]`, numOfNodes: 2},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-rw", value: `["leaf-list-3"]`, numOfNodes: 3},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-rw", value: `["leaf-list-3"]`, numOfNodes: 3},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-rw/leaf-list-4", value: "", numOfNodes: 4},
		{wantInsertErr: true, wantDeleteErr: true, path: "/sample/leaf-list-rw[.=leaf-list-5]", value: "", numOfNodes: 4},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-rw[.=leaf-list-5]", value: "leaf-list-5", numOfNodes: 5},
	}
	for _, tt := range rwLeafListTest {
		t.Run(fmt.Sprintf("Edit.%s %v", tt.path, tt.value), func(t *testing.T) {
			editopt := &EditOption{EditOp: EditMerge}
			err := Edit(editopt, RootData, tt.path, tt.value)
			if (err != nil) != tt.wantInsertErr {
				t.Errorf("Edit() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.path)
				return
			}
			if sample := RootData.Get("sample"); sample != nil {
				if sample.Len() != tt.numOfNodes {
					t.Errorf("Edit() error = unexpected number of nodes in %s, expected num %d, got %d", tt.path, tt.numOfNodes, sample.Len())
					return
				}
			}
		})
	}
	for i := len(rwLeafListTest) - 1; i >= 0; i-- {
		t.Run(fmt.Sprintf("Delete.%s", rwLeafListTest[i].path), func(t *testing.T) {
			// err := Delete(RootData, rwLeafListTest[i].path)
			editopt := &EditOption{EditOp: EditRemove}
			err := Edit(editopt, RootData, rwLeafListTest[i].path, rwLeafListTest[i].value)
			if (err != nil) != rwLeafListTest[i].wantDeleteErr {
				t.Errorf("Set() error = %v, wantDeleteErr = %v path = %s", err, rwLeafListTest[i].wantDeleteErr, rwLeafListTest[i].path)
			}
		})
	}

	roLeafListTest := []struct {
		path          string
		value         string
		wantInsertErr bool
		wantDeleteErr bool
		numOfNodes    int
	}{
		// // Read-only leaf-list
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-ro", value: "[]", numOfNodes: 0}, // do nothing
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-ro", value: `["leaf-list-1", "leaf-list-2"]`, numOfNodes: 2},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-ro", value: `["leaf-list-2"]`, numOfNodes: 3},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-ro", value: `["leaf-list-3"]`, numOfNodes: 4},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-ro", value: `["leaf-list-3"]`, numOfNodes: 5},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-ro/leaf-list-4", value: "", numOfNodes: 6},
		{wantInsertErr: true, wantDeleteErr: true, path: "/sample/leaf-list-ro[.=leaf-list-5]", value: "", numOfNodes: 6},
		{wantInsertErr: false, wantDeleteErr: false, path: "/sample/leaf-list-ro[.=leaf-list-5]", value: "leaf-list-5", numOfNodes: 7},
	}
	for _, tt := range roLeafListTest {
		t.Run(fmt.Sprintf("Edit.%s %v", tt.path, tt.value), func(t *testing.T) {
			editopt := &EditOption{EditOp: EditMerge}
			err := Edit(editopt, RootData, tt.path, tt.value)
			if (err != nil) != tt.wantInsertErr {
				t.Errorf("Edit() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.path)
				return
			}
			if sample := RootData.Get("sample"); sample != nil {
				if sample.Len() != tt.numOfNodes {
					t.Errorf("Edit() error = unexpected number of nodes in %s, expected num %d, got %d", tt.path, tt.numOfNodes, sample.Len())
					return
				}
			}
		})
	}
	for i := len(roLeafListTest) - 1; i >= 0; i-- {
		t.Run(fmt.Sprintf("Delete.%s", roLeafListTest[i].path), func(t *testing.T) {
			// err := Delete(RootData, roLeafListTest[i].path)
			editopt := &EditOption{EditOp: EditRemove}
			err := Edit(editopt, RootData, roLeafListTest[i].path, roLeafListTest[i].value)
			if (err != nil) != roLeafListTest[i].wantDeleteErr {
				t.Errorf("Set() error = %v, wantDeleteErr = %v path = %s", err, roLeafListTest[i].wantDeleteErr, roLeafListTest[i].path)
			}
		})
	}
}
