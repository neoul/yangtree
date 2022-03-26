[![GoDoc](https://godoc.org/github.com/neoul/yangtree?status.svg)](https://godoc.org/github.com/neoul/yangtree)

# yangtree (YANG Tree)

yangtree is a Go utilities that can be used to:

- Build a runtime data tree and enumerated values for a set of YANG modules.
- Verify the contents of the data tree against the YANG schema. (e.g. range, pattern, when and must statements of the YANG schema)
- Render the data tree to multiple output formats. For example, `XML`, `YAML`, `JSON`, `JSON_IETF`, `gNMI message`, etc.
- Provide the retrieval of the config, state data nodes separately.
- Supports the data node access and control using XPath.

## Usage


```go
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

  // {
  //  "sample:sample": {
  //   "multiple-key-list": [
  //    {
  //     "integer": 1,
  //     "str": "first"
  //    }
  //   ],
  //   "single-key-list": [
  //    {
  //     "country-code": "KR",
  //     "list-key": "A",
  //     "uint32-range": 200
  //    },
  //    {
  //     "country-code": "KR",
  //     "decimal-range": 1.01,
  //     "empty-node": [
  //      null
  //     ],
  //     "list-key": "B",
  //     "uint32-range": 100,
  //     "uint64-node": "1234567890"
  //    }
  //   ],
  //   "str-val": "hello yangtree!"
  //  }
  // }


	// Print the data tree using JSON
	b, err = yangtree.MarshalJSONIndent(rootdata, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

  // {
  //  "sample": {
  //   "multiple-key-list": {
  //    "first": {
  //     "1": {
  //      "integer": 1,
  //      "str": "first"
  //     }
  //    }
  //   },
  //   "single-key-list": {
  //    "A": {
  //     "country-code": "KR",
  //     "list-key": "A",
  //     "uint32-range": 200
  //    },
  //    "B": {
  //     "country-code": "KR",
  //     "decimal-range": 1.01,
  //     "empty-node": null,
  //     "list-key": "B",
  //     "uint32-range": 100,
  //     "uint64-node": 1234567890
  //    }
  //   },
  //   "str-val": "hello yangtree!"
  //  }
  // }

	// Print the data tree using YAML
	b, err = yangtree.MarshalYAMLIndent(rootdata, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

  // sample:
  //  multiple-key-list:
  //   first:
  //    1:
  //     integer: 1
  //     str: first
  //  single-key-list:
  //   A:
  //    country-code: KR
  //    list-key: A
  //    uint32-range: 200
  //   B:
  //    country-code: KR
  //    decimal-range: 1.01
  //    empty-node: 
  //    list-key: B
  //    uint32-range: 100
  //    uint64-node: 1234567890
  //  str-val: hello yangtree!


	// Print the data tree using XML
	b, err = yangtree.MarshalXMLIndent(rootdata, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

  // <root xmlns="https://github.com/neoul/yangtree">
  //  <sample xmlns="urn:network">
  //   <multiple-key-list>
  //    <integer>1</integer>
  //    <str>first</str>
  //   </multiple-key-list>
  //   <single-key-list>
  //    <country-code>KR</country-code>
  //    <list-key>A</list-key>
  //    <uint32-range>200</uint32-range>
  //   </single-key-list>
  //   <single-key-list>
  //    <country-code>KR</country-code>
  //    <decimal-range>1.01</decimal-range>
  //    <empty-node></empty-node>
  //    <list-key>B</list-key>
  //    <uint32-range>100</uint32-range>
  //    <uint64-node>1234567890</uint64-node>
  //   </single-key-list>
  //   <str-val>hello yangtree!</str-val>
  //  </sample>
  // </root>
```

## Sorting by data node key

- **container**: schema_name
- **list**: schema_name + [key1=val1]
- **non-key-list**: schema_name
- **leaf**: schema_name
- **leaf-list**: schema_name

## XPath syntax

- XPATH: https://tools.ietf.org/html/rfc7950#section-6.4.1
- Path: https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md

# ordered-by

yangtree supports `ordered-by` statement that is used for the ordering of the list and leaf-list entries.

- The `ordered-by` statement can be present in lists and leaf-lists with the following types.
  - `system`: Sorted by the system
  - `user`: Sorted by the user
- This statement is ignored if the list represents state data, RPC output parameters, or notification content.
- The list and leaf-list are defined to `ordered-by system` by default.
- `insert` attribute (metadata) is used for the data node set operation if the list or leaf-list nodes are defined with `ordered-by user`.
- The `insert` attribute has {`first`, `last`, `before`, `after`}.
