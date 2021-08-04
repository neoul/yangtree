package gnmi

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/neoul/yangtree"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestValueToTypedValue(t *testing.T) {
	RootSchema, err := yangtree.Load([]string{"../testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	RootData, err := yangtree.New(RootSchema)
	if err != nil {
		t.Fatal(err)
	}
	jbytes := `
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
	if err := yangtree.UnmarshalJSON(RootData, []byte(jbytes)); err != nil {
		t.Error(err)
	}

	tests := []struct {
		name    string
		val     interface{}
		enc     gnmipb.Encoding
		want    *gnmipb.TypedValue
		wantErr bool
	}{
		{
			name: "container",
			val:  RootData,
			enc:  gnmipb.Encoding_JSON,
			want: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_JsonVal{JsonVal: []byte(jbytes)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValueToTypedValue(tt.val, tt.enc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValueToTypedValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if b := tt.want.GetJsonVal(); b != nil {
				jval1 := map[string]interface{}{}
				err := json.Unmarshal(b, &jval1)
				if err != nil {
					t.Errorf("ValueToTypedValue() unmarshalling error = %v", err)
				}
				jval2 := map[string]interface{}{}
				err = json.Unmarshal([]byte(jbytes), &jval2)
				if err != nil {
					t.Errorf("ValueToTypedValue() unmarshalling error = %v", err)
				}
				if !reflect.DeepEqual(jval1, jval2) {
					t.Errorf("ValueToTypedValue() = %v, want %v", jval1, jval2)
				}
			} else {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ValueToTypedValue() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
