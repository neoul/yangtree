package main

import (
	"fmt"
	"log"

	"github.com/neoul/yangtree"
)

func main() {
	// Load shema from YANG files.
	// Load() will load all YANG files from testdata/sample directory.
	rootschema, err := yangtree.Load([]string{"../../testdata/sample"}, nil, nil)
	if err != nil {
		log.Fatalln(err)
	}

	// Create a data node from the root schema
	// New() creates new data node from the schema node.
	rootdata, err := yangtree.New(rootschema)
	if err != nil {
		log.Fatalln(err)
	}

	// Update data tree using simple XPaths and data.
	yangtree.SetValueString(rootdata, "/sample/str-val", nil, "hello yangtree!")
	yangtree.SetValueString(rootdata, "/sample/single-key-list[list-key=A]/country-code", nil, "KR")
	yangtree.SetValueString(rootdata, "/sample/single-key-list[list-key=A]/decimal-range", nil, "10.1")
	yangtree.SetValueString(rootdata, "/sample/single-key-list[list-key=A]/empty", nil)
	yangtree.SetValueString(rootdata, "/sample/single-key-list[list-key=A]/uint32-range", nil, "200")
	yangtree.SetValueString(rootdata, "/sample/single-key-list[list-key=A]/uint64-node", nil, "0987654321")

	yangtree.SetValue(rootdata, "sample/multiple-key-list[integer=1][str=first]", nil,
		map[interface{}]interface{}{"integer": 1, "str": "first"})
	yangtree.SetValue(rootdata, "sample/single-key-list", nil,
		[]interface{}{
			map[interface{}]interface{}{
				"country-code":  "KR",
				"decimal-range": 1.01,
				"empty-node":    nil,
				"list-key":      "B",
				"uint32-range":  100,
				"uint64-node":   1234567890},
		})

	// rootdata.Create("sample")

	// Print the data tree using RFC7951 (JSON_IETF)
	b, err := yangtree.MarshalJSONIndent(rootdata, "", " ", yangtree.RFC7951Format{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Print the data tree using JSON
	b, err = yangtree.MarshalJSONIndent(rootdata, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Print the data tree using YAML
	b, err = yangtree.MarshalYAMLIndent(rootdata, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Print the data tree using XML
	b, err = yangtree.MarshalXMLIndent(rootdata, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}
