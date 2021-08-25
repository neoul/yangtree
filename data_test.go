package yangtree

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
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

	// Set
	err = Set(root1, "/sample/container-val/enum-val", "enum1")
	if err != nil {
		t.Error(err)
	}

	// Equal
	if equal := Equal(root1, root3); equal {
		t.Errorf("equal(root1, root3) is not equal")
	}

	// Merge
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
	node, _ := Find(root2, "sample/multiple-key-list[integer=2][str=first]")
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
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
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
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
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
		// {wantInsertErr: false, path: "/sample/str-val", value: []string{"abc"}},
		// {wantInsertErr: false, path: "/sample/empty-val", value: []string{"true"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/", value: nil},
		// {wantInsertErr: false, path: "/sample/single-key-list", value: []string{`{"list-key": "ZZZ"}`}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/country-code", value: []string{"KR"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/uint32-range", value: []string{"100"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/decimal-range", value: []string{"1.01"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/empty-node", value: nil},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/uint64-node[.=1234567890]", value: nil},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=BBB]/uint64-node[.=1234567890]", value: nil},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=BBB]/uint32-range", value: []string{"200"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=CCC]/uint32-range", value: []string{"300"}},
		// {wantInsertErr: false, path: "/sample/single-key-list[list-key=DDD]/uint32-range", value: []string{"400"}},
		// {wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=1]/ok", value: []string{"true"}},
		// {wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=2]/str", value: []string{"first"}},
		// {wantInsertErr: false, path: "/sample/multiple-key-list[str=second][integer=1]/str", value: []string{"second"}},
		// {wantInsertErr: false, path: "/sample/multiple-key-list[sample:str=second][integer=2]/str", value: []string{"second"}},
		{wantInsertErr: false, path: "/sample:sample/container-val", value: nil},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: nil},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: []string{"leaf-list-first", "leaf-list-second"}},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: []string{"leaf-list-third"}},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val/leaf-list-fourth", value: nil},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val[.=leaf-list-fifth]", value: nil},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/sample:enum-val", value: []string{"enum2"}},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/sample:test-default", value: []string{"11"}},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/a", value: []string{"A"}},
		// {wantInsertErr: false, path: "/sample:sample/non-key-list", value: []string{`{"uintval": "11", "strval": "XYZ"}`}},
		// {wantInsertErr: false, path: "/sample:sample/non-key-list", value: []string{`{"uintval": "12", "strval": "XYZ"}`}},
		// {wantInsertErr: false, path: "/sample:sample/non-key-list[uintval=13][strval=ABC]", value: nil},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/test-instance-identifier", value: []string{"/sample:sample/sample:container-val/a"}},
		// {wantInsertErr: false, path: "/sample:sample/sample:container-val/test-must", value: []string{"5"}},
		// {wantInsertErr: true, path: "/sample/single-key-list[list-ke=first]", value: []string{"true"}},
		// {wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/uint32-range", value: []string{"493"}},
		// {wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/int8-range", value: []string{"500"}},
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

	// // gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
	// testfinds := []struct {
	// 	expectedNum int
	// 	path        string
	// 	findOption  Option
	// }{
	// 	{expectedNum: 1, path: "/sample/container-val/leaf-list-val[.=leaf-list-fourth]"},
	// 	{expectedNum: 1, path: "/sample/multiple-key-list[str=first][integer=*]/ok"},
	// 	{expectedNum: 1, path: "/sample/single-key-list[sample:list-key=AAA]/list-key"},
	// 	{expectedNum: 1, path: "/sample/single-key-list[list-key='AAA']"},
	// 	{expectedNum: 4, path: "/sample/single-key-list[list-key=*]"},
	// 	{expectedNum: 13, path: "/sample/single-key-list/*"},
	// 	{expectedNum: 14, path: "/sample/*"},
	// 	{expectedNum: 49, path: "/sample/..."},
	// 	{expectedNum: 4, path: "/sample/...", findOption: StateOnly{}},
	// 	{expectedNum: 1, path: "/sample/.../enum-val"},
	// 	{expectedNum: 34, path: "/sample/*/*/"},
	// 	{expectedNum: 3, path: "/sample//non-key-list"},
	// 	{expectedNum: 2, path: "/sample/multiple-key-list[str=first][integer=*]"},
	// 	{expectedNum: 4, path: "/sample/multiple-key-list"},
	// 	{expectedNum: 1, path: "/sample/non-key-list[2]"},
	// 	{expectedNum: 2, path: "/sample/single-key-list[list-key='BBB' or list-key='CCC']"},
	// }
	// for _, tt := range testfinds {
	// 	t.Run(fmt.Sprintf("Find(%s,%v)", tt.path, tt.findOption), func(t *testing.T) {
	// 		var err error
	// 		var node []DataNode
	// 		node, err = Find(RootData, tt.path, tt.findOption)
	// 		if err != nil {
	// 			t.Errorf("Find() path %v error = %v", tt.path, err)
	// 		}
	// 		t.Logf("Find %s (expected num: %d, result: %d)", tt.path, tt.expectedNum, len(node))
	// 		for j := range node {
	// 			jj, _ := MarshalJSON(node[j], RFC7951Format{})
	// 			t.Log(" - found", j+1, "", node[j].Path(), string(jj))
	// 		}
	// 		if tt.expectedNum != len(node) {
	// 			t.Errorf("find error for %s (expected num: %d, result: %d)", tt.path, tt.expectedNum, len(node))
	// 		}
	// 	})
	// }

	// if node, err := Find(RootData, "/sample"); err != nil {
	// 	t.Errorf("Find() path %v error = %v", "/sample", err)
	// } else {
	// 	if b, err := MarshalJSON(node[0], StateOnly{}); err != nil {
	// 		t.Errorf("MarshalJSON() StateOnly error = %v", err)
	// 	} else {
	// 		j := `{"single-key-list":{"AAA":{"uint32-range":100},"BBB":{"uint32-range":200},"CCC":{"uint32-range":300},"DDD":{"uint32-range":400}}}`
	// 		if string(b) != j {
	// 			t.Errorf("MarshalJSON(StateOnly) returns unexpected json  = %v", string(b))
	// 			t.Logf(" Required json: %s", string(j))
	// 		}
	// 	}
	// 	if b, err := MarshalJSON(node[0], RFC7951Format{}, StateOnly{}); err != nil {
	// 		t.Errorf("MarshalJSON(RFC7951Format) StateOnly error = %v", err)
	// 	} else {
	// 		j := `{"sample:single-key-list":[{"uint32-range":100},{"uint32-range":200},{"uint32-range":300},{"uint32-range":400}]}`
	// 		if string(b) != j {
	// 			t.Errorf("MarshalJSON(RFC7951Format, StateOnly) returns unexpected json  = %v", string(b))
	// 			t.Logf(" Required json: %s", string(j))
	// 		}
	// 	}
	// }

	// path := []string{
	// 	"/sample/container-val/leaf-list-val[.=leaf-list-fourth]",
	// }
	// result := []interface{}{
	// 	[]string{"leaf-list-fourth"},
	// }
	// for i := range path {
	// 	value, err := FindValueString(RootData, path[i])
	// 	if err != nil {
	// 		t.Errorf("Find() path %v error = %v", path[i], err)
	// 	}
	// 	t.Logf("FindValue %s", path[i])
	// 	if !reflect.DeepEqual(value, result[i]) {
	// 		t.Error("not equal", value, result[i])
	// 	}
	// 	for j := range value {
	// 		v := value[j]
	// 		t.Log(" - Find", j+1, "", ValueToString(v))
	// 	}
	// }

	// // nodes := RootData.Get("sample")
	// // t.Log(nodes[0].Get("single-key-list[list-key=AAA]"))
	// // t.Log(nodes[0].Lookup("s"))

	// // jj, err := MarshalJSONIndent(RootData, "", " ", RFC7951Format{})
	// // if err != nil {
	// // 	t.Error(err)
	// // }
	// // t.Log(string(jj))

	// for i := len(tests) - 1; i >= 0; i-- {
	// 	tt := tests[i]
	// 	if tt.wantInsertErr {
	// 		continue
	// 	}
	// 	t.Run("Delete."+tt.path, func(t *testing.T) {
	// 		if err := Delete(RootData, tt.path, tt.value...); (err != nil) != tt.wantDeleteErr {
	// 			t.Errorf("Delete() error = %v, wantDeleteErr %v", err, tt.wantDeleteErr)
	// 		}
	// 	})
	// }

	// jsonietf, err := MarshalJSONIndent(RootData, "", " ", RFC7951Format{})
	// if err != nil {
	// 	t.Error(err)
	// }
	// if string(jsonietf) != "{}" {
	// 	t.Error("all nodes are not removed")
	// 	t.Log(string(jsonietf))
	// 	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
	// }
}

