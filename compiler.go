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

// Node represents an AST node
type Node struct {
	Type     string
	DType    string
	Value    string
	Declared []*Node
	Params   []*Node
	Returns  []*Node
	Body     []*Node
	Left     *Node
	Right    *Node
}

var line int

func main() {
	inputFile := parseFlags()
	code := readLines(inputFile)
	root := parse(code, &Node{})
	traverseAST(root.Body)
}

// parseFlags handles command-line arguments and validates file input
func parseFlags() string {
	inputFile := flag.String("file", "", "File to compile")
	flag.Parse()
	if *inputFile == "" {
		log.Fatalf("Error: No file to compile provided")
	}
	return *inputFile
}

// readLines reads all lines from the given input file and tokenizes each line
func readLines(inputFile string) []string {
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	var code []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tokens := strings.Fields(scanner.Text())
		splitTokens(&tokens)
		tokens = append(tokens, "\n")
		code = append(code, tokens...)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	return code
}

// parse recursively processes tokens and builds the AST
func parse(tokens []string, root *Node) *Node {
	body := []*Node{}
	for i := 0; i < len(tokens); {
		token := tokens[i]
		switch token {
		case "func":
			funcNode := parseFunc(tokens[i:slices.Index(tokens[i:], "{")+i+1], line)
			if !symbolCheck(root, funcNode) {
				log.Fatalf("Error: Function %s already declared on line %d", funcNode.Value, line)
			}
			root.Params = append(root.Params, funcNode)
			body = append(body, funcNode)
			i += advanceToEnd(tokens[i:], "{", "}") // advance to end of function block
		case "int":
			declNode := parseDeclaration(tokens[i:advanceToEnd(tokens[i:], "\n")+i], line)
			if !symbolCheck(root, declNode) {
				log.Fatalf("Error: Variable %s already declared on line %d", declNode.Value, line)
			}
			root.Params = append(root.Params, declNode)
			body = append(body, declNode)
			i += advanceToEnd(tokens[i:], "\n")
		case "[":
			arrayNode := parseArrayDecl(tokens[i:advanceToEnd(tokens[i:], "\n")+i], line)
			if !symbolCheck(root, arrayNode) {
				log.Fatalf("Error: Array %s already declared on line %d", arrayNode.Value, line)
			}
			root.Params = append(root.Params, arrayNode)
			body = append(body, arrayNode)
			i += advanceToEnd(tokens[i:], "\n")

			var arrayDecl *Node

			declLine := tokens[i : advanceToEnd(tokens[i:], "\n")+i]

			if tokens[i+1] == "]" {
				arrayDecl = parseArrayDecl(declLine, line)
			}

			if !symbolCheck(root, arrayDecl) {
				fmt.Println(arrayDecl.Value + " has already been declared!")
				os.Exit(3)
			}

			root.Params = append(root.Params, arrayNode)

			if len(declLine) > 3 {

				if declLine[4] == "=" {
					newNode := parseGeneric(declLine[3:], line)
					newNode.Right.DType = arrayDecl.DType
					body = append(body, newNode)
				}
			}

			i += advanceToEnd(tokens[i:], "\n")
		case "\n", ";":
			i++
		default:
			fmt.Println(tokens[i : advanceToEnd(tokens[i:], "\n")+i])
			assignNode := parseGeneric(tokens[i:advanceToEnd(tokens[i:], "\n")+i], line)
			fmt.Println("parse: " + assignNode.Value)
			if symbolCheck(root, assignNode) {
				body = append(body, assignNode)
			} else {
				log.Fatalf("Error: Undeclared variable %s assignment on line %d", tokens[i], line)
			}
			i += advanceToEnd(tokens[i:], "\n")
		}
	}
	root.Body = body
	return root
}

// symbolCheck checks for existing declarations to avoid re-declaration errors.
// If it's an assignment, it checks the left node to ensure the symbol is declared.
func symbolCheck(root *Node, newNode *Node) bool {
	// Check if the node is an assignment
	if newNode.Type == "ASSIGN" {
		// Traverse the left node to check for declaration
		if newNode.Left != nil {
			for _, param := range root.Params {
				fmt.Println("Checking assignment for:", param.Value, "vs", newNode.Left.Value)
				if param.Value == newNode.Left.Value {
					return true // Symbol exists, valid assignment
				}
			}
			fmt.Println("Undeclared variable in assignment:", newNode.Left.Value)
			return false // Symbol not declared, invalid assignment
		}
	}

	// For functions, check directly in root.Params without left/right traversal
	if newNode.Type == "FUNCTION" {
		for _, param := range root.Params {
			fmt.Println("Checking function parameter:", param.Value, "vs", newNode.Value)
			if param.Value == newNode.Value {
				return true // Function name is already declared
			}
		}
		return false // New function declaration allowed
	}

	// Default case: assume no re-declaration errors
	return true
}

