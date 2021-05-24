[![GoDoc](https://godoc.org/github.com/neoul/yangtree?status.svg)](https://godoc.org/github.com/neoul/yangtree)

# yangtree (YANG Tree)

yangtree is a Go utilities that can be used to:

- Build a runtime data tree and enumerated values for a set of YANG modules.
- Verify the contents of the data tree against the YANG schema. (e.g. range, pattern, when and must statements of the YANG schema)
- Render the data tree to multiple output formats. For example, JSON, JSON_IETF, gNMI message, etc.
- Provide the retrieval of the config, state data nodes separately.
- Supports the data node access and control using XPath.

## To be implemented

- NewWithDefault(), Clone(), Merge(), Replace(), Update()
- config, state retrieval

## XPath syntax
- XPATH: https://tools.ietf.org/html/rfc7950#section-6.4.1
- Path: https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md

## NETCONF Operations

```
      operation:  Elements in the <config> subtree MAY contain an
         "operation" attribute, which belongs to the NETCONF namespace
         defined in Section 3.1.  The attribute identifies the point in
         the configuration to perform the operation and MAY appear on
         multiple elements throughout the <config> subtree.

         If the "operation" attribute is not specified, the
         configuration is merged into the configuration datastore.

         The "operation" attribute has one of the following values:

         merge:  The configuration data identified by the element
            containing this attribute is merged with the configuration
            at the corresponding level in the configuration datastore
            identified by the <target> parameter.  This is the default
            behavior.

         replace:  The configuration data identified by the element
            containing this attribute replaces any related configuration
            in the configuration datastore identified by the <target>
            parameter.  If no such configuration data exists in the
            configuration datastore, it is created.  Unlike a
            <copy-config> operation, which replaces the entire target
            configuration, only the configuration actually present in
            the <config> parameter is affected.

         create:  The configuration data identified by the element
            containing this attribute is added to the configuration if
            and only if the configuration data does not already exist in
            the configuration datastore.  If the configuration data
            exists, an <rpc-error> element is returned with an
            <error-tag> value of "data-exists".

         delete:  The configuration data identified by the element
            containing this attribute is deleted from the configuration
            if and only if the configuration data currently exists in
            the configuration datastore.  If the configuration data does
            not exist, an <rpc-error> element is returned with an
            <error-tag> value of "data-missing".

         remove:  The configuration data identified by the element
            containing this attribute is deleted from the configuration
            if the configuration data currently exists in the
            configuration datastore.  If the configuration data does not
            exist, the "remove" operation is silently ignored by the
            server.
```