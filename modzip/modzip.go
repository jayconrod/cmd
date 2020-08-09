package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	modzip "golang.org/x/mod/zip"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	outPath, err := run(wd, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", outPath)
}

func run(wd string, args []string) (outPath string, err error) {
	var version string
	fs := flag.NewFlagSet("modzip", flag.ContinueOnError)
	fs.StringVar(&version, "version", "", "semantic version to create a zip file for")
	fs.StringVar(&outPath, "o", "", "output file path (defaults to <version>.zip)")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if v := semver.Canonical(version); v != version {
		return "", fmt.Errorf("version %q is not a canonical version", version)
	}

	goModPath, err := findGoMod(wd)
	if err != nil {
		return "", err
	}
	modRoot := filepath.Dir(goModPath)
	goModData, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	modPath := modfile.ModulePath(goModData)
	if modPath == "" {
		return "", fmt.Errorf("%s: could not read module path", goModPath)
	}

	if outPath == "" {
		outPath = filepath.Join(filepath.Dir(modRoot), version+".zip")
	}
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	mv := module.Version{Path: modPath, Version: version}
	if err := modzip.CreateFromDir(f, mv, modRoot); err != nil {
		return "", err
	}

	return outPath, nil
}

func findGoMod(dir string) (string, error) {
	d := dir
	for {
		goModPath := filepath.Join(d, "go.mod")
		fi, err := os.Lstat(goModPath)
		if err == nil {
			if !fi.Mode().IsRegular() {
				return "", fmt.Errorf("%s: go.mod must be a regular file", goModPath)
			}
			return goModPath, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("%s: could not find go.mod in any parent directory", dir)
		}
		d = parent
	}
}
