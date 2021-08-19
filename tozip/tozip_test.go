// tozip - generates go byte arrary of the input yang files.
package tozip

import (
	"fmt"
	"testing"
)

func TestZipUnzip(t *testing.T) {

	tests := []struct {
		name string
		file string
	}{
		{name: "builtInYangMetadata", file: "../modules/ietf-yang-metadata@2016-08-05.yang"},
		{name: "builtInYanglib2016", file: "../modules/ietf-yang-library@2016-06-21.yang"},
		{name: "builtInYanglib2019", file: "../modules/ietf-yang-library@2019-01-04.yang"},
		{name: "yangtreeRoot", file: "../modules/yangtree.yang"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipped, err := Zip(tt.file)
			if err != nil {
				t.Errorf("Zip() error = %v", err)
				return
			}
			fmt.Printf("var %s = ", tt.name)
			fmt.Println(GenerateGoCodes(zipped))
			// unzipped, err := Unzip(zipped)
			// if err != nil {
			// 	t.Errorf("Unzip() error = %v", err)
			// }
			// fmt.Println(string(unzipped))
		})
	}
}
