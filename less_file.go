package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
)

type lessFile struct {
	Name   string      `json:"name"`
	Dir    *os.File    `json:"-"`
	CSSDir *os.File    `json:"-"`
	File   os.FileInfo `json:"-"`
	Path   string      `json:"-"`

	Imports []*lessImport `json:"imports,omitempty"`
	Hash    string        `json:"hash"`

	tokens []string
}

type lessImport struct {
	Options []string  `json:"options,omitempty"`
	File    *lessFile `json:"file"`
}

func newLessFile(name string, lessDir, cssDir *os.File, inputLessFile os.FileInfo) (*lessFile, error) {

	l := new(lessFile)
	l.Name = name
	l.Dir = lessDir
	l.File = inputLessFile
	l.CSSDir = cssDir
	l.Path = filepath.Join(l.Dir.Name(), l.File.Name())

	lessContent, err := ioutil.ReadFile(l.Path)
	if err != nil {
		return nil, fmt.Errorf("can't read file %s: %s\n", l.Path, err)
	}

	hash := sha1.Sum(lessContent)
	str := hex.EncodeToString(hash[:])
	l.Hash = str

	l.tokens = tokenize(lessContent)
	l.Imports = make([]*lessImport, 0)

	err = l.findImports()
	if err != nil {
		return nil, fmt.Errorf("import parse error: %s", err)
	}

	l.Dir.Close()

	return l, nil
}

func (l *lessFile) String() string {
	return l.prefixString(0) + "\n"
}

func (l *lessFile) prefixString(width int) string {
	if len(l.Imports) == 0 {
		return fmt.Sprintf("File %s imports no files (%s)", l.Name, l.Hash)
	}

	str := fmt.Sprintf("File %s imports %d files: (%s)", l.Name, len(l.Imports), l.Hash)
	for _, v := range l.Imports {
		str += "\n"
		for i := 0; i < width; i++ {
			str += fmt.Sprintf(" ")
		}
		str += fmt.Sprintf(" - %s", v.File.prefixString(width+2))
	}
	return str
}

func (l *lessFile) findImports() error {
	i := 0
	for i < len(l.tokens) {
		token := l.tokens[i]

		switch token {
		case importToken:
			slice, err := sliceUntil(l.tokens, ";", i, 0)
			if err != nil {
				return fmt.Errorf("error parsing import: missing semicolon")
			}

			imp, err := l.NewLESSImport(slice)
			if err != nil {
				return fmt.Errorf("error parsing import: %s", err)
			}

			if imp != nil {
				l.Imports = append(l.Imports, imp)
			}

			i += len(slice)

		default:
			i++
		}
	}

	return nil
}

func (l *lessFile) NewLESSImport(in []string) (imp *lessImport, err error) {
	i := 1
	if len(in) <= 1 {
		return nil, fmt.Errorf("not enough parameters")
	}

	imp = new(lessImport)
	imp.Options = []string{}

	if in[1] == lParenToken {
		opts := []string{}
		opts, err = sliceUntilMatching(in, lParenToken, rParenToken, 1, 0)
		if err != nil {
			return nil, fmt.Errorf("missing a )")
		}

		imp.Options = opts[1 : len(opts)-1]

		i += len(opts)
	}

	// remove quotes
	path := in[i][1 : len(in[i])-1]

	if u, err := url.Parse(path); err == nil {
		if u.IsAbs() {
			// this is an absolute url
			return nil, nil
		}
	}

	dirName, fileName := filepath.Split(path)

	var dir, file *os.File
	if dirName == "" {
		dir = l.Dir
	} else if !filepath.IsAbs(dirName) {
		dir, err = os.Open(filepath.Join(l.Dir.Name(), dirName))
	} else {
		dir, err = os.Open(filepath.Join(dirName))
	}

	if err != nil {
		return nil, fmt.Errorf("import path %s is not valid: %s", path, err)
	}

	ext := filepath.Ext(fileName)
	if ext == "" {
		fileName = fileName + ".less"
	}

	file, err = os.Open(filepath.Join(dir.Name(), fileName))
	if err != nil {
		return nil, fmt.Errorf("import path %s is not valid: %s", path, err)
	}

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("can't stat path %s: %s", path, err)
	}

	imp.File, err = newLessFile(path, dir, nil, fi)

	dir.Close()
	file.Close()

	return imp, err
}
