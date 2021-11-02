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
	HasMultipleValues() bool  // HasMultipleValues() returns true if the node has a set of values.

	IsLeaf() bool      // IsLeaf() returns true if the data node is an yang leaf.
	IsLeafList() bool  // IsLeafList() returns true if the data node is an yang leaf-list.
	IsList() bool      // IsList() returns true if the data node is an yang list.
	IsContainer() bool // IsContainer returns true if the data node is an yang container node.

	Name() string                      // Name() returns the name of the data node.
	QName(rfc7951 bool) (string, bool) // QName() returns the namespace-qualified name e.g. module-name:node-name or module-prefix:node-name.
	ID() string                        // ID() returns the data node ID. The ID is an XPath element combined with XPath predicates to identify the node instance.

	Schema() *SchemaNode  // Schema() returns the schema of the data node.
	Parent() DataNode     // Parent() returns the parent if it is present.
	Children() []DataNode // Children() returns all child nodes.

	Insert(child DataNode, insert InsertOption) (DataNode, error) // Insert() inserts a new child node. It replaces and returns the old one.
	Delete(child DataNode) error                                  // Delete() deletes the child node if it is present.
	Replace(src DataNode) error                                   // Replace() replaces itself to the src node.
	Merge(src DataNode) error                                     // Merge() merges the src node including all children to the current data node.

	SetString(value ...string) error     // SetString() writes the values to the data node. The value must be string.
	SetStringSafe(value ...string) error // SetStringSafe() writes the values to the data node. It will recover the value if failed.
	UnsetString(value ...string) error   // UnsetString() clear the value of the data node to the default.

	Remove() error // Remove() removes itself.

	// GetOrNew() gets or creates a node having the id (NODE[KEY=VALUE]) and returns
	// the found or created node with the boolean value that
	// indicates the returned node is created.
	GetOrNew(id string, insert InsertOption) (DataNode, bool, error)

	Create(id string, value ...string) (DataNode, error) // Create() creates a child using the node id (NODE_NAME[KEY=VALUE]).
	Update(id string, value ...string) (DataNode, error) // Update() updates a child that has the node id using the input values.

	CreateByMap(pmap map[string]interface{}) error // CreateByMap() updates the data node using pmap (path predicate map) and string values.
	UpdateByMap(pmap map[string]interface{}) error // UpdateByMap() updates the data node using pmap (path predicate map) and string values.

	Exist(id string) bool              // Exist() is used to check a data node is present.
	Get(id string) DataNode            // Get() is used to get the first child has the id.
	GetValue(id string) interface{}    // GetValue() is used to get the value of the child that has the ID.
	GetValueString(id string) string   // GetValueString() is used to get the value, converted to string, of the child that has the ID.
	GetAll(id string) []DataNode       // GetAll() is used to get all children that have the id.
	Lookup(idPrefix string) []DataNode // Lookup() is used to get all children on which their keys start with the prefix string of the node ID.

	Len() int                 // Len() returns the number of children or the number of values.
	Index(id string) int      // Index() finds all children by the node id and returns the position.
	Child(index int) DataNode // Child() gets the child of the index.

	String() string                    // String() returns a string to identify the node.
	Path() string                      // Path() returns the path from the root to the current data node.
	PathTo(descendant DataNode) string // PathTo() returns a relative path to a descendant node.

	Value() interface{}         // Value() returns the raw data of the data node.
	Values() []interface{}      // Values() returns its values using []interface{} slice
	ValueString() string        // ValueString() returns the string value of the data node.
	HasValue(value string) bool // HasValue() returns true if the data node value has the value.
}

// yangtree Option
type Option interface {
	IsOption()
}
