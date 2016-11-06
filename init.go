package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/uk702/gvt/fileutils"
	vendor "github.com/uk702/gvt/gbvendor"
)

func addInitFlags(fs *flag.FlagSet) {
	fs.StringVar(&branch, "branch", "", "branch of the package")
	fs.StringVar(&revision, "revision", "", "revision of the package")
	fs.StringVar(&tag, "tag", "", "tag of the package")
	fs.BoolVar(&noRecurse, "no-recurse", false, "do not fetch recursively")
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
	fs.BoolVar(&tests, "t", false, "fetch _test.go files and testdata")
	fs.BoolVar(&all, "a", false, "fetch all files and subfolders")
	fs.BoolVar(&verbose, "v", false, "verbose show checkout progress")
}

var cmdInit = &Command{
	Name:      "init",
	UsageLine: "init",
	Short:     "scan and download all dependence",
	Long:      `sacn all source files and download all dependence`,
	Run: func(args []string) error {
		m, err := vendor.ReadManifest(manifestFile)
		level := 0

		path, _ := os.Getwd()
		// dir, _ := ioutil.TempDir("", "gvt-")
		deps, err := vendor.ParseImports(path, path, "", tests, all)

		if err != nil {
			return fmt.Errorf("failed to parse imports: %s", err)
		}

		for d := range deps {
			if strings.Index(d, ".") == -1 { // TODO: replace this silly heuristic
				continue
			}

			// 已经存在于 vendor 目录，跳过
			if fileutils.IsFileExist(filepath.Join(vendorDir, d)) {
				// fmt.Println(filepath.Join(vendorDir, d)+"is exist, skip ", d)
				continue
			}

			// 检查是否已经是源代码树中的一部分
			isSourcePath := false

			// 通过 $GOPATH 检查是否已经在编译路径中，包括当前项目内
			for _, ds := range srcTree {
				dstpath := filepath.Join(ds, d)
				if fileutils.IsFileExist(dstpath) {
					// fmt.Println(dstpath+"is exist, skip ", d)
					isSourcePath = true
					break
				}
			}

			if isSourcePath {
				continue
			}

			if err := fetchRecursive(m, d, level+1); err != nil {
				if strings.HasPrefix(err.Error(), "error fetching") {
					fmt.Println(err)
				}
			}
		}

		return err
	},
	AddFlags: addInitFlags,
}
