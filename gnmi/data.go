package gnmi

import (
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
	return yangtree.New(schema, valstr)
}
