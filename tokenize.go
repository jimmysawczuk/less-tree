package main

import (
	"bytes"
	"fmt"
	// "strings"
	"unicode"
)

var _ = fmt.Sprintf

func tokenize(in []byte) []string {

	content := bytes.Runes(in)

	working := ""
	i := 0
	tokens := []string{}

	for i < len(content) {
		chr := content[i]

		switch {
		case unicode.IsSpace(chr):
			if working != "" {
				tokens = append(tokens, working)
				working = ""
			}
			i++

		case chr == '/':
			if i+1 < len(content) && content[i+1] == '/' {
				if working != "" {
					tokens = append(tokens, working)
					working = ""
				}

				comment := readUntilNewline(content, i+2)
				i += len(comment)
			} else if i+1 < len(content) && content[i+1] == '*' {
				if working != "" {
					tokens = append(tokens, working)
					working = ""
				}
				comment := readUntilMatch(content, []rune("*/"), i, 0)
				i += len(comment)
			} else {
				if working != "" {
					tokens = append(tokens, working)
					working = ""
				}
				tokens = append(tokens, "/")
				i++
			}

		default:
			switch chr {
			case '(', ')', ';', ':', ',', '=', '[', ']', '{', '}':
				if working != "" {
					tokens = append(tokens, working)
					working = ""
				}
				tokens = append(tokens, string(chr))
				i++

			case '"':
				if working != "" {
					tokens = append(tokens, working)
					working = ""
				}

				match := readUntilMatch(content, []rune(`"`), i, 1)

				tokens = append(tokens, string(match))
				i += len(match)

			case '\'':
				if working != "" {
					tokens = append(tokens, working)
					working = ""
				}

				match := readUntilMatch(content, []rune(`'`), i, 1)

				tokens = append(tokens, string(match))
				i += len(match)

			default:
				working += string(chr)
				i++
			}
		}
	}

	if working != "" {
		tokens = append(tokens, working)
		working = ""
	}

	return tokens
}

func readUntilNewline(haystack []rune, start int) []rune {
	if start < 0 {
		return []rune("")
	}

	for i := start; i < len(haystack); i++ {
		switch rune(haystack[i]) {
		case '\u000A', '\u000D':
			return haystack[start : i+2]
		}
	}

	return haystack[start:]
}

func readUntilMatch(haystack []rune, match []rune, start int, offset int) []rune {
	if start < 0 {
		return []rune("")
	}

	var i, j int
	for i = start + offset; i < len(haystack)-len(match); i++ {
		for j = 0; j < len(match); j++ {
			if haystack[i+j] != match[j] {
				break
			}
		}

		if j == len(match) {
			ret := haystack[start : i+len(match)]
			return ret
		}
	}

	return haystack[start:]
}
