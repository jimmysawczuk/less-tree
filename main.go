package main

import (
	// 	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var pathToLESS string
var workingDirectory string
var maxProcesses int

func main() {
	var err error
	workingDirectory, err = os.Getwd()
	if err != nil {
		log.Fatalf("Can't find the working directory. Massive fail.")
		os.Exit(1)
		return
	}

	flag.StringVar(&pathToLESS, "path", "lessc", "Path to the lessc executable")
	flag.IntVar(&maxProcesses, "max-processes", 1, "Max number of processes to have open")

	flag.Parse()

	runtime.GOMAXPROCS(maxProcesses)

	args := flag.Args()
	chans := make([]chan int, len(args))
	for i, v := range args {
		chans[i] = make(chan int)
		go compileFromRoot(v, chans[i])
	}

	for i, _ := range chans {
		<-chans[i]
	}
}

func compileFromRoot(dir string, ch chan int) {
	var fq_dir *os.File
	var err error

	if dir[0:1] != "/" {
		fq_dir, err = os.Open(workingDirectory + "/" + dir)
	} else {
		fq_dir, err = os.Open(dir)
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	less_dir, err := os.Open(fq_dir.Name() + "/less")
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("No /less directory exists at %s", fq_dir.Name())
			return
		} else {
			log.Println(err)
			return
		}
	}

	css_dir, err := os.Open(fq_dir.Name() + "/css")
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(fq_dir.Name()+"/css", 0755)
			if err != nil {
				fmt.Println("Can't create css directory")
				return
			} else {
				css_dir, _ = os.Open(fq_dir.Name() + "/css")
			}
		} else {
			log.Println(err)
			return
		}
	}

	dir_chan := make(chan int)
	go compileDirectory("", less_dir, css_dir, dir_chan)
	<-dir_chan
	ch <- 1
}

func compileDirectory(prefix string, less_dir, css_dir *os.File, ch chan int) {
	files, err := less_dir.Readdir(-1)
	if err != nil {
		log.Panicf("Can't parse %s", less_dir.Name())
	}

	chs := make([]chan int, 0)

	for _, v := range files {
		if v.IsDir() {

			less_deeper, _ := os.Open(less_dir.Name() + "/" + v.Name())
			css_deeper, err := os.Open(css_dir.Name() + "/" + v.Name())
			if err != nil {
				if os.IsNotExist(err) {
					err = os.Mkdir(css_dir.Name()+"/"+v.Name(), 0755)
					if err != nil {
						fmt.Println("Can't create css directory")
						return
					} else {
						css_deeper, _ = os.Open(css_dir.Name() + "/" + v.Name())
					}
				}
			}

			dir_ch := make(chan int)
			chs = append(chs, dir_ch)
			go compileDirectory(v.Name()+"/", less_deeper, css_deeper, dir_ch)

		} else if v.Name()[0:1] != "_" {
			file_ch := make(chan int)
			chs = append(chs, file_ch)
			go compileFile(less_dir, css_dir, v, file_ch, prefix+v.Name())
		}
	}

	for i, _ := range chs {
		<-chs[i]
	}

	ch <- 1
}

func compileFile(less_dir, css_dir *os.File, less_file os.FileInfo, ch chan int, log_text string) {

	fmt.Println(log_text)

	(func() {
		cmd := exec.Command(pathToLESS, less_dir.Name()+"/"+less_file.Name())

		result, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
		} else {
			dest_file, err := os.OpenFile(css_dir.Name()+"/"+strings.Replace(less_file.Name(), ".less", ".css", 1), os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
			if err != nil {
				log.Println(err)
			} else {
				dest_file.Write(result)
			}
		}
	})()

	(func() {
		cmd := exec.Command(pathToLESS, "-x", less_dir.Name()+"/"+less_file.Name())

		result, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
		} else {
			dest_file, err := os.OpenFile(css_dir.Name()+"/"+strings.Replace(less_file.Name(), ".less", ".min.css", 1), os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
			if err != nil {
				log.Println(err)
			} else {
				dest_file.Write(result)
			}
		}
	})()

	ch <- 1
}
