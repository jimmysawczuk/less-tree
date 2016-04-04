package less

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type LESSFile struct {
	Name           string
	Dir            *os.File
	File           os.FileInfo
	ProducesOutput bool

	Imports []*LESSImport
	Hash    string

	tokens []string
}

type LESSImport struct {
	Options []string
	File    *LESSFile
}

func New(name string, less_dir *os.File, less_file os.FileInfo, produces_output bool) (*LESSFile, error) {

	l := new(LESSFile)
	l.Name = name
	l.Dir = less_dir
	l.File = less_file
	l.ProducesOutput = produces_output

	full := l.fullPath()
	less_content, err := ioutil.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("can't read file %s: %s\n", full, err)
	}

	hash := md5.Sum(less_content)
	str := hex.EncodeToString(hash[:])
	l.Hash = str

	l.tokens = tokenize(less_content)
	l.Imports = make([]*LESSImport, 0)

	err = l.findImports()
	if err != nil {
		return nil, fmt.Errorf("import parse error: %s", err)
	}

	return l, nil
}

func (l *LESSFile) fullPath() string {
	return filepath.Join(l.Dir.Name(), l.File.Name())
}

func (l *LESSFile) String() string {
	full := l.fullPath()

	if len(l.Imports) == 0 {
		return fmt.Sprintf("File %s imports no files\n", full)
	} else {
		str := fmt.Sprintf("File %s imports %d files\n", full, len(l.Imports))
		for _, v := range l.Imports {
			str += fmt.Sprintf(" - %s\n", v.File.Name)
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

			l.Imports = append(l.Imports, imp)
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
	dir, file := filepath.Split(path)

	var dir_p *os.File
	if dir == "" {
		dir_p = p.Dir
	} else if !filepath.IsAbs(dir) {
		dir_p, err = os.Open(filepath.Join(p.Dir.Name(), dir))
	} else {
		dir_p, err = os.Open(filepath.Join(dir))
	}

	file_p, err := os.Open(filepath.Join(dir_p.Name(), file))
	fi, _ := file_p.Stat()

	l.File, _ = New(path, dir_p, fi, false)

	return l, nil
}
