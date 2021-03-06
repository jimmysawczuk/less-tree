package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"
)

var headerTemplate = template.Must(template.New("header").Parse(`/* generated by less-tree v{{ .Version }} (github.com/jimmysawczuk/less-tree) at {{ .Date }} */` + "\n"))

type cssJob struct {
	Name string

	LESSDir  *os.File
	CSSDir   *os.File
	LESSFile os.FileInfo

	lesscArgs []string

	lessIn    string
	cssOut    string
	cssMinOut string
	lessHash  string

	cmd    *exec.Cmd
	cmdMin *exec.Cmd

	exitCode int
}

func newCSSJob(name string, lessDir, cssDir *os.File, file os.FileInfo, lesscArgs []string) *cssJob {

	c := &cssJob{}
	c.Name = name
	c.LESSDir = lessDir
	c.CSSDir = cssDir
	c.LESSFile = file
	c.lesscArgs = lesscArgs

	c.init()

	return c
}

func (j *cssJob) init() {
	j.lessIn = j.LESSDir.Name() + string(os.PathSeparator) + j.LESSFile.Name()
	j.cssOut, j.cssMinOut = j.getCSSFilename(false), j.getCSSFilename(true)

	lesscArgs := []string{}
	if len(j.lesscArgs) > 0 {
		lesscArgs = append(lesscArgs, j.lesscArgs...)
	}
	lesscArgs = append(lesscArgs, j.lessIn)

	j.cmd = exec.Command(pathToLessc, lesscArgs...)
	if enableCSSMin {
		j.cmdMin = exec.Command(pathToCSSMin, j.cssOut)
	}
}

func (j *cssJob) OutputFilesExist() bool {
	var exists = true
	var err error

	_, err = os.Open(j.getCSSFilename(false))
	if err != nil && os.IsNotExist(err) {
		exists = false
	}

	if j.cmdMin != nil {
		_, err = os.Open(j.getCSSFilename(true))
		if err != nil && os.IsNotExist(err) {
			exists = false
		}
	}

	return exists
}

func (j *cssJob) getCSSFilename(min bool) (css string) {
	lessFilename := j.LESSFile.Name()
	cssFilename := ""

	last := strings.LastIndex(lessFilename, ".less")

	if last > 0 {
		if min {
			cssFilename = lessFilename[0:last] + ".min.css"
		} else {
			cssFilename = lessFilename[0:last] + ".css"
		}
	} else {
		// this shouldn't really ever happen, since we tested for it before calling this function
		cssFilename = lessFilename
	}

	return path.Join(j.CSSDir.Name(), cssFilename)
}

func (j *cssJob) buildCSSOutput() error {
	result, err := j.cmd.CombinedOutput()
	if err != nil {
		return lessError{Message: bytes.NewBuffer(result).String(), indent: 3}
	}

	destFile, err := os.OpenFile(j.cssOut, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("File write error: %s\n", err)
	}

	err = j.writeOutput(result, destFile, true)
	return err
}

func (j *cssJob) buildMinCSSOutput() error {
	result, err := j.cmdMin.Output()
	if err != nil {
		return lessError{Message: bytes.NewBuffer(result).String(), indent: 3}
	}

	destFile, err := os.OpenFile(j.cssMinOut, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("File write error: %s\n", err)
	}

	err = j.writeOutput(result, destFile, true)
	return err
}

func (j *cssJob) writeOutput(contents []byte, fp *os.File, includeHeader bool) error {
	if includeHeader {
		headerTemplate.Execute(fp, struct {
			Date    string
			Hash    string
			Version string
		}{
			Date:    time.Now().Format(time.RFC3339),
			Hash:    j.lessHash,
			Version: version,
		})
	}

	fp.Write(contents)
	return nil
}

func (j *cssJob) Run() {

	var err error

	if isVerbose {
		fmt.Printf("build: %s\n", j.Name)
	}

	err = j.buildCSSOutput()
	if err == nil && j.cmdMin != nil {
		err = j.buildMinCSSOutput()
	}

	if err != nil {
		switch err.(type) {
		case lessError:
			fmt.Printf("err: %s\n%s", j.Name, err)
			j.exitCode = 1
			return
		default:
			fmt.Printf("err: %s: %s", j.Name, err)
			j.exitCode = 1
			return
		}
	}

	if isVerbose {
		fmt.Printf("ok: %s\n", j.Name)
	}
}
