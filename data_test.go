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
	root1, err := New(RootSchema, jbyte)
	if err != nil {
		t.Fatal(err)
	}
	j, _ := MarshalJSON(root1)
	t.Log(string(j))

	root2, err := New(RootSchema, jbyte)
	if err != nil {
		t.Fatal(err)
	}
	j, _ = MarshalJSON(root2)
	t.Log(string(j))

	if equal := Equal(root1, root2); !equal {
		t.Errorf("equal(root1, root2) is failed for the same tree")
	}
	root3 := Clone(root1)
	if root3 == nil {
		t.Errorf("clone a data node is failed")
	}
	if equal := Equal(root1, root3); !equal {
		t.Errorf("equal(root1, root3) is failed for the same tree")
	}

	err = Set(root1, "/sample/container-val/enum-val", "enum1")
	if err != nil {
		t.Error(err)
	}
	if equal := Equal(root1, root3); equal {
		t.Errorf("equal(root1, root3) is not equal")
	}

	mergingData := `{
	 "integer": 2,
	 "str": "first",
	 "ok": false
	}`
	s := FindSchema(RootSchema, "/sample/multiple-key-list")
	if s == nil {
		t.Error("schema multiple-key-list not found")
	}
	mnode, err := New(s, mergingData)
	if err != nil {
		t.Error("new failed", err)
	}
	node, _ := root2.Find("sample/multiple-key-list[integer=2][str=first]")
	if err := node[0].Merge(mnode); err != nil {
		t.Error("merge failed:", err)
	}
	if equal := Equal(root2, root3); equal {
		t.Errorf("equal(root2, root3) is not equal")
	}

	j, _ = MarshalJSON(root2)
	t.Log(string(j))
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
		// "/sample/single-key-list[list-key=A3]/int8-range",
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

	tests := []struct {
		path          string
		value         []string
		wantInsertErr bool
		wantDeleteErr bool
	}{
		{wantInsertErr: false, path: "/sample"},
		{wantInsertErr: false, path: "/sample/str-val", value: []string{"abc"}},
		{wantInsertErr: false, path: "/sample/empty-val", value: []string{"true"}},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/", value: nil},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/country-code", value: []string{"KR"}},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/uint32-range", value: []string{"100"}},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/decimal-range", value: []string{"1.01"}},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/empty-node", value: nil},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/uint64-node[.=1234567890]", value: nil},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=BBB]/uint64-node[.=1234567890]", value: nil},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=BBB]/uint32-range", value: []string{"200"}},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=CCC]/uint32-range", value: []string{"300"}},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=DDD]/uint32-range", value: []string{"400"}},
		{wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=1]/ok", value: []string{"true"}},
		{wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=2]/str", value: []string{"first"}},
		{wantInsertErr: false, path: "/sample/multiple-key-list[str=second][integer=1]/str", value: []string{"second"}},
		{wantInsertErr: false, path: "/sample/multiple-key-list[sample:str=second][integer=2]/str", value: []string{"second"}},
		{wantInsertErr: false, path: "/sample:sample/container-val", value: nil},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: nil},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: []string{"leaf-list-first", "leaf-list-second"}},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: []string{"leaf-list-third"}},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val/leaf-list-fourth", value: nil},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val[.=leaf-list-fifth]", value: nil},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/sample:enum-val", value: []string{"enum2"}},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/sample:test-default", value: []string{"11"}},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/a", value: []string{"A"}},
		{wantInsertErr: false, path: "/sample:sample/non-key-list", value: []string{`{"uintval": "11", "strval": "XYZ"}`}},
		{wantInsertErr: false, path: "/sample:sample/non-key-list", value: []string{`{"uintval": "12", "strval": "XYZ"}`}},
		{wantInsertErr: false, path: "/sample:sample/non-key-list[uintval=13][strval=ABC]", value: nil},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/test-instance-identifier", value: []string{"/sample:sample/sample:container-val/a"}},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/test-must", value: []string{"5"}},

		{wantInsertErr: true, path: "/sample/single-key-list[list-ke=first]", value: []string{"true"}},
		{wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/uint32-range", value: []string{"493"}},
		{wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/int8-range", value: []string{"500"}},
	}
	for _, tt := range tests {
		t.Run("Set."+tt.path, func(t *testing.T) {
			err := Set(RootData, tt.path, tt.value...)
			if (err != nil) != tt.wantInsertErr {
				t.Errorf("Set() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.path)
			}
		})
	}
	if err := Validate(RootData); err != nil {
		t.Error(err)
	}

	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
	testfinds := []struct {
		expectedNum int
		path        string
		findState   bool
	}{
		{expectedNum: 1, path: "/sample/container-val/leaf-list-val[.=leaf-list-fourth]"},
		{expectedNum: 1, path: "/sample/multiple-key-list[str=first][integer=*]/ok"},
		{expectedNum: 1, path: "/sample/single-key-list[sample:list-key=AAA]/list-key"},
		{expectedNum: 1, path: "/sample/single-key-list[list-key='AAA']"},
		{expectedNum: 4, path: "/sample/single-key-list[list-key=*]"},
		{expectedNum: 13, path: "/sample/single-key-list/*"},
		{expectedNum: 14, path: "/sample/*"},
		{expectedNum: 49, path: "/sample/..."},
		{expectedNum: 4, path: "/sample/...", findState: true},
		{expectedNum: 1, path: "/sample/.../enum-val"},
		{expectedNum: 34, path: "/sample/*/*/"},
		{expectedNum: 3, path: "/sample//non-key-list"},
		{expectedNum: 2, path: "/sample/multiple-key-list[str=first][integer=*]"},
		{expectedNum: 4, path: "/sample/multiple-key-list"},
		{expectedNum: 1, path: "/sample/non-key-list[2]"},
		{expectedNum: 2, path: "/sample/single-key-list[list-key='BBB' or list-key='CCC']"},
	}
	for _, tt := range testfinds {
		t.Run("Find."+tt.path, func(t *testing.T) {
			var err error
			var node []DataNode
			if tt.findState {
				node, err = Find(RootData, tt.path, OptionGetState{})
			} else {
				node, err = Find(RootData, tt.path)
			}
			if err != nil {
				t.Errorf("Find() path %v error = %v", tt.path, err)
			}
			t.Logf("Find %s (expected num: %d, result: %d)", tt.path, tt.expectedNum, len(node))
			for j := range node {
				jj, _ := MarshalJSON(node[j], RFC7951Format{})
				t.Log(" - found", j+1, "", node[j].Path(), string(jj))
			}
			if tt.expectedNum != len(node) {
				t.Errorf("find error for %s (expected num: %d, result: %d)", tt.path, tt.expectedNum, len(node))
			}
		})
	}

	path := []string{
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

	jj, err := MarshalJSONIndent(RootData, "", " ", RFC7951Format{})
	if err != nil {
		t.Error(err)
	}
	t.Log(string(jj))

	for i := len(tests) - 1; i >= 0; i-- {
		tt := tests[i]
		if tt.wantInsertErr {
			continue
		}
		t.Run("Delete."+tt.path, func(t *testing.T) {
			if err := Delete(RootData, tt.path, tt.value...); (err != nil) != tt.wantDeleteErr {
				t.Errorf("Delete() error = %v, wantDeleteErr %v", err, tt.wantDeleteErr)
			}
		})
	}

	jsonietf, err := MarshalJSONIndent(RootData, "", " ", RFC7951Format{})
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

func TestCreatedWithDefault(t *testing.T) {
	rootschema, err := Load(
		[]string{
			"data/modules/default.yang",
		}, nil, nil, SchemaOption{CreatedWithDefault: true})
	if err != nil {
		t.Fatal(err)
	}
	rootdata, err := New(rootschema)
	if err != nil {
		t.Fatal(err)
	}
	test, err := rootdata.New("test")
	if err != nil {
		t.Error(err)
	}
	_, err = test.New("config")
	if err != nil {
		t.Error(err)
	}
	_, err = test.New("state")
	if err != nil {
		t.Error(err)
	}

	node, err := test.Find("config/d1")
	if err != nil {
		t.Error(err)
	}
	if len(node) != 1 {
		t.Errorf("d1 node must be created.=%d\n", len(node))
	}
	if len(node) == 1 && node[0].Value() == 100 {
		t.Errorf("d1 node must be created with default. d1=%v\n", node[0].Value())
	}

	j, err := rootdata.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	t.Log(string(j))
}
