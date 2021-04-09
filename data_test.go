package yangtree

import (
	"fmt"
	"testing"

	"github.com/neoul/gdump"
)

func TestDataNode(t *testing.T) {
	RootSchema, err := Load([]string{"data"}, nil, nil)
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
			args:          args{path: "/"},
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
				value: []string{"true"},
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
		{
			name: "uint64-node",
			args: args{
				path:  "/sample/single-key-list[list-key=AAA]/uint64-node",
				value: []string{"1234567890"},
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
				path:  "/sample/container-val/enum-val",
				value: []string{"enum2"},
			},
			wantInsertErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+".Insert", func(t *testing.T) {
			if err := Insert(RootData, tt.args.path, tt.args.value...); (err != nil) != tt.wantInsertErr {
				t.Errorf("Insert() error = %v, wantInsertErr %v", err, tt.wantInsertErr)
			}
		})
	}
	gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
	path := []string{
		// "/sample/multiple-key-list[str=first][integer=*]/ok",
		// "/sample/single-key-list[list-key=AAA]/list-key",
		// "/sample/single-key-list[list-key=AAA]",
		"/sample/single-key-list[list-key=*]",
		// "/sample/single-key-list/*",
		// "/sample/*",
		// "/sample/...",
		// "/sample/.../enum-val",
		// "/sample/*/*/",
	}
	for i := range path {
		node, err := RootData.Retrieve(path[i])
		if err != nil {
			t.Errorf("Retrieve() path %v error = %v", path[i], err)
		}
		fmt.Println(node)
		for j := range node {
			j, _ := MarshalJSON(node[j], true)
			fmt.Println(path[i], string(j))
		}
	}
	// node := RootData.Find("/sample")
	// // j, _ := node.MarshalJSON()
	// j, _ := MarshalJSONIndent(node, "", " ", false)
	// fmt.Println(string(j))

	jj, err := MarshalJSONIndent(RootData, "", " ", false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(jj))

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

	// jsonietf, err = MarshalJSONIndent(RootData, "", " ", true)
	// if err != nil {
	// 	t.Error(err)
	// }
	// fmt.Println(string(jsonietf))
	// gdump.ValueDump(RootData, 12, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
}
