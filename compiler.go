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
}

type Symbol struct {
	dtype string
	value string
}

var line int

func main() {
	line++
	root := Node{}
	var inputFile string = getFlags()
	code := readLines(inputFile)
	fmt.Println("Parsing...")
	newRoot := parse(code, &root)
	fmt.Println("Finished Parsing! Beginning Optimization...")
	optimizedAST := optimizer(newRoot)
	fmt.Println("Finished Optimization! Outputting Tac...")
	optimize_tac(&optimizedAST, "output.tac")
	fmt.Println("Tac Complete! Compiling to MIPS...")
	tac2Mips("output.tac")
	fmt.Println("Finished! 【=◈ ︿ ◈=】")
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
		lineText := scanner.Text()
		fmt.Printf("Raw Line: %q\n", lineText)

		// Updated regex pattern to handle strings with spaces and escaped quotes
		re := regexp.MustCompile(`"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|\S+`)
		tokens := re.FindAllString(lineText, -1)
		splitStringInPlace(&tokens)
		tokens = append(tokens, "\n")

		fmt.Println("Tokens:", tokens)

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

		switch {
		case token == "write":
			endLineIndex := findEndLine(tokens[i:]) + i

			writeNode := parseWrite(tokens[i:endLineIndex], line, root)

			body = append(body, &writeNode)

			i = endLineIndex

		case token == "func":

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

			body = append(body, funcNode)

			tokensTraversed := closingBraceIndex - i

			i += tokensTraversed
		case token == "int" || token == "string" || token == "char" || token == "float" || token == "bool":

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
		case token == "if":
			// Parse the 'if' statement
			endIndex := parseIfStatement(tokens[i:], line, root, &body)
			i += endIndex

		case token == "[":
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

		case token == "return":
			endLineIndex := findEndLine(tokens[i:]) + i

			if len(tokens[i:endLineIndex]) > 2 {
				fmt.Println("Only one return argument allowed. Error: line " + strconv.Itoa(line))
				os.Exit(3)
			}

			newNode := parseReturn(tokens[i:endLineIndex], line, root)

			for _, declarations := range root.Declared {
				if declarations.Value == newNode.Value {
					newNode.DType = declarations.DType
					break
				}
			}

			checkFunctionReturnType(root, newNode)

			root.Returns = append(root.Returns, newNode)
			root.Body = append(root.Body, newNode)

			i = endLineIndex
		case token == "\n":
			line++
			i++
		case token == ";":
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
		fmt.Println("Unexpected return in function "+root.Value+" which is void of returns! Line:", line)
		os.Exit(3)
	} else if returnNode.DType != root.DType {
		fmt.Println("Returned variable "+returnNode.Value+" in "+root.Value+" does not match function return type! Line:", line)
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
		DType: strings.ToUpper(tokens[0]),
		Value: tokens[1],
	}

	if !isIdentifier(tokens[1]) {
		fmt.Println("Expected variable name declaration got " + tokens[1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}
	return &newNode
}

// Parse return declarations
func parseReturn(tokens []string, lineNumber int, root *Node) *Node {

	returnNode := parseGeneric(tokens[1:], lineNumber, root)

	newNode := Node{
		Type:  "RETURN",
		Value: "return",
	}

	newNode.Body = append(newNode.Body, returnNode)
	newNode.DType = returnNode.DType

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

func parseArrayIndex(tokens []string, lineNumber int, root *Node) Node {
	arrayNode := Node{}

	arrayNode.Type = "ARRAY_INDEX"

	indexStart := slices.Index(tokens, "[")

	arrayNode.Value = strings.Join(tokens[:indexStart], "")

	indexTokens := tokens[indexStart+1 : len(tokens)-1]

	arrayNode.Body = append(arrayNode.Body, parseGeneric(indexTokens, lineNumber, root))

	arrayNode.DType = returnType(root, &arrayNode)[2:]

	return arrayNode
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
		fmt.Println("Unrecognized function \""+newNode.Value+"\" Line:", line)
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
		DType: strings.ToUpper(strings.Join(tokens[0:3], "")),
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
	pattern := regexp.MustCompile(`"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|\b\d+\.\d+\b|>=|<=|==|!=|[a-zA-Z0-9]+|[(){}[\];,+\-*/%=<>!]`)
	var result []string

	for _, str := range *arr {
		// If the string is a quoted string, don't split it
		if len(str) >= 2 && (str[0] == '"' && str[len(str)-1] == '"') ||
			(str[0] == '\'' && str[len(str)-1] == '\'') {
			result = append(result, str)
			continue
		}
		// Find all matches based on the regex pattern
		matches := pattern.FindAllString(str, -1)
		result = append(result, matches...)
	}

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
		fmt.Println("Character not found in expression. Line:", line)
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
		fmt.Println("Did not recognize direction: "+direction+" Line:", line)
		os.Exit(3)
	}

	return tokens
}

func operatorTypeComparison(node *Node) {
	if node.Left.DType != node.Right.DType {
		fmt.Println("Type mismatch between " + node.Left.Value + " (" + node.Left.DType + ") " + "and " + node.Right.Value + " (" + node.Right.DType + ") " + " Error: line " + strconv.Itoa(line))
		os.Exit(3)
	}

	// For comparison operators, set DType to BOOL
	comparisonOperators := map[string]bool{
		"GREATER_THAN":  true,
		"LESS_THAN":     true,
		"GREATER_EQUAL": true,
		"LESS_EQUAL":    true,
		"EQUAL_TO":      true,
		"NOT_EQUAL":     true,
	}

	if comparisonOperators[node.Type] {
		node.DType = "BOOL"
	} else {
		node.DType = node.Left.DType
	}
}

func parseGeneric(tokens []string, lineNumber int, root *Node) *Node {

	var newNode Node

	dataType := detectType(strings.Join(tokens, ""))

	if dataType != "unknown" && dataType != "" {
		// Handle literals
		newNode = Node{
			Type:  dataType,
			DType: dataType,
			Value: strings.Join(tokens, ""),
		}
	} else {
		if slices.Contains(tokens, "=") && !slices.Contains(tokens, "==") {
			// Handle assignment '=' but not '=='
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

		} else if slices.Contains(tokens, "==") {
			newNode = Node{
				Type:  "EQUAL_TO",
				DType: "BOOL",
				Value: "==",
				Left:  parseGeneric(bisect(tokens, "==", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "==", "right"), lineNumber, root),
			}
			operatorTypeComparison(&newNode)
		} else if slices.Contains(tokens, "!=") {
			newNode = Node{
				Type:  "NOT_EQUAL",
				DType: "BOOL",
				Value: "!=",
				Left:  parseGeneric(bisect(tokens, "!=", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "!=", "right"), lineNumber, root),
			}
			operatorTypeComparison(&newNode)
		} else if slices.Contains(tokens, ">=") {
			newNode = Node{
				Type:  "GREATER_EQUAL",
				DType: "BOOL",
				Value: ">=",
				Left:  parseGeneric(bisect(tokens, ">=", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, ">=", "right"), lineNumber, root),
			}
			operatorTypeComparison(&newNode)
		} else if slices.Contains(tokens, "<=") {
			newNode = Node{
				Type:  "LESS_EQUAL",
				DType: "BOOL",
				Value: "<=",
				Left:  parseGeneric(bisect(tokens, "<=", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "<=", "right"), lineNumber, root),
			}
			operatorTypeComparison(&newNode)
		} else if slices.Contains(tokens, ">") {
			newNode = Node{
				Type:  "GREATER_THAN",
				DType: "BOOL",
				Value: ">",
				Left:  parseGeneric(bisect(tokens, ">", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, ">", "right"), lineNumber, root),
			}
			operatorTypeComparison(&newNode)
		} else if slices.Contains(tokens, "<") {
			newNode = Node{
				Type:  "LESS_THAN",
				DType: "BOOL",
				Value: "<",
				Left:  parseGeneric(bisect(tokens, "<", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "<", "right"), lineNumber, root),
			}
			operatorTypeComparison(&newNode)
		} else if slices.Contains(tokens, "*") {
			// ... existing code for multiplication ...
		} else if slices.Contains(tokens, "/") {
			// ... existing code for division ...
		} else if slices.Contains(tokens, "+") {
			// ... existing code for addition ...
		} else if slices.Contains(tokens, "-") {
			// ... existing code for subtraction ...
		} else if slices.Contains(tokens, "(") {
			newNode = parseFunctionCall(tokens, lineNumber, root)
		} else if slices.Contains(tokens, "{") {
			newNode = parseArray(tokens, lineNumber, root)
		} else if slices.Contains(tokens, "[") {
			newNode = parseArrayIndex(tokens, lineNumber, root)
		} else if isIdentifier(tokens[0]) {
			// Handle identifiers
			newNode = Node{
				Type:  "IDENTIFIER",
				Value: tokens[0],
			}
			returnType := returnType(root, &newNode)
			newNode.DType = returnType
			isValid := symbolMan(root, &newNode)
			if !isValid {
				fmt.Println("Previously undeclared variable assignment: "+tokens[0]+" on line", line)
				os.Exit(3)
			}
		} else {
			fmt.Println("Unrecognized character \"" + tokens[0] + "\" on line " + strconv.Itoa(line))
			os.Exit(3)
		}
	}

	return &newNode
}

// detectType analyzes a slice of string tokens and returns a map of the token and its detected type
func detectType(tokens string) string {
	var types string

	switch {
	// Check for booleans
	case tokens == "True" || tokens == "False":
		types = "BOOL"

	// Check for floats
	case isFloat(tokens):
		types = "FLOAT"

	// Check for integers
	case isInt(tokens):
		types = "INT"

	// Check for strings
	case isString(tokens):
		types = "STRING"

	// Check for chars
	case isChar(tokens):
		types = "CHAR"

	// If none matched, mark as "unknown"
	default:
		types = "unknown"
	}

	return types
}

// Helper function to check if token is an integer
func isInt(token string) bool {
	_, err := strconv.Atoi(token)
	return err == nil
}

// Helper function to check if token is a float
func isFloat(token string) bool {
	_, err := strconv.ParseFloat(token, 64)
	return err == nil && containsDecimal(token)
}

// Helper function to check if token is a string (starts and ends with double quotes)
func isString(token string) bool {
	return len(token) >= 2 && token[0] == '"' && token[len(token)-1] == '"'
}

// Helper function to check if token is a character (single character surrounded by single quotes)
func isChar(token string) bool {
	return len(token) == 3 && token[0] == '\'' && token[2] == '\''
}

// Helper function to check if the float contains a decimal point
func containsDecimal(token string) bool {
	for _, char := range token {
		if char == '.' {
			return true
		}
	}
	return false
}
func parseIfStatement(tokens []string, lineNumber int, root *Node, body *[]*Node) int {
	i := 1 // Skip the 'if' token

	// Ensure the next token is '('
	if tokens[i] != "(" {
		fmt.Println("Expected '(' after 'if' on line", lineNumber)
		os.Exit(3)
	}
	i++

	// Find the matching ')'
	conditionEndIndex := findMatchingToken(tokens, i-1, "(", ")")
	if conditionEndIndex == -1 {
		fmt.Println("Expected ')' to close 'if' condition on line", lineNumber)
		os.Exit(3)
	}
	conditionTokens := tokens[i:conditionEndIndex]
	conditionNode := parseGeneric(conditionTokens, lineNumber, root)
	i = conditionEndIndex + 1

	// Ensure the next token is '{'
	if tokens[i] != "{" {
		fmt.Println("Expected '{' after 'if' condition on line", lineNumber)
		os.Exit(3)
	}
	i++

	// Find the matching '}'
	bodyEndIndex := findMatchingToken(tokens, i-1, "{", "}")
	if bodyEndIndex == -1 {
		fmt.Println("Expected '}' to close 'if' body on line", lineNumber)
		os.Exit(3)
	}
	bodyTokens := tokens[i:bodyEndIndex]
	ifBodyNode := parse(bodyTokens, root)
	i = bodyEndIndex + 1

	// Create the 'if' node
	ifNode := Node{
		Type: "IF_STATEMENT",
		Left: conditionNode,
		Body: ifBodyNode.Body,
	}

	// Check for 'else' clause
	if i < len(tokens) && tokens[i] == "else" {
		i++

		// Ensure the next token is '{'
		if tokens[i] != "{" {
			fmt.Println("Expected '{' after 'else' on line", lineNumber)
			os.Exit(3)
		}
		i++

		// Find the matching '}'
		elseBodyEndIndex := findMatchingToken(tokens, i-1, "{", "}")
		if elseBodyEndIndex == -1 {
			fmt.Println("Expected '}' to close 'else' body on line", lineNumber)
			os.Exit(3)
		}
		elseBodyTokens := tokens[i:elseBodyEndIndex]
		elseBodyNode := parse(elseBodyTokens, root)
		i = elseBodyEndIndex + 1

		elseNode := Node{
			Type: "ELSE_STATEMENT",
			Body: elseBodyNode.Body,
		}

		ifNode.Right = &elseNode
	}

	// Append the 'if' node to the current body
	*body = append(*body, &ifNode)

	// Return the number of tokens consumed
	return i
}

func findMatchingToken(tokens []string, startIndex int, openToken string, closeToken string) int {
	depth := 0
	for i := startIndex; i < len(tokens); i++ {
		if tokens[i] == openToken {
			depth++
		} else if tokens[i] == closeToken {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	// If we reach here, no matching closing token found
	return -1
}
