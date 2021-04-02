package main

import (
	"fmt"
	"os"
	"yangtree"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/spf13/pflag"
)

var (
	pathFiles      = pflag.StringArray("file", []string{}, "yang files to get the paths")
	pathExclude    = pflag.StringArray("exclude", []string{}, "yang modules to be excluded from path generation")
	pathDir        = pflag.StringArray("dir", []string{}, "directories to search yang includes and imports")
	pathWithPrefix = pflag.Bool("with-prefix", false, "include module/submodule prefix in path elements")
	// pathTypes      = pflag.Bool("types", false, "print leaf type")
	// pathPathType   = pflag.String("path-type", "xpath", "path type xpath or gnmi")
)

func main() {
	pflag.Parse()
	schemaroot, err := yangtree.Load(*pathFiles, *pathDir, *pathExclude)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	collected := make([]*yang.Entry, 0, 256)
	for _, entry := range schemaroot.Dir {
		collected = append(collected, yangtree.CollectSchemaEntries(entry, true)...)
	}
	for _, entry := range collected {
		fmt.Println(yangtree.GeneratePath(entry, *pathWithPrefix))
	}
}
