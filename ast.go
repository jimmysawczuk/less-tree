package main

import (
	"fmt"
)

const (
	importToken = "@import"
	lParenToken = "("
	rParenToken = ")"
	lCurlyToken = "{"
	rCurlyToken = "}"
)

func sliceUntil(tokens []string, search string, start int, offset int) ([]string, error) {
	for i := start + offset; i < len(tokens); i++ {
		if tokens[i] == search {
			return tokens[start : i+1], nil
		}
	}

	return []string{}, fmt.Errorf("search token not found: %s", search)
}

func sliceUntilMatching(tokens []string, opener, closer string, start int, offset int) ([]string, error) {
	open := 0
	for i := start + offset; i < len(tokens); i++ {
		if tokens[i] == opener {
			open++
		} else if tokens[i] == closer {
			open--
		}

		if open == 0 {
			return tokens[start : i+1], nil
		}
	}

	return []string{}, fmt.Errorf("matching search token not found: %s", closer)
}
