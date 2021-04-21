package yangtree

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	RootSchema, err := Load([]string{"data"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	jbyte := `
	{
		"sample:sample": {
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
		 "empty-val": [
		  null
		 ],
		 "multiple-key-list": [
		  {
		   "integer": 1,
		   "ok": true,
		   "str": "first"
		  },
		  {
		   "integer": 2,
		   "str": "first"
		  }
		 ],
		 "non-key-list": [
		  {
		   "strval": "XYZ",
		   "uintval": 10
		  }
		 ],
		 "single-key-list": [
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
		 ],
		 "str-val": "abc"
		}
	   }
	`
	RootData, err := New(RootSchema, jbyte)
	if err != nil {
		t.Fatal(err)
	}
	j, _ := MarshalJSONIndent(RootData, "", " ", false)
	t.Log(string(j))

	cschema := GetSchema(RootSchema, "sample")
	t.Log(New(cschema, `{"str-val": "ok"}`))
}

func TestChildDataNodeListing(t *testing.T) {
	RootSchema, err := Load([]string{"data"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	RootData, err := New(RootSchema)
	if err != nil {
		t.Fatal(err)
	}
	input := []string{
		"/sample/single-key-list[list-key=A1]/uint32-range",
		"/sample/single-key-list[list-key=A12]/uint32-range",
		"/sample/single-key-list[list-key=A123]/uint32-range",
		"/sample/single-key-list[list-key=A1234]/uint32-range",
		// "/sample/single-key-list[list-key=A24]/uint32-range",
		// "/sample/single-key-list[list-key=A3]/int8-range",
		// "/sample/single-key-list[list-key=A4]/decimal-range",
		// "/sample/single-key-list[list-key=A5]/empty-node",
		// "/sample/single-key-list[list-key=A6]/uint64-node",
		// "/sample/single-key-list[list-key=A0]/uint64-node",
		// "/sample/multiple-key-list[str=first][integer=1]/ok",
	}
	for i := range input {
		Set(RootData, input[i])
	}
	// n, _ := RootData.Retrieve("/sample")
}

func TestDataNode(t *testing.T) {
	RootSchema, err := Load([]string{"data"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	RootData, err := New(RootSchema)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		path  string
		value []string
	}
	tests := []struct {
		name          string
		args          args
		wantInsertErr bool
		wantDeleteErr bool
	}{
		{
			name:          "test-item",
			args:          args{path: "/sample"},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/str-val",
				value: []string{"abc"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/empty-val",
				value: []string{"true"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/",
				value: nil,
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/single-key-list[list-ke=first]",
				value: []string{"true"},
			},
			wantInsertErr: true,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/country-code",
				value: []string{"KR"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/uint32-range",
				value: []string{"100"},
			},
			wantInsertErr: false,
		},
		{
			name: "range-check",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/uint32-range",
				value: []string{"493"},
			},
			wantInsertErr: true,
		},
		{
			name: "range-check",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/int8-range",
				value: []string{"500"},
			},
			wantInsertErr: true,
		},
		{
			name: "decimal-range",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/decimal-range",
				value: []string{"1.01"},
			},
			wantInsertErr: false,
		},
		{
			name: "empty-node",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/empty-node",
				value: nil,
			},
			wantInsertErr: false,
		},
		{
			name: "uint64-node",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/uint64-node",
				value: []string{"1234567890"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/multiple-key-list[str=first][integer=1]/ok",
				value: []string{"true"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/multiple-key-list[str=first][integer=2]/str",
				value: []string{"first"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample:sample/container-val",
				value: nil,
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/leaf-list-val",
				value: nil,
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/leaf-list-val",
				value: []string{"leaf-list-first", "leaf-list-second"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/leaf-list-val",
				value: []string{"leaf-list-third"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/leaf-list-val/leaf-list-fourth",
				value: nil,
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample:sample/sample:container-val/sample:enum-val",
				value: []string{"enum2"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample:sample/sample:container-val/sample:test-default",
				value: []string{"11"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-choice",
			args: args{
				path:  "/sample:sample/sample:container-val/a",
				value: []string{"A"},
			},
			wantInsertErr: false,
		},
		{
			name: "non-key-list",
			args: args{
				path:  "/sample:sample/non-key-list",
				value: []string{`{"uintval": "10", "strval": "XYZ"}`},
			},
			wantInsertErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+".Set", func(t *testing.T) {
			if err := Set(RootData, tt.args.path, tt.args.value...); (err != nil) != tt.wantInsertErr {
				t.Errorf("Set() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.args.path)
			}
		})
	}

	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")

	path := []string{
		"/sample/multiple-key-list[str=first][integer=*]/ok",
		"/sample/single-key-list[list-key=AAA]/list-key",
		"/sample/single-key-list[list-key=AAA]",
		"/sample/single-key-list[list-key=*]",
		"/sample/single-key-list/*",
		"/sample/*",
		"/sample/...",
		"/sample/.../enum-val",
		"/sample/*/*/",
	}
	for i := range path {
		node, err := RootData.Retrieve(path[i])
		if err != nil {
			t.Errorf("Retrieve() path %v error = %v", path[i], err)
		}
		for j := range node {
			t.Log("Retrieve", i, path[i], "::::", node[j].Path(), node[j])
			// j, _ := MarshalJSON(node[j], true)
			// t.Log("Retrieve", i, "", path[i], string(j))
		}
	}
	// node := RootData.Find("/sample")
	// // j, _ := node.MarshalJSON()
	// j, _ := MarshalJSONIndent(node, "", " ", false)
	// t.Log(string(j))

	jj, err := MarshalJSONIndent(RootData, "", " ", false)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(jj))

	for i := len(tests) - 1; i >= 0; i-- {
		tt := tests[i]
		if tt.wantInsertErr {
			continue
		}
		t.Run(tt.name+".Delete", func(t *testing.T) {
			if err := Delete(RootData, tt.args.path, tt.args.value...); (err != nil) != tt.wantDeleteErr {
				t.Errorf("Delete() error = %v, wantDeleteErr %v", err, tt.wantDeleteErr)
			}
		})
	}

	jsonietf, err := MarshalJSONIndent(RootData, "", " ", true)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(jsonietf))
	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
}

func TestParseXPath(t *testing.T) {

	tests := []struct {
		path  string
		elems []string
		attrs map[string]string
	}{
		{path: "/interfaces/interface[name=1/1]", elems: []string{"interfaces", "interface"}, attrs: map[string]string{"name": "1/1"}},
		{path: "/abc:interfaces/id[name=1/1]/xyz=10", elems: []string{"abc", "interfaces", "id", "xyz"}, attrs: map[string]string{"name": "1/1"}},
		// {path: "//", elems: nil},
		// {path: "/[?=what]", result: nil},
	}
	for _, tt := range tests {
		t.Run("ParseXPath", func(t *testing.T) {
			var pos int
			var err error
			var prefix, elem string
			var attrs map[string]string
			var result []string
			rattrs := map[string]string{}
			for pos < len(tt.path) {
				prefix, elem, attrs, pos, err = ParseXPath(&tt.path, pos, len(tt.path))
				if err != nil {
					t.Error(err)
					break
				}
				if prefix != "" {
					result = append(result, prefix)
				}
				if elem != "" {
					result = append(result, elem)
				}
				for k, v := range attrs {
					rattrs[k] = v
				}
			}

			if !reflect.DeepEqual(result, tt.elems) && !reflect.DeepEqual(rattrs, tt.attrs) {
				t.Errorf("not equal with got = %v %v, expect = %v %v", result, rattrs, tt.elems, tt.attrs)
			}
		})
	}
}
