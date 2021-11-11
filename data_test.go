package yangtree

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSetValue(t *testing.T) {
	schema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := New(schema)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path    string
		value   interface{}
		wantErr bool
	}{
		{path: "sample"},
		{path: "sample/container-val"},
		{path: "sample/container-val/a", value: "A"}, // SetValue(root, "sample/container-val/a", nil, "A")
		{path: "sample/container-val/enum-val", value: "enum1"},
		{path: "sample/container-val/enum-val", value: "enum2"},
		{path: "sample/container-val/test-default", value: 11},
		{path: "sample/container-val/leaf-list-val", value: []interface{}{"leaf-list-first", "leaf-list-fourth", "leaf-list-second", "leaf-list-third"}},
		{path: "sample/empty-val"},
		{path: "sample/non-key-list", value: []interface{}{map[interface{}]interface{}{"strval": "XYZ", "uintval": 10}}},
		{path: "sample/multiple-key-list[integer=1][str=first]", value: map[interface{}]interface{}{"integer": 1, "str": "first"}},
		{path: "sample/multiple-key-list", value: map[interface{}]interface{}{
			"first": map[interface{}]interface{}{
				1: map[interface{}]interface{}{
					"integer": 1,
					"ok":      true,
					"str":     "first",
				},
				2: map[interface{}]interface{}{
					"integer": 2,
					"str":     "first",
				},
			}},
		},
		{path: "sample/single-key-list", value: []interface{}{
			map[interface{}]interface{}{"country-code": "KR", "decimal-range": 1.01, "empty-node": nil, "list-key": "AAA", "uint32-range": 100, "uint64-node": 1234567890}},
		},
		{path: "sample/str-val", value: "abc"},
	}
	for _, tt := range tests {
		t.Run("SetValue."+tt.path, func(t *testing.T) {
			if tt.value == nil {
				err = SetValue(root, tt.path, nil)
			} else {
				err = SetValue(root, tt.path, nil, tt.value)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("SetValue() error = %v, wantErr = %v path = %s", err, tt.wantErr, tt.path)
			}
		})
	}

	f, err := os.Open("testdata/json/sample.json")
	if err != nil {
		t.Fatal(err)
	}
	jbyte, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("err in file read: %v", err)
	}
	f.Close()
	refroot, err := New(schema)
	if err := json.Unmarshal(jbyte, refroot); err != nil {
		t.Fatal(err)
	}
	if !Equal(root, refroot) {
		t.Errorf("an yangtree instance built by SetValue is not equal to expected one")
		yb, err := MarshalYAML(root)
		if err != nil {
			t.Fatal(err)
		}
		t.Error(string(yb))

		yb, err = MarshalYAML(refroot)
		if err != nil {
			t.Fatal(err)
		}
		t.Error(string(yb))
	}
	// pretty.Print(root.Value(), root.Values())
	// err = root.SetValue(
	// 	map[interface{}]interface{}{"sample": map[interface{}]interface{}{
	// 		"container-val": map[interface{}]interface{}{
	// 			"a":        "A",
	// 			"enum-val": "enum2",
	// 			"leaf-list-val": []interface{}{
	// 				"leaf-list-first",
	// 				"leaf-list-fourth",
	// 				"leaf-list-second",
	// 				"leaf-list-third",
	// 			},
	// 			"test-default": 11,
	// 		},
	// 		"empty-val": nil,
	// 		"non-key-list": []interface{}{
	// map[interface{}]interface{}{
	// 	"strval":  "XYZ",
	// 	"uintval": 10,
	// },
	// 		},
	// 	}})
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// yb, err := MarshalYAML(root)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Println(string(yb))
	// SetValue(root, "sample/multiple-key-list", nil,
	// 	[]interface{}{
	// 		map[interface{}]interface{}{
	// 			"integer": 1,
	// 			"ok":      true,
	// 			"str":     "first",
	// 		},
	// 		map[interface{}]interface{}{
	// 			"integer": 2,
	// 			"str":     "first",
	// 		},
	// 	})

}

