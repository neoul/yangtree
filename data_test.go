package yangtree

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	RootSchema, err := Load([]string{"data/sample"}, nil, nil)
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
	RootSchema, err := Load([]string{"data/sample"}, nil, nil)
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

		"/sample/single-key-list[list-key=A122]/uint32-range",
		"/sample/single-key-list[list-key=A123]/uint32-range",
		"/sample/single-key-list[list-key=A1234]/uint32-range",
		"/sample/single-key-list[list-key=A24]/uint32-range",
		"/sample/single-key-list[list-key=A5]/empty-node",
		"/sample/single-key-list[list-key=A3]/int8-range",
		"/sample/single-key-list[list-key=A4]/decimal-range",

		"/sample/single-key-list[list-key=A6]/uint64-node",
		"/sample/single-key-list[list-key=A0]/uint64-node",
		"/sample/multiple-key-list[str=first][integer=1]/ok",
	}
	for i := range input {
		Set(RootData, input[i])
	}
	// sort.Strings(input)
	// pretty.Print(input)

	path := []string{
		"/sample/*",
		"/sample/single-key-list[list-key]",
		"/sample/multiple-key-list[str=first]/ok",
		"/sample/single-key-list[position()=last()]",
		"/sample/single-key-list[2]",
		// "/sample/single-key-list[list-key=A12]/uint32-range",
		// "/sample/single-key-list[list-key=A123]/uint32-range",
		// "/sample/single-key-list[list-key=A1234]/uint32-range",
		// "/sample/single-key-list[list-key=A24]/uint32-range",
		// "/sample/single-key-list[list-key=A3]/int8-range",
		// "/sample/single-key-list[list-key=A4]/decimal-range",
		// "/sample/single-key-list[list-key=A5]/empty-node",
		// "/sample/single-key-list[list-key=A6]/uint64-node",
		// "/sample/single-key-list[list-key=A0]/uint64-node",
		// "/sample/multiple-key-list[str=first][integer=1]/ok",
	}
	for i := range path {
		node, err := Find(RootData, path[i])
		if err != nil {
			t.Errorf("Find() path %v error = %v", path[i], err)
		}
		t.Logf("Find (%s)", path[i])
		for j := range node {
			t.Logf(" - %s, %s (%p)", node[j].Path(), node[j], node[j])
			// j, _ := MarshalJSON(node[j], true)
			// t.Log("Find", i, "", path[i], string(j))
		}
	}
}

