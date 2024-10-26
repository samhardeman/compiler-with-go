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

var line int

func main() {
	root := Node{}
	var inputFile string = getFlags()
	code := readLines(inputFile)
	newRoot := parse(code, &root)
	traverseAST(newRoot.Body)
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
		// splits line into tokens
		tokens := strings.Fields(scanner.Text())
		splitStringInPlace(&tokens)
		tokens = append(tokens, "\n")
		code = append(code, tokens...)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return code
}

// Parse (big slay)
func parse(tokens []string, root *Node) *Node {
	body := []*Node{}

	// iterate through code
	for i := 0; i < len(tokens); i += 0 {

		token := tokens[i]

		switch token {
		case "func":
			line++
			endFunctionDeclIndex := slices.Index(tokens[i:], "{") + i + 1
			closingBraceIndex := slices.Index(tokens[i:], "}") + i + 1
			funcNode := parseFunc(tokens[i:endFunctionDeclIndex], line)
			body = append(body, funcNode)
			parse(tokens[endFunctionDeclIndex:closingBraceIndex], funcNode)

			tokensTraversed := closingBraceIndex - i

			i += tokensTraversed
		case "int":
			line++
			endLineIndex := findEndLine(tokens[i:]) + i
			fmt.Println(tokens[i:endLineIndex])
			root.Params = append(root.Params, parseDecl(tokens[i:endLineIndex], line))

			i += endLineIndex
			fmt.Println(tokens[i])

		case "\n":
			line++
			i++
		case ";":
			i++
		default:
			fmt.Println(tokens[i : findEndLine(tokens[i:])+i])
			newNode := parseGeneric(tokens[i:findEndLine(tokens[i:])+i], line)
			linked := false

			for i := 0; i < len(root.Params); i++ {
				if root.Params[i].Value == newNode.Left.Value {
					body = append(body, newNode)
					linked = true
					break
				}
			}

			if !linked {
				fmt.Println("Previously undeclared variable assignment: " + tokens[0] + " on line " + strconv.Itoa(line+1))
				os.Exit(3)
			}
			i++
		}

	}

	root.Body = body

	return root
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

	if len(tokens) > 2 {
		if tokens[2] == "=" {
			newNode.Right = parseGeneric(tokens[1:], lineNumber)
		}
	}

	return &newNode
}

