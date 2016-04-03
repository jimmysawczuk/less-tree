package less

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

type LESSFile struct {
	dir    *os.File
	fi     os.FileInfo
	ast    *AST
	tokens []string
}

func New(less_dir *os.File, less_file os.FileInfo) (*LESSFile, error) {

	full := path.Join(less_dir.Name(), less_file.Name())
	less_content, err := ioutil.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("can't read file %s: %s\n", full, err)
	}

	l := new(LESSFile)
	l.dir = less_dir
	l.fi = less_file
	l.tokens = tokenize(less_content)
	l.ast = buildAST(l.tokens)
	l.ast.Print()

	return l, nil
}
