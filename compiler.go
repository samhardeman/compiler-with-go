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
	Type     string
	DType    string
	Value    string
	Declared []*Node
	Params   []*Node
	Returns  []*Node
	Body     []*Node
	Left     *Node
	Right    *Node
	Elements []*Node // For array elements
	Index    *Node   // For array indexing
}

type Symbol struct {
	dtype string
	value string
}

var line int

func main() {
	line++
	root := Node{}
	inputFile := getFlags()
	code := readLines(inputFile)
	newRoot := parse(code, &root)

	// Print the AST
	fmt.Println("Abstract Syntax Tree:")
	traverseAST(newRoot.Body)

	// Print the symbol table
	fmt.Println("\nSymbol Table:")
	printSymbolTable(newRoot, "")

	// Generate TAC and write to output.tac
	create_tac(newRoot, "output.tac")

	// Optimize the AST
	optimizedAST := optimizer(newRoot)

	// Generate optimized TAC and write to optimized_output.tac
	optimize_tac(optimizedAST, "optimized_output.tac")
}

func getFlags() string {
	inputFile := flag.String("file", "", "")
	flag.Parse()
	if *inputFile == "" {
		fmt.Printf("No file to compile provided")
		os.Exit(3)
	}
	return *inputFile
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
		case "write":
			endLineIndex := findEndLine(tokens[i:]) + i

			writeNode := parseWrite(tokens[i:endLineIndex], line, root)

			body = append(body, &writeNode)

			i = endLineIndex
			line++
		case "func":
			line++
			endFunctionDeclIndex := slices.Index(tokens[i:], "{") + i + 1
			closingBraceIndex := slices.Index(tokens[i:], "}") + i + 1
			funcNode := parseFunc(tokens[i:endFunctionDeclIndex], line)

			isValid := symbolMan(root, funcNode)

			parse(tokens[endFunctionDeclIndex:closingBraceIndex-1], funcNode)

			if !isValid {
				fmt.Println(funcNode.Value + " has already been declared! Error line: " + strconv.Itoa(line))
				os.Exit(3)
			}

			root.Declared = append(root.Declared, symbolNode(funcNode.Value, funcNode.Type, funcNode.DType))

			if tokens[closingBraceIndex] == "\n" {
				line++
			}

			body = append(body, funcNode)

			tokensTraversed := closingBraceIndex - i

			i += tokensTraversed
		case "int":
			line++
			endLineIndex := findEndLine(tokens[i:]) + i
			declLine := tokens[i:endLineIndex]
			declNode := parseDecl(declLine, line)

			isValid := symbolMan(root, declNode)

			if !isValid {
				fmt.Println(declNode.Value + " has already been declared!")
				os.Exit(3)
			}

			root.Declared = append(root.Declared, symbolNode(declNode.Value, declNode.Type, declNode.DType))

			if len(declLine) > 2 {
				if declLine[2] == "=" {
					newNode := parseGeneric(declLine[1:], line, root)
					body = append(body, newNode)
				}
			}

			i = endLineIndex
		case "[":
			var arrayDecl *Node
			endLineIndex := findEndLine(tokens[i:]) + i

			declLine := tokens[i:endLineIndex]

			if tokens[i+1] == "]" {
				arrayDecl = parseArrayDecl(tokens[i:endLineIndex], line)
			}

			isValid := symbolMan(root, arrayDecl)

			if !isValid {
				fmt.Println(arrayDecl.Value + " has already been declared!")
				os.Exit(3)
			}

			root.Declared = append(root.Declared, symbolNode(arrayDecl.Value, arrayDecl.Type, arrayDecl.DType))

			if len(declLine) > 3 {

				if declLine[4] == "=" {
					newNode := parseGeneric(declLine[3:], line, root)
					newNode.Right.DType = arrayDecl.DType
					body = append(body, newNode)
				}
			}

			i = endLineIndex

		case "return":
			line++
			endLineIndex := findEndLine(tokens[i:]) + i

			if len(tokens[i:endLineIndex]) > 2 {
				fmt.Println("Only one return argument allowed. Error: line " + strconv.Itoa(line))
				os.Exit(3)
			}

			newNode := parseReturn(tokens[i:endLineIndex], line, root)

			isValid := symbolMan(root, newNode)

			if isValid {
				root.Returns = append(root.Returns, newNode)
			}

			if !isValid {
				fmt.Println("Previously undeclared variable assignment: " + tokens[i] + " on line " + strconv.Itoa(line))
				os.Exit(3)
			}

			for _, declarations := range root.Declared {
				if declarations.Value == newNode.Value {
					newNode.DType = declarations.DType
					break
				}
			}

			checkFunctionReturnType(root, newNode)

			root.Returns = append(root.Returns, newNode)

			i = endLineIndex
		case "\n":
			i++
		case ";":
			i++
		default:
			endLineIndex := findEndLine(tokens[i:]) + i
			newNode := parseGeneric(tokens[i:endLineIndex], line, root)

			isValid := symbolMan(root, newNode)

			if isValid {
				body = append(body, newNode)
			}

			if !isValid {
				fmt.Println("Previously undeclared variable assignment: " + tokens[i] + " on line " + strconv.Itoa(line))
				os.Exit(3)
			}
			i = endLineIndex + 1
		}

	}

	root.Body = body

	return root
}

