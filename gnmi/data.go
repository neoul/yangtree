package gnmi

import (
	"fmt"

	"github.com/neoul/yangtree"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
)

func Find(root yangtree.DataNode, gpath *gnmipb.Path, option ...yangtree.Option) ([]yangtree.DataNode, error) {
	path := ToPath(false, gpath)
	return yangtree.Find(root, path, option...)
}

func New(schema *yang.Entry, typedvalue *gnmipb.TypedValue) (yangtree.DataNode, error) {
	valstr, err := TypedValueToString(typedvalue)
	if err != nil {
		return nil, err
	}
	return yangtree.NewWithValue(schema, valstr)
}

// Replace() replaces the target data node to the new data node in the path.
// It will return a new created node or the replaced top node.
func Replace(root yangtree.DataNode, path string, new yangtree.DataNode) (yangtree.DataNode, error) {
	if !yangtree.IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	if !yangtree.IsValid(new) {
		return nil, fmt.Errorf("invalid new data node")
	}
	pathnode, err := yangtree.ParsePath(&path)
	if err != nil {
		return nil, err
	}
	var key string
	var found, created yangtree.DataNode
	var pmap map[string]interface{}
	for i := range pathnode {
		if pathnode[i].Select == yangtree.NodeSelectAll ||
			pathnode[i].Select == yangtree.NodeSelectAllChildren {
			err = fmt.Errorf("cannot replace multiple nodes")
			break
		}
		key, pmap, err = yangtree.KeyGen(root.Schema(), pathnode[i])
		if err != nil {
			break
		}
		found = root.Get(key)
		if found == nil {
			found, err = root.New(key)
			if err != nil {
				break
			}
			if created == nil {
				created = found
			}
		}
		root = found
	}
	if err == nil {
		if yangtree.IsEqualSchema(root, new) && len(pmap) > 0 {
			yangtree.UpdateByMap(new, pmap)
		}
		err = root.Replace(new)
	}
	if err != nil {
		if yangtree.IsValid(created) {
			created.Remove()
		}
		return nil, err
	}
	if created != nil {
		return created, nil
	}
	return new, nil
}

// Update() updates the target data node using the new data node in the path.
// It will return a new created node or the updated top node.
func Update(root yangtree.DataNode, path string, new yangtree.DataNode) (yangtree.DataNode, error) {
	if !yangtree.IsValid(root) {
		return nil, fmt.Errorf("invalid root data node")
	}
	if !yangtree.IsValid(new) {
		return nil, fmt.Errorf("invalid new data node")
	}
	node := root
	pathnode, err := yangtree.ParsePath(&path)
	if err != nil {
		return nil, err
	}
	var key string
	var found, created yangtree.DataNode
	for i := range pathnode {
		if pathnode[i].Select == yangtree.NodeSelectAll ||
			pathnode[i].Select == yangtree.NodeSelectAllChildren {
			err = fmt.Errorf("cannot Update multiple nodes")
			break
		}
		key, _, err = yangtree.KeyGen(node.Schema(), pathnode[i])
		if err != nil {
			break
		}
		found = node.Get(key)
		if found == nil {
			found, err = node.New(key)
			if err != nil {
				break
			}
			if created == nil {
				created = found
			}
		}
		node = found
	}
	if err == nil {
		err = node.Merge(new)
	}
	if err != nil {
		if yangtree.IsValid(created) {
			created.Remove()
		}
		return nil, err
	}
	if created != nil {
		return created, nil
	}
	return node, nil
}