// splitTokens separates mixed tokens in a line (e.g., "func(" to "func", "(")
func splitTokens(tokens *[]string) {
	pattern := regexp.MustCompile(`[a-zA-Z0-9]+|[(){}[\];,+\-*/%=<>!]`)
	var result []string
	for _, str := range *tokens {
		result = append(result, pattern.FindAllString(str, -1)...)
	}
	*tokens = result
}

// traverseAST prints the AST recursively
func traverseAST(nodes []*Node) {
	for _, node := range nodes {
		printNode(node, "", true)
	}
}

// printNode displays details of an AST node in a formatted tree-like structure
func printNode(node *Node, prefix string, isTail bool) {
	fmt.Printf("%s%s [Type: %s, DType: %s, Value: %s]\n", prefix, getBranch(isTail), node.Type, node.DType, node.Value)
	newPrefix := prefix
	if isTail {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}
	if len(node.Params) > 0 {
		fmt.Println(newPrefix + "Params:")
		for i, param := range node.Params {
			printNode(param, newPrefix, i == len(node.Params)-1)
		}
	}
	if len(node.Body) > 0 {
		fmt.Println(newPrefix + "Body:")
		for i, bodyNode := range node.Body {
			printNode(bodyNode, newPrefix, i == len(node.Body)-1)
		}
	}
	if node.Left != nil {
		fmt.Println(newPrefix + "Left:")
		printNode(node.Left, newPrefix, false)
	}
	if node.Right != nil {
		fmt.Println(newPrefix + "Right:")
		printNode(node.Right, newPrefix, true)
	}
}

// getBranch determines the branch characters for tree-like printing
func getBranch(isTail bool) string {
	if isTail {
		return "└── "
	}
	return "├── "
}

