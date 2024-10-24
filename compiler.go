package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Node struct {
	Type    string
	DType   string
	Value   string
	Params  []*Node
	Returns []*Node
	Body    []*Node
	Left    *Node
	Right   *Node
}

type Symbol struct {
	dtype string
	value string
}

func main() {
	var inputFile string = getFlags()
	readLines(inputFile)
}

func getFlags() string {
	inputFile := flag.String("file", "", "")
	flag.Parse()
	if string(*inputFile) == "" {
		fmt.Printf("no file to compile provided")
		os.Exit(3)
	}
	return string(*inputFile)
}

// read lines
func readLines(inputFile string) []string {
	code := []string{}
	// open file
	f, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}

	// close please
	defer f.Close()

	// read the file line by line using scanner
	scanner := bufio.NewScanner(f)

	// for each line, append the line to the code array
	for scanner.Scan() {
		code = append(code, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return code
}

// Parse (big slay)
func parse(code []string) {
	root := Node{}
	body := root.Body

	openBrace := 0

	// iterate through code
	for i := 0; i < len(code); i++ {
		line := i
		// splits line into tokens
		tokens := strings.Fields(code[line])

		if tokens[0] == "func" {
			funcNode, funcBrace := parseFunc(tokens, line)
			openBrace = funcBrace
			body = append(body, funcNode)
		} else if tokens[0] == "int" {
			parseDecl(tokens, line)
		} else if openBrace != 1 {
			body[len(body)].Body = append(body[len(body)].Body, parseGeneric(tokens, line))
		} else {
			body = append(body, parseGeneric(tokens, line))
		}

	}
}

func parseDecl(tokens []string, lineNumber int) *Node {
	newNode := Node{
		Type:  "DECLARATION",
		DType: tokens[0],
		Value: tokens[1],
	}

	if !isIdentifier(tokens[1]) {
		fmt.Println("Expected variable name declaration got " + tokens[1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	if tokens[2] == "=" {
		newNode.Right = parseGeneric(tokens[2:], lineNumber)
	}

	return &newNode
}

// Parse Function Declarations
func parseFunc(tokens []string, lineNumber int) (*Node, int) {
	var newNode Node
	splitStringInPlace(&tokens)
	openParen, openBrace := 0, 0

	newNode.Type = "FUNCTION"

	if !isIdentifier(tokens[1]) {
		fmt.Println("Expected function name declaration got " + tokens[1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	} else {
		newNode.Value = tokens[1]
	}

	if tokens[2] != "(" {
		fmt.Println("Expected \"(\" got " + tokens[2] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	} else {
		openParen++
	}

	closeParenIndex := slices.Index(tokens, ")")

	if closeParenIndex == -1 {
		fmt.Println("Expected \")\" got " + tokens[2] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	} else {
		openParen--
	}

	if closeParenIndex != 3 {
		params := tokens[2:closeParenIndex]

		for i := 1; i < (len(params) + 1/3); i += 3 {
			newNode.Params = append(newNode.Params, parseDecl(params[i:i+1], lineNumber))
		}
	}

	if tokens[closeParenIndex+1] == "int" {
		newNode.DType = "int"
	} else if tokens[closeParenIndex+1] != "{" {
		fmt.Println("Expected \"{\" got " + tokens[closeParenIndex+1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	} else {
		openBrace++
	}

	return &newNode, openBrace

}

// Validates identifiers (variable names, function names, etc.)
func isIdentifier(word string) bool {
	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return validIdentifier.MatchString(word)
}

// SplitStringInPlace splits mixed strings (like "add(int" or "b)") in the array in place
func splitStringInPlace(arr *[]string) {
	// Define a regex pattern to match sequences of letters, digits, or special characters, including arithmetic operators and equal signs
	pattern := regexp.MustCompile(`[a-zA-Z0-9]+|[(){}[\];,+\-*/%=<>!]`)

	// Create a new slice to store the modified array
	var result []string

	for _, str := range *arr {
		// Find all matches based on the regex pattern
		matches := pattern.FindAllString(str, -1)
		// Append the split matches to the result array
		result = append(result, matches...)
	}

	// Replace the original array content with the new split elements
	*arr = result
}

func rootManager(line string) {
	root := Node{}

	root.Type = "ROOT"

	tokens := strings.Fields(line)

	format := ""

	fmt.Println(tokens)

	format = strings.TrimSpace(format)
	root.Body = append(root.Body, parser(tokens))

	if root.Body[0].Type == "ASSIGN" {
		fmt.Println(root.Body[0].Left.Value + " = " + root.Body[0].Right.Value)
	}
}

func HelperPreOrder(node *Node, processFunc func(v string)) {
	if node != nil {
		processFunc(node.value)
		HelperPreOrder(node.left, processFunc)
		HelperPreOrder(node.right, processFunc)
	}
}

func traverseAST(root []*Node) []string {
	var res []string
	for i := 0; i < len(root); i++ {
		processFunc := func(v string) {
			res = append(res, v)
		}
		HelperPreOrder(root[i], processFunc)
	}
	return res
}

func bisect(expression []string, character string, direction string) []string {
	index := slices.Index(expression, character)

	// Check if character is not found
	if index == -1 {
		fmt.Println("Character not found in expression.")
		os.Exit(3)
	}

	tokens := []string{}
	if direction == "right" {
		for i := index + 1; i < len(expression); i++ {
			tokens = append(tokens, expression[i])
		}
	} else if direction == "left" {
		for i := 0; i < index; i++ {
			tokens = append(tokens, expression[i])
		}
	} else {
		fmt.Println("Did not recognize direction: " + direction)
		os.Exit(3)
	}

	return tokens
}

func parseGeneric(tokens []string, lineNumber int) *Node {

	var newNode Node

	numbers := strings.Split("1234567890", "")
	letters := strings.Split("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "")

	if slices.Contains(tokens, "=") {
		newNode = Node{
			class: "ASSIGN",
			dtype: "CHAR",
			value: "=",
			left:  parseGeneric(bisect(tokens, "=", "left")),
			right: parseGeneric(bisect(tokens, "=", "right")),
		}
	} else if slices.Contains(tokens, "*") {
		newNode = Node{
			class: "MULT",
			dtype: "CHAR",
			value: "*",
			left:  parseGeneric(bisect(tokens, "*", "left")),
			right: parseGeneric(bisect(tokens, "*", "right")),
		}

	} else if slices.Contains(tokens, "/") {
		newNode = Node{
			class: "DIV",
			dtype: "CHAR",
			value: "/",
			left:  parseGeneric(bisect(tokens, "/", "left")),
			right: parseGeneric(bisect(tokens, "/", "right")),
		}

	} else if slices.Contains(tokens, "+") {
		newNode = Node{
			class: "ADD",
			dtype: "CHAR",
			value: "+",
			left:  parseGeneric(bisect(tokens, "+", "left")),
			right: parseGeneric(bisect(tokens, "+", "right")),
		}

	} else if slices.Contains(tokens, "-") {
		newNode = Node{
			class: "SUB",
			dtype: "CHAR",
			value: "-",
			left:  parseGeneric(bisect(tokens, "-", "left")),
			right: parseGeneric(bisect(tokens, "-", "right")),
		}

	} else if slices.Contains(numbers, tokens[0]) {
		newNode = Node{
			class: "NUMBER",
			dtype: "INT",
			value: tokens[0],
		}
	} else if slices.Contains(letters, tokens[0]) {
		newNode = Node{
			class: "IDENTIFIER",
			dtype: "CHAR",
			value: tokens[0],
		}
	} else {
		fmt.Println("Unrecognized character")
		os.Exit(3)

	}

	return &newNode
}

func createNode(expression []string, format string) *Node {
	var newNode Node
	switch format {
	case "TYPE IDENTIFIER":
		fmt.Println(format)
		newNode = Node{
			class: "IDENTIFIER",
			dtype: "CHAR",
			value: expression[1],
		}

	case "IDENTIFIER ASSIGN NUMBER":
		fmt.Println(format)
		newNode = Node{
			class: "ASSIGN",
			dtype: "CHAR",
			value: expression[1],
			left:  createNode([]string{expression[0]}, "IDENTIFIER"),
			right: createNode([]string{expression[2]}, "NUMBER"),
		}

	case "IDENTIFIER":
		fmt.Println(format)
		newNode = Node{
			class: "IDENTIFIER",
			dtype: "CHAR",
			value: expression[0],
		}

	case "NUMBER":
		fmt.Println(format)
		newNode = Node{
			class: "NUMBER",
			dtype: "INT",
			value: expression[0],
		}
	}

	return &newNode
}