func TestComplexModel(t *testing.T) {
	rootschema, err := Load(
		[]string{
			"testdata/modules/choice-case-example.yang",
			"testdata/modules/pattern.yang",
			"testdata/modules/openconfig-simple-target.yang",
			"testdata/modules/openconfig-simple-augment.yang",
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
			"testdata/modules/default.yang",
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

	node, err := Find(test, "config/d1")
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

func TestReplace(t *testing.T) {
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
	root, err := New(rootschema)
	if err != nil {
		t.Fatal(err)
	}
	schema := FindSchema(rootschema, "interfaces/interface")
	for i := 1; i < 5; i++ {
		v := fmt.Sprintf(`{"name":"e%d", "config": {"enabled":"true"}}`, i)
		new, err := New(schema, v)
		if err != nil {
			t.Error(err)
		}
		err = Replace(root, "/interfaces/interface", new)
		if err != nil {
			t.Error(err)
		}
	}
	for i := 3; i < 7; i++ {
		v := `{ "config": {"enabled":"true"}, "state": {"admin-status":"UP"}}`
		new, err := New(schema, v)
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

	b, _ := root.MarshalJSON()
	t.Log(string(b))
}

func TestLeafList(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
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
		// Read-write leaf-list
		{wantInsertErr: false, path: "/sample/leaf-list-rw", value: nil},
		{wantInsertErr: false, path: "/sample/leaf-list-rw", value: []string{"leaf-list-1", "leaf-list-2"}},
		{wantInsertErr: false, path: "/sample/leaf-list-rw", value: []string{"leaf-list-3"}},
		{wantInsertErr: false, path: "/sample/leaf-list-rw/leaf-list-4", value: nil},
		{wantInsertErr: false, path: "/sample/leaf-list-rw[.=leaf-list-5]", value: nil},
		{wantInsertErr: false, path: "/sample/leaf-list-rw", value: []string{"leaf-list-3"}},
		// Read-only leaf-list
		{wantInsertErr: false, path: "/sample/leaf-list-ro", value: []string{"leaf-list-1", "leaf-list-2"}},
		{wantInsertErr: false, path: "/sample/leaf-list-ro", value: []string{"leaf-list-3"}},
		{wantInsertErr: false, path: "/sample/leaf-list-ro/leaf-list-4", value: nil},
		{wantInsertErr: false, path: "/sample/leaf-list-ro[.=leaf-list-5]", value: nil},
		{wantInsertErr: false, path: "/sample/leaf-list-ro", value: []string{"leaf-list-3"}},
		{wantInsertErr: false, path: "/sample/leaf-list-ro", value: nil},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Set.%s %v", tt.path, tt.value), func(t *testing.T) {
			err := Set(RootData, tt.path, tt.value...)
			if (err != nil) != tt.wantInsertErr {
				t.Errorf("Set() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.path)
			}
		})
	}
	fmt.Println(RootData)
}
