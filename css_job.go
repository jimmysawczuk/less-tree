package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

var headerTemplate = template.Must(template.New("header").Parse(`/* generated by less-tree v{{ .Version }} (github.com/jimmysawczuk/less-tree) at {{ .Date }} */` + "\n"))

type LESSError struct {
	indent  int
	Message string
}

type CSSJob struct {
	Name string

	LessDir   *os.File
	CssDir    *os.File
	LessFile  os.FileInfo
	LesscArgs []string

	less_in     string
	css_out     string
	css_min_out string
	less_hash   string

	cmd     *exec.Cmd
	cmd_min *exec.Cmd

	exit_code int
}

func NewCSSJob(short_name string, less_dir, css_dir *os.File, less_file os.FileInfo, lessc_args []string) *CSSJob {

	c := &CSSJob{}
	c.Name = short_name
	c.LessDir = less_dir
	c.CssDir = css_dir
	c.LessFile = less_file
	c.LesscArgs = lessc_args

	c.init()

	return c
}

func (j *CSSJob) init() {
	j.less_in = j.LessDir.Name() + string(os.PathSeparator) + j.LessFile.Name()
	j.css_out = j.getCSSFilename(false)
	j.css_min_out = j.getCSSFilename(true)

	lessc_args := []string{}
	if len(j.LesscArgs) > 0 {
		lessc_args = append(lessc_args, j.LesscArgs...)
	}
	lessc_args = append(lessc_args, j.less_in)

	j.cmd = exec.Command(pathToLessc, lessc_args...)
	if enableCssMin && pathToCssMin != "" {
		j.cmd_min = exec.Command(pathToCssMin, j.css_out)
	}
}

func (j *CSSJob) getCSSFilename(min bool) (css string) {
	less_filename := j.LessFile.Name()
	css_filename := ""

	last := strings.LastIndex(less_filename, ".less")

	if last > 0 {
		if min {
			css_filename = less_filename[0:last] + ".min.css"
		} else {
			css_filename = less_filename[0:last] + ".css"
		}
	} else {
		// this shouldn't really ever happen, since we tested for it before calling this function
		css_filename = less_filename
	}

	return j.CssDir.Name() + string(os.PathSeparator) + css_filename
}

func (j *CSSJob) buildCSSOutput() error {
	result, err := j.cmd.CombinedOutput()
	if err != nil {
		return LESSError{Message: bytes.NewBuffer(result).String(), indent: 3}
	}

	dest_file, err := os.OpenFile(j.css_out, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("File write error: %s\n", err)
	}

	err = j.writeOutput(result, dest_file, true)
	return err
}

func (j *CSSJob) buildMinCSSOutput() error {
	result, err := j.cmd_min.Output()
	if err != nil {
		return LESSError{Message: bytes.NewBuffer(result).String(), indent: 3}
	}

	dest_file, err := os.OpenFile(j.css_min_out, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("File write error: %s\n", err)
	}

	err = j.writeOutput(result, dest_file, true)
	return err
}

func (j *CSSJob) writeOutput(contents []byte, fp *os.File, includeHeader bool) error {
	if includeHeader {
		headerTemplate.Execute(fp, struct {
			Date    string
			Hash    string
			Version string
		}{
			Date:    time.Now().Format(time.RFC3339),
			Hash:    j.less_hash,
			Version: version,
		})
	}

	fp.Write(contents)
	return nil
}

func (j *CSSJob) Run() {

	var err error

	if isVerbose {
		fmt.Printf("build: %s\n", j.Name)
	}

	err = j.buildCSSOutput()
	if err == nil && j.cmd_min != nil {
		err = j.buildMinCSSOutput()
	}

	if err != nil {
		switch err.(type) {
		case LESSError:
			fmt.Printf("err: %s\n%s", j.Name, err)
			j.exit_code = 1
			return
		default:
			fmt.Printf("err: %s: %s", j.Name, err)
			j.exit_code = 1
			return
		}
	}

	if isVerbose {
		fmt.Printf("ok: %s\n", j.Name)
	}
}
