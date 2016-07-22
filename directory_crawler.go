package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type addFunc func(crawler *directoryCrawler, less_dir, css_dir *os.File, less_file os.FileInfo)

type directoryCrawler struct {
	root     *os.File
	rootCSS  *os.File
	rootLESS *os.File

	addFunc addFunc
}

func newDirectoryCrawler(path string, addFunc addFunc) (*directoryCrawler, error) {
	var root *os.File
	var err error
	if !filepath.IsAbs(path) {
		root, err = os.Open(filepath.Join(workingDirectory, path))
	} else {
		root, err = os.Open(filepath.Clean(path))
	}

	if err != nil {
		return nil, fmt.Errorf("error parsing directory %s: %s", path, err)
	}

	c := &directoryCrawler{
		root:    root,
		addFunc: addFunc,
	}

	lessDir, err := os.Open(filepath.Join(c.root.Name(), "less"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory %s doesn't exist", filepath.Join(c.root.Name(), "less"))
		}
		return nil, fmt.Errorf("can't open %s: %s", filepath.Join(c.root.Name(), "less"), err)

	}
	c.rootLESS = lessDir

	cssDir, err := os.Open(filepath.Join(c.root.Name(), "css"))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("can't open %s: %s", filepath.Join(c.root.Name(), "css"), err)
		}

		err = os.Mkdir(filepath.Join(c.root.Name(), "css"), 0755)
		if err != nil {
			return nil, fmt.Errorf("can't create %s: %s", filepath.Join(c.root.Name(), "css"), err)
		}

		cssDir, _ = os.Open(filepath.Join(c.root.Name(), "css"))
	}
	c.rootCSS = cssDir

	return c, nil
}

func (c *directoryCrawler) Parse() error {
	c.parseDirectory("", c.rootLESS, c.rootCSS)
	return nil
}

func (c *directoryCrawler) parseDirectory(prefix string, lessDir, cssDir *os.File) {
	files, err := lessDir.Readdir(-1)
	if err != nil {
		fmt.Printf("Can't scan %s for files", lessDir.Name())
		return
	}

	for _, v := range files {
		if v.IsDir() {
			if strings.HasPrefix(v.Name(), "_") {
				// We're dealing with an underscore-prefixed directory.
				if isVerbose {
					dir, _ := filepath.Rel(c.rootLESS.Name(), filepath.Join(lessDir.Name(), v.Name()))
					fmt.Printf("skip: %s\n", dir+"/*")
				}

				continue
			}

			lessDeeper, _ := os.Open(lessDir.Name() + string(os.PathSeparator) + v.Name())
			cssDeeper, err := os.Open(cssDir.Name() + string(os.PathSeparator) + v.Name())
			if err != nil {
				if os.IsNotExist(err) {
					err = os.Mkdir(cssDir.Name()+string(os.PathSeparator)+v.Name(), 0755)
					if err != nil {
						fmt.Println("Can't create css directory")
						return
					}
					cssDeeper, _ = os.Open(cssDir.Name() + string(os.PathSeparator) + v.Name())
				}
			}

			c.parseDirectory(v.Name()+string(os.PathSeparator), lessDeeper, cssDeeper)
		}

		if !v.IsDir() && lessFilename.MatchString(v.Name()) {
			if strings.HasPrefix(v.Name(), "_") {

				// We're dealing with an underscore-prefixed file (an include).
				if isVerbose {
					filename, _ := filepath.Rel(c.rootLESS.Name(), filepath.Join(lessDir.Name(), v.Name()))
					fmt.Printf("skip: %s\n", filename)
				}

				continue
			}

			c.addFunc(c, lessDir, cssDir, v)
		}
	}
}
