package gta

import (
	"errors"
	"go/build"
)

var (
	// ErrNoDiffer is returned when there is no differ set on the GTA
	ErrNoDiffer = errors.New("there is no differ set")
	// ErrNoPackager is returned when there is no packager set on the GTA
	ErrNoPackager = errors.New("there is no packager set")
)

// GTA
type GTA struct {
	differ   Differ
	packager Packager
}

// New returns a new GTA with various options passed in
func New(opts ...Option) (*GTA, error) {
	gta := &GTA{}

	for _, opt := range opts {
		err := opt(gta)
		if err != nil {
			return nil, err
		}
	}

	return gta, nil
}

// DirtyPackages uses the differ and packager to build a list of dirty packages
// where dirty is defined as "changed"
func (g *GTA) DirtyPackages() ([]*build.Package, error) {
	if g.differ == nil {
		return nil, ErrNoDiffer
	}
	if g.packager == nil {
		return nil, ErrNoPackager
	}

	// get our diff'd directories
	dirs, err := g.differ.Diff()
	if err != nil {
		return nil, err
	}

	// we build our set of initial dirty packages from the git diff
	changed := make(map[string]bool)
	for dir := range dirs {
		pkg, err := g.packager.PackageFromDir(dir)
		if err != nil {
			return nil, err
		}

		// we create a simple set of changed pkgs by import path
		changed[pkg.ImportPath] = false
	}

	// we build the dependent graph
	graph, err := g.packager.DependentGraph()
	if err != nil {
		return nil, err
	}

	// we copy the map since iterating over a map
	// while its being mutated is undefined behavior
	marked := make(map[string]bool)
	for k, v := range changed {
		marked[k] = v
	}

	for change := range changed {
		// we traverse the graph and build our list of mark all dependents
		graph.Traverse(change, &marked)
	}

	// build our packages
	var packages []*build.Package
	for path := range marked {
		pkg, err := g.packager.PackageFromImport(path)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}
