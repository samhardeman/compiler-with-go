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
	Params        []string
	Returns       []string
	Body          []string
	ArrayType     string
	ArrayElements []string
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

	entireCode := []string{}

	// Process each line and generate TAC
	for i := 0; i < len(lines); i++ {
		lexerLines := lexer(lines[i])
		for _, line := range lexerLines {
			entireCode = append(entireCode, line.Type)
		}
	}

	for _, declaration := range entireCode {
		fmt.Println(declaration)
	}

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

var tempCounter int

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

// Token represents a lexical token.
type Token struct {
	Type    string
	Literal string
}

func splitStringInPlace(arr *[]string) {
	// Define a regex pattern to match sequences of letters, digits, or special characters
	pattern := regexp.MustCompile(`[a-zA-Z0-9]+|[(){}[\];,]`)

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

// Lexer (simple) to split input into tokens.
func lexer(input string) []Token {
	var tokens []Token
	words := strings.Fields(input)

	// Call the function to modify the array in place
	splitStringInPlace(&words)

	for _, word := range words {
		switch word {
		case "func":
			tokens = append(tokens, Token{Type: "FUNC", Literal: word})
		case "+":
			tokens = append(tokens, Token{Type: "PLUS", Literal: word})
		case "-":
			tokens = append(tokens, Token{Type: "SUB", Literal: word})
		case "*":
			tokens = append(tokens, Token{Type: "MULT", Literal: word})
		case "/":
			tokens = append(tokens, Token{Type: "DIV", Literal: word})
		case ";":
			tokens = append(tokens, Token{Type: "SEMI", Literal: word})
		case "=":
			tokens = append(tokens, Token{Type: "ASSIGN", Literal: word})
		case "{":
			tokens = append(tokens, Token{Type: "LBRACE", Literal: word})
		case "}":
			tokens = append(tokens, Token{Type: "RBRACE", Literal: word})
		case "(":
			tokens = append(tokens, Token{Type: "LPAREN", Literal: word})
		case ")":
			tokens = append(tokens, Token{Type: "RPAREN", Literal: word})
		case "[":
			tokens = append(tokens, Token{Type: "LBRACKET", Literal: word})
		case "]":
			tokens = append(tokens, Token{Type: "RBRACKET", Literal: word})
		case ",":
			tokens = append(tokens, Token{Type: "COMMA", Literal: word})
		default:
			// Assume identifier, type, or literal value for simplicity
			if strings.Contains(word, "int") || strings.Contains(word, "string") || strings.Contains(word, "[]string") {
				tokens = append(tokens, Token{Type: "TYPE", Literal: word})
			} else if strings.HasPrefix(word, "\"") && strings.HasSuffix(word, "\"") {
				// Detect string literals
				tokens = append(tokens, Token{Type: "STRING_LITERAL", Literal: word})
			} else {
				tokens = append(tokens, Token{Type: "IDENTIFIER", Literal: word})
			}
		}
	}
	return tokens
}
