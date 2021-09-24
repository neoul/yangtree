package yangtree

// TrvsCallOption is an argument of Traverse() to decide where user-defined traverser() is called.
//  - TrvsCalledAtEnter TrvsCallOption // call user-defined traverser() at the entrance of of child nodes.
//  - TrvsCalledAtExit                     // call user-defined traverser() at the exit of of child nodes.
//  - TrvsCalledAtBoth                     // call user-defined traverser() at the entrance and exit of child nodes.
type TrvsCallOption int

const (
	TrvsCalledAtEnter TrvsCallOption = 0 // call user-defined traverser() at the entrance of of child nodes.
	TrvsCalledAtExit  TrvsCallOption = 2 // call user-defined traverser() at the exit of of child nodes.
	TrvsCalledAtBoth  TrvsCallOption = 1 // call user-defined traverser() at the entrance and exit of child nodes.
)

// Traverse() loops the node's all children, descendants and itself to execute traverser() at each node.
func Traverse(node DataNode, traverser func(DataNode, TrvsCallOption) error, calledAt TrvsCallOption, depth int, leafOnly bool) error {
	if !IsValid(node) {
		return Errorf(EAppTagInvalidArg, "invalid data node inserted")
	}
	if traverser == nil {
		return Errorf(EAppTagInvalidArg, "no traverser")
	}
	err := traverse(node, &traverseArg{
		traverser: traverser,
		calledAt:  calledAt,
		leafOnly:  leafOnly,
		depth:     depth,
	})
	// if e, ok := err.(*YError); !ok {
	// 	return Errorf(EAppTagInvalidArg, "%v", e)
	// }
	return err
}

type traverseArg struct {
	traverser func(DataNode, TrvsCallOption) error
	calledAt  TrvsCallOption
	leafOnly  bool
	depth     int
}

func traverse(node DataNode, arg *traverseArg) error {
	if arg.depth == 0 {
		return nil
	}
	switch n := node.(type) {
	case *DataBranch:
		if arg.depth > 0 {
			arg.depth--
		}
		if !arg.leafOnly && (arg.calledAt <= TrvsCalledAtBoth) {
			if err := arg.traverser(n, TrvsCalledAtEnter); err != nil {
				return err
			}
		}
		for i := 0; i < len(n.children); i++ {
			if err := traverse(n.children[i], arg); err != nil {
				return err
			}
		}
		if !arg.leafOnly && (arg.calledAt >= TrvsCalledAtBoth) {
			if err := arg.traverser(n, TrvsCalledAtExit); err != nil {
				return err
			}
		}
	case *DataLeaf:
		if err := arg.traverser(n, TrvsCalledAtEnter); err != nil {
			return err
		}
	}
	return nil
}