func TestNewWithValueString(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("testdata/json/sample.json")
	if err != nil {
		t.Fatal(err)
	}
	jbyte, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("err in file read: %v", err)
	}
	f.Close()

	root1, err := NewWithValueString(RootSchema, string(jbyte))
	if err != nil {
		t.Fatal(err)
	}
	j, _ := MarshalJSON(root1)
	t.Log(string(j))

	root2, err := NewWithValueString(RootSchema, string(jbyte))
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
	err = SetValueString(root1, "/sample/container-val/enum-val", nil, "enum1")
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
	s := RootSchema.FindSchema("/sample/multiple-key-list")
	if s == nil {
		t.Error("schema multiple-key-list not found")
	}
	mnode, err := NewWithValueString(s, mergingData)
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
	RootData, err := NewWithValueString(RootSchema)
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
		SetValueString(RootData, input[i], nil, "")
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
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil, YANGTreeOption{LeafListValueAsKey: true})
	if err != nil {
		t.Fatal(err)
	}
	RootData, err := NewWithValueString(RootSchema)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path          string
		value         string
		wantInsertErr bool
		wantDeleteErr bool
	}{
		{wantInsertErr: false, path: "/sample"},
		{wantInsertErr: false, path: "/sample/str-val", value: "abc"},
		{wantInsertErr: false, path: "/sample/empty-val", value: "true"},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/", value: ""},
		{wantInsertErr: false, path: "/sample/single-key-list", value: `{"ZZZ":{"list-key": "ZZZ"}}`},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/country-code", value: "KR"},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/uint32-range", value: "100"},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/decimal-range", value: "1.01"},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=AAA]/empty-node", value: ""},
		{wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/uint64-node[.=1234567890]", value: ""}, // failed if leaf key != value
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=BBB]/uint64-node", value: "1234567890"},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=BBB]/uint32-range", value: "200"},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=CCC]/uint32-range", value: "300"},
		{wantInsertErr: false, path: "/sample/single-key-list[list-key=DDD]/uint32-range", value: "400"},
		{wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=1]/ok", value: "true"},
		{wantInsertErr: false, path: "/sample/multiple-key-list[str=first][integer=2]/str", value: "first"},
		{wantInsertErr: false, path: "/sample/multiple-key-list[str=second][integer=1]/str", value: "second"},
		{wantInsertErr: false, path: "/sample/multiple-key-list[sample:str=second][integer=2]/str", value: "second"},
		{wantInsertErr: false, path: "/sample:sample/container-val", value: ""},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: `["leaf-list-first"]`},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: `["leaf-list-second"]`},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val", value: `["leaf-list-third"]`},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val/leaf-list-fourth", value: ""},
		{wantInsertErr: false, path: "/sample/container-val/leaf-list-val[.=leaf-list-fifth]", value: "leaf-list-fifth"},
		{wantInsertErr: true, path: "/sample/container-val/leaf-list-val[.=leaf-list-fifth]", value: ""}, // failed if leaf-list key != value
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/sample:enum-val", value: "enum2"},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/sample:test-default", value: "11"},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/a", value: "A"},
		{wantInsertErr: false, path: "/sample:sample/non-key-list", value: `[{"uintval":"11","strval":"XYZ"}]`},
		{wantInsertErr: false, path: "/sample:sample/non-key-list", value: `[{"uintval":"12","strval":"XYZ"}]`},
		{wantInsertErr: false, path: "/sample:sample/non-key-list", value: `[{"uintval":13,"strval":"ABC"}]`},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/test-instance-identifier", value: "/sample:sample/sample:container-val/a"},
		{wantInsertErr: false, path: "/sample:sample/sample:container-val/test-must", value: "5"},
		{wantInsertErr: true, path: "/sample/single-key-list[list-ke=first]", value: "true"},
		{wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/uint32-range", value: "493"},
		{wantInsertErr: true, path: "/sample/single-key-list[list-key=AAA]/int8-range", value: "500"},
	}
	for _, tt := range tests {
		t.Run("SetValueString."+tt.path, func(t *testing.T) {
			err := SetValueString(RootData, tt.path, nil, tt.value)
			if (err != nil) != tt.wantInsertErr {
				t.Errorf("SetValueString() error = %v, wantInsertErr = %v path = %s", err, tt.wantInsertErr, tt.path)
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
		findOption  Option
	}{
		{expectedNum: 1, path: "/sample/container-val/leaf-list-val[.=leaf-list-fourth]"},
		{expectedNum: 1, path: "/sample/multiple-key-list[str=first][integer=*]/ok"},
		{expectedNum: 1, path: "/sample/single-key-list[sample:list-key=AAA]/list-key"},
		{expectedNum: 1, path: "/sample/single-key-list[list-key='AAA']"},
		{expectedNum: 5, path: "/sample/single-key-list[list-key=*]"},
		{expectedNum: 13, path: "/sample/single-key-list/*"},
		{expectedNum: 5, path: "/sample/single-key-list"},
		{expectedNum: 15, path: "/sample/*"},
		{expectedNum: 54, path: "/sample/..."},
		{expectedNum: 4, path: "/sample/...", findOption: StateOnly{}},
		{expectedNum: 1, path: "/sample/.../enum-val"},
		{expectedNum: 38, path: "/sample/*/*/"},
		{expectedNum: 3, path: "/sample//non-key-list"},
		{expectedNum: 2, path: "/sample/multiple-key-list[str=first][integer=*]"},
		{expectedNum: 4, path: "/sample/multiple-key-list"},
		{expectedNum: 1, path: "/sample/non-key-list[2]"},
		{expectedNum: 2, path: "/sample/single-key-list[list-key='BBB' or list-key='CCC']"},
		{expectedNum: 5, path: "/sample/container-val/leaf-list-val"},
	}
	for _, tt := range testfinds {
		t.Run(fmt.Sprintf("Find(%s,%v)", tt.path, tt.findOption), func(t *testing.T) {
			var err error
			var node []DataNode
			node, err = Find(RootData, tt.path, tt.findOption)
			if err != nil {
				t.Errorf("Find() path %v error = %v", tt.path, err)
				return
			}
			t.Logf("Find %s (expected num: %d, result: %d)", tt.path, tt.expectedNum, len(node))
			for j := range node {
				jj, _ := MarshalJSON(node[j], RFC7951Format{})
				t.Log(" - found", j+1, "", node[j].Path(), string(jj))
			}
			if tt.expectedNum != len(node) {
				t.Errorf("find error for %s (expected num: %d, result: %d)", tt.path, tt.expectedNum, len(node))
				return
			}
		})
	}

	if node, err := Find(RootData, "/sample"); err != nil {
		t.Errorf("Find() path %v error = %v", "/sample", err)
	} else {
		if b, err := MarshalJSON(node[0], StateOnly{}); err != nil {
			t.Errorf("MarshalJSON() StateOnly error = %v", err)
		} else {
			j := `{"single-key-list":{"AAA":{"uint32-range":100},"BBB":{"uint32-range":200},"CCC":{"uint32-range":300},"DDD":{"uint32-range":400},"ZZZ":{}}}`
			if string(b) != j {
				t.Errorf("MarshalJSON(StateOnly) returns unexpected json  = %v", string(b))
				t.Logf(" Required json: %s", string(j))
			}
		}
		if b, err := MarshalJSON(node[0], RFC7951Format{}, StateOnly{}); err != nil {
			t.Errorf("MarshalJSON(RFC7951Format) StateOnly error = %v", err)
		} else {
			j := `{"sample:single-key-list":[{"uint32-range":100},{"uint32-range":200},{"uint32-range":300},{"uint32-range":400},{}]}`
			if string(b) != j {
				t.Errorf("MarshalJSON(RFC7951Format, StateOnly) returns unexpected json  = %v", string(b))
				t.Logf(" Required json: %s", string(j))
			}
		}
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
			t.Log(" - Find", j+1, "", ValueToValueString(v))
		}
	}

	// nodes := RootData.Get("sample")
	// t.Log(nodes[0].Get("single-key-list[list-key=AAA]"))
	// t.Log(nodes[0].Lookup("s"))

	// jj, err := MarshalJSONIndent(RootData, "", " ", RFC7951Format{})
	// if err != nil {
	// 	t.Error(err)
	// }
	// t.Log(string(jj))

	for i := len(tests) - 1; i >= 0; i-- {
		tt := tests[i]
		if tt.wantInsertErr {
			continue
		}
		t.Run("Delete."+tt.path, func(t *testing.T) {
			if err := Delete(RootData, tt.path); (err != nil) != tt.wantDeleteErr {
				t.Errorf("Delete() error = %v, wantDeleteErr %v", err, tt.wantDeleteErr)
			}
		})
	}

	jsonietf, err := MarshalJSONIndent(RootData, "", " ", RFC7951Format{})
	if err != nil {
		t.Error(err)
	}
	if string(jsonietf) != "{}" {
		t.Error("all nodes are not removed")
		t.Log(string(jsonietf))
		// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
	}
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
	rootdata, err := NewWithValueString(rootschema)
	if err != nil {
		t.Fatal(err)
	}
	simpleChoiceCase, err := rootdata.Create("simple-choice-case")
	if err != nil {
		t.Error(err)
	}
	_, err = simpleChoiceCase.Update("a", "a.value")
	if err != nil {
		t.Error(err)
	}
	_, err = simpleChoiceCase.Update("b", "b.value")
	if err != nil {
		t.Error(err)
	}
	choiceCaseAnonymousCase, err := rootdata.Create("choice-case-anonymous-case")
	if err != nil {
		t.Error(err)
	}
	_, err = choiceCaseAnonymousCase.Update("foo/a", "a.value")
	if err == nil {
		t.Error("choice and case should not be present in the tree.")
	}
	_, err = choiceCaseAnonymousCase.Update("a", "a.value")
	if err != nil {
		t.Error(err)
	}
	_, err = choiceCaseAnonymousCase.Update("b", "b.value")
	if err != nil {
		t.Error(err)
	}
	choiceCaseWithLeafref, err := rootdata.Create("choice-case-with-leafref")
	if err != nil {
		t.Error(err)
	}
	_, err = choiceCaseWithLeafref.Update("referenced", "referenced.value")
	if err != nil {
		t.Error(err)
	}
	node, err := choiceCaseWithLeafref.Update("ptr", "ok?")
	if err != nil {
		t.Error(err)
	}
	if err := Validate(node); err == nil {
		t.Error("leafref value must be present in the tree.")
	}
	node, err = choiceCaseWithLeafref.Update("ptr", "referenced.value")
	if err != nil {
		t.Error(err)
	}
	if err := Validate(node); err != nil {
		t.Error(err)
	}

	if _, err = rootdata.Update("pattern-type", "x"); err == nil {
		t.Error(err)
	}
	if _, err = rootdata.Update("pattern-type", "abc"); err != nil {
		t.Error(err)
	}

	j, err := MarshalJSON(rootdata)
	if err != nil {
		t.Error(err)
	}

	t.Log(string(j))
}

func TestCreatedWithDefault(t *testing.T) {
	rootschema, err := Load(
		[]string{
			"testdata/modules/default.yang",
		}, nil, nil, YANGTreeOption{CreatedWithDefault: true})
	if err != nil {
		t.Fatal(err)
	}
	rootdata, err := NewWithValueString(rootschema)
	if err != nil {
		t.Fatal(err)
	}
	test, err := rootdata.Create("test")
	if err != nil {
		t.Error(err)
	}
	_, err = test.Create("config")
	if err != nil {
		t.Error(err)
	}
	_, err = test.Create("state")
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

	j, err := MarshalJSON(rootdata)
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
	root, err := NewWithValueString(rootschema)
	if err != nil {
		t.Fatal(err)
	}
	schema := rootschema.FindSchema("interfaces/interface")
	for i := 1; i < 5; i++ {
		v := fmt.Sprintf(`{"name":"e%d", "config": {"enabled":"true"}}`, i)
		new, err := NewWithValueString(schema, v)
		if err != nil {
			t.Error(err)
		}
		err = Replace(root, fmt.Sprintf("/interfaces/interface[name=e%d]", i), new)
		if err != nil {
			t.Error(err)
		}
	}
	for i := 3; i < 7; i++ {
		v := `{ "config": {"enabled":"true"}, "state": {"admin-status":"UP"}}`
		new, err := NewWithValueString(schema, v)
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

	b, _ := MarshalJSON(root)
	t.Log(string(b))
}

func TestEdit(t *testing.T) {
	schema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewWithValueString(schema)
	if err != nil {
		t.Fatal(err)
	}
	var updated, deleted []DataNode
	callback := func(op EditOp, old, new []DataNode) error {
		for i := range new {
			updated = append(updated, new[i])
		}
		for i := range old {
			deleted = append(deleted, old[i])
		}
		return nil
	}
	tests := []struct {
		path     string
		value    interface{}
		opt      EditOption
		expected int
		wantErr  bool
	}{
		// editing a leaf node
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/str-val", value: "C1", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/str-val", value: "C2", expected: 0, wantErr: true},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/str-val", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/str-val", expected: 0, wantErr: true},
		{opt: EditOption{EditOp: EditReplace, Callback: callback}, path: "/sample/str-val", value: "R1", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditReplace, Callback: callback}, path: "/sample/str-val", value: "R2", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditRemove, Callback: callback}, path: "/sample/str-val", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditRemove, Callback: callback}, path: "/sample/str-val", expected: 0, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, Callback: callback}, path: "/sample/str-val", value: "M1", expected: 1, wantErr: false},
		// editing a container node
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/sample:container-val", value: `{"enum-val":"enum2"}`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditReplace, Callback: callback}, path: "/sample/sample:container-val", value: `{"enum-val":"enum3"}`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, Callback: callback}, path: "/sample/sample:container-val", value: `{"enum-val":"enum1"}`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/sample:container-val", value: `{"enum-val":"enum2"}`, expected: 1, wantErr: true},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/sample:container-val", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/sample:container-val", expected: 1, wantErr: true},
		{opt: EditOption{EditOp: EditRemove, Callback: callback}, path: "/sample/sample:container-val", expected: 0, wantErr: false},
		// editing leaf-list nodes
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/container-val/leaf-list-val", value: `["first"]`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/container-val/leaf-list-val", value: `["first"]`, expected: 0, wantErr: true},
		{opt: EditOption{EditOp: EditReplace, Callback: callback}, path: "/sample/container-val/leaf-list-val", value: `["second","third"]`, expected: 2, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, Callback: callback}, path: "/sample/container-val/leaf-list-val", value: `["fifth","fourth"]`, expected: 2, wantErr: false},
		{opt: EditOption{EditOp: EditRemove, Callback: callback}, path: "/sample/container-val/leaf-list-val[.=third]", value: `third`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/container-val/leaf-list-val[.=third]", value: `third`, expected: 0, wantErr: true},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/container-val/leaf-list-val", expected: 3, wantErr: false},
		{opt: EditOption{EditOp: EditReplace, Callback: callback}, path: "/sample/container-val/leaf-list-val", value: `["second","third"]`, expected: 2, wantErr: false},
		{opt: EditOption{EditOp: EditRemove, Callback: callback}, path: "/sample/container-val/leaf-list-val", expected: 2, wantErr: false},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/container-val/leaf-list-val", expected: 0, wantErr: true},
		// editing list nodes
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/single-key-list", expected: 0, wantErr: true},
		{opt: EditOption{EditOp: EditRemove, Callback: callback}, path: "/sample/single-key-list", expected: 0, wantErr: false},
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/single-key-list", value: `[{"list-key":"AAA","uint32-range":100,"uint64-node":1234567890}]`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/single-key-list", value: `{"AAA":{"uint32-range":100,"uint64-node":123456789}}`, expected: 1, wantErr: true},
		{opt: EditOption{EditOp: EditReplace, Callback: callback}, path: "/sample/single-key-list", value: `{"BBB":{"uint32-range":101,"uint64-node":123456789}}`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, Callback: callback}, path: "/sample/single-key-list", value: `{"CCC":{"uint32-range":151,"uint64-node":123456789}}`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/single-key-list[list-key=AAA]", value: `{"uint32-range":200,"uint64-node":123456789}`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditReplace, Callback: callback}, path: "/sample/single-key-list[list-key=DDD]", value: `{"uint32-range":201,"uint64-node":123456789}`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, Callback: callback}, path: "/sample/single-key-list[list-key=EEE]", value: `{"uint32-range":202,"uint64-node":123456789}`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, Callback: callback}, path: "/sample/single-key-list[list-key=BBB]/empty-node", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/single-key-list[list-key=B]", expected: 0, wantErr: true},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/single-key-list[list-key=BBB]", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/single-key-list[list-key=BBB]", expected: 0, wantErr: true},
		{opt: EditOption{EditOp: EditRemove, Callback: callback}, path: "/sample/single-key-list[list-key=BBB]", expected: 0, wantErr: false},
		{opt: EditOption{EditOp: EditRemove, Callback: callback}, path: "/sample/single-key-list[list-key=DDD]", expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditDelete, Callback: callback}, path: "/sample/single-key-list", expected: 3, wantErr: false},
		{opt: EditOption{EditOp: EditCreate, Callback: callback}, path: "/sample/non-key-list", value: `[{"uintval":"1","strval":"a"}]`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditReplace, Callback: callback}, path: "/sample/non-key-list", value: `[{"uintval":"2","strval":"b"}]`, expected: 1, wantErr: false}, // replace all existent nodes.
		{opt: EditOption{EditOp: EditMerge, Callback: callback}, path: "/sample/non-key-list", value: `[{"uintval":"3","strval":"c"}]`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, InsertOption: InsertToFirst{}, Callback: callback}, path: "/sample/non-key-list", value: `[{"uintval":"1","strval":"first"}]`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, InsertOption: InsertToLast{}, Callback: callback}, path: "/sample/non-key-list", value: `[{"uintval":"1","strval":"last"}]`, expected: 1, wantErr: false},
		{opt: EditOption{EditOp: EditMerge, InsertOption: InsertToAfter{Key: "uintval"}, Callback: callback}, path: "/sample/non-key-list", value: `[{"uintval":"1","strval":"last"}]`, expected: 0, wantErr: true},
	}
	for _, tt := range tests {
		name := tt.path + "," + tt.opt.String()
		if tt.value != nil {
			name = name + "," + tt.value.(string)
		}
		if tt.wantErr {
			name = name + ",wantError"
		}
		t.Run(name, func(t *testing.T) {
			val := []string{}
			if tt.value != nil {
				val = append(val, tt.value.(string))
			}
			deleted = nil
			updated = nil
			err := SetValueString(root, tt.path, &tt.opt, val...)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetValueString(%s) error = %v, wantErr %v", name, err, tt.wantErr)
				return
			}
			var got []DataNode
			switch tt.opt.GetOperation() {
			case EditRemove, EditDelete:
				got = deleted
			default:
				got = nil
				for i := range updated {
					equal := false
					for j := range deleted {
						if Equal(updated[i], deleted[j]) {
							equal = true
						}
					}
					if !equal {
						got = append(got, updated[i])
					}
				}
			}

			if !tt.wantErr {
				if len(got) != tt.expected {
					t.Errorf("SetValueString(%s) num = %v, expected %v", name, len(got), tt.expected)
					for i := range updated {
						t.Logf("updated %s\n", updated[i].Path())
					}
					for i := range deleted {
						t.Logf("deleted %s\n", deleted[i].Path())
					}
					return
				}
				_, err := Find(root, tt.path)
				if err != nil {
					t.Errorf("Find(%s) error: %v", name, err)
					return
				}
				if len(got) == 0 {
					return
				}
				switch tt.opt.EditOp {
				case EditRemove, EditDelete:
					return
					// if len(found) > 0 {
					// 	t.Errorf("SetValueString(%s) error: %v", name, fmt.Errorf("deleting data node %q found", tt.path))
					// 	return
					// }
				}
				switch {
				case got[0].IsLeafList():
					group, err := ConvertToGroup(nil, got)
					if err != nil {
						t.Errorf("SetValueString() error: %v", fmt.Errorf("converting to data node group failed: %v", err))
						return
					}
					b, err := group.MarshalJSON()
					if err != nil {
						t.Errorf("SetValueString() error: %v", fmt.Errorf("marshalling json for %q failed: %v", tt.path, err))
						return
					}
					if string(b) != tt.value {
						t.Errorf("SetValueString() error: %v", fmt.Errorf(
							"the value %q of the editing data %q is not equal to %q",
							string(b), tt.path, tt.value))
						return
					}
				case got[0].IsLeaf():
					if tt.value != nil && got[0].ValueString() != tt.value {
						t.Errorf("SetValueString() error: %v", fmt.Errorf(
							"the value %q of the editing data %q is not equal to %q",
							got[0].ValueString(), tt.path, tt.value))
						return
					}
				}
			}
		})
	}
	y, err := MarshalYAML(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("\n%s\n", string(y))
}

