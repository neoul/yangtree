package yangtree

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestDataNode_JSON(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	RootData, err := New(RootSchema)
	if err != nil {
		t.Fatal(err)
	}
	jbyte := `
	{
		"sample": {
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
		 "empty-val": null,
		 "multiple-key-list": {
		  "first": {
		   "1": {
			"integer": 1,
			"ok": true,
			"str": "first"
		   },
		   "2": {
			"integer": 2,
			"str": "first"
		   }
		  }
		 },
		 "non-key-list": [
		  {
		   "strval": "XYZ",
		   "uintval": 10
		  }
		 ],
		 "single-key-list": {
		  "AAA": {
		   "country-code": "KR",
		   "decimal-range": 1.01,
		   "empty-node": null,
		   "list-key": "AAA",
		   "uint32-range": 100,
		   "uint64-node": 1234567890
		  }
		 },
		 "str-val": "abc"
		}
	   }
	`
	if err := RootData.UnmarshalJSON([]byte(jbyte)); err != nil {
		t.Error(err)
	}
	var jdata1 interface{}
	var jdata2 interface{}
	json.Unmarshal([]byte(jbyte), &jdata1)
	if err != nil {
		t.Error(err)
	}

	jbyte2, err := RootData.MarshalJSON()
	if err != nil {
		t.Error(err)
	}
	json.Unmarshal([]byte(jbyte2), &jdata2)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(jdata1, jdata2) {
		t.Errorf("unmarshaled data is not equal.")
		t.Log(jdata1)
		t.Log(jdata2)
	}
}

func TestDataNode_JSON_IETF(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	RootData, err := New(RootSchema)
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
	if err := RootData.UnmarshalJSON([]byte(jbyte)); err != nil {
		t.Error(err)
	}
	var jdata1 interface{}
	var jdata2 interface{}
	json.Unmarshal([]byte(jbyte), &jdata1)
	if err != nil {
		t.Error(err)
	}

	jbyte2, err := RootData.MarshalJSON_IETF()
	if err != nil {
		t.Error(err)
	}
	json.Unmarshal([]byte(jbyte2), &jdata2)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(jdata1, jdata2) {
		t.Errorf("unmarshaled data is not equal.")
		pretty.Print(jdata1)
		pretty.Print(jdata2)
	}

	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
}

func TestDataNode_FindState(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	RootData, err := New(RootSchema)
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
		   "leaf-list-fifth",
		   "leaf-list-first",
		   "leaf-list-fourth",
		   "leaf-list-second",
		   "leaf-list-third"
		  ],
		  "test-default": 11,
		  "test-instance-identifier": "/sample:sample/sample:container-val/a",
		  "test-must": 5
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
		  },
		  {
		   "integer": 1,
		   "str": "second"
		  },
		  {
		   "integer": 2,
		   "str": "second"
		  }
		 ],
		 "non-key-list": [
		  {
		   "strval": "XYZ",
		   "uintval": 11
		  },
		  {
		   "strval": "XYZ",
		   "uintval": 12
		  },
		  {
		   "strval": "ABC",
		   "uintval": 13
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
		  },
		  {
		   "list-key": "BBB",
		   "uint32-range": 200,
		   "uint64-node": "1234567890"
		  },
		  {
		   "list-key": "CCC",
		   "uint32-range": 300
		  },
		  {
		   "list-key": "DDD",
		   "uint32-range": 400
		  }
		 ],
		 "str-val": "abc"
		}
	   }	   
	`
	if err := UnmarshalJSON(RootData, []byte(jbyte)); err != nil {
		t.Error(err)
	}
	var jdata1 interface{}
	var jdata2 interface{}
	json.Unmarshal([]byte(jbyte), &jdata1)
	if err != nil {
		t.Error(err)
	}

	jbyte2, err := MarshalJSON(RootData, RFC7951Format{})
	if err != nil {
		t.Error(err)
	}
	json.Unmarshal([]byte(jbyte2), &jdata2)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(jdata1, jdata2) {
		t.Errorf("unmarshaled data is not equal.")
		pretty.Print(jdata1)
		pretty.Print(jdata2)
	}
	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
}
