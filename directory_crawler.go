package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type addFunc func(crawler *DirectoryCrawler, less_dir, css_dir *os.File, less_file os.FileInfo)

type DirectoryCrawler struct {
	root     *os.File
	rootCSS  *os.File
	rootLESS *os.File

	addFunc addFunc
}

func NewDirectoryCrawler(path string, addFunc addFunc) (*DirectoryCrawler, error) {
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

	c := &DirectoryCrawler{
		root:    root,
		addFunc: addFunc,
	}

	less_dir, err := os.Open(filepath.Join(c.root.Name(), "less"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory %s doesn't exist", filepath.Join(c.root.Name(), "less"))
		} else {
			return nil, fmt.Errorf("can't open %s: %s", filepath.Join(c.root.Name(), "less"), err)
		}
	}
	c.rootLESS = less_dir

	css_dir, err := os.Open(filepath.Join(c.root.Name(), "css"))
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(filepath.Join(c.root.Name(), "css"), 0755)
			if err != nil {
				return nil, fmt.Errorf("can't create %s: %s", filepath.Join(c.root.Name(), "css"), err)
			} else {
				css_dir, _ = os.Open(filepath.Join(c.root.Name(), "css"))
			}
		} else {
			return nil, fmt.Errorf("can't open %s: %s", filepath.Join(c.root.Name(), "css"), err)
		}
	}
	c.rootCSS = css_dir

	return c, nil
}

func (c *DirectoryCrawler) Parse() error {
	c.parseDirectory("", c.rootLESS, c.rootCSS)
	return nil
}

func (c *DirectoryCrawler) parseDirectory(prefix string, less_dir, css_dir *os.File) {
	files, err := less_dir.Readdir(-1)
	if err != nil {
		fmt.Printf("Can't scan %s for files", less_dir.Name())
		return
	}

	for _, v := range files {
		if v.IsDir() {
			if strings.HasPrefix(v.Name(), "_") {
				// We're dealing with an underscore-prefixed directory.
				if isVerbose {
					dir, _ := filepath.Rel(c.rootLESS.Name(), filepath.Join(less_dir.Name(), v.Name()))
					fmt.Printf("skip: %s\n", dir+"/*")
				}

				continue
			}

			less_deeper, _ := os.Open(less_dir.Name() + string(os.PathSeparator) + v.Name())
			css_deeper, err := os.Open(css_dir.Name() + string(os.PathSeparator) + v.Name())
			if err != nil {
				if os.IsNotExist(err) {
					err = os.Mkdir(css_dir.Name()+string(os.PathSeparator)+v.Name(), 0755)
					if err != nil {
						fmt.Println("Can't create css directory")
						return
					} else {
						css_deeper, _ = os.Open(css_dir.Name() + string(os.PathSeparator) + v.Name())
					}
				}
			}

			c.parseDirectory(v.Name()+string(os.PathSeparator), less_deeper, css_deeper)
		}

		if !v.IsDir() && lessFilename.MatchString(v.Name()) {
			if strings.HasPrefix(v.Name(), "_") {

				// We're dealing with an underscore-prefixed file (an include).
				if isVerbose {
					filename, _ := filepath.Rel(c.rootLESS.Name(), filepath.Join(less_dir.Name(), v.Name()))
					fmt.Printf("skip: %s\n", filename)
				}

				continue
			}

			c.addFunc(c, less_dir, css_dir, v)
		}
	}
}