func TestAnyData(t *testing.T) {
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
		  },
		  {
			"strval": "ABC",
			"uintval": 11
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
	root1, err := NewWithValueString(RootSchema, jbyte)
	if err != nil {
		t.Fatal(err)
	}
	j, _ := MarshalJSON(root1)
	t.Log(string(j))

	root2, err := NewWithValueString(RootSchema, jbyte)
	if err != nil {
		t.Fatal(err)
	}
	j, _ = MarshalJSON(root2)
	t.Log(string(j))

	if err := SetValueString(root1, "sample/any", &EditOption{}); err != nil {
		t.Fatal(err)
	}
	nodes, err := Find(root1, "sample/any")
	if err != nil {
		t.Fatal(err)
	}
	any := nodes[0]
	nodes, err = Find(root2, "sample/non-key-list")
	if err != nil {
		t.Fatal(err)
	}

	if _nodes, err := Find(root2, "sample/container-val"); err != nil {
		t.Fatal(err)
	} else {
		nodes = append(nodes, _nodes...)
	}
	for _, node := range nodes {
		_, err = any.Insert(node, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	var anystr = `container-val:
 a: A
 enum-val: enum2
 leaf-list-val:
  - leaf-list-first
  - leaf-list-fourth
  - leaf-list-second
  - leaf-list-third
 test-default: 11
non-key-list:
 - strval: XYZ
   uintval: 10
 - strval: ABC
   uintval: 11
`
	y, _ := MarshalYAML(any)
	if !strings.Contains(anystr, string(y)) {
		t.Errorf("any has different values: %v", string(y))
	}

	for _, node := range nodes {
		err = any.Delete(node)
		if err != nil {
			t.Fatal(err)
		}
	}
	y, _ = MarshalYAML(any)
	if anystr == `` {
		t.Errorf("any has different values: %v", string(y))
	}

	// collector := NewCollector()
	// collector.Insert(root2)
	// y, _ = MarshalYAML(collector)
	// fmt.Println(string(y))
}
