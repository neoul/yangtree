package yangtree

import (
	"fmt"
	"testing"

	"github.com/neoul/gdump"
)

func TestDataBranch_JSON(t *testing.T) {
	RootSchema, err := Load([]string{"data"}, nil, nil)
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
		  "enum-val": "enum2",
		  "leaf-list-val": [
		   "leaf-list-first",
		   "leaf-list-second",
		   "leaf-list-third",
		   "leaf-list-fourth"
		  ]
		 },
		 "empty-val": true,
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

	gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")

	// type fields struct {
	// 	schema   *yang.Entry
	// 	parent   *DataBranch
	// 	key      string
	// 	Children map[string]DataNode
	// }
	// tests := []struct {
	// 	name    string
	// 	fields  fields
	// 	want    []byte
	// 	wantErr bool
	// }{
	// 	// TODO: Add test cases.
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		branch := &DataBranch{
	// 			schema:   tt.fields.schema,
	// 			parent:   tt.fields.parent,
	// 			key:      tt.fields.key,
	// 			Children: tt.fields.Children,
	// 		}
	// 		got, err := branch.MarshalJSON()
	// 		if (err != nil) != tt.wantErr {
	// 			t.Errorf("DataBranch.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
	// 			return
	// 		}
	// 		if !reflect.DeepEqual(got, tt.want) {
	// 			t.Errorf("DataBranch.MarshalJSON() = %v, want %v", got, tt.want)
	// 		}
	// 	})
	// }
}

func TestDataBranch_JSON_IETF(t *testing.T) {
	RootSchema, err := Load([]string{"data"}, nil, nil)
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
		  "enum-val": "enum2",
		  "leaf-list-val": [
		   "leaf-list-first",
		   "leaf-list-second",
		   "leaf-list-third",
		   "leaf-list-fourth"
		  ]
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

	gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
}
