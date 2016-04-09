package main

import (
	"github.com/jimmysawczuk/less-tree/less"

	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type LessTreeCache struct {
	Version   string                    `json:"version"`
	Generated time.Time                 `json:"generated"`
	Files     map[string]*less.LESSFile `json:"files"`

	rootDir *os.File
}

func NewLessTreeCache(dir *os.File) *LessTreeCache {
	cm := &LessTreeCache{
		Version:   version,
		Generated: time.Now(),
		Files:     make(map[string]*less.LESSFile, 0),
		rootDir:   dir,
	}

	return cm
}

func (c *LessTreeCache) Load() error {
	contents, err := ioutil.ReadFile(filepath.Join(c.rootDir.Name(), ".less-tree-cache"))
	if err != nil {
		return err
	}

	err = json.Unmarshal(contents, c)

	return err
}

func (c *LessTreeCache) Save() error {
	c.Version = version
	c.Generated = time.Now()

	contents, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(c.rootDir.Name(), ".less-tree-cache"), contents, 0644)

	return err
}

func (c *LessTreeCache) Test(current *less.LESSFile) bool {

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

func (c *LessTreeCache) testImports(current, cached *less.LESSFile) bool {
	var cur_file, cached_file *less.LESSFile

	for _, a := range current.Imports {
		match := false
		for _, b := range cached.Imports {
			if a.File.Name == b.File.Name {
				match = a.File.Hash == b.File.Hash
				cur_file = a.File
				cached_file = b.File
				break
			}
		}

		if match {
			res := c.testImports(cur_file, cached_file)
			if !res {
				return false
			}
		} else {
			return false
		}
	}

	return true
}
