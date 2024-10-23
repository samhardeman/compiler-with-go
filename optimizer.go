package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func optimizer() {

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
