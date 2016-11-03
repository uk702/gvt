package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/uk702/gvt/fileutils"
	"github.com/uk702/gvt/gbvendor"
)

var (
	branch    string
	revision  string // revision (commit)
	tag       string
	noRecurse bool
	insecure  bool // Allow the use of insecure protocols
	tests     bool
	all       bool

	// Lilx
	verbose bool
)

func addFetchFlags(fs *flag.FlagSet) {
	fs.StringVar(&branch, "branch", "", "branch of the package")
	fs.StringVar(&revision, "revision", "", "revision of the package")
	fs.StringVar(&tag, "tag", "", "tag of the package")
	fs.BoolVar(&noRecurse, "no-recurse", false, "do not fetch recursively")
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
	fs.BoolVar(&tests, "t", false, "fetch _test.go files and testdata")
	fs.BoolVar(&all, "a", false, "fetch all files and subfolders")
	fs.BoolVar(&verbose, "v", false, "verbose show checkout progress")
}

var cmdFetch = &Command{
	Name:      "fetch",
	UsageLine: "fetch [-branch branch] [-revision rev | -tag tag] [-precaire] [-no-recurse] [-t|-a] importpath",
	Short:     "fetch a remote dependency",
	Long: `fetch vendors an upstream import path.

Recursive dependencies are fetched (at their master/tip/HEAD revision), unless they
or their parent package are already present.

If a subpackage of a dependency being fetched is already present, it will be deleted.

The import path may include a url scheme. This may be useful when fetching dependencies
from private repositories that cannot be probed.

Flags:
	-t
		fetch also _test.go files and testdata.
	-a
		fetch all files and subfolders, ignoring ONLY .git, .hg and .bzr.
	-branch branch
		fetch from the named branch. Will also be used by gvt update.
		If not supplied the default upstream branch will be used.
	-no-recurse
		do not fetch recursively.
	-tag tag
		fetch the specified tag.
	-revision rev
		fetch the specific revision from the branch or repository.
		If no revision supplied, the latest available will be fetched.
	-precaire
		allow the use of insecure protocols.

`,
	Run: func(args []string) error {
		switch len(args) {
		case 0:
			return fmt.Errorf("fetch: import path missing")
		case 1:
			path := args[0]
			return fetch(path)
		default:
			return fmt.Errorf("more than one import path supplied")
		}
	},
	AddFlags: addFetchFlags,
}

var (
	fetchRoot    string   // where the current session started
	rootRepoURL  string   // the url of the repo from which the root comes from
	fetchedToday []string // packages fetched during this session
	mapMirrorUrl map[string]string
)

func fetch(path string) error {
	m, err := vendor.ReadManifest(manifestFile)
	if err != nil {
		return fmt.Errorf("could not load manifest: %v", err)
	}

	// Lilx
	// 读入 mirrorUrls
	mapMirrorUrl = make(map[string]string, 30)
	if fileutils.IsFileExist(mirrorUrls) {
		// 读入各个url
		content, err := ioutil.ReadFile(mirrorUrls)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if len(strings.TrimSpace(line)) > 0 {
					//fmt.Println(line)
					str := strings.Split(line, " ")
					if len(str) == 2 {
						dst := strings.TrimSpace(str[1])
						mapMirrorUrl[str[0]] = dst
					}
				}
			}
		}
	}

	if path == "fix" {
		fmt.Println("--- fix fail urls ---")

		// Lilx
		// 如果存在 failFetchUrls 这个文件，就单独下载失败的 url
		if fileutils.IsFileExist(failFetchUrls) {
			// 读入各个url
			content, err := ioutil.ReadFile(failFetchUrls)
			if err == nil {
				// 删除 failFetchUrls
				os.Remove(failFetchUrls)

				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					if len(strings.TrimSpace(line)) > 0 {
						err = fetchRecursive(m, line, 0)
						if err != nil {
							fmt.Println(err)
						}
					}
				}
			}

		}

		return nil
	}

	fetchRoot = stripscheme(path)
	err = fetchRecursive(m, path, 0)

	return err
}

func replaceMirrorPath(fullPath, branch string) (string, string) {
	for k, v := range mapMirrorUrl {
		if strings.HasPrefix(fullPath, k) {
			fullPath = strings.Replace(fullPath, k, v, 1)
		}
	}

	return fullPath, branch
}