func symbolMan(root *Node, newNode *Node) bool {
	var declared bool
	var isValid bool

	switch newNode.Type {
	case "DECLARATION":
		for i := 0; i < len(root.Declared); i++ {
			if root.Declared[i].Value == newNode.Value {
				declared = true
				break
			}
		}
		for i := 0; i < len(root.Params); i++ {
			if root.Params[i].Value == newNode.Value {
				declared = true
				break
			}
		}
		if declared {
			isValid = false
		} else {
			isValid = true
		}

	case "FUNCTION_DECL":
		for i := 0; i < len(root.Declared); i++ {
			if root.Declared[i].Value == newNode.Value {
				declared = true
				break
			}
		}
		for i := 0; i < len(root.Params); i++ {
			if root.Params[i].Value == newNode.Value {
				declared = true
				break
			}
		}
		if declared {
			isValid = false
		} else {
			isValid = true
		}

	case "ARRAY_DECL":
		for i := 0; i < len(root.Declared); i++ {
			if root.Declared[i].Value == newNode.Value {
				declared = true
				break
			}
		}
		for i := 0; i < len(root.Params); i++ {
			if root.Params[i].Value == newNode.Value {
				declared = true
				break
			}
		}
		if declared {
			isValid = false
		} else {
			isValid = true
		}
		if declared {
			isValid = false
		} else {
			isValid = true
		}

	case "RETURN":
		for i := 0; i < len(root.Declared); i++ {
			if root.Declared[i].Value == newNode.Value {
				declared = true
				break
			}
		}
		for i := 0; i < len(root.Params); i++ {
			if root.Params[i].Value == newNode.Value {
				declared = true
				break
			}
		}
		if !declared {
			isValid = false
		} else {
			isValid = true
		}

	case "ASSIGN":
		for i := 0; i < len(root.Declared); i++ {
			if root.Declared[i].Value == newNode.Left.Value {
				declared = true
				break
			}
		}

		for i := 0; i < len(root.Params); i++ {
			if root.Params[i].Value == newNode.Left.Value {
				declared = true
				break
			}
		}

		if !declared {
			isValid = false
		} else {
			isValid = true
		}
	case "IDENTIFIER":
		for i := 0; i < len(root.Declared); i++ {
			if root.Declared[i].Value == newNode.Value {
				declared = true
				break
			}
		}

		for i := 0; i < len(root.Params); i++ {
			if root.Params[i].Value == newNode.Value {
				declared = true
				break
			}
		}

		if !declared {
			isValid = false
		} else {
			isValid = true
		}

	default:
		isValid = true
	}

	return isValid
}

func checkFunctionReturnType(root *Node, returnNode *Node) {

	if root.DType == "void" {
		fmt.Println("Unexpected return in function " + root.Value + " which is void of returns!")
		os.Exit(3)
	} else if returnNode.DType != root.DType {
		fmt.Println("Returned variable " + returnNode.Value + " in " + root.Value + " does not match function return type!")
		os.Exit(3)
	}

}

func returnType(root *Node, searchedNode *Node) string {
	var returnType string

	for _, declared := range root.Declared {
		if declared.Value == searchedNode.Value {
			returnType = declared.DType
			break
		}
	}

	for _, params := range root.Params {
		if params.Value == searchedNode.Value {
			returnType = params.DType
			break
		}
	}

	return returnType

}

func symbolNode(name string, decltype string, dtype string) *Node {
	newNode := Node{
		Type:  decltype,
		DType: dtype,
		Value: name,
	}

	return &newNode
}

