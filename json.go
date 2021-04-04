package yangtree

import "encoding/json"

func (branch *DataBranch) MarshalJSON() ([]byte, error) {
	return nil, nil
}
func (leaf *DataLeaf) MarshalJSON() ([]byte, error) {
	if leaf == nil {
		return nil, nil
	}
	return json.Marshal(leaf.Value)
}
func (leaflist *DataLeafList) MarshalJSON() ([]byte, error) {
	if leaflist == nil {
		return nil, nil
	}
	return json.Marshal(leaflist.Value)
}
