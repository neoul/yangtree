package yangtree

import (
	"reflect"
	"testing"

	"github.com/neoul/gdump"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		path    string
		want    []*PathNode
		wantErr bool
	}{
		{
			path: "/interfaces/interface[name=1/1]",
			want: []*PathNode{
				&PathNode{Name: "interfaces", Select: NodeSelectFromRoot},
				&PathNode{Name: "interface", Select: NodeSelectChild, Predicates: []string{"name=1/1"}},
			},
		},
		{
			path: "/abc:interfaces/id[name=1/10]/name=1/10",
			want: []*PathNode{
				&PathNode{Prefix: "abc", Name: "interfaces", Select: NodeSelectFromRoot},
				&PathNode{Name: "id", Select: NodeSelectChild, Predicates: []string{"name=1/10"}},
				&PathNode{Name: "name", Select: NodeSelectChild, Value: "1/10"},
			},
		},
		{
			path: "/library/book/isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: NodeSelectFromRoot},
				&PathNode{Name: "book", Select: NodeSelectChild},
				&PathNode{Name: "isbn", Select: NodeSelectChild},
			},
		},
		{
			path: "/library/book/isbn/",
			want: []*PathNode{
				&PathNode{Name: "library", Select: NodeSelectFromRoot},
				&PathNode{Name: "book", Select: NodeSelectChild},
				&PathNode{Name: "isbn", Select: NodeSelectChild},
			},
		},
		{
			path: "library/*/isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: NodeSelectChild},
				&PathNode{Name: "*", Select: NodeSelectAllChildren},
				&PathNode{Name: "isbn", Select: NodeSelectChild},
			},
		},
		{
			path: "/library/book/../book/./isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: NodeSelectFromRoot},
				&PathNode{Name: "book", Select: NodeSelectChild},
				&PathNode{Name: "..", Select: NodeSelectParent},
				&PathNode{Name: "book", Select: NodeSelectChild},
				&PathNode{Name: ".", Select: NodeSelectSelf},
				&PathNode{Name: "isbn", Select: NodeSelectChild},
			},
		},
		{
			path: "/library/book/character[born='1950-10-04']/name",
			want: []*PathNode{
				&PathNode{Name: "library", Select: NodeSelectFromRoot},
				&PathNode{Name: "book", Select: NodeSelectChild},
				&PathNode{Name: "character", Select: NodeSelectChild, Predicates: []string{"born='1950-10-04'"}},
				&PathNode{Name: "name", Select: NodeSelectChild},
			},
		},
		{
			path: "library//isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: NodeSelectChild},
				&PathNode{Name: "", Select: NodeSelectAll},
				&PathNode{Name: "isbn", Select: NodeSelectChild},
			},
		},
		// for gnmi
		{
			path: "library/.../isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: NodeSelectChild},
				&PathNode{Name: "...", Select: NodeSelectAll},
				&PathNode{Name: "isbn", Select: NodeSelectChild},
			},
		},
		{
			path: "library/.../",
			want: []*PathNode{
				&PathNode{Name: "library", Select: NodeSelectChild},
				&PathNode{Name: "...", Select: NodeSelectAll},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := ParsePath(&tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePath() = %v, want %v", got, tt.want)
				gdump.Print(got)
				gdump.Print(tt.want)
			}
		})
	}
}
