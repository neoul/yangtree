package yangtree

import (
	"fmt"
	"os"
	"testing"

	"github.com/neoul/gdump"
)

func TestDataNode(t *testing.T) {
	RootSchema, err := Load([]string{"data"}, nil, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "test-item",
			args:    args{path: "/"},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/str-val",
				value: []string{"abc"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/empty-val",
				value: []string{"true"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/single-key-list[list-key=first]/",
				value: []string{"true"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/single-key-list[list-ke=first]",
				value: []string{"true"},
			},
			wantErr: true,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/single-key-list[list-key=first]/country-code",
				value: []string{"KR"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/single-key-list[list-key=first]/uint32-range",
				value: []string{"100"},
			},
			wantErr: false,
		},
		{
			name: "range-check",
			args: args{
				path:  "/sample/single-key-list[list-key=first]/uint32-range",
				value: []string{"493"},
			},
			wantErr: true,
		},
		// {
		// 	name: "range-check",
		// 	args: args{
		// 		path:  "/sample/single-key-list[list-key=first]/int8-range",
		// 		value: []string{"500"},
		// 	},
		// 	wantErr: false,
		// },
		{
			name: "decimal-range",
			args: args{
				path:  "/sample/single-key-list[list-key=first]/decimal-range",
				value: []string{"1.01"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/multiple-key-list[str=first][integer=1]/ok",
				value: []string{"true"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/multiple-key-list[str=first][integer=2]/str",
				value: []string{"first"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/leaf-list-val",
				value: nil,
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/leaf-list-val",
				value: []string{"leaf-list-first", "leaf-list-second"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/leaf-list-val",
				value: []string{"leaf-list-third"},
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/leaf-list-val/leaf-list-fourth",
				value: nil,
			},
			wantErr: false,
		},
		{
			name: "test-item",
			args: args{
				path:  "/sample/container-val/enum-val",
				value: []string{"enum2"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+".Insert", func(t *testing.T) {
			if err := Insert(RootData, tt.args.path, tt.args.value...); (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
	node := RootData.Find("/sample/single-key-list[list-key=first]")
	// j, _ := node.MarshalJSON()
	j, _ := MarshalJSONIndent(node, "", " ", true)
	fmt.Println(string(j))

	// jsonietf, err := MarshalJSONIndent(RootData, "", " ", true)
	// if err != nil {
	// 	t.Error(err)
	// }
	// fmt.Println(string(jsonietf))

	// for i := len(tests) - 1; i >= 0; i-- {
	// 	tt := tests[i]
	// 	if tt.wantErr {
	// 		continue
	// 	}
	// 	t.Run(tt.name+".Delete", func(t *testing.T) {
	// 		if err := Delete(RootData, tt.args.path, tt.args.value...); (err != nil) != tt.wantErr {
	// 			t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
	// 		}
	// 	})
	// }

	// jsonietf, err = MarshalJSONIndent(RootData, "", " ", true)
	// if err != nil {
	// 	t.Error(err)
	// }
	// fmt.Println(string(jsonietf))
	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
}
