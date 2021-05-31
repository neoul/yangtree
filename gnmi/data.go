package gnmi

import (
	"fmt"

	"github.com/neoul/yangtree"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
)

func Find(root yangtree.DataNode, gpath *gnmipb.Path, option ...yangtree.Option) ([]yangtree.DataNode, error) {
	path := ToPath(gpath)
	return yangtree.Find(root, path, option...)
}

func New(schema *yang.Entry, typedvalue *gnmipb.TypedValue) (yangtree.DataNode, error) {
	valstr, err := TypedValueToString(typedvalue)
	if err != nil {
		return nil, err
	}
	return yangtree.New(schema, valstr)
}

func insert(root, node yangtree.DataNode, elem []*gnmipb.PathElem) error {
	if root == nil {
		return fmt.Errorf("null node")
	}
	if len(elem) == 0 {
		return fmt.Errorf("no path for the target data node")
	}
	if len(elem) == 1 {
		return root.Insert(node)
	}
	switch elem[0].Name {
	case ".":
		return insert(root, node, elem[1:])
	case "..":
		return insert(root.Parent(), node, elem[1:])
	case "*":
		return fmt.Errorf("unable to specify the target data node")
	case "...":
		return fmt.Errorf("unable to specify the target data node")
	default:
		for _, v := range elem[0].Key {
			if v == "*" {
				return fmt.Errorf("unable to specify the target data node")
			}
		}
	}
	key := GNMIPathElemToXPATH(elem[:1], false)
	got := root.Get(key)
	if got == nil {
		new, err := root.New(key)
		if err != nil {
			return err
		}
		return insert(new, node, elem[1:])
	}
	return insert(got, node, elem[1:])
}

func Insert(root, node yangtree.DataNode, gpath *gnmipb.Path) error {
	return insert(root, node, gpath.Elem)
}

// func Set(node yangtree.DataNode, gpath *gnmipb.Path, typedvalue *gnmipb.TypedValue) (yangtree.DataNode, error) {
// 	return set(node, gpath.Elem, typedvalue)
// }

// func set(node yangtree.DataNode, elem []*gnmipb.PathElem, typedvalue *gnmipb.TypedValue) (yangtree.DataNode, error) {
// 	if typedvalue == nil {
// 		return nil, fmt.Errorf("null typed-value")
// 	}
// 	if node == nil {
// 		return nil, fmt.Errorf("null node")
// 	}
// 	if len(elem) == 0 {
// 		tvstr, err := TypedValueToString(typedvalue)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return nil, node.Set(tvstr)
// 	}
// 	switch elem[0].Name {
// 	case ".":
// 		return set(node, elem[1:], typedvalue)
// 	case "..":
// 		return set(node.Parent(), elem[1:], typedvalue)
// 	case "*":
// 		return nil, fmt.Errorf("unable to identify a data node for set")
// 	case "...":
// 		return nil, fmt.Errorf("unable to identify a data node for set")
// 	default:
// 		for _, v := range elem[0].Key {
// 			if v == "*" {
// 				return nil, fmt.Errorf("unable to identify a data node for set")
// 			}
// 		}
// 	}
// 	key := GNMIPathElemToXPATH(elem[:1], false)
// 	if err := node.Update(key); err != nil {
// 		return nil, err
// 	}

// 	return nil
// }
