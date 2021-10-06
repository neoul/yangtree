package yangtree

import (
	"github.com/google/go-cmp/cmp"
)

// DiffUpdated() returns created or updated nodes.
// It returns all created, replaced nodes in node2 (including itself) against node1.
// The deleted nodes can be obtained by the reverse input.
// if disDupCmp (disable duplicatable node comparison) is set, duplicatable nodes are not compared.
func DiffUpdated(node1, node2 DataNode, disDupCmp bool) ([]DataNode, []DataNode) {
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
		created := []DataNode{}
		replaced := []DataNode{}
		// created, replaced
		for first := 0; first < len(d2.children); first++ {
			// duplicatable data nodes (non-key list and ro leaf-list node) must have the same position.
			duplicatable := IsDuplicatable(d2.children[first].Schema())
			if duplicatable && disDupCmp {
				c, r := DiffUpdated(nil, d2.children[first], disDupCmp)
				created = append(created, c...)
				replaced = append(replaced, r...)
			} else if duplicatable {
				name := d2.children[first].Name()
				d1children := d1.GetAll(name)
				d2children := d2.GetAll(name)
				for i := range d2children {
					if i < len(d1children) {
						c, r := DiffUpdated(d1children[i], d2children[i], disDupCmp)
						created = append(created, c...)
						replaced = append(replaced, r...)
					} else {
						c, r := DiffUpdated(nil, d2children[i], disDupCmp)
						created = append(created, c...)
						replaced = append(replaced, r...)
					}
				}
				first = len(d2children) - 1
			} else {
				id := d2.children[first].ID()
				c, r := DiffUpdated(d1.Get(id), d2.children[first], disDupCmp)
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
	}
	return nil, nil
}

// DiffCreated() returns created nodes.
// It returns all created nodes in node2 (including itself) against node1.
// The deleted nodes can be obtained by the reverse input.
func DiffCreated(node1, node2 DataNode, disDupCmp bool) []DataNode {
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
		created := []DataNode{}
		// created
		for first := 0; first < len(d2.children); first++ {
			duplicatable := IsDuplicatable(d2.children[first].Schema())
			if duplicatable && disDupCmp {
				c := DiffCreated(nil, d2.children[first], disDupCmp)
				created = append(created, c...)
			} else if duplicatable {
				name := d2.children[first].Name()
				d1children := d1.GetAll(name)
				d2children := d2.GetAll(name)
				for i := range d2children {
					if i < len(d1children) {
						c := DiffCreated(d1children[i], d2children[i], disDupCmp)
						created = append(created, c...)
					} else {
						c := DiffCreated(nil, d2children[i], disDupCmp)
						created = append(created, c...)
					}
				}
				first = len(d2children) - 1
			} else {
				id := d2.children[first].ID()
				c := DiffCreated(d1.Get(id), d2.children[first], disDupCmp)
				created = append(created, c...)
			}
		}
		return created
	case *DataLeaf:
		return nil
	}
	return nil
}

// Diff() returns differences between nodes.
// It returns all created, replaced and deleted nodes in node2 (including itself) against node1.
func Diff(node1, node2 DataNode) ([]DataNode, []DataNode, []DataNode) {
	c, r := DiffUpdated(node1, node2, false)
	d := DiffCreated(node2, node1, false)
	return c, r, d
}

// SetDiff() = Set() + Diff()
func SetDiff(root DataNode, path string, value ...string) ([]DataNode, []DataNode, error) {
	if !IsValid(root) {
		return nil, nil, Errorf(EAppTagDataNodeMissing, "invalid root node")
	}
	new, err := NewDataNode(root.Schema())
	if err != nil {
		return nil, nil, err
	}
	if err = Set(new, path, value...); err != nil {
		return nil, nil, err
	}
	c, d := DiffUpdated(root, new, true)
	if err := root.Merge(new); err != nil {
		return nil, nil, err
	}
	return c, d, nil
}

// MergeDiff() = Merge() + Diff()
func MergeDiff(root DataNode, path string, node DataNode) ([]DataNode, []DataNode, error) {
	if !IsValid(root) {
		return nil, nil, Errorf(EAppTagDataNodeMissing, "invalid root node")
	}
	new, err := NewDataNode(root.Schema())
	if err != nil {
		return nil, nil, err
	}
	if err := Merge(new, path, node); err != nil {
		return nil, nil, err
	}
	c, d := DiffUpdated(root, new, true)
	if err := root.Merge(new); err != nil {
		return nil, nil, err
	}
	return c, d, nil
}
