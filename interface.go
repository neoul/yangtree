package yangtree

// yangtree consists of the data node.
type DataNode interface {
	IsDataNode()
	IsNil() bool              // IsNil() is used to check the data node is null.
	IsBranchNode() bool       // IsBranchNode() returns true if the data node is a DataBranch (a container or a list node).
	IsLeafNode() bool         // IsLeafNode() returns true if the data node is a DataLeaf (a leaf or a multiple leaf-list node) or DataLeafList (a single leaf-list node).
	IsDuplicatableNode() bool // IsDuplicatable() returns true if multiple nodes having the same ID can exist in the tree.
	IsListableNode() bool     // IsListable() returns true if the nodes that has the same schema are listed in the tree.
	IsStateNode() bool        // IsStateNode() returns true if the node is a config=false node.
	HasStateNode() bool       // HasStateNode() returns true if the node has a config=false child node.
	HasMultipleValues() bool  // HasMultipleValues() returns true if the node has a set of values (= *DataLeafList).

	IsLeaf() bool      // IsLeaf() returns true if the data node is an yang leaf.
	IsLeafList() bool  // IsLeafList() returns true if the data node is an yang leaf-list.
	IsList() bool      // IsList() returns true if the data node is an yang list.
	IsContainer() bool // IsContainer returns true if the data node is an yang container node.

	Name() string                      // Name() returns the name of the data node.
	QName(rfc7951 bool) (string, bool) // QName() returns the namespace-qualified name e.g. module-name:node-name or module-prefix:node-name.
	ID() string                        // ID() returns the data node ID (NODE[KEY=VALUE]). The ID is an XPath element combined with XPath predicates to identify the node instance.

	Schema() *SchemaNode  // Schema() returns the schema of the data node.
	Parent() DataNode     // Parent() returns the parent if it is present.
	Children() []DataNode // Children() returns all child nodes.

	Insert(child DataNode, i InsertOption) (DataNode, error) // Insert() inserts a new child node. It replaces and returns the old one.
	Delete(child DataNode) error                             // Delete() deletes the child node if it is present.
	Replace(src DataNode) error                              // Replace() replaces itself to the src node.
	Merge(src DataNode) error                                // Merge() merges the src node including all children to the current data node.
	Remove() error                                           // Remove() removes itself.

	// GetOrNew() gets or creates a node having the id (NODE_NAME or NODE_NAME[KEY=VALUE]) and returns
	// the found or created node with the boolean value that
	// indicates the returned node is created.
	GetOrNew(id string, i InsertOption) (DataNode, bool, error)

	Create(id string, value ...string) (DataNode, error) // Create() creates a child using the node id (NODE_NAME or NODE_NAME[KEY=VALUE]).
	Update(id string, value ...string) (DataNode, error) // Update() updates a child that has the node id (NODE_NAME or NODE_NAME[KEY=VALUE]) using the input values.

	CreateByMap(pmap map[string]interface{}) error // CreateByMap() updates the data node using pmap (path predicate map) and string values.
	UpdateByMap(pmap map[string]interface{}) error // UpdateByMap() updates the data node using pmap (path predicate map) and string values.

	Exist(id string) bool              // Exist() is used to check a data node is present.
	Get(id string) DataNode            // Get() is used to get the first child has the id.
	GetValue(id string) interface{}    // GetValue() is used to get the value of the child that has the id.
	GetValueString(id string) string   // GetValueString() is used to get the value, converted to string, of the child that has the id.
	GetAll(id string) []DataNode       // GetAll() is used to get all children that have the id.
	Lookup(idPrefix string) []DataNode // Lookup() is used to get all children on which their keys start with the prefix string of the node id.

	Len() int                 // Len() returns the number of children or the number of values.
	Index(id string) int      // Index() finds all children by the node id and returns the position.
	Child(index int) DataNode // Child() gets the child of the index.

	String() string                    // String() returns a string to identify the node.
	Path() string                      // Path() returns the path from the root to the current data node.
	PathTo(descendant DataNode) string // PathTo() returns a relative path to a descendant node.

	SetValue(value ...interface{}) error     // SetValue() writes the values to the data node.
	SetValueSafe(value ...interface{}) error // SetValueSafe() writes the values to the data node. It will recover the value if failed.
	UnsetValue(value ...interface{}) error   // UnsetValue() clear the value of the data node to the default.

	SetValueString(value ...string) error     // SetValueString() writes the values to the data node. The value must be string.
	SetValueStringSafe(value ...string) error // SetValueStringSafe() writes the values to the data node. It will recover the value if failed.
	UnsetValueString(value ...string) error   // UnsetValueString() clear the value of the data node to the default.
	HasValueString(value string) bool         // HasValueString() returns true if the data node value has the value.

	Value() interface{}    // Value() returns the raw data of the data node.
	Values() []interface{} // Values() returns its values using []interface{} slice
	ValueString() string   // ValueString() returns the string value of the data node.
}

// yangtree Option
type Option interface {
	IsOption()
}

// SetValueString(node, editopt, valuestring ...)
// SetValue(node, editopt, value ...)
// New YANGTreeOption

// YANGTreeOption is used to store yangtree options.
type YANGTreeOption struct {
	// If SingleLeafList is enabled, leaf-list data represents to a single leaf-list node that contains several values.
	// If disabled, leaf-list data represents to multiple leaf-list nodes that contains each single value.
	SingleLeafList     bool
	LeafListValueAsKey bool   // leaf-list value can be represented to the xpath if it is set to true.
	CreatedWithDefault bool   // DataNode (data node) is created with the default value of the schema if set.
	YANGLibrary2016    bool   // Load ietf-yang-library@2016-06-21
	YANGLibrary2019    bool   // Load ietf-yang-library@2019-01-04
	SchemaSetName      string // The name of the schema set
	// DefaultValueString [json, yaml, xml]
}

func (schemaoption YANGTreeOption) IsOption() {}
