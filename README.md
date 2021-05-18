# yangtree (YANG Tree)

yangtree is a Go utilities that can be used to:

- Build a runtime data tree and enumerated values for a set of YANG modules.
- Verify the contents of the data tree against the YANG schema. (e.g. range, pattern, when and must statements of the YANG schema)
- Render the data tree to multiple output formats. For example, JSON, JSON_IETF, gNMI message, etc.
- Provide the retrieval of the config, state data nodes separately.
- Supports the data node access and control using XPath.

## XPath syntax
- XPATH: https://tools.ietf.org/html/rfc7950#section-6.4.1
- Path: https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md