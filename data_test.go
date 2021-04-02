package yangtree

import (
	"fmt"
	"os"
	"testing"

	"github.com/neoul/gdump"
)

func TestDataNode_Insert(t *testing.T) {
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
			name:    "add",
			args:    args{path: "/"},
			wantErr: false,
		},
		{
			name: "add",
			args: args{
				path:  "/sample/str-val",
				value: []string{"abc"},
			},
			wantErr: false,
		},
		{
			name: "add",
			args: args{
				path:  "/sample/empty-val",
				value: []string{"true"},
			},
			wantErr: false,
		},
		{
			name: "add",
			args: args{
				path:  "/sample/single-key-list[list-key=first]",
				value: []string{"true"},
			},
			wantErr: false,
		},
		{
			name: "add",
			args: args{
				path:  "/sample/single-key-list[list-ke=first]",
				value: []string{"true"},
			},
			wantErr: true,
		},
		{
			name: "add",
			args: args{
				path:  "/sample/single-key-list[list-ke=first]/",
				value: []string{"true"},
			},
			wantErr: true,
		},
		{
			name: "add",
			args: args{
				path:  "/sample/single-key-list[list-key=first]/country-code",
				value: []string{"KR"},
			},
			wantErr: false,
		},
		{
			name: "add",
			args: args{
				path:  "/sample/single-key-list[list-key=first]/dial-code",
				value: []string{"100"},
			},
			wantErr: false,
		},
		{
			name: "add",
			args: args{
				path:  "/sample/multiple-key-list[str=first][integer=1]",
				value: []string{"100"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Insert(RootData, tt.args.path, tt.args.value...); (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Insert(RootData, tt.args.path, tt.args.value...); (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	gdump.ValueDump(RootData, 5, func(a ...interface{}) { fmt.Print(a...) }, "schema", "parent")
}
