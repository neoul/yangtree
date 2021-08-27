package yangtree

import "github.com/openconfig/goyang/pkg/yang"

type DataNode interface {
	IsYangDataNode()
	IsNil() bool        // IsNil() is used to check the data node is null.
	IsDataBranch() bool // IsDataBranch() returns true if the data node is a DataBranch.
	IsDataLeaf() bool   // IsDataLeaf() returns true if the data node is a DataLeaf.
	IsLeaf() bool       // IsLeaf() returns true if the data node is a leaf.
	IsLeafList() bool   // IsLeafList() returns true if the data node is a leaf-list.
	Name() string       // Name() returns the name of the data node.
	Key() string        // Key() returns the key string of the data node. The key is an XPath element combined with XPath predicates.

	Schema() *yang.Entry // Schema() returns the schema of the data node.
	Parent() DataNode    // Parent() returns the parent if it is present.

	Insert(child DataNode, opt ...Option) error // Insert() inserts a new child node. It replaces the old one.
	Delete(child DataNode) error                // Delete() deletes the child node if it is present.
	Replace(src DataNode) error                 // Replace() replaces itself to the src node.
	Merge(src DataNode) error                   // Merge() merges the src node including all children to the current data node.

	Set(value string) error // Set() writes the values to the data node. The value must be string.
	Remove() error          // Remote() removes the value if the value is inserted or itself if the value is not specified.

	// New() creates a cild using the key.
	// The key is an XPath element combined with xpath predicates.
	// For example, interface[KEY=VALUE]
	New(key string) (DataNode, error)
	// Update() updates a child that can be identified by the key using the input values.
	Update(key string, value string) (DataNode, error)

	Exist(key string) bool            // Exist() is used to check a data node is present.
	Get(key string) DataNode          // Get() is used to get the first child has the key.
	GetValue(key string) interface{}  // GetValue() is used to get the value of the child that has the key.
	GetValueString(key string) string // GetValueString() is used to get the value, converted to string, of the child that has the key.

	GetAll(key string) []DataNode    // GetAll() is used to get all children that have the key.
	Lookup(prefix string) []DataNode // Lookup() is used to get all children on which their keys start with the prefix.

	Len() int                    // Len() returns the length of children.
	Index(key string) (int, int) // Index() finds all children by the key and returns the range found.
	Child(index int) DataNode    // Child() gets the child of the index.

	String() string
	Path() string                      // Path() returns the path from the root to the current data node.
	PathTo(descendant DataNode) string // PathTo() returns a relative path to a descendant node.
	Value() interface{}                // Value() returns the raw data of the data node.
	ValueString() string               // ValueString() returns the string value of the data node.

	MarshalJSON() ([]byte, error)      // MarshalJSON() encodes the data node to JSON bytes.
	MarshalJSON_IETF() ([]byte, error) // MarshalJSON_IETF() encodes the data node to JSON_IETF (RFC7951) bytes.
	UnmarshalJSON([]byte) error        // UnmarshalJSON() assembles the data node using JSON or JSON_IETF (rfc7951) bytes.

	MarshalYAML() ([]byte, error)         // MarshalYAML() encodes the data node to a YAML bytes.
	MarshalYAML_RFC7951() ([]byte, error) // MarshalYAML_RFC7951() encodes the data node to a YAML bytes.
	UnmarshalYAML([]byte) error           // UnmarshalYAML() assembles the data node using a YAML bytes
}

// yangtree Option
type Option interface {
	IsOption()
}
