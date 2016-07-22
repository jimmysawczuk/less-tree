package main

import (
	"github.com/jimmysawczuk/less-tree/less"

	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type lessTreeCache struct {
	Version   string                    `json:"version"`
	Generated time.Time                 `json:"generated"`
	Files     map[string]*less.LESSFile `json:"files"`

	rootDir *os.File
}

func newLessTreeCache(dir *os.File) *lessTreeCache {
	cm := &lessTreeCache{
		Version:   version,
		Generated: time.Now(),
		Files:     make(map[string]*less.LESSFile, 0),
		rootDir:   dir,
	}

	return cm
}

func (c *lessTreeCache) Load() error {
	contents, err := ioutil.ReadFile(filepath.Join(c.rootDir.Name(), ".less-tree-cache"))
	if err != nil {
		return err
	}

	err = json.Unmarshal(contents, c)

	return err
}

func (c *lessTreeCache) Save() error {
	c.Version = version
	c.Generated = time.Now()

	contents, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(c.rootDir.Name(), ".less-tree-cache"), contents, 0644)

	return err
}

func (c *lessTreeCache) Test(current *less.LESSFile) bool {

	cached, exists := c.Files[current.Name]
	if !exists {
		c.Files[current.Name] = current
		return false
	}

	if cached.Hash != current.Hash {
		c.Files[current.Name] = current
		return false
	}

	res := c.testImports(current, cached)

	c.Files[current.Name] = current
	return res
}

func (c *lessTreeCache) testImports(current, cached *less.LESSFile) bool {
	var curFile, cachedFile *less.LESSFile

	for _, a := range current.Imports {
		match := false
		for _, b := range cached.Imports {
			if a.File.Name == b.File.Name {
				match = a.File.Hash == b.File.Hash
				curFile = a.File
				cachedFile = b.File
				break
			}
		}

		if match {
			res := c.testImports(curFile, cachedFile)
			if !res {
				return false
			}
		} else {
			return false
		}
	}

	return true
}
