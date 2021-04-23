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
				&PathNode{Name: "interfaces", Select: PathSelectFromRoot},
				&PathNode{Name: "interface", Select: PathSelectChild, Predicates: []string{"name=1/1"}},
			},
		},
		{
			path: "/abc:interfaces/id[name=1/10]/name=1/10",
			want: []*PathNode{
				&PathNode{Prefix: "abc", Name: "interfaces", Select: PathSelectFromRoot},
				&PathNode{Name: "id", Select: PathSelectChild, Predicates: []string{"name=1/10"}},
				&PathNode{Name: "name", Select: PathSelectChild, Value: "1/10"},
			},
		},
		{
			path: "/library/book/isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: PathSelectFromRoot},
				&PathNode{Name: "book", Select: PathSelectChild},
				&PathNode{Name: "isbn", Select: PathSelectChild},
			},
		},
		{
			path: "/library/book/isbn/",
			want: []*PathNode{
				&PathNode{Name: "library", Select: PathSelectFromRoot},
				&PathNode{Name: "book", Select: PathSelectChild},
				&PathNode{Name: "isbn", Select: PathSelectChild},
				&PathNode{Name: "", Select: PathSelectChild},
			},
		},
		{
			path: "library/*/isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: PathSelectChild},
				&PathNode{Name: "*", Select: PathSelectAllChildren},
				&PathNode{Name: "isbn", Select: PathSelectChild},
			},
		},
		{
			path: "/library/book/../book/./isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: PathSelectFromRoot},
				&PathNode{Name: "book", Select: PathSelectChild},
				&PathNode{Name: "..", Select: PathSelectParent},
				&PathNode{Name: "book", Select: PathSelectChild},
				&PathNode{Name: ".", Select: PathSelectSelf},
				&PathNode{Name: "isbn", Select: PathSelectChild},
			},
		},
		{
			path: "/library/book/character[born='1950-10-04']/name",
			want: []*PathNode{
				&PathNode{Name: "library", Select: PathSelectFromRoot},
				&PathNode{Name: "book", Select: PathSelectChild},
				&PathNode{Name: "character", Select: PathSelectChild, Predicates: []string{"born='1950-10-04'"}},
				&PathNode{Name: "name", Select: PathSelectChild},
			},
		},
		{
			path: "library//isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: PathSelectChild},
				&PathNode{Name: "", Select: PathSelectAllMatched},
				&PathNode{Name: "isbn", Select: PathSelectChild},
			},
		},
		// for gnmi
		{
			path: "library/.../isbn",
			want: []*PathNode{
				&PathNode{Name: "library", Select: PathSelectChild},
				&PathNode{Name: "...", Select: PathSelectAllMatched},
				&PathNode{Name: "isbn", Select: PathSelectChild},
			},
		},
		// {
		// 	path: "library/.../",
		// 	want: []*PathNode{
		// 		&PathNode{Name: "library", Select: PathSelectChild},
		// 		&PathNode{Name: "...", Select: PathSelectAllMatched},
		// 		&PathNode{Name: "isbn", Select: PathSelectChild},
		// 	},
		// },
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
