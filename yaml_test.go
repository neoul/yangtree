package yangtree

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYAML(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatalf("model open err: %v\n", err)
	}
	max := 3
	root := make([]DataNode, max)
	for i := 0; i < max; i++ {
		root[i], err = NewWithValueString(RootSchema)
		if err != nil {
			t.Errorf("yangtree creation error: %v\n", err)
		}
		file, err := os.Open(fmt.Sprint("testdata/yaml/sample", i+1, ".yaml"))
		if err != nil {
			t.Errorf("file open err: %v\n", err)
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			t.Errorf("file read error: %v\n", err)
		}
		file.Close()
		// if err := UnmarshalYAML(root[i], b); err != nil {
		// 	t.Errorf("unmarshalling error: %v\n", err)
		// }
		if err := yaml.Unmarshal(b, root[i]); err != nil {
			t.Errorf("unmarshalling error: %v\n", err)
		}

		if i >= 1 {
			if !Equal(root[i-1], root[i]) {
				t.Errorf("unmarshaled data is not equal")
				b1, err := MarshalJSON(root[i-1])
				if err != nil {
					t.Error(err)
				}
				b2, err := MarshalJSON(root[i])
				if err != nil {
					t.Error(err)
				}
				t.Log(string(b1))
				t.Log(string(b2))
			}
		}
	}

	option := []Option{InternalFormat{}, RFC7951Format{}, nil}
	reversed := make([]DataNode, max)
	for i := 0; i < max; i++ {
		if len(option) > 0 {
			option = option[:len(option)-1]
		}
		b, err := MarshalYAML(root[i], option...)
		if err != nil {
			t.Errorf("yaml marshalling error: %v\n", err)
		}
		reversed[i], err = NewWithValueString(RootSchema)
		if err != nil {
			t.Errorf("yangtree creation error: %v\n", err)
		}
		if err := UnmarshalYAML(reversed[i], b); err != nil {
			t.Errorf("unmarshalling error: %v\n", err)
		}
		if i >= 1 {
			if !Equal(reversed[i-1], reversed[i]) {
				t.Errorf("unmarshaled data is not equal")
				b1, err := MarshalJSON(reversed[i-1])
				if err != nil {
					t.Error(err)
				}
				b2, err := MarshalJSON(reversed[i])
				if err != nil {
					t.Error(err)
				}
				t.Log(string(b1))
				t.Log(string(b2))
			}
		}
	}
}

func BenchmarkYAMLmarshallingOld(b *testing.B) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		b.Fatalf("model open err: %v\n", err)
	}
	max := 3
	root := make([]DataNode, max)
	for i := 0; i < max; i++ {
		root[i], err = NewWithValueString(RootSchema)
		if err != nil {
			b.Errorf("yangtree creation error: %v\n", err)
		}
		file, err := os.Open(fmt.Sprint("testdata/yaml/sample", i+1, ".yaml"))
		if err != nil {
			b.Errorf("file open err: %v\n", err)
		}
		fb, err := ioutil.ReadAll(file)
		if err != nil {
			b.Errorf("file read error: %v\n", err)
		}
		file.Close()
		if err := UnmarshalYAML(root[i], fb); err != nil {
			b.Errorf("unmarshalling error: %v\n", err)
		}
	}
	for n := 0; n < b.N; n++ {
		for i := 0; i < max; i++ {
			_, err := MarshalJSON(root[i])
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkYAMLmarshallingNew(b *testing.B) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		b.Fatalf("model open err: %v\n", err)
	}
	max := 3
	root := make([]DataNode, max)
	for i := 0; i < max; i++ {
		root[i], err = NewWithValueString(RootSchema)
		if err != nil {
			b.Errorf("yangtree creation error: %v\n", err)
		}
		file, err := os.Open(fmt.Sprint("testdata/yaml/sample", i+1, ".yaml"))
		if err != nil {
			b.Errorf("file open err: %v\n", err)
		}
		fb, err := ioutil.ReadAll(file)
		if err != nil {
			b.Errorf("file read error: %v\n", err)
		}
		file.Close()
		if err := UnmarshalYAML(root[i], fb); err != nil {
			b.Errorf("unmarshalling error: %v\n", err)
		}
	}
	for n := 0; n < b.N; n++ {
		for i := 0; i < max; i++ {
			_, err := yaml.Marshal(root[i])
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func TestRepresentItself(t *testing.T) {
	RootSchema, err := Load([]string{"testdata/sample"}, nil, nil)
	if err != nil {
		t.Fatalf("model open err: %v\n", err)
	}
	root, err := NewWithValueString(RootSchema)
	if err != nil {
		t.Errorf("yangtree creation error: %v\n", err)
	}
	file, err := os.Open("testdata/yaml/sample1.yaml")
	if err != nil {
		t.Errorf("file open err: %v\n", err)
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		t.Errorf("file read error: %v\n", err)
	}
	file.Close()
	// if err := UnmarshalYAML(root[i], b); err != nil {
	// 	t.Errorf("unmarshalling error: %v\n", err)
	// }
	if err := yaml.Unmarshal(b, root); err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}
	// if j, _ := MarshalYAMLIndent(root.Get("sample"), "", " ", RepresentItself{}); len(j) > 0 {
	// 	fmt.Println(string(j))
	// }
	// if j, _ := MarshalJSONIndent(root.Get("sample"), "", " ", RepresentItself{}); len(j) > 0 {
	// 	fmt.Println(string(j))
	// }
	// if j, _ := MarshalXMLIndent(root.Get("sample"), "", " ", RepresentItself{}); len(j) > 0 {
	// 	fmt.Println(string(j))
	// }

	enumSchema := RootSchema.FindSchema("/sample/container-val/enum-val")
	datanode, _ := NewWithValue(enumSchema, "enum1")
	if j, _ := MarshalJSON(datanode); string(j) != `"enum1"` {
		t.Errorf("unexpected marshalling result: %v %v\n", string(j), "enum1")
	}
	if j, _ := MarshalJSON(datanode, RepresentItself{}); string(j) != `{"enum-val":"enum1"}` {
		t.Errorf("unexpected marshalling result: %v %v\n", string(j), `{"enum-val":"enum1"}`)
	}
}