func TestDataNode(t *testing.T) {
	RootSchema, err := Load([]string{"data/sample"}, nil, nil)
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
		// {
		// 	name: "uint64-node",
		// 	args: args{
		// 		path:  "/sample/single-key-list[list-key=AAA]/uint64-node",
		// 		value: []string{"1234567890"},
		// 	},
		// 	wantInsertErr: false,
		// },
		{
			name: "uint64-node-with-predicates",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/uint64-node[.=1234567890]",
				value: nil,
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
				path:  "/sample/multiple-key-list[str=second][integer=1]/str",
				value: []string{"second"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/multiple-key-list[sample:str=second][integer=2]/str",
				value: []string{"second"},
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
				path:  "/sample/container-val/leaf-list-val[.=leaf-list-fifth]",
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
				value: []string{`{"uintval": "11", "strval": "XYZ"}`},
			},
			wantInsertErr: false,
		},
		{
			name: "non-key-list",
			args: args{
				path:  "/sample:sample/non-key-list",
				value: []string{`{"uintval": "12", "strval": "XYZ"}`},
			},
			wantInsertErr: false,
		},
		{
			name: "non-key-list",
			args: args{
				path:  "/sample:sample/non-key-list[uintval=13][strval=ABC]",
				value: nil,
			},
			wantInsertErr: false,
		},
		{
			name: "test-instance-identifier",
			args: args{
				path:  "/sample:sample/sample:container-val/test-instance-identifier",
				value: []string{"/sample:sample/sample:container-val/a"},
			},
			wantInsertErr: false,
		},
		{
			name: "test-must",
			args: args{
				path:  "/sample:sample/sample:container-val/test-must",
				value: []string{"5"},
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
	if err := Validate(RootData); err != nil {
		t.Error(err)
	}

	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")

	path := []string{
		"/sample/container-val/leaf-list-val[.=leaf-list-fourth]",
		"/sample/multiple-key-list[str=first][integer=*]/ok",
		"/sample/single-key-list[sample:list-key=AAA]/list-key",
		"/sample/single-key-list[list-key='AAA']",
		"/sample/single-key-list[list-key=*]",
		"/sample/single-key-list/*",
		"/sample/*",
		"/sample/...",
		"/sample/.../enum-val",
		"/sample/*/*/",
		"/sample//non-key-list",
		"/sample/multiple-key-list[str=first][integer=*]",
		"/sample/multiple-key-list",
		"/sample/non-key-list[2]",
	}
	for i := range path {
		node, err := Find(RootData, path[i])
		if err != nil {
			t.Errorf("Find() path %v error = %v", path[i], err)
		}
		t.Logf("Find %s", path[i])
		for j := range node {
			jj, _ := MarshalJSON(node[j], true)
			t.Log(" - Find", j, "", node[j].Path(), string(jj))
		}
	}

	path = []string{
		"/sample/container-val/leaf-list-val[.=leaf-list-fourth]",
	}
	result := []interface{}{
		[]string{"leaf-list-fourth"},
	}
	for i := range path {
		value, err := FindValueString(RootData, path[i])
		if err != nil {
			t.Errorf("Find() path %v error = %v", path[i], err)
		}
		t.Logf("FindValue %s", path[i])
		if !reflect.DeepEqual(value, result[i]) {
			t.Error("not equal", value, result[i])
		}
		for j := range value {
			v := value[j]
			t.Log(" - Find", j, "", ValueToString(v))
		}
	}

	// nodes := RootData.Get("sample")
	// t.Log(nodes[0].Get("single-key-list[list-key=AAA]"))
	// t.Log(nodes[0].Lookup("s"))

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

func TestComplexModel(t *testing.T) {
	rootschema, err := Load(
		[]string{
			"data/modules/choice-case-example.yang",
			"data/modules/pattern.yang",
			"data/modules/openconfig-simple-target.yang",
			"data/modules/openconfig-simple-augment.yang",
		}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	rootdata, err := New(rootschema)
	if err != nil {
		t.Fatal(err)
	}
	simpleChoiceCase, err := rootdata.New("simple-choice-case")
	if err != nil {
		t.Error(err)
	}
	_, err = simpleChoiceCase.New("a", "a.value")
	if err != nil {
		t.Error(err)
	}
	_, err = simpleChoiceCase.New("b", "b.value")
	if err != nil {
		t.Error(err)
	}
	choiceCaseAnonymousCase, err := rootdata.New("choice-case-anonymous-case")
	if err != nil {
		t.Error(err)
	}
	_, err = choiceCaseAnonymousCase.New("foo/a", "a.value")
	if err == nil {
		t.Error("choice and case should not be present in the tree.")
	}
	_, err = choiceCaseAnonymousCase.New("a", "a.value")
	if err != nil {
		t.Error(err)
	}
	_, err = choiceCaseAnonymousCase.New("b", "b.value")
	if err != nil {
		t.Error(err)
	}
	choiceCaseWithLeafref, err := rootdata.New("choice-case-with-leafref")
	if err != nil {
		t.Error(err)
	}
	_, err = choiceCaseWithLeafref.New("referenced", "referenced.value")
	if err != nil {
		t.Error(err)
	}
	node, err := choiceCaseWithLeafref.New("ptr", "ok?")
	if err != nil {
		t.Error(err)
	}
	if err := Validate(node); err == nil {
		t.Error("leafref value must be present in the tree.")
	}
	node, err = choiceCaseWithLeafref.New("ptr", "referenced.value")
	if err != nil {
		t.Error(err)
	}
	if err := Validate(node); err != nil {
		t.Error(err)
	}

	if _, err = rootdata.New("pattern-type", "x"); err == nil {
		t.Error(err)
	}
	if _, err = rootdata.New("pattern-type", "abc"); err != nil {
		t.Error(err)
	}

	j, err := rootdata.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	t.Log(string(j))
}
