package gnmi

import (
	"github.com/neoul/yangtree"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func Find(root yangtree.DataNode, gpath *gnmipb.Path, option ...yangtree.Option) ([]yangtree.DataNode, error) {
	path := ToPath(gpath)
	return yangtree.Find(root, path, option...)
}
