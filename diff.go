package yangtree

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
)

// DiffUpdated() returns updated nodes.
// It returns all created, replaced nodes in node2 (including itself) against node1.
// The deleted nodes can be obtained by the reverse input. e.g. node1 ==> node2, node2 ==> node1
func DiffUpdated(node1, node2 DataNode) ([]DataNode, []DataNode) {
	if node1 == node2 {
		return nil, nil
	}
	if node1 == nil {
		created, _ := Find(node2, "...")
		return created, nil
	}
	if node2 == nil {
		return nil, nil
	}
	if node1.Schema() != node2.Schema() {
		return nil, nil
	}
	switch d1 := node1.(type) {
	case *DataBranch:
		d2 := node2.(*DataBranch)
		// created or replaced nodes
		created := []DataNode{}
		replaced := []DataNode{}
		// created, replaced
		for first := 0; first < len(d2.children); first++ {
			key := d2.children[first].Key()
			if IsDuplicatedList(d2.children[first].Schema()) {
				d1children := d1.GetAll(key)
				d2children := d2.GetAll(key)
				for i := range d2children {
					if i < len(d1children) {
						c, r := DiffUpdated(d1children[i], d2children[i])
						created = append(created, c...)
						replaced = append(replaced, r...)
					} else {
						c, r := DiffUpdated(nil, d2children[i])
						created = append(created, c...)
						replaced = append(replaced, r...)
					}
				}
				first = len(d2children) - 1
			} else {
				c, r := DiffUpdated(d1.Get(key), d2.children[first])
				created = append(created, c...)
				replaced = append(replaced, r...)
			}
		}
		return created, replaced
	case *DataLeaf:
		d2 := node2.(*DataLeaf)
		if cmp.Equal(d1.value, d2.value) {
			return nil, nil
		}
		return nil, []DataNode{d2}
	case *DataLeafList:
		d2 := node2.(*DataLeafList)
		if Equal(d1, d2) {
			return nil, nil
		}
		return nil, []DataNode{d2}
	}
	return nil, nil
}

// DiffUpdated() returns updated nodes.
// It returns all created, replaced nodes in node2 (including itself) against node1.
// The deleted nodes can be obtained by the reverse input. e.g. node1 ==> node2, node2 ==> node1
func DiffCreated(node1, node2 DataNode) []DataNode {
	if node1 == node2 {
		return nil
	}
	if node1 == nil {
		created, _ := Find(node2, "...")
		return created
	}
	if node2 == nil {
		return nil
	}
	if node1.Schema() != node2.Schema() {
		return nil
	}
	switch d1 := node1.(type) {
	case *DataBranch:
		d2 := node2.(*DataBranch)
		// created nodes
		created := []DataNode{}
		// created
		for first := 0; first < len(d2.children); first++ {
			key := d2.children[first].Key()
			if IsDuplicatedList(d2.children[first].Schema()) {
				d1children := d1.GetAll(key)
				d2children := d2.GetAll(key)
				for i := range d2children {
					if i < len(d1children) {
						c := DiffCreated(d1children[i], d2children[i])
						created = append(created, c...)
					} else {
						c := DiffCreated(nil, d2children[i])
						created = append(created, c...)
					}
				}
				first = len(d2children) - 1
			} else {
				c := DiffCreated(d1.Get(key), d2.children[first])
				created = append(created, c...)
			}
		}
		return created
	case *DataLeaf:
		return nil
	case *DataLeafList:
		return nil
	}
	return nil
}

// Diff() returns differences between nodes.
// It returns all created, replaced and deleted nodes in node2 (including itself) against node1.
func Diff(node1, node2 DataNode) ([]DataNode, []DataNode, []DataNode) {
	c, r := DiffUpdated(node1, node2)
	d := DiffCreated(node2, node1)
	return c, r, d
}

// SetDiff() = Set() + Diff()
func SetDiff(root DataNode, path string, value ...string) ([]DataNode, []DataNode, error) {
	if !IsValid(root) {
		return nil, nil, fmt.Errorf("invalid root node")
	}
	new, err := New(root.Schema())
	if err != nil {
		return nil, nil, err
	}
	if err := Set(new, path, value...); err != nil {
		return nil, nil, err
	}
	c, d := DiffUpdated(root, new)
	if err := root.Merge(new); err != nil {
		return nil, nil, err
	}
	return c, d, nil
}
