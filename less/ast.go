package less

import (
	"fmt"
)

type AST struct {
	head Node
	tail Node
}

func (a *AST) Add(n Node) {
	if a.head == nil {
		a.head = n
		a.tail = n
	} else {
		a.tail.SetNext(n)
		a.tail = n
	}
}

func (a *AST) Print() {
	if a.head == nil {
		return
	}

	n := a.head
	for {
		fmt.Printf("%T\n", n)
		n = n.Next()

		if n == nil {
			break
		}
	}
	fmt.Println("----------")
}

const (
	importToken = "@import"
	lParenToken = "("
	rParenToken = ")"
	lCurlyToken = "{"
	rCurlyToken = "}"
)

type Node interface {
	Next() Node
	SetNext(Node)
}

type ASTNode struct {
	next Node
}

func (n *ASTNode) Next() Node {
	return n.next
}

func (n *ASTNode) SetNext(next Node) {
	n.next = next
}

type ImportNode struct {
	ASTNode
	options []string
	path    string
}

type SelectorNode struct {
	ASTNode
	identifier []string
	subTree    Node
}

func (n *ImportNode) build(tokens []string) error {
	i := 1
	for i < len(tokens) {
		if tokens[i] == lParenToken {
			opts, _ := sliceUntil(tokens, rParenToken, i, 1)
			n.options = opts[1 : len(opts)-1]
			i += len(opts)
		}

		n.path = tokens[i]
		break
	}

	return nil
}

func buildAST(tokens []string) *AST {

	tree := new(AST)

	i := 0
	for i < len(tokens) {
		token := tokens[i]

		switch token {
		case importToken:
			slice, _ := sliceUntil(tokens, ";", i, 0)
			n := new(ImportNode)
			n.build(slice)
			tree.Add(n)
			i += len(slice)

		default:
			n, resume := parseSelectorNode(tokens, i)
			_ = n
			i = resume
		}
	}

	return tree
}

func parseSelectorNode(tokens []string, start int) (Node, int) {

	i := start
	identifier := []string{}
	parameters := []string{}
	body := []string{}
	br := false

	for i < len(tokens) && !br {

		token := tokens[i]
		switch token {
		case lParenToken:
			parameters, _ = sliceUntilMatching(tokens, "(", ")", i, 0)
			i += len(parameters)

		case lCurlyToken:
			body, _ = sliceUntilMatching(tokens, "{", "}", i, 0)
			i += len(body)
			br = true

		default:
			identifier = append(identifier, token)
			i++
		}
	}

	return nil, i
}

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
