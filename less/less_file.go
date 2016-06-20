package less

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
)

type LESSFile struct {
	Name   string      `json:"name"`
	Dir    *os.File    `json:"-"`
	CSSDir *os.File    `json:"-"`
	File   os.FileInfo `json:"-"`
	Path   string      `json:"-"`

	Imports []*LESSImport `json:"imports,omitempty"`
	Hash    string        `json:"hash"`

	tokens []string
}

type LESSImport struct {
	Options []string  `json:"options,omitempty"`
	File    *LESSFile `json:"file"`
}

func New(name string, less_dir *os.File, less_file os.FileInfo, css_dir *os.File) (*LESSFile, error) {

	l := new(LESSFile)
	l.Name = name
	l.Dir = less_dir
	l.File = less_file
	l.CSSDir = css_dir
	l.Path = filepath.Join(l.Dir.Name(), l.File.Name())

	less_content, err := ioutil.ReadFile(l.Path)
	if err != nil {
		return nil, fmt.Errorf("can't read file %s: %s\n", l.Path, err)
	}

	hash := sha1.Sum(less_content)
	str := hex.EncodeToString(hash[:])
	l.Hash = str

	l.tokens = tokenize(less_content)
	l.Imports = make([]*LESSImport, 0)

	err = l.findImports()
	if err != nil {
		return nil, fmt.Errorf("import parse error: %s", err)
	}

	l.Dir.Close()

	return l, nil
}

func (l *LESSFile) String() string {
	return l.prefixString(0) + "\n"
}

func (l *LESSFile) prefixString(width int) string {
	if len(l.Imports) == 0 {
		return fmt.Sprintf("File %s imports no files (%s)", l.Name, l.Hash)
	} else {
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
}

func (l *LESSFile) findImports() error {
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

func (p *LESSFile) NewLESSImport(in []string) (l *LESSImport, err error) {
	i := 1
	if len(in) <= 1 {
		return nil, fmt.Errorf("not enough parameters")
	}

	l = new(LESSImport)
	l.Options = []string{}

	if in[1] == lParenToken {
		opts := []string{}
		opts, err = sliceUntilMatching(in, lParenToken, rParenToken, 1, 0)
		if err != nil {
			return nil, fmt.Errorf("missing a )")
		}

		l.Options = opts[1 : len(opts)-1]

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

	dir, file := filepath.Split(path)

	var dir_p, file_p *os.File
	if dir == "" {
		dir_p = p.Dir
	} else if !filepath.IsAbs(dir) {
		dir_p, err = os.Open(filepath.Join(p.Dir.Name(), dir))
	} else {
		dir_p, err = os.Open(filepath.Join(dir))
	}

	if err != nil {
		return nil, fmt.Errorf("import path %s is not valid: %s", path, err)
	}

	ext := filepath.Ext(file)
	if ext == "" {
		file = file + ".less"
	}

	file_p, err = os.Open(filepath.Join(dir_p.Name(), file))
	if err != nil {
		return nil, fmt.Errorf("import path %s is not valid: %s", path, err)
	}

	fi, err := file_p.Stat()
	if err != nil {
		return nil, fmt.Errorf("can't stat path %s: %s", path, err)
	}

	l.File, err = New(path, dir_p, fi, nil)

	dir_p.Close()
	file_p.Close()

	return l, err
}
