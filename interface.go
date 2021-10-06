package yangtree

import "github.com/openconfig/goyang/pkg/yang"

// yangtree consists of the data node.
type DataNode interface {
	IsYangDataNode()
	IsNil() bool        // IsNil() is used to check the data node is null.
	IsDataBranch() bool // IsDataBranch() returns true if the data node is a DataBranch.
	IsDataLeaf() bool   // IsDataLeaf() returns true if the data node is a DataLeaf.
	IsLeaf() bool       // IsLeaf() returns true if the data node is a leaf.
	IsLeafList() bool   // IsLeafList() returns true if the data node is a leaf-list.
	Name() string       // Name() returns the name of the data node.
	ID() string         // ID() returns the ID of the data node. The ID is an XPath element combined with XPath predicates to identify the node instance.

	Schema() *yang.Entry  // Schema() returns the schema of the data node.
	Parent() DataNode     // Parent() returns the parent if it is present.
	Children() []DataNode // Children() returns all child nodes.

	Insert(child DataNode, edit *EditOption) (DataNode, error) // Insert() inserts a new child node. It replaces and returns the old one.
	Delete(child DataNode) error                               // Delete() deletes the child node if it is present.
	Replace(src DataNode) error                                // Replace() replaces itself to the src node.
	Merge(src DataNode) error                                  // Merge() merges the src node including all children to the current data node.

	Set(value ...string) error     // Set() writes the values to the data node. The value must be string.
	SetSafe(value ...string) error // SetSafe() writes the values to the data node. It will recover the value if failed.
	Unset(value ...string) error   // Unset() clear the value of the data node to the default.

	Remove() error // Remove() removes itself.

	// GetOrNew() gets or creates a node having the id and returns
	// the found or created node with the boolean value that
	// indicates the returned node is created.
	GetOrNew(id string, opt *EditOption) (DataNode, bool, error)

	NewDataNode(id string, value ...string) (DataNode, error) // NewDataNode() creates a cild using the node id (NODE_NAME[KEY=VALUE]).
	Update(id string, value ...string) (DataNode, error)      // Update() updates a child that has the node id using the input values.
	UpdateByMap(pmap map[string]interface{}) error            // UpdateByMap() updates the data node using pmap (path predicate map) and string values.

	Exist(id string) bool              // Exist() is used to check a data node is present.
	Get(id string) DataNode            // Get() is used to get the first child has the id.
	GetValue(id string) interface{}    // GetValue() is used to get the value of the child that has the ID.
	GetValueString(id string) string   // GetValueString() is used to get the value, converted to string, of the child that has the ID.
	GetAll(id string) []DataNode       // GetAll() is used to get all children that have the id.
	Lookup(idPrefix string) []DataNode // Lookup() is used to get all children on which their keys start with the prefix string of the node ID.

	Len() int                 // Len() returns the length of children.
	Index(id string) int      // Index() finds all children by the node id and returns the position.
	Child(index int) DataNode // Child() gets the child of the index.

	String() string
	Path() string                      // Path() returns the path from the root to the current data node.
	PathTo(descendant DataNode) string // PathTo() returns a relative path to a descendant node.
	Value() interface{}                // Value() returns the raw data of the data node.
	ValueString() string               // ValueString() returns the string value of the data node.

	MarshalJSON() ([]byte, error)         // MarshalJSON() encodes the data node to JSON bytes.
	MarshalJSON_RFC7951() ([]byte, error) // MarshalJSON_RFC7951() encodes the data node to JSON_IETF (RFC7951) bytes.
	UnmarshalJSON([]byte) error           // UnmarshalJSON() assembles the data node using JSON or JSON_IETF (rfc7951) bytes.

	MarshalYAML() ([]byte, error)         // MarshalYAML() encodes the data node to a YAML bytes.
	MarshalYAML_RFC7951() ([]byte, error) // MarshalYAML_RFC7951() encodes the data node to a YAML bytes.
	UnmarshalYAML([]byte) error           // UnmarshalYAML() assembles the data node using a YAML bytes
}

// yangtree Option
type Option interface {
	IsOption()
}