func parseDecl(tokens []string, lineNumber int) *Node {
	newNode := Node{
		Type:  "DECLARATION",
		Value: tokens[1],
	}

	if strings.HasPrefix(tokens[0], "int") || strings.HasPrefix(tokens[0], "string") || strings.HasPrefix(tokens[0], "char") {
		newNode.DType = strings.ToUpper(tokens[0])
	} else if tokens[0] == "[" && tokens[1] == "]" {
		// Handling array declarations
		baseType := strings.ToUpper(tokens[2])
		newNode.DType = "[]" + baseType
		newNode.Type = "ARRAY_DECLARATION"
	} else {
		fmt.Println("Unknown type in declaration on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	if !isIdentifier(newNode.Value) {
		fmt.Println("Expected variable name declaration got " + newNode.Value + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}
	return &newNode
}

func parseReturn(tokens []string, lineNumber int, root *Node) *Node {
	newNode := Node{
		Type:  "RETURN",
		Value: tokens[1],
	}

	newNode.DType = returnType(root, &newNode)

	return &newNode
}

// Parse Function Declarations
func parseFunc(tokens []string, lineNumber int) *Node {
	var newNode Node
	openParen := 0

	newNode.Type = "FUNCTION_DECL"
	newNode.DType = "VOID"

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

	if isIdentifier(tokens[closeParenIndex+1]) {
		newNode.DType = strings.ToUpper(tokens[closeParenIndex+1])
	} else if tokens[closeParenIndex+1] != "{" {
		fmt.Println("Expected \"{\" got " + tokens[closeParenIndex+1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	return &newNode

}

func parseArray(tokens []string, lineNumber int, root *Node) Node {

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
			newNode.Body = append(newNode.Body, parseGeneric(args[i:i+1], lineNumber, root))
		}
	}

	return newNode
}

func parseWrite(tokens []string, lineNumber int, root *Node) Node {
	newNode := Node{
		Type:  "FUNCTION_CALL",
		Value: tokens[0], // The function name (e.g., 'write')
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

	if len(args) > 1 {
		fmt.Println("Unexpected character \"" + args[0] + "\" in parameters call on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	} else {
		newNode.Params = append(newNode.Params, parseGeneric(args[0:1], lineNumber, root))
	}

	return newNode
}

func parseFunctionCall(tokens []string, lineNumber int, root *Node) Node {
	newNode := Node{
		Type:  "FUNCTION_CALL",
		Value: tokens[0], // The function name (e.g., 'print')
	}

	functionDeclared := false

	for _, declared := range root.Declared {
		if declared.Value == newNode.Value {
			newNode.DType = declared.DType
			functionDeclared = true
			break
		}
	}

	if !functionDeclared {
		fmt.Println("Unrecognized function \"" + newNode.Value + "\"")
		os.Exit(3)
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
			newNode.Params = append(newNode.Params, parseGeneric(args[i:i+1], lineNumber, root))
		}
	}

	return newNode
}

func parseArrayDecl(tokens []string, lineNumber int) *Node {
	newNode := Node{
		Type:  "ARRAY_DECL",
		DType: tokens[0] + tokens[1] + tokens[2],
		Value: tokens[3],
	}

	if !isIdentifier(tokens[3]) {
		fmt.Println("Expected variable name declaration got " + tokens[3] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	return &newNode
}

// Validates identifiers (variable names, function names, etc.)
func isIdentifier(word string) bool {
	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return validIdentifier.MatchString(word)
}

// SplitStringInPlace splits mixed strings (like "add(int" or "b)") in the array in place
func splitStringInPlace(arr *[]string) {
	// Define a regex pattern to match numbers, identifiers, and operators
	pattern := regexp.MustCompile(`\d+(\.\d+)?|\w+|[(){}[\];,+\-*/%=<>!]`)

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

func traverseAST(nodes []*Node) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		printNode(node, "", isLast)
	}
}

func printNode(node *Node, prefix string, isTail bool) {
	// Print the current node
	fmt.Printf("%s%s [Type: %s, DType: %s, Value: %s]\n",
		prefix, getBranch(isTail), node.Type, node.DType, node.Value)

	// Prepare the prefix for child nodes
	childPrefix := prefix
	if isTail {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Print Declared symbols in this node (if any)
	if len(node.Declared) > 0 {
		fmt.Printf("%s%sDeclared Symbols:\n", childPrefix, getBranch(false))
		for i, decl := range node.Declared {
			isLastDecl := i == len(node.Declared)-1
			printNode(decl, childPrefix+"    ", isLastDecl)
		}
	}

	// Print Parameters (if any)
	if len(node.Params) > 0 {
		fmt.Printf("%s%sParameters:\n", childPrefix, getBranch(false))
		for i, param := range node.Params {
			isLastParam := i == len(node.Params)-1
			printNode(param, childPrefix+"    ", isLastParam)
		}
	}

	// Print Body (if any)
	if len(node.Body) > 0 {
		fmt.Printf("%s%sBody:\n", childPrefix, getBranch(false))
		for i, child := range node.Body {
			isLastChild := i == len(node.Body)-1
			printNode(child, childPrefix+"    ", isLastChild)
		}
	}

	// Print Left and Right nodes (if any)
	if node.Left != nil {
		fmt.Printf("%s%sLeft:\n", childPrefix, getBranch(false))
		printNode(node.Left, childPrefix+"    ", false)
	}
	if node.Right != nil {
		fmt.Printf("%s%sRight:\n", childPrefix, getBranch(true))
		printNode(node.Right, childPrefix+"    ", true)
	}
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

func operatorTypeComparison(node *Node) {
	if node.Left.DType != node.Right.DType {
		fmt.Println("Type mismatch between " + node.Left.Value + " (" + node.Left.DType + ") " + "and " + node.Right.Value + " (" + node.Right.DType + ") " + " Error: line " + strconv.Itoa(line))
		os.Exit(3)
	}
}

func parseGeneric(tokens []string, lineNumber int, root *Node) *Node {

	var newNode Node

	numbers := strings.Split("1234567890", "")
	letters := strings.Split("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "")

	if slices.Contains(tokens, "=") {
		newNode = Node{
			Type:  "ASSIGN",
			DType: "OP",
			Value: "=",
			Left:  parseGeneric(bisect(tokens, "=", "left"), lineNumber, root),
			Right: parseGeneric(bisect(tokens, "=", "right"), lineNumber, root),
		}

		if newNode.Right.Value == "{}" {
			expectedElementType := strings.ToUpper(strings.Split(newNode.Left.DType, "]")[1])

			if newNode.Left.DType != "[]any" {
				for _, element := range newNode.Right.Body {
					if element.DType != expectedElementType {
						fmt.Println("Array element " + element.Value + " does not match array type " + expectedElementType)
						os.Exit(3)
					}
				}
			}
		} else if newNode.Right.DType != "OP" {
			operatorTypeComparison(&newNode)
		}

	} else if slices.Contains(tokens, "{") && slices.Contains(tokens, "}") {
		newNode = parseArrayInitialization(tokens, lineNumber, root)
	} else if slices.Contains(tokens, "*") {
		newNode = Node{
			Type:  "MULT",
			DType: "OP",
			Value: "*",
			Left:  parseGeneric(bisect(tokens, "*", "left"), lineNumber, root),
			Right: parseGeneric(bisect(tokens, "*", "right"), lineNumber, root),
		}

		operatorTypeComparison(&newNode)

		newNode.DType = newNode.Left.DType

	} else if slices.Contains(tokens, "/") {
		newNode = Node{
			Type:  "DIV",
			DType: "OP",
			Value: "/",
			Left:  parseGeneric(bisect(tokens, "/", "left"), lineNumber, root),
			Right: parseGeneric(bisect(tokens, "/", "right"), lineNumber, root),
		}

		operatorTypeComparison(&newNode)

		newNode.DType = newNode.Left.DType

	} else if slices.Contains(tokens, "+") {
		newNode = Node{
			Type:  "ADD",
			DType: "OP",
			Value: "+",
			Left:  parseGeneric(bisect(tokens, "+", "left"), lineNumber, root),
			Right: parseGeneric(bisect(tokens, "+", "right"), lineNumber, root),
		}

		operatorTypeComparison(&newNode)

		newNode.DType = newNode.Left.DType

	} else if slices.Contains(tokens, "-") {
		newNode = Node{
			Type:  "SUB",
			DType: "OP",
			Value: "-",
			Left:  parseGeneric(bisect(tokens, "-", "left"), lineNumber, root),
			Right: parseGeneric(bisect(tokens, "-", "right"), lineNumber, root),
		}

		operatorTypeComparison(&newNode)

		newNode.DType = newNode.Left.DType

	} else if slices.Contains(tokens, "(") && slices.Contains(tokens, ")") {
		newNode = parseFunctionCall(tokens, line, root)
	} else if slices.Contains(tokens, "{") && slices.Contains(tokens, "}") {
		newNode = parseArray(tokens, line, root)
	} else if slices.Contains(tokens, "[") && slices.Contains(tokens, "]") {
		var index string
		for _, token := range tokens {
			index = index + token
		}
		newNode = Node{
			Type:  "ARRAY_INDEX",
			DType: "INT",
			Value: index,
		}
	} else if slices.Contains(numbers, tokens[0]) {
		newNode = Node{
			Type:  "NUMBER",
			DType: "INT",
			Value: tokens[0],
		}
	} else if slices.Contains(letters, tokens[0]) {

		newNode = Node{
			Type:  "IDENTIFIER",
			Value: tokens[0],
		}

		returnType := returnType(root, &newNode)

		newNode.DType = returnType

		isValid := symbolMan(root, &newNode)

		if !isValid {
			fmt.Println("Previously undeclared variable assignment: " + tokens[0] + " on line " + strconv.Itoa(line))
			os.Exit(3)
		}

	} else {
		fmt.Println("Unrecognized character \"" + tokens[0] + "\" on line " + strconv.Itoa(line))
		os.Exit(3)

	}

	return &newNode
}

func create_tac(root *Node, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating TAC file: %v\n", err)
		os.Exit(3)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	generateTAC(root, writer)
	writer.Flush()
}

func generateTAC(node *Node, writer *bufio.Writer) {
	if node == nil {
		return
	}

	switch node.Type {
	case "ASSIGN":
		// Generate TAC for assignment
		left := node.Left.Value
		right := generateExpressionTAC(node.Right, writer)
		line := fmt.Sprintf("%s = %s\n", left, right)
		writer.WriteString(line)
	case "FUNCTION_DECL":
		// Handle function declaration
		writer.WriteString(fmt.Sprintf("func %s:\n", node.Value))
		for _, stmt := range node.Body {
			generateTAC(stmt, writer)
		}
		writer.WriteString("end func\n")
	case "FUNCTION_CALL":
		// Handle function call
		writer.WriteString(fmt.Sprintf("call %s\n", node.Value))
	case "RETURN":
		// Handle return statement
		writer.WriteString(fmt.Sprintf("return %s\n", node.Value))
	default:
		// Handle other node types if necessary
	}

	// Recursively generate TAC for child nodes
	if node.Left != nil {
		generateTAC(node.Left, writer)
	}
	if node.Right != nil {
		generateTAC(node.Right, writer)
	}
	for _, child := range node.Body {
		generateTAC(child, writer)
	}
}

func generateExpressionTAC(node *Node, writer *bufio.Writer) string {
	if node == nil {
		return ""
	}

	switch node.Type {
	case "NUMBER", "IDENTIFIER":
		return node.Value
	case "ADD", "SUB", "MULT", "DIV":
		left := generateExpressionTAC(node.Left, writer)
		right := generateExpressionTAC(node.Right, writer)
		tempVar := getTempVar()
		line := fmt.Sprintf("%s = %s %s %s\n", tempVar, left, node.Value, right)
		writer.WriteString(line)
		return tempVar
	default:
		return ""
	}
}

var tempVarCounter int

func getTempVar() string {
	tempVarCounter++
	return fmt.Sprintf("t%d", tempVarCounter)
}
func printSymbolTable(node *Node, indent string) {
	if node == nil {
		return
	}

	// Print the symbols declared at this node's scope
	if len(node.Declared) > 0 {
		fmt.Printf("%sScope (%s):\n", indent, node.Value)
		for _, sym := range node.Declared {
			fmt.Printf("%s    Name: %s, Type: %s, DType: %s\n",
				indent, sym.Value, sym.Type, sym.DType)
		}
	}

	// Recursively print symbol tables of child nodes
	for _, child := range node.Body {
		printSymbolTable(child, indent+"    ")
	}
}
func getBranch(isTail bool) string {
	if isTail {
		return "└── "
	}
	return "├── "
}

// Array shit that really works
func parseArrayInitialization(tokens []string, lineNumber int, root *Node) Node {
	newNode := Node{
		Type:     "ARRAY_INITIALIZATION",
		Value:    "{}",
		Elements: []*Node{},
	}

	// Extract elements between braces
	startIndex := slices.Index(tokens, "{")
	endIndex := slices.Index(tokens, "}")

	if startIndex == -1 || endIndex == -1 || endIndex <= startIndex {
		fmt.Println("Invalid array initialization on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	elementsTokens := tokens[startIndex+1 : endIndex]
	elements := parseArrayElements(elementsTokens, lineNumber, root)
	newNode.Elements = elements

	return newNode
}
func parseArrayElements(tokens []string, lineNumber int, root *Node) []*Node {
	var elements []*Node
	currentElementTokens := []string{}

	for _, token := range tokens {
		if token == "," {
			elementNode := parseGeneric(currentElementTokens, lineNumber, root)
			elements = append(elements, elementNode)
			currentElementTokens = []string{}
		} else {
			currentElementTokens = append(currentElementTokens, token)
		}
	}

	// Add the last element
	if len(currentElementTokens) > 0 {
		elementNode := parseGeneric(currentElementTokens, lineNumber, root)
		elements = append(elements, elementNode)
	}

	return elements
}
