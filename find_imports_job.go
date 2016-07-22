package main

import (
	"fmt"
	"os"
)

type findImportsJob struct {
	File *lessFile
	Name string

	inDir  *os.File
	outDir *os.File
	inFile os.FileInfo

	outCh chan *lessFile
	errCh chan error
}

func newFindImportsJob(name string, lessDir, cssDir *os.File, inputLessFile os.FileInfo, outCh chan *lessFile, errCh chan error) *findImportsJob {
	j := &findImportsJob{
		Name:   name,
		inDir:  lessDir,
		outDir: cssDir,
		inFile: inputLessFile,
		outCh:  outCh,
		errCh:  errCh,
	}

	return j
}

func (j *findImportsJob) Run() {
	if isVerbose {
		fmt.Println("analyze:", j.Name)
	}

	l, err := newLessFile(j.Name, j.inDir, j.outDir, j.inFile)
	if err != nil {
		j.errCh <- err
		return
	}

	j.outCh <- l
}
