package yangtree

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestUnmarshalYAML(t *testing.T) {
	RootSchema, err := Load([]string{"data/sample"}, nil, nil)
	if err != nil {
		t.Fatalf("model open err: %v\n", err)
	}
	root := make([]DataNode, 3)
	for i := 0; i < 3; i++ {
		root[i], err = New(RootSchema)
		if err != nil {
			t.Fatalf("yangtree creation error: %v\n", err)
		}
		file, err := os.Open(fmt.Sprint("data/yaml/sample", i+1, ".yaml"))
		if err != nil {
			t.Fatalf("file open err: %v\n", err)
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			t.Fatalf("file read error: %v\n", err)
		}
		file.Close()
		if err := root[i].UnmarshalYAML(b); err != nil {
			t.Fatalf("unmarshalling error: %v\n", err)
		}

		if i >= 1 {
			if !Equal(root[i-1], root[i]) {
				t.Errorf("unmarshaled data is not equal")
				b1, err := root[i-1].MarshalJSON()
				if err != nil {
					t.Error(err)
				}
				b2, err := root[i].MarshalJSON()
				if err != nil {
					t.Error(err)
				}
				t.Log(string(b1))
				t.Log(string(b2))
			}
		}
	}
}
