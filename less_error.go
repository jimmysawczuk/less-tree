package main

import (
	"fmt"
	"strings"
)

type lessError struct {
	indent  int
	Message string
}

func (e lessError) Error() string {
	indent := ""
	for i := 0; i < e.indent; i++ {
		indent = indent + " "
	}

	str := strings.Replace(fmt.Sprintf("\n%s", e.Message), "\n", "\n"+indent, -1)
	return str + "\n"
}
