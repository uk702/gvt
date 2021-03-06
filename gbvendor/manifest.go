package vendor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
)

// gb-vendor manifest support

// Manifest describes the layout of $PROJECT/vendor/manifest.
type Manifest struct {
	// Manifest version. Current manifest version is 0.
	Version int `json:"version"`

	// Depenencies is a list of vendored dependencies.
	Dependencies []Dependency `json:"dependencies"`
}

var (
	DepPresent       = errors.New("dependency already present")
	DepSubPkgPresent = errors.New("subpackages of this dependency are already present")
	DepMissing       = errors.New("dependency does not exist")
)

// AddDependency adds a Dependency to the current Manifest.
// If the dependency exists already then it returns and error.
func (m *Manifest) AddDependency(dep Dependency) error {
	if m.HasImportpath(dep.Importpath) {
		return DepPresent
	}
	if m.GetSubpackages(dep.Importpath) != nil {
		return DepSubPkgPresent
	}
	m.Dependencies = append(m.Dependencies, dep)
	return nil
}

// RemoveDependency removes a Dependency from the current Manifest.
// If the dependency does not exist then it returns an error.
func (m *Manifest) RemoveDependency(dep Dependency) error {
	for i, d := range m.Dependencies {
		if reflect.DeepEqual(d, dep) {
			m.Dependencies = append(m.Dependencies[:i], m.Dependencies[i+1:]...)
			return nil
		}
	}
	return DepMissing
}

// HasImportpath reports whether the Manifest contains the import path,
// or a parent of it.
func (m *Manifest) HasImportpath(path string) bool {
	_, err := m.GetDependencyForImportpath(path)
	return err == nil
}

// GetDependencyForRepository return a dependency for specified import
// path. Note that it might be a parent of the specified path.
// If the dependency does not exist it returns an error.
func (m *Manifest) GetDependencyForImportpath(path string) (Dependency, error) {
	for _, d := range m.Dependencies {
		if path == d.Importpath || strings.HasPrefix(path, d.Importpath+"/") {
			return d, nil
		}
	}
	return Dependency{}, fmt.Errorf("dependency for %s does not exist", path)
}

// GetSubpackages returns any Dependency in the Manifest that is a subpackage
// of the given import path.
func (m *Manifest) GetSubpackages(path string) (deps []Dependency) {
	for _, d := range m.Dependencies {
		if path != d.Importpath && strings.HasPrefix(d.Importpath, path+"/") {
			deps = append(deps, d)
		}
	}
	return
}

// Dependency describes one vendored import path of code
// A Dependency is an Importpath sources from a Respository
// at Revision from Path.
type Dependency struct {
	// Importpath is name by which this dependency is known.
	Importpath string `json:"importpath"`

	// Repository is the remote DVCS location that this
	// dependency was fetched from.
	Repository string `json:"repository"`

	// VCS is the DVCS system found at Repository.
	VCS string `json:"vcs"`

	// Revision is the revision that describes the dependency's
	// remote revision.
	Revision string `json:"revision"`

	// Branch is the branch the Revision was located on.
	// Can be blank if not needed.
	Branch string `json:"branch"`

	// Path is the path inside the Repository where the
	// dependency was fetched from.
	Path string `json:"path,omitempty"`

	// NoTests indicates that test files were ignored.
	// In the negative for backwards compatibility.
	NoTests bool `json:"notests,omitempty"`

	// AllFiles indicates that no files were ignored.
	AllFiles bool `json:"allfiles,omitempty"`
}

// WriteManifest writes a Manifest to the path. If the manifest does
// not exist, it is created. If it does exist, it will be overwritten.
// If the manifest file is empty (0 dependencies) it will be deleted.
// The dependencies will be ordered by import path to reduce churn when making
// changes.
// TODO(dfc) write to temporary file and move atomically to avoid
// destroying a working vendorfile.
func WriteManifest(path string, m *Manifest) error {
	if len(m.Dependencies) == 0 {
		err := os.Remove(path)
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := writeManifest(f, m); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func writeManifest(w io.Writer, m *Manifest) error {
	sort.Sort(byImportpath(m.Dependencies))
	buf, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		return err
	}
	_, err = io.Copy(w, bytes.NewReader(buf))
	return err
}

// ReadManifest reads a Manifest from path. If the Manifest is not
// found, a blank Manifest will be returned.
func ReadManifest(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return new(Manifest), nil
		}
		return nil, err
	}
	defer f.Close()

	var m Manifest
	d := json.NewDecoder(f)
	if err := d.Decode(&m); err != nil {
		return new(Manifest), nil
		// return nil, err
	}

	// Pass all dependencies through AddDependency to detect overlap
	deps := m.Dependencies
	m.Dependencies = nil
	sort.Sort(byImportpath(deps)) // so that subpackages come after parents
	for _, d := range deps {
		if err := m.AddDependency(d); err == DepPresent {
			log.Println("WARNING: overlapping dependency detected:", d.Importpath)
			log.Println("The subpackage will be ignored to fix undefined behavior. See https://git.io/vr8Mu")
		} else if err != nil {
			return nil, err
		}
	}

	return &m, err
}

type byImportpath []Dependency

func (s byImportpath) Len() int           { return len(s) }
func (s byImportpath) Less(i, j int) bool { return s[i].Importpath < s[j].Importpath }
func (s byImportpath) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
