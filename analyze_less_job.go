package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
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

	j.init()

	return j
}

func (j *AnalyzeJob) init() {
	j.less_in = path.Join(j.LessDir.Name(), j.LessFile.Name())
}

func (j *AnalyzeJob) Run() {
	less_content, err := ioutil.ReadFile(j.less_in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't read file %s: %s\n", j.Name, err)
	}

	tokens := tokenize(less_content)

	_ = tokens

	if j.Name == "test.less" {
		fmt.Println(j.Name)
		for _, v := range tokens {
			fmt.Printf("%v\n", v)
		}
		fmt.Println("------")
	}
}
