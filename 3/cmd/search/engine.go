package main

import (
	"errors"
	"fmt"
	"regexp"
)

type void = struct{}

type expressionNode struct {
	Value string
	Left  *expressionNode
	Right *expressionNode
}

type Engine struct {
	index   InvertedIndex
	allDocs map[string]void
}

func New(index InvertedIndex) *Engine {
	allDocs := make(map[string]void)
	for _, docs := range index {
		for _, doc := range docs {
			allDocs[doc] = void{}
		}
	}

	return &Engine{index: index, allDocs: allDocs}
}

func (e *Engine) Search(query string) ([]string, error) {
	tokens, err := parseQuery(query)
	if err != nil {
		return nil, err
	}

	rpn := rPN(tokens)

	opTree, err := buildOperationTree(rpn)
	if err != nil {
		return nil, err
	}

	resultSet, err := evaluateTree(opTree, e.index, e.allDocs)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(resultSet))
	for doc := range resultSet {
		result = append(result, doc)
	}

	return result, nil
}

func parseQuery(query string) (tokens []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("не удалось распарсить запрос")
		}
	}()

	re := regexp.MustCompile(`!|[\p{L}]+|[&|]`)
	tokens = re.FindAllString(query, -1)
	return
}

func not(index InvertedIndex, allDocs map[string]void, term string) map[string]void {
	excludeDocs := make(map[string]void)
	for _, doc := range index[term] {
		excludeDocs[doc] = void{}
	}

	result := make(map[string]void)
	for doc := range allDocs {
		if _, found := excludeDocs[doc]; !found {
			result[doc] = void{}
		}
	}
	return result
}

func and(set1, set2 map[string]void) map[string]void {
	result := make(map[string]void)
	for doc := range set1 {
		if _, found := set2[doc]; found {
			result[doc] = void{}
		}
	}
	return result
}

func or(set1, set2 map[string]void) map[string]void {
	result := make(map[string]void)
	for doc := range set1 {
		result[doc] = void{}
	}
	for doc := range set2 {
		result[doc] = void{}
	}
	return result
}

func listToSet(docs []string) map[string]void {
	result := make(map[string]void)
	for _, doc := range docs {
		result[doc] = void{}
	}
	return result
}

var precedence = map[string]int{
	"!": 3,
	"&": 2,
	"|": 1,
}

func rPN(tokens []string) []string {
	var output []string
	var operators []string

	for _, token := range tokens {
		switch token {
		case "&", "|":
			for len(operators) > 0 {
				top := operators[len(operators)-1]
				if precedence[top] >= precedence[token] {
					output = append(output, top)
					operators = operators[:len(operators)-1]
				} else {
					break
				}
			}
			operators = append(operators, token)

		case "!":
			operators = append(operators, token)

		default:
			output = append(output, token)
		}
	}

	for len(operators) > 0 {
		output = append(output, operators[len(operators)-1])
		operators = operators[:len(operators)-1]
	}

	return output
}

func buildOperationTree(rpn []string) (*expressionNode, error) {
	var stack []*expressionNode

	for _, token := range rpn {
		node := &expressionNode{Value: token}

		if token == "&" || token == "|" {
			if len(stack) < 2 {
				return nil, errors.New("не найдены выраежния около терминов")
			}
			node.Right = stack[len(stack)-1]
			node.Left = stack[len(stack)-2]
			stack = stack[:len(stack)-2]
		} else if token == "!" {
			if len(stack) < 1 {
				return nil, errors.New("не найден термин, который отрицают")
			}
			node.Left = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		}

		stack = append(stack, node)
	}

	if len(stack) != 1 {
		return nil, errors.New(("не верный формат запроса"))
	}

	return stack[0], nil
}

func evaluateTree(node *expressionNode, index InvertedIndex, allDocs map[string]void) (map[string]void, error) {
	if node == nil {
		return nil, nil
	}

	if node.Left == nil && node.Right == nil {
		return listToSet(index[node.Value]), nil
	}

	leftSet, err := evaluateTree(node.Left, index, allDocs)
	if err != nil {
		return nil, err
	}
	rightSet, err := evaluateTree(node.Right, index, allDocs)
	if err != nil {
		return nil, err
	}

	switch node.Value {
	case "!":
		return not(index, allDocs, node.Left.Value), nil
	case "&":
		return and(leftSet, rightSet), nil
	case "|":
		return or(leftSet, rightSet), nil
	default:
		return nil, fmt.Errorf("не поддерживаемый операнд %s", node.Value)
	}
}
