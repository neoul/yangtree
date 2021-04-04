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
	RootData := New(RootSchema)

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
				path:  "/sample/single-key-list[list-key=first]",
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
				path:  "/sample/single-key-list[list-ke=first]/",
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
				path:  "/sample/single-key-list[list-key=first]/dial-code",
				value: []string{"100"},
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
	leaf := RootData.Find("/sample/multiple-key-list[str=first][integer=1]/ok")
	j, _ := leaf.MarshalJSON()
	fmt.Println(string(j))

	for i := len(tests) - 1; i >= 0; i-- {
		tt := tests[i]
		t.Run(tt.name+".Delete", func(t *testing.T) {
			if err := Delete(RootData, tt.args.path, tt.args.value...); (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
