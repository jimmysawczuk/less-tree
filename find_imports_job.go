package main

import (
	"less-tree/less"

	"fmt"
	"os"
)

type FindImportsJob struct {
	File *less.LESSFile
	Name string

	in_dir  *os.File
	out_dir *os.File
	in_file os.FileInfo

	out_ch chan *less.LESSFile
	err_ch chan error
}

func NewFindImportsJob(short_name string, less_dir, css_dir *os.File, less_file os.FileInfo, out_ch chan *less.LESSFile, err_ch chan error) *FindImportsJob {
	j := &FindImportsJob{
		Name:    short_name,
		in_dir:  less_dir,
		out_dir: css_dir,
		in_file: less_file,
		out_ch:  out_ch,
		err_ch:  err_ch,
	}

	return j
}

func (j *FindImportsJob) Run() {
	if isVerbose {
		fmt.Println("analyze:", j.Name)
	}

	l, err := less.New(j.Name, j.in_dir, j.in_file, true)
	if err != nil {
		j.err_ch <- err
	}

	j.out_ch <- l

	for _, v := range l.Imports {
		j.out_ch <- v.File
	}
}
