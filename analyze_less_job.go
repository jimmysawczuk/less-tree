package main

import (
	"less-tree/less"

	"fmt"
	"os"
)

type AnalyzeJob struct {
	Name     string
	LessDir  *os.File
	LessFile os.FileInfo

	less_in string
}

func NewAnalyzeJob(short_name string, less_dir *os.File, less_file os.FileInfo) *AnalyzeJob {
	j := &AnalyzeJob{
		Name:     short_name,
		LessDir:  less_dir,
		LessFile: less_file,
	}

	return j
}

func (j *AnalyzeJob) Run() {
	l, err := less.New(j.LessDir, j.LessFile)
	// fmt.Println("-------")
	_, _ = l, err
	_ = fmt.Sprintf
	// fmt.Println(l, err)
}