func fetchRecursive(m *vendor.Manifest, fullPath string, level int) error {
	path := stripscheme(fullPath)

	// Don't even bother the user about skipping packages we just fetched
	for _, p := range fetchedToday {
		if contains(p, path) {
			return nil
		}
	}

	// First, check if this or a parent is already vendored
	if m.HasImportpath(path) {
		if level == 0 {
			return fmt.Errorf("%s or a parent of it is already vendored", path)
		} else {
			// TODO: print a different message for packages fetched during this session
			logIndent(level, "Skipping (existing):", path)
			return nil
		}
	}

	// Next, check if we are trying to vendor from the same repository we are in
	if importPath != "" && contains(importPath, path) {
		if level == 0 {
			return fmt.Errorf("refusing to vendor a subpackage of \".\"")
		} else {
			logIndent(level, "Skipping (subpackage of \".\"):", path)
			return nil
		}
	}

	if level == 0 {
		log.Println("Fetching:", path)
	} else {
		logIndent(level, "Fetching recursive dependency:", path)
	}

	// Finally, check if we already vendored a subpackage and remove it
	for _, subp := range m.GetSubpackages(path) {
		if !contains(subp.Importpath, fetchRoot) { // ignore parents of the root
			ignore := false
			for _, d := range fetchedToday {
				if contains(d, subp.Importpath) {
					ignore = true // No need to warn the user if we just downloaded it
				}
			}
			if !ignore {
				logIndent(level, "Deleting existing subpackage to prevent overlap:", subp.Importpath)
			}
		}
		if err := m.RemoveDependency(subp); err != nil {
			return fmt.Errorf("failed to remove subpackage: %v", err)
		}
	}
	if err := fileutils.RemoveAll(filepath.Join(vendorDir, path)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing folder: %v", err)
	}

	// Find and download the repository
	replacePathWithMirror, replaceBranch := replaceMirrorPath(fullPath, branch)
	fmt.Println("replacePathWithMirror = " + replacePathWithMirror)
	repo, extra, err := GlobalDownloader.DeduceRemoteRepo(replacePathWithMirror, insecure)
	if err != nil {
		// Lilx
		// 如下载失败，则将 url 加入到 failFetchUrls 中
		// 测试 gvt fetch -a -v golang.org/x/text/transform
		of, _ := os.OpenFile(failFetchUrls, os.O_CREATE|os.O_APPEND, 0666)
		defer of.Close()

		of.WriteString(path + "\n")

		return err
	}

	if level == 0 {
		rootRepoURL = repo.URL()
	}

	var wc vendor.WorkingCopy
	if repo.URL() == rootRepoURL {
		wc, err = GlobalDownloader.Get(repo, replaceBranch, tag, revision, verbose)
	} else {
		wc, err = GlobalDownloader.Get(repo, "", "", "", verbose)
	}
	if err != nil {
		// Lilx
		// 如下载失败，则将 url 加入到 failFetchUrls 中
		of, _ := os.OpenFile(failFetchUrls, os.O_CREATE|os.O_APPEND, 0666)
		defer of.Close()

		of.WriteString(path + "\n")
		return err
	}

	// Add the dependency to the manifest

	rev, err := wc.Revision()
	if err != nil {
		return err
	}

	b, err := wc.Branch()
	if err != nil {
		return err
	}

	dep := vendor.Dependency{
		Importpath: path,
		Repository: repo.URL(),
		VCS:        repo.Type(),
		Revision:   rev,
		Branch:     b,
		Path:       extra,
		NoTests:    !tests,
		AllFiles:   all,
	}

	if err := m.AddDependency(dep); err != nil {
		return err
	}

	// Copy the code to the vendor folder

	dst := filepath.Join(vendorDir, dep.Importpath)
	src := filepath.Join(wc.Dir(), dep.Path)

	if err := fileutils.Copypath(dst, src, !dep.NoTests, dep.AllFiles); err != nil {
		return err
	}

	if err := fileutils.CopyLicense(dst, wc.Dir()); err != nil {
		return err
	}

	if err := vendor.WriteManifest(manifestFile, m); err != nil {
		return err
	}

	// Recurse

	fetchedToday = append(fetchedToday, path)

	if !noRecurse {
		// Look for dependencies in src, not going past wc.Dir() when looking for /vendor/,
		// knowing that wc.Dir() corresponds to rootRepoPath
		if !strings.HasSuffix(dep.Importpath, dep.Path) {
			return fmt.Errorf("unable to derive the root repo import path")
		}
		rootRepoPath := strings.TrimRight(strings.TrimSuffix(dep.Importpath, dep.Path), "/")
		deps, err := vendor.ParseImports(src, wc.Dir(), rootRepoPath, tests, all)
		if err != nil {
			return fmt.Errorf("failed to parse imports: %s", err)
		}

		for d := range deps {
			if strings.Index(d, ".") == -1 { // TODO: replace this silly heuristic
				continue
			}
			if err := fetchRecursive(m, d, level+1); err != nil {
				if strings.HasPrefix(err.Error(), "error fetching") { // I know, ok?
					//Lilx
					//return err
					continue
				} else {
					//Lilx
					//return fmt.Errorf("error fetching %s: %s", d, err)
					continue
				}
			}
		}
	}

	return nil
}

func logIndent(level int, v ...interface{}) {
	prefix := strings.Repeat("·", level)
	v = append([]interface{}{prefix}, v...)
	log.Println(v...)
}

// stripscheme removes any scheme components from url like paths.
func stripscheme(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u.Host + u.Path
}

// Package a contains package b?
func contains(a, b string) bool {
	return a == b || strings.HasPrefix(b, a+"/")
}
