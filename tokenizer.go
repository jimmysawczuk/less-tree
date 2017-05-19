package main

import (
	"bytes"
	"unicode"
)

func appendNonEmptyToken(working string, tokens []string) (string, []string) {
	if working != "" {
		tokens = append(tokens, working)
		working = ""
	}
	return working, tokens
}

func tokenize(in []byte) []string {
	content := bytes.Runes(in)
	working := ""
	i := 0
	tokens := []string{}

	for i < len(content) {
		chr := content[i]

		switch {
		case unicode.IsSpace(chr):
			working, tokens = appendNonEmptyToken(working, tokens)
			i++

		case chr == '/':
			if i+1 < len(content) && content[i+1] == '/' {
				working, tokens = appendNonEmptyToken(working, tokens)
				comment := readUntilNewline(content, i+2)
				i += len(comment) + 2
			} else if i+1 < len(content) && content[i+1] == '*' {
				working, tokens = appendNonEmptyToken(working, tokens)
				comment := readUntilMatch(content, []rune("*/"), i, 0)
				i += len(comment)
			} else {
				working, tokens = appendNonEmptyToken(working, tokens)
				tokens = append(tokens, "/")
				i++
			}

		default:
			switch chr {
			case '(', ')', ';', ':', ',', '=', '{', '}', '[', ']':
				working, tokens = appendNonEmptyToken(working, tokens)
				_, tokens = appendNonEmptyToken(string(chr), tokens)
				i++

			case '"':
				working, tokens = appendNonEmptyToken(working, tokens)
				match := readUntilMatch(content, []rune(`"`), i, 1)
				_, tokens = appendNonEmptyToken(string(match), tokens)
				i += len(match)

			case '\'':
				working, tokens = appendNonEmptyToken(working, tokens)
				match := readUntilMatch(content, []rune(`'`), i, 1)
				_, tokens = appendNonEmptyToken(string(match), tokens)
				i += len(match)

			default:
				working += string(chr)
				i++
			}
		}
	}

	working, tokens = appendNonEmptyToken(working, tokens)

	return tokens
}

func readUntilNewline(haystack []rune, start int) []rune {
	if start < 0 {
		return []rune("")
	}

	for i := start; i < len(haystack); i++ {
		switch haystack[i] {
		case '\r', '\n':
			return haystack[start : i+1]
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
