package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type CacheMap struct {
	Version   string         `json:"version"`
	Generated time.Time      `json:"generated"`
	Files     []CacheMapFile `json:"files"`

	dir *os.File
}

type CacheMapFile struct {
	Name    string            `json:"name"`
	Hash    string            `json:"hash"`
	Imports map[string]string `json:"imports"`
}

func NewCacheMap(dir *os.File) *CacheMap {
	cm := &CacheMap{
		Version:   version,
		Generated: time.Now(),
		Files:     make([]CacheMapFile, 0),
		dir:       dir,
	}

	return cm
}

func (c *CacheMap) Load() error {
	contents, err := ioutil.ReadFile(filepath.Join(c.dir.Name(), ".less-tree"))
	if err != nil {
		return err
	}

	err = json.Unmarshal(contents, c)

	return err
}

func (c *CacheMap) TestMain(name string, hash string) bool {
	for _, v := range c.Files {
		if v.Name == name {
			return v.Hash != hash
		}
	}

	return true
}
