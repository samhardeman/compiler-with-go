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
	"time"
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
	Scope    string
}

type Symbol struct {
	dtype string
	value string
}

var line int

func main() {
	startTime := time.Now()

	debug := false // set to true to print trees before and after optimization

	line++
	root := Node{}
	var inputFile string = getFlags()
	code := readLines(inputFile)

	startParsing := time.Now()
	newRoot := parse(code, &root)
	fmt.Printf("Parsing took %v\n", time.Since(startParsing))
	if debug {
		printAST(newRoot)
	}
	startOptimization := time.Now()
	optimizedAST := optimizer(newRoot)
	finalRound(&optimizedAST)
	fmt.Printf("Optimization took %v\n", time.Since(startOptimization))
	if debug {
		printAST(&optimizedAST)
	}
	startTacGeneration := time.Now()
	optimize_tac(&optimizedAST, "output.tac")
	fmt.Printf("TAC Generation took %v\n", time.Since(startTacGeneration))

	startMipsCompilation := time.Now()
	tac2Mips("output.tac")
	fmt.Printf("MIPS Compilation took %v\n", time.Since(startMipsCompilation))

	totalTime := time.Since(startTime)

	fmt.Println("Finished! 【=◈ ︿◈ =】Total:", totalTime)
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
		re := regexp.MustCompile(`"(.*?)"|\S+`)
		tokens := re.FindAllString(scanner.Text(), -1)
		splitStringInPlace(&tokens)
		removeComments(&tokens)
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

		switch {
		case token == "write":
			endLineIndex := findEndLine(tokens[i:]) + i

			writeNode := parseWrite(tokens[i:endLineIndex], line, root)

			body = append(body, &writeNode)

			i = endLineIndex

		case token == "func":

			endFunctionDeclIndex := slices.Index(tokens[i:], "{") + i
			closingBraceIndex := findMatchingBrace(tokens[endFunctionDeclIndex:], 0) + endFunctionDeclIndex
			if closingBraceIndex == -1 {
				fmt.Println("No closing brace found! Line:", line)
				os.Exit(3)
			}

			funcNode := parseFunc(tokens[i:endFunctionDeclIndex+1], line)

			funcNode.Declared = append(funcNode.Declared, passGlobals(root)...)

			isValid := symbolMan(root, funcNode)

			parse(tokens[endFunctionDeclIndex+1:closingBraceIndex], funcNode)

			if !isValid {
				fmt.Println(funcNode.Value + " has already been declared! Error line: " + strconv.Itoa(line))
				os.Exit(3)
			}

			root.Declared = append(root.Declared, symbolNode(funcNode.Value, funcNode.Type, funcNode.DType, "LOCAL"))

			body = append(body, funcNode)

			i = closingBraceIndex + 1

		case token == "int" || token == "string" || token == "char" || token == "float" || token == "bool":

			endLineIndex := findEndLine(tokens[i:]) + i
			declLine := tokens[i:endLineIndex]
			declNode := parseDecl(declLine, line)

			// check if valid
			isValid := symbolMan(root, declNode)
			if !isValid {
				fmt.Println(declNode.Value + " has already been declared!")
				os.Exit(3)
			}

			declNode.Scope = "LOCAL"
			root.Declared = append(root.Declared, symbolNode(declNode.Value, declNode.Type, declNode.DType, declNode.Scope))

			if len(declLine) > 2 {
				if declLine[2] == "=" {
					newNode := parseGeneric(declLine[1:], line, root)
					body = append(body, newNode)
				}
			}

			i = endLineIndex

		case token == "global":
			endLineIndex := findEndLine(tokens[i:]) + i

			// skip the global token, and parse like a regular data type
			// there really should be a check here to make sure after global is a int/char/string/etc
			i++
			declLine := tokens[i:endLineIndex]
			declNode := parseDecl(declLine, line)

			// check if valid
			isValid := symbolMan(root, declNode)
			if !isValid {
				fmt.Println(declNode.Value + " has already been declared!")
				os.Exit(3)
			}

			declNode.Scope = "GLOBAL"

			root.Declared = append(root.Declared, symbolNode(declNode.Value, declNode.Type, declNode.DType, declNode.Scope))

			if len(declLine) > 2 {
				if declLine[2] == "=" {
					newNode := parseGeneric(declLine[1:], line, root)
					body = append(body, newNode)
				}
			}

			i = endLineIndex

		case token == "if":
			// Pass the entire slice from 'if' onward to parseIfStatement
			ifNode, tokensConsumed := parseIfStatement(tokens[i:], line, root)
			body = append(body, &ifNode)
			i += tokensConsumed

		case token == "for":
			endForLoopDeclIndex := slices.Index(tokens[i:], "{") + i

			closingBraceIndex := findMatchingBrace(tokens[endForLoopDeclIndex:], 0) + endForLoopDeclIndex
			if closingBraceIndex == -1 {
				fmt.Println("No closing brace found!")
				os.Exit(3)
			}

			forLoopNode := parseForLoop(tokens[i:endForLoopDeclIndex], root, line)

			initNode := forLoopNode.Body[0]

			// what are you doing, stepnode?
			stepNode := forLoopNode.Body[1]

			parse(tokens[endForLoopDeclIndex+1:closingBraceIndex], forLoopNode)

			forLoopCore := forLoopIf(forLoopNode)

			forLoopNode.Body = nil

			forLoopNode.Body = append(forLoopNode.Body, initNode)
			forLoopNode.Body = append(forLoopNode.Body, forLoopCore)
			forLoopNode.Body = append(forLoopNode.Body, stepNode)

			body = append(body, forLoopNode)

			i = closingBraceIndex + 2

		case token == "while":
			endWhileDeclIndex := slices.Index(tokens[i:], "{") + i

			closingBraceIndex := findMatchingBrace(tokens[endWhileDeclIndex:], 0) + endWhileDeclIndex
			if closingBraceIndex == -1 {
				fmt.Println("No closing brace found!")
				os.Exit(3)
			}

			whileLoop := parseWhile(tokens[i:endWhileDeclIndex], root, line)

			parse(tokens[endWhileDeclIndex+1:closingBraceIndex], whileLoop)

			whileLoopCore := forLoopIf(whileLoop)

			whileLoop.Body = nil

			whileLoop.Body = append(whileLoop.Body, whileLoopCore)

			body = append(body, whileLoop)

			i = closingBraceIndex + 2

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

			root.Declared = append(root.Declared, symbolNode(arrayDecl.Value, arrayDecl.Type, arrayDecl.DType, "LOCAL"))

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

			i = endLineIndex + 1

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

func symbolNode(name string, decltype string, dtype string, scope string) *Node {
	newNode := Node{
		Type:  decltype,
		DType: dtype,
		Value: name,
		Scope: scope,
	}

	return &newNode
}

func parseForLoop(tokens []string, root *Node, lineNumber int) *Node {
	var newNode Node
	openParen := 0

	newNode.Type = "FOR_LOOP"
	newNode.DType = "FOR_LOOP"
	newNode.Value = "for"

	// append declared variables to the for loop so that it has access to them
	newNode.Declared = append(newNode.Declared, root.Declared...)

	// Expect first open parentheses
	if tokens[1] != "(" {
		fmt.Println("Expected \"(\" got " + tokens[1] + " on line " + strconv.Itoa(lineNumber))
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

	firstStatementEndIndex := slices.Index(tokens, ";") + 1
	secondStatementEndIndex := slices.Index(tokens[firstStatementEndIndex:], ";") + 1

	parse(tokens[2:firstStatementEndIndex], &newNode)
	condition := parseGeneric(tokens[firstStatementEndIndex:firstStatementEndIndex+secondStatementEndIndex-1], line, &newNode)
	step := parseGeneric(tokens[firstStatementEndIndex+secondStatementEndIndex:len(tokens)-1], line, &newNode)

	newNode.Params = append(newNode.Params, condition)
	newNode.Body = append(newNode.Body, step)

	return &newNode
}

func parseWhile(tokens []string, root *Node, lineNumber int) *Node {
	var newNode Node
	openParen := 0

	newNode.Type = "WHILE_LOOP"
	newNode.DType = "WHILE_LOOP"
	newNode.Value = "while"

	// append declared variables to the for loop so that it has access to them
	newNode.Declared = append(newNode.Declared, root.Declared...)

	// Expect first open parentheses
	if tokens[1] != "(" {
		fmt.Println("Expected \"(\" got " + tokens[1] + " on line " + strconv.Itoa(lineNumber))
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

	condition := parseGeneric(tokens[2:closeParenIndex], line, &newNode)

	newNode.Params = append(newNode.Params, condition)

	return &newNode
}

func forLoopIf(node *Node) *Node {
	ifNode := Node{
		Type:  "IF_STATEMENT",
		Value: "if",
		Body:  []*Node{},
	}

	ifNode.Body = node.Body
	ifNode.Left = node.Params[0]

	return &ifNode
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
	// Special case - skip if it's an else statement
	if tokens[0] == "else" {
		return Node{} // Return empty node which will be handled by the if statement parsing
	}

	newNode := Node{
		Type:  "FUNCTION_CALL",
		Value: tokens[0],
	}

	functionDeclared := false

	// Check if this is a built-in function
	if tokens[0] == "write" {
		functionDeclared = true
	} else {
		// Check if the function has been declared
		for _, declared := range root.Declared {
			if declared.Value == newNode.Value {
				newNode.DType = declared.DType
				functionDeclared = true
				break
			}
		}
	}

	if !functionDeclared {
		fmt.Println("Unrecognized function \""+newNode.Value+"\" Line:", line)
		os.Exit(3)
	}

	// Rest of the function remains the same
	if tokens[1] != "(" {
		fmt.Println("Expected \"(\" after function name, got " + tokens[1] + " on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	closeParenIndex := slices.Index(tokens, ")")
	if closeParenIndex == -1 {
		fmt.Println("Expected \")\" to close function call on line " + strconv.Itoa(lineNumber))
		os.Exit(3)
	}

	args := tokens[2:closeParenIndex]

	for i := 0; i < len(args); i += 2 {
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
	// Updated regex pattern:
	// 1. Matches quoted strings: "..." or '...'
	// 2. Matches decimal numbers as a single token (e.g., 123.45)
	// 3. Matches multi-character operators like ==, >=, <=, and //
	// 4. Matches single-character operators, symbols, and identifiers
	pattern := regexp.MustCompile(`"[^"]*"|'[^']*'|\b\d+\.\d+\b|==|>=|<=|//|[a-zA-Z0-9]+|[(){}[\];,+\-*/%=<>!]`)
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

// RemoveComments removes everything from // (inclusive) to \n in the array of strings
func removeComments(arr *[]string) {
	var result []string
	skip := false // Flag to indicate whether we are skipping tokens within a comment

	for _, str := range *arr {
		// Check for the start of a comment
		if str == "//" {
			skip = true // Start skipping until we hit \n
			continue
		}

		// If we encounter a newline, stop skipping and include the newline
		if str == "\n" {
			skip = false
			result = append(result, str)
			continue
		}

		// If we're not in a comment, include the current token
		if !skip {
			result = append(result, str)
		}
	}

	// Replace the original array content with the filtered array
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
	bracketCount := 0

	for i, token := range chunk {
		switch token {
		case "{":
			bracketCount++
		case "}":
			bracketCount--
			if bracketCount == 0 {
				return i
			}
		case "\n":
			if bracketCount == 0 {
				return i
			}
		case ";":
			if bracketCount == 0 {
				return i
			}
		}
	}

	// If we're still in a block, return the length of the chunk
	if bracketCount > 0 {
		return len(chunk)
	}

	return 0
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
}

func parseGeneric(tokens []string, lineNumber int, root *Node) *Node {

	var newNode Node

	dataType := detectType(strings.Join(tokens, ""))

	if dataType != "unknown" && dataType != "" {
		switch dataType {
		case "INT":
			newNode = Node{
				Type:  dataType,
				DType: dataType,
				Value: strings.Join(tokens, ""),
			}
		case "STRING":
			newNode = Node{
				Type:  dataType,
				DType: dataType,
				Value: strings.Join(tokens, ""),
			}
		case "CHAR":
			newNode = Node{
				Type:  dataType,
				DType: dataType,
				Value: strings.Join(tokens, ""),
			}
		case "FLOAT":
			newNode = Node{
				Type:  dataType,
				DType: dataType,
				Value: strings.Join(tokens, ""),
			}
		case "BOOL":
			newNode = Node{
				Type:  dataType,
				DType: dataType,
				Value: strings.Join(tokens, ""),
			}
		}
	} else {
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

		} else if slices.Contains(tokens, ">") {
			newNode = Node{
				Type:  "GREATER_THAN",
				DType: "BOOL",
				Value: ">",
				Left:  parseGeneric(bisect(tokens, ">", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, ">", "right"), lineNumber, root),
			}

			// Check that we're comparing compatible types
			if newNode.Left.DType != newNode.Right.DType {
				fmt.Printf("Cannot compare values of different types: %s and %s on line %d\n",
					newNode.Left.DType, newNode.Right.DType, lineNumber)
				os.Exit(3)
			}

		} else if slices.Contains(tokens, "<") {
			newNode = Node{
				Type:  "LESS_THAN",
				DType: "BOOL",
				Value: "<",
				Left:  parseGeneric(bisect(tokens, "<", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "<", "right"), lineNumber, root),
			}

			if newNode.Left.DType != newNode.Right.DType {
				fmt.Printf("Cannot compare values of different types: %s and %s on line %d\n",
					newNode.Left.DType, newNode.Right.DType, lineNumber)
				os.Exit(3)
			}

		} else if slices.Contains(tokens, "==") {
			newNode = Node{
				Type:  "EQUALS",
				DType: "BOOL",
				Value: "==",
				Left:  parseGeneric(bisect(tokens, "==", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "==", "right"), lineNumber, root),
			}

			if newNode.Left.DType != newNode.Right.DType {
				fmt.Printf("Cannot compare values of different types: %s and %s on line %d\n",
					newNode.Left.DType, newNode.Right.DType, lineNumber)
				os.Exit(3)
			}

		} else if slices.Contains(tokens, "%") {
			newNode = Node{
				Type:  "MODULO",
				DType: "OP",
				Value: "%",
				Left:  parseGeneric(bisect(tokens, "%", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "%", "right"), lineNumber, root),
			}

			newNode.DType = newNode.Left.DType

		} else if slices.Contains(tokens, "*") {
			newNode = Node{
				Type:  "MULT",
				DType: "OP",
				Value: "*",
				Left:  parseGeneric(bisect(tokens, "*", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "*", "right"), lineNumber, root),
			}

			//operatorTypeComparison(&newNode)

			newNode.DType = newNode.Left.DType

		} else if slices.Contains(tokens, "/") {
			newNode = Node{
				Type:  "DIV",
				DType: "OP",
				Value: "/",
				Left:  parseGeneric(bisect(tokens, "/", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "/", "right"), lineNumber, root),
			}

			//operatorTypeComparison(&newNode)

			newNode.DType = newNode.Left.DType

		} else if slices.Contains(tokens, "+") {
			newNode = Node{
				Type:  "ADD",
				DType: "OP",
				Value: "+",
				Left:  parseGeneric(bisect(tokens, "+", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "+", "right"), lineNumber, root),
			}

			//operatorTypeComparison(&newNode)

			newNode.DType = newNode.Left.DType

		} else if slices.Contains(tokens, "-") {
			newNode = Node{
				Type:  "SUB",
				DType: "OP",
				Value: "-",
				Left:  parseGeneric(bisect(tokens, "-", "left"), lineNumber, root),
				Right: parseGeneric(bisect(tokens, "-", "right"), lineNumber, root),
			}

			//operatorTypeComparison(&newNode)

			newNode.DType = newNode.Left.DType

		} else if slices.Contains(tokens, "(") {
			newNode = parseFunctionCall(tokens, line, root)
		} else if slices.Contains(tokens, "{") {
			newNode = parseArray(tokens, line, root)
		} else if slices.Contains(tokens, "[") {
			newNode = parseArrayIndex(tokens, lineNumber, root)
		} else if isIdentifier(tokens[0]) {

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
func findIfBlockEnd(tokens []string) int {
	braceCount := 0
	for i, token := range tokens {
		if token == "{" {
			braceCount++
		} else if token == "}" {
			braceCount--
			if braceCount == 0 {
				return i + 1 // return the position after the closing brace
			}
		}
	}
	return len(tokens) // fallback if braces are malformed
}

// Update parseIfStatement to handle the blocks properly
func parseIfStatement(tokens []string, lineNumber int, root *Node) (Node, int) {
	newNode := Node{
		Type:  "IF_STATEMENT",
		Value: "if",
		Body:  []*Node{},
	}

	openParenIndex := slices.Index(tokens, "(")
	closeParenIndex := slices.Index(tokens, ")")
	if openParenIndex == -1 || closeParenIndex == -1 {
		fmt.Printf("Missing parentheses in if statement on line %d\n", lineNumber)
		os.Exit(3)
	}

	conditionTokens := tokens[openParenIndex+1 : closeParenIndex]
	condition := parseGeneric(conditionTokens, lineNumber, root)
	newNode.Left = condition

	// Find '{' that starts the if block
	ifBlockStart := slices.Index(tokens[closeParenIndex:], "{")
	if ifBlockStart == -1 {
		fmt.Printf("Missing '{' for if block on line %d\n", lineNumber)
		os.Exit(3)
	}
	ifBlockStart += closeParenIndex

	// Find matching '}'
	ifBlockEnd := findMatchingBrace(tokens, ifBlockStart)
	if ifBlockEnd == -1 {
		fmt.Println(tokens)
		fmt.Printf("Missing closing '}' for if block on line %d\n", lineNumber)
		os.Exit(3)
	}

	// Parse if block body
	ifBlockTokens := tokens[ifBlockStart+1 : ifBlockEnd]
	ifBlockNode := parse(ifBlockTokens, root)
	newNode.Body = ifBlockNode.Body

	tokensConsumed := ifBlockEnd + 1

	// Check for else
	if tokensConsumed < len(tokens) && tokens[tokensConsumed] == "else" {
		tokensConsumed++ // skip 'else'
		if tokensConsumed < len(tokens) && tokens[tokensConsumed] == "{" {
			elseStart := tokensConsumed
			elseEnd := findMatchingBrace(tokens, elseStart)
			if elseEnd == -1 {
				fmt.Printf("Missing closing '}' for else block on line %d\n", lineNumber)
				os.Exit(3)
			}

			elseTokens := tokens[elseStart+1 : elseEnd]
			elseBlockNode := parse(elseTokens, root)
			elseNode := Node{
				Type:  "ELSE_STATEMENT",
				Value: "else",
				Body:  elseBlockNode.Body,
			}
			newNode.Right = &elseNode

			tokensConsumed = elseEnd + 1
		} else {
			fmt.Printf("Missing '{' after else on line %d\n", lineNumber)
			os.Exit(3)
		}
	}

	return newNode, tokensConsumed
}

// Helper function to find matching closing brace
func findMatchingBrace(tokens []string, openIndex int) int {
	count := 1
	for i := openIndex + 1; i < len(tokens); i++ {
		if tokens[i] == "{" {
			count++
		} else if tokens[i] == "}" {
			count--
			if count == 0 {
				return i
			}
		}
	}
	return -1
}

func passGlobals(root *Node) []*Node {
	var globals []*Node
	for _, node := range root.Declared {
		if node.Scope == "GLOBAL" {
			globals = append(globals, node)
		}
		if node.Type == "FUNCTION_DECL" {
			globals = append(globals, node)
		}
	}
	return globals
}