// Parse Function Declarations
func parseFunc(tokens []string, lineNumber int) *Node {
	var newNode Node
	splitStringInPlace(&tokens)
	openParen := 0

	newNode.Type = "FUNCTION_DECL"

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

		for i := 1; i < (len(params) + 1); i += 3 {
			newNode.Params = append(newNode.Params, parseDecl(params[i:i+2], lineNumber))
		}
	}

	if tokens[closeParenIndex+1] == "int" {
		newNode.DType = "int"
	} else if tokens[closeParenIndex+1] != "{" {
		fmt.Println("Expected \"{\" got " + tokens[closeParenIndex+1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	return &newNode

}

func parseFunctionCall(tokens []string, lineNumber int) Node {
	newNode := Node{
		Type:  "FUNCTION_CALL",
		Value: tokens[0], // The function name (e.g., 'print')
	}

	// Expect the second token to be an opening parenthesis
	if tokens[1] != "(" {
		fmt.Println("Expected \"(\" after function name, got " + tokens[1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	// Find the closing parenthesis
	closeParenIndex := slices.Index(tokens, ")")
	if closeParenIndex == -1 {
		fmt.Println("Expected \")\" to close function call on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	// Extract the arguments between parentheses
	args := tokens[2:closeParenIndex]

	// Parse each argument and add it to the function's Params

	for i := 0; i < (len(args)); i += 2 {
		if args[i] == "," {
			fmt.Println("Unexpected character \"" + args[i] + "\" in parameters call on line " + strconv.Itoa(lineNumber))
			os.Exit(3)
		} else {
			newNode.Params = append(newNode.Params, parseGeneric(args[i:i+1], lineNumber))
		}
	}

	return newNode
}

func parseArray(tokens []string, lineNumber int) *Node {
	var newNode Node

	newNode.DType = tokens[0] + tokens[1] + tokens[2]

	return &newNode
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

func traverseAST(root []*Node) {
	for i := 0; i < len(root); i++ {
		printNode(root[i], "", true)
	}
}

func printNode(node *Node, prefix string, isTail bool) {
	// Construct the current node's details (Type, DType, Value)
	nodeRepresentation := fmt.Sprintf("%s [Type: %s, DType: %s, Value: %s]", node.Value, node.Type, node.DType, node.Value)

	// Print the node, with graphical tree branches (└── or ├──)
	fmt.Printf("%s%s%s\n", prefix, getBranch(isTail), nodeRepresentation)

	// Prepare the prefix for children
	newPrefix := prefix
	if isTail {
		newPrefix += "    " // For the last child, indent
	} else {
		newPrefix += "│   " // For other children, continue the branch
	}

	// Handle Params, if any
	if len(node.Params) > 0 {
		fmt.Println(newPrefix + "Params:")
		for i := 0; i < len(node.Params); i++ {
			printNode(node.Params[i], newPrefix, i == len(node.Params)-1)
		}
	}

	// Handle Body, if any
	if len(node.Body) > 0 {
		fmt.Println(newPrefix + "Body:")
		for i := 0; i < len(node.Body); i++ {
			printNode(node.Body[i], newPrefix, i == len(node.Body)-1)
		}
	}

	// Handle Left and Right children (for operations like +, -, *, /)
	if node.Left != nil {
		fmt.Println(newPrefix + "Left:")
		printNode(node.Left, newPrefix, false)
	}
	if node.Right != nil {
		fmt.Println(newPrefix + "Right:")
		printNode(node.Right, newPrefix, true)
	}
}

// getBranch returns the appropriate branch characters for the tree
func getBranch(isTail bool) string {
	if isTail {
		return "└── "
	}
	return "├── "
}

func findEndLine(chunk []string) int {
	var endLineIndex int

	newLineIndex := slices.Index(chunk, "\n")

	semiIndex := slices.Index(chunk, ";")

	if semiIndex == -1 {
		endLineIndex = newLineIndex

	} else if newLineIndex > semiIndex {
		endLineIndex = newLineIndex

	} else {
		endLineIndex = semiIndex

	}

	return endLineIndex

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
			Type:  "ASSIGN",
			DType: "CHAR",
			Value: "=",
			Left:  parseGeneric(bisect(tokens, "=", "left"), lineNumber),
			Right: parseGeneric(bisect(tokens, "=", "right"), lineNumber),
		}
	} else if slices.Contains(tokens, "*") {
		newNode = Node{
			Type:  "MULT",
			DType: "CHAR",
			Value: "*",
			Left:  parseGeneric(bisect(tokens, "*", "left"), lineNumber),
			Right: parseGeneric(bisect(tokens, "*", "right"), lineNumber),
		}

	} else if slices.Contains(tokens, "/") {
		newNode = Node{
			Type:  "DIV",
			DType: "CHAR",
			Value: "/",
			Left:  parseGeneric(bisect(tokens, "/", "left"), lineNumber),
			Right: parseGeneric(bisect(tokens, "/", "right"), lineNumber),
		}

	} else if slices.Contains(tokens, "+") {
		newNode = Node{
			Type:  "ADD",
			DType: "CHAR",
			Value: "+",
			Left:  parseGeneric(bisect(tokens, "+", "left"), lineNumber),
			Right: parseGeneric(bisect(tokens, "+", "right"), lineNumber),
		}

	} else if slices.Contains(tokens, "-") {
		newNode = Node{
			Type:  "SUB",
			DType: "CHAR",
			Value: "-",
			Left:  parseGeneric(bisect(tokens, "-", "left"), lineNumber),
			Right: parseGeneric(bisect(tokens, "-", "right"), lineNumber),
		}
	} else if slices.Contains(tokens, "(") && slices.Contains(tokens, ")") {
		newNode = parseFunctionCall(tokens, line)
	} else if slices.Contains(numbers, tokens[0]) {
		newNode = Node{
			Type:  "NUMBER",
			DType: "INT",
			Value: tokens[0],
		}
	} else if slices.Contains(letters, tokens[0]) {
		newNode = Node{
			Type:  "IDENTIFIER",
			DType: "CHAR",
			Value: tokens[0],
		}
	} else {
		fmt.Println("Unrecognized character \"" + tokens[1] + " \" on line " + strconv.Itoa(line))
		os.Exit(3)

	}

	return &newNode
}
