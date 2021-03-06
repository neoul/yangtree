package main

import (
	"fmt"
	"os"

	"github.com/neoul/yangtree"
)

func main() {
	file := []string{
		"../../../YangModels/yang/standard/ietf/RFC/iana-if-type@2017-01-19.yang",
		"../../../openconfig/public/release/models/interfaces/openconfig-interfaces.yang",
		"../../../openconfig/public/release/models/system/openconfig-messages.yang",
		"../../../openconfig/public/release/models/telemetry/openconfig-telemetry.yang",
		"../../../openconfig/public/release/models/openflow/openconfig-openflow.yang",
		"../../../openconfig/public/release/models/platform/openconfig-platform.yang",
		"../../../openconfig/public/release/models/system/openconfig-system.yang",
		"../testdata/modules/openconfig-simple-target.yang",
		"../testdata/modules/openconfig-simple-augment.yang",
		"../testdata/modules/openconfig-simple-deviation.yang",
	}
	dir := []string{"../../../openconfig/public/", "../../../YangModels/yang"}
	excluded := []string{"ietf-interfaces"}
	schema, err := yangtree.Load(file, dir, excluded, yangtree.YANGTreeOption{YANGLibrary2019: true})
	if err != nil {
		if merr, ok := err.(yangtree.MultipleError); ok {
			for i := range merr {
				fmt.Fprintf(os.Stderr, "error in loading: %v", merr[i])
			}
		} else {
			fmt.Fprintf(os.Stderr, "error in loading: %v", err)
		}
		os.Exit(1)
	}
	allschema := yangtree.CollectSchemaEntries(schema, true)
	for i := range allschema {
		fmt.Println(yangtree.GeneratePath(allschema[i], true, false))
	}
	os.Exit(0)
}
