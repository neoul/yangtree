package yangtree

import (
	"strings"
	"testing"
)

func TestNewDataGroup(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	{
		// multiple leaf-list node
		jleaflist := `["first","fourth","second","third"]`
		schema := RootSchema.FindSchema("sample/container-val/leaf-list-val")
		jleaflistnodes, err := NewDataNodeGroup(schema, jleaflist)
		if err != nil {
			t.Fatal(err)
		}
		values := []string{"first", "fourth", "second", "third"}
		expected := []DataNode{}
		for _, value := range values {
			n, _ := NewDataNode(schema, value)
			expected = append(expected, n)
		}
		for i := 0; i < 4; i++ {
			if !Equal(jleaflistnodes.Nodes[i], expected[i]) {
				t.Errorf("not equal: %d (%v, %v)", i, jleaflistnodes.Nodes[i], expected[i])
			}
		}
		j, err := jleaflistnodes.MarshalJSON()
		if err != nil {
			t.Errorf("leaflist json marshalling: %v", err)
		}
		if string(j) != jleaflist {
			t.Errorf("leaflist json marshalling failed: %s", string(j))
			return
		}
	}
	{
		// list node
		jlist := `[
			{
				"sample:country-code": "KR",
				"sample:decimal-range": 1.01,
				"sample:empty-node": [
					null
				],
				"sample:list-key": "AAA",
				"sample:uint32-range": 100,
				"sample:uint64-node": "1234567890"
			},
			{
				"sample:country-code": "KR",
				"sample:decimal-range": 1.01,
				"sample:empty-node": [
					null
				],
				"sample:list-key": "BBB"
			},
			{
				"sample:list-key": "CCC"
			}
		]`

		schema := RootSchema.FindSchema("sample/single-key-list")
		jlistnodes, err := NewDataNodeGroup(schema, jlist)
		if err != nil {
			t.Fatal(err)
		}

		values := []string{
			`{"sample:country-code":"KR","sample:decimal-range":1.01,"sample:empty-node":[null],"sample:list-key":"AAA","sample:uint32-range":100,"sample:uint64-node":"1234567890"}`,
			`{"sample:country-code":"KR","sample:decimal-range":1.01,"sample:empty-node":[null],"sample:list-key":"BBB"}`,
			`{"sample:list-key":"CCC"}`,
		}
		expected := []DataNode{}
		for _, value := range values {
			n, _ := NewDataNode(schema, value)
			expected = append(expected, n)
		}
		for i := 0; i < 3; i++ {
			if !Equal(jlistnodes.Nodes[i], expected[i]) {
				t.Errorf("not equal: %d (%v, %v)", i, jlistnodes.Nodes[i], expected[i])
			}
		}
		j, err := MarshalJSON(jlistnodes, RFC7951Format{})
		if err != nil {
			t.Fatal(err)
		}
		jlist = strings.ReplaceAll(jlist, " ", "")
		jlist = strings.ReplaceAll(jlist, "\t", "")
		jlist = strings.ReplaceAll(jlist, "\n", "")
		if string(j) != jlist {
			t.Errorf("leaflist json marshalling failed: %s", string(j))
			t.Errorf("leaflist json marshalling failed: %s", jlist)
			return
		}
	}
}
