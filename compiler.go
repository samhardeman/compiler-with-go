package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Node struct {
	class string
	dtype string
	value string
	left  *Node
	right *Node
}

type Symbol struct {
	dtype  string
	value  string
	isUsed bool
}

var symbolTable map[string]Symbol

func main() {
	var inputFile string = getFlags()
	lines := readLines(inputFile)

	// Open file for writing TAC
	tacFile, err := os.Create("output.tac")
	if err != nil {
		log.Fatal("Cannot create output file:", err)
	}
	defer tacFile.Close()

	// Initialize the symbol table
	symbolTable = make(map[string]Symbol)

	// Process each line and generate TAC
	for i := 0; i < len(lines); i++ {
		doohickey(lines[i], tacFile)
	}

	// Optimize the TAC file
	optimizeTAC("output.tac", "optimized_output.tac")
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

func readLines(inputFile string) []string {
	lines := []string{}

	// open file
	f, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	// remember to close the file at the end of the program
	defer f.Close()

	// read the file line by line using scanner
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return lines

}

func doohickey(line string, file *os.File) {
	root := []*Node{}
	tokens := strings.Fields(line)

	// Parse tokens into an AST
	root = append(root, parser(tokens))

	// Generate and write TAC to the file
	for _, astRoot := range root {
		generateTAC(astRoot, file)
	}
}

func HelperPreOrder(node *Node, processFunc func(v string)) {
	if node != nil {
		processFunc(node.value)
		fmt.Println("traverseAST: " + node.value)
		HelperPreOrder(node.left, processFunc)
		HelperPreOrder(node.right, processFunc)
	}
}

func traverseAST(root []*Node) []string {
	var res []string
	processFunc := func(v string) {
		res = append(res, v)
	}
	for i := 0; i < len(root); i++ {
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

func parser(tokens []string) *Node {

	var newNode Node

	numbers := strings.Split("1234567890", "")
	letters := strings.Split("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "")

	if slices.Contains(tokens, "int") {
		newNode = Node{
			class: "INITIALIZE",
			dtype: "STRING",
			value: "int",
		}

	} else if slices.Contains(tokens, "=") {
		newNode = Node{
			class: "ASSIGN",
			dtype: "CHAR",
			value: "=",
			left:  parser(bisect(tokens, "=", "left")),
			right: parser(bisect(tokens, "=", "right")),
		}
	} else if slices.Contains(tokens, "*") {
		newNode = Node{
			class: "MULT",
			dtype: "CHAR",
			value: "*",
			left:  parser(bisect(tokens, "*", "left")),
			right: parser(bisect(tokens, "*", "right")),
		}

	} else if slices.Contains(tokens, "/") {
		newNode = Node{
			class: "DIV",
			dtype: "CHAR",
			value: "/",
			left:  parser(bisect(tokens, "/", "left")),
			right: parser(bisect(tokens, "/", "right")),
		}

	} else if slices.Contains(tokens, "+") {
		newNode = Node{
			class: "ADD",
			dtype: "CHAR",
			value: "+",
			left:  parser(bisect(tokens, "+", "left")),
			right: parser(bisect(tokens, "+", "right")),
		}

	} else if slices.Contains(tokens, "-") {
		newNode = Node{
			class: "SUB",
			dtype: "CHAR",
			value: "-",
			left:  parser(bisect(tokens, "-", "left")),
			right: parser(bisect(tokens, "-", "right")),
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

var tempCounter int

// Generates a TAC and writes it to the file
func generateTAC(node *Node, file *os.File) string {
	if node == nil {
		return ""
	}

	// Traverse left and right subtrees first (post-order)
	leftVar := generateTAC(node.left, file)
	rightVar := generateTAC(node.right, file)

	writer := bufio.NewWriter(file)

	switch node.class {
	case "ADD", "SUB", "MULT", "DIV":
		// Generate new temporary variable for the operation result
		temp := newTemp()
		tacLine := fmt.Sprintf("%s = %s %s %s\n", temp, leftVar, node.value, rightVar)
		writer.WriteString(tacLine)
		writer.Flush() // Ensure the content is written immediately
		return temp
	case "ASSIGN":
		// For assignment, simply assign the right expression to the left variable
		tacLine := fmt.Sprintf("%s = %s\n", leftVar, rightVar)
		writer.WriteString(tacLine)
		writer.Flush()

		// Track the symbol in the symbol table for constant propagation
		if node.left.class == "IDENTIFIER" {
			symbolTable[node.left.value] = Symbol{value: rightVar, dtype: node.left.dtype, isUsed: false}
		}
		return leftVar
	case "NUMBER", "IDENTIFIER":
		// Return the value of the leaf nodes (either a number or an identifier)
		return node.value
	case "INITIALIZE":
	default:
		log.Fatal("Unrecognized node type")
	}
	return ""
}

// Function to generate a new temporary variable
func newTemp() string {
	tempCounter++
	return fmt.Sprintf("t%d", tempCounter)
}

// 1. Constant Folding
func constantFolding(lines []string) []string {
	foldedLines := []string{}
	for _, line := range lines {
		// Check for arithmetic operations and try to fold constants
		parts := strings.Fields(line)
		if len(parts) == 5 && isOperator(parts[2]) {
			left, leftErr := strconv.Atoi(parts[1])
			right, rightErr := strconv.Atoi(parts[3])
			if leftErr == nil && rightErr == nil {
				// Perform the operation
				result := performOperation(left, right, parts[2])
				foldedLine := fmt.Sprintf("%s = %d\n", parts[0], result)
				foldedLines = append(foldedLines, foldedLine)
				continue
			}
		}
		foldedLines = append(foldedLines, line)
	}
	return foldedLines
}

func isOperator(op string) bool {
	return op == "+" || op == "-" || op == "*" || op == "/"
}

func performOperation(left, right int, op string) int {
	switch op {
	case "+":
		return left + right
	case "-":
		return left - right
	case "*":
		return left * right
	case "/":
		return left / right
	}
	return 0
}

// 2. Constant Propagation
func constantPropagation(lines []string) []string {
	propagatedLines := []string{}
	constantMap := make(map[string]string)

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 3 && parts[1] == "=" {
			// Check if it's an assignment of a constant
			if _, err := strconv.Atoi(parts[2]); err == nil {
				constantMap[parts[0]] = parts[2]
			}
		}

		// Replace variables with constants
		for i, part := range parts {
			if val, ok := constantMap[part]; ok {
				parts[i] = val
			}
		}

		propagatedLines = append(propagatedLines, strings.Join(parts, " "))
	}
	return propagatedLines
}

// 3. Dead Code Elimination
func deadCodeElimination(lines []string) []string {
	usedVariables := make(map[string]bool)
	optimizedLines := []string{}

	// Track variables that are used
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) > 2 && isOperator(parts[2]) {
			usedVariables[parts[1]] = true
			usedVariables[parts[3]] = true
		}
	}

	// Eliminate dead code
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 3 && parts[1] == "=" {
			if _, used := usedVariables[parts[0]]; used {
				optimizedLines = append(optimizedLines, line)
			}
		} else {
			optimizedLines = append(optimizedLines, line)
		}
	}

	return optimizedLines
}

// Ties all optimizations together
func optimizeTAC(inputFile string, outputFile string) {
	// Read TAC from the file
	lines := readLines(inputFile)

	// Apply constant folding
	lines = constantFolding(lines)

	// Apply constant propagation
	lines = constantPropagation(lines)

	// Apply dead code elimination
	lines = deadCodeElimination(lines)

	// Write the optimized TAC to a new file
	file, err := os.Create(outputFile)
	if err != nil {
		log.Fatal("Cannot create optimized output file:", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		writer.WriteString(line + "\n")
	}
	writer.Flush()
}
