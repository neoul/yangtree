package yangtree

import (
	"testing"
)

func TestTraverse(t *testing.T) {
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
	root1, err := NewWithValue(RootSchema, jbyte)
	if err != nil {
		t.Fatal(err)
	}
	j, _ := MarshalJSON(root1)
	t.Log(string(j))

	var count int
	traverser := func(n DataNode, at TrvsCallOption) error {
		count++
		return nil
	}
	err = Traverse(root1, traverser, TrvsCalledAtEnter, -1, false)
	if err != nil {
		t.Fatal(err)
	}
	if count != 32 {
		t.Errorf("invalid number traversing nodes, %d", count)
	}
	count = 0
	err = Traverse(root1, traverser, TrvsCalledAtEnter, -1, true)
	if err != nil {
		t.Fatal(err)
	}
	if count != 24 {
		t.Errorf("invalid number traversing nodes, %d", count)
	}
}
