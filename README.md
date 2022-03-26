[![GoDoc](https://godoc.org/github.com/neoul/yangtree?status.svg)](https://godoc.org/github.com/neoul/yangtree)

# yangtree (YANG Tree)

yangtree is a Go utilities that can be used to:

- Build a runtime data tree and enumerated values for a set of YANG modules.
- Verify the contents of the data tree against the YANG schema. (e.g. range, pattern, when and must statements of the YANG schema)
- Render the data tree to multiple output formats. For example, `XML`, `YAML`, `JSON`, `JSON_IETF`, `gNMI message`, etc.
- Provide the retrieval of the config, state data nodes separately.
- Supports the data node access and control using XPath.

## Usage

### Loading YANG files

```go
   // Load shema from YANG files.
   // Load() will load all YANG files from testdata/sample directory.
	RootSchema, err := yanagtree.Load([]string{"testdata/sample"}, nil, nil, YANGTreeOption{LeafListValueAsKey: true})
	if err != nil {
		t.Fatal(err)
	}

   // Create a data node from the root schema
   // New()
   RootData, err := New(RootSchema)
	if err != nil {
		t.Fatal(err)
	}
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
