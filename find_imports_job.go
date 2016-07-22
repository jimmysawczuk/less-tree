package main

import (
	"github.com/jimmysawczuk/less-tree/less"

	"fmt"
	"os"
)

type findImportsJob struct {
	File *less.LESSFile
	Name string

	inDir  *os.File
	outDir *os.File
	inFile os.FileInfo

	outCh chan *less.LESSFile
	errCh chan error
}

func newFindImportsJob(name string, lessDir, cssDir *os.File, lessFile os.FileInfo, outCh chan *less.LESSFile, errCh chan error) *findImportsJob {
	j := &findImportsJob{
		Name:   name,
		inDir:  lessDir,
		outDir: cssDir,
		inFile: lessFile,
		outCh:  outCh,
		errCh:  errCh,
	}

	return j
}

func (j *findImportsJob) Run() {
	if isVerbose {
		fmt.Println("analyze:", j.Name)
	}

	l, err := less.New(j.Name, j.inDir, j.inFile, j.outDir)
	if err != nil {
		j.errCh <- err
		return
	}

	j.outCh <- l
}