// parseFunc parses a function declaration
func parseFunc(tokens []string, lineNumber int) *Node {
	if len(tokens) < 3 {
		fmt.Println("Insufficient tokens for function declaration on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	newNode := &Node{
		Type:  "FUNCTION_DECL",
		DType: "void",
	}

	if !isValidIdentifier(tokens[1]) {
		fmt.Println("Expected function name declaration got " + tokens[1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	} else {
		newNode.Value = tokens[1]
	}

	if tokens[2] != "(" {
		fmt.Println("Expected \"(\" got " + tokens[2] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	// Check if closing parenthesis exists
	closeParenIndex := slices.Index(tokens, ")")
	if closeParenIndex == -1 {
		fmt.Println("Expected \")\" but none found on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	// Parse function parameters if they exist
	if closeParenIndex > 3 {
		newNode.Params = parseParams(tokens[3:closeParenIndex], lineNumber)
	}

	// Ensure there’s a function body
	if closeParenIndex+1 < len(tokens) && tokens[closeParenIndex+1] == "{" {
		return newNode
	}

	fmt.Println("Expected \"{\" after function declaration on line " + strconv.Itoa(lineNumber))
	os.Exit(3)

	return newNode
}

// parseDeclaration parses a variable declaration
func parseDeclaration(tokens []string, lineNumber int) *Node {
	node := &Node{Type: "DECLARATION", DType: tokens[0], Value: tokens[1]}
	if !isValidIdentifier(tokens[1]) {
		log.Fatalf("Error: Invalid variable name %s on line %d", tokens[1], lineNumber)
	}
	return node
}

// parseParams extracts and parses function parameters from tokens
func parseParams(tokens []string, lineNumber int) []*Node {
	var params []*Node
	for i := 0; i < len(tokens); i += 3 {
		fmt.Println(tokens)
		// Ensure there are enough tokens to form a parameter (dtype and name)
		if i+1 >= len(tokens) {
			fmt.Println("Incomplete parameter declaration on line " + strconv.Itoa(lineNumber))
			os.Exit(3)
		}

		// Validate the data type
		dtype := tokens[i]
		if !isValidDataType(dtype) {
			fmt.Println("Invalid data type " + dtype + " for parameter on line " + strconv.Itoa(lineNumber))
			os.Exit(3)
		}

		// Validate the parameter name
		paramName := tokens[i+1]
		if !isValidIdentifier(paramName) {
			fmt.Println("Invalid identifier " + paramName + " for parameter name on line " + strconv.Itoa(lineNumber))
			os.Exit(3)
		}

		// Create a Node for the parameter
		paramNode := &Node{
			Type:  "PARAM",
			DType: dtype,
			Value: paramName,
		}
		params = append(params, paramNode)
	}
	return params
}

// isValidIdentifier validates if a token is a valid identifier
func isValidIdentifier(word string) bool {
	return regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(word)
}

// isValidDataType checks if the provided data type is valid
func isValidDataType(dtype string) bool {
	validDataTypes := []string{"int", "float", "string", "bool"}
	for _, validType := range validDataTypes {
		if dtype == validType {
			return true
		}
	}
	return false
}

// advanceToEnd processes tokens until a specified end delimiter or a default end-of-line delimiter
func advanceToEnd(tokens []string, delimiter ...string) int {
	endDelimiter := "\n" // Default end delimiter
	if len(delimiter) > 0 {
		endDelimiter = delimiter[0] // Use provided delimiter if available
	}

	for i, token := range tokens {
		if token == endDelimiter {
			return i
		}
	}
	return len(tokens) // Return length if end delimiter is not found
}

// parseArrayDecl parses an array declaration, supporting syntax like []int d or []int d = {1, 2, 3}
func parseArrayDecl(tokens []string, lineNumber int) *Node {
	newNode := Node{
		Type:  "ARRAY_DECL",
		DType: tokens[0] + tokens[1] + tokens[2],
		Value: tokens[3],
	}

	if !isValidIdentifier(tokens[3]) {
		fmt.Println("Expected variable name declaration got " + tokens[3] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	return &newNode
}

func parseArray(tokens []string, lineNumber int) Node {

	newNode := Node{
		Type:  "ARRAY",
		Value: "{}",
	}

	// Expect the second token to be an opening parenthesis
	if tokens[0] != "{" {
		fmt.Println("Expected \"{\" after function name, got " + tokens[0] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	// Find the closing parenthesis
	closeBracketIndex := slices.Index(tokens, "}")
	if closeBracketIndex == -1 {
		fmt.Println("Expected \"}\" to close function call on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	// Extract the arguments between brackets
	args := tokens[1:closeBracketIndex]

	// Parse each element

	for i := 0; i < (len(args)); i += 2 {
		if args[i] == "," {
			fmt.Println("Unexpected character \"" + args[i] + "\" in array setting on line " + strconv.Itoa(lineNumber))
			os.Exit(3)
		} else {
			newNode.Body = append(newNode.Body, parseGeneric(args[i:i+1], lineNumber))
		}
	}

	return newNode
}

// parseGeneric handles assignments and operations
func parseGeneric(tokens []string, lineNumber int) *Node {
	fmt.Println("parseGeneric: " + tokens[0])
	switch {
	case len(tokens) > 2 && tokens[1] == "=":
		left := &Node{Type: "IDENTIFIER", Value: tokens[0]}
		right := parseExpression(tokens[2:], lineNumber)
		return &Node{Type: "ASSIGNMENT", Value: tokens[1], Left: left, Right: right}
	default:
		return parseExpression(tokens, lineNumber)
	}
}

// parseExpression handles arithmetic and logical expressions recursively
func parseExpression(tokens []string, lineNumber int) *Node {
	fmt.Println(tokens)
	for _, op := range []string{"=", "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">="} {
		if index := indexOf(tokens, op); index != -1 {
			return &Node{
				Type:  "EXPRESSION",
				Value: op,
				Left:  parseExpression(tokens[:index], lineNumber),
				Right: parseExpression(tokens[index+1:], lineNumber),
			}
		}
	}
	return &Node{Type: "LITERAL", Value: tokens[0]}
}

// indexOf finds the first occurrence of a string in a slice
func indexOf(tokens []string, search string) int {
	for i, token := range tokens {
		if token == search {
			return i
		}
	}
	return -1
}

// Helper functions for error handling and code analysis

// raiseError handles error messages with line numbers
func raiseError(line int, message string) {
	log.Fatalf("Error on line %d: %s", line, message)
}
