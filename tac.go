package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Function to generate optimized TAC and write to a file
func optimize_tac(root *Node, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating optimized TAC file: %v\n", err)
		os.Exit(3)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	generateOptimizedTAC(root, writer)
	writer.Flush()
}

func generateOptimizedTAC(node *Node, writer *bufio.Writer) {
	if node == nil {
		return
	}

	switch node.Type {
	case "ASSIGN":
		// Existing assignment handling
		left := node.Left.Value
		right := node.Right.Value
		line := fmt.Sprintf("%s = %s\n", left, right)
		writer.WriteString(line)

	case "FUNCTION_DECL":
		// Existing function declaration handling
		writer.WriteString(fmt.Sprintf("func %s:\n", node.Value))
		for _, stmt := range node.Body {
			generateOptimizedTAC(stmt, writer)
		}
		writer.WriteString("end func\n")

	case "FUNCTION_CALL":
		// Existing function call handling
		args := []string{}
		for _, param := range node.Params {
			arg := generateOptimizedExpressionTAC(param, writer)
			args = append(args, arg)
		}
		line := fmt.Sprintf("call %s %s\n", node.Value, strings.Join(args, ", "))
		writer.WriteString(line)

	case "RETURN":
		// Existing return handling
		expr := generateOptimizedExpressionTAC(node.Right, writer)
		writer.WriteString(fmt.Sprintf("return %s\n", expr))

	case "IF_STATEMENT":
		// Handle if statement
		condition := generateOptimizedExpressionTAC(node.Left, writer)
		labelTrue := getLabel()
		labelEnd := getLabel()

		// Write condition evaluation and conditional jump
		line := fmt.Sprintf("if %s goto %s\n", condition, labelTrue)
		writer.WriteString(line)
		// If condition is false, jump to else or end
		if node.Right != nil {
			// Else exists
			writer.WriteString(fmt.Sprintf("goto %s\n", labelEnd))
		} else {
			// No else, jump to end
			writer.WriteString(fmt.Sprintf("goto %s\n", labelEnd))
		}

		// Label for true condition
		writer.WriteString(fmt.Sprintf("%s:\n", labelTrue))
		// Generate TAC for the 'if' body
		for _, stmt := range node.Body {
			generateOptimizedTAC(stmt, writer)
		}
		// After 'if' body, jump to end
		writer.WriteString(fmt.Sprintf("goto %s\n", labelEnd))

		if node.Right != nil {
			// Label for else
			writer.WriteString(fmt.Sprintf("%s:\n", labelEnd))
			elseNode := node.Right
			for _, stmt := range elseNode.Body {
				generateOptimizedTAC(stmt, writer)
			}
			// End label
			writer.WriteString(fmt.Sprintf("%s:\n", labelEnd))
		} else {
			// End label
			writer.WriteString(fmt.Sprintf("%s:\n", labelEnd))
		}

	case "ELSE_STATEMENT":
		// Handled within IF_STATEMENT
		// No action needed here
	default:
		// Handle other node types if necessary
	}

	// Recursively generate TAC for child nodes
	if node.Left != nil && node.Type != "IF_STATEMENT" { // Prevent duplicate handling for IF_STATEMENT
		generateOptimizedTAC(node.Left, writer)
	}
	if node.Right != nil && node.Type != "IF_STATEMENT" { // Prevent duplicate handling for IF_STATEMENT
		generateOptimizedTAC(node.Right, writer)
	}
	for _, child := range node.Body {
		generateOptimizedTAC(child, writer)
	}
}

func generateOptimizedExpressionTAC(node *Node, writer *bufio.Writer) string {
	if node == nil {
		return ""
	}

	switch node.Type {
	case "NUMBER", "IDENTIFIER":
		return node.Value
	case "ADD", "SUB", "MULT", "DIV":
		left := generateOptimizedExpressionTAC(node.Left, writer)
		right := generateOptimizedExpressionTAC(node.Right, writer)
		tempVar := getOptimizedTempVar()
		line := fmt.Sprintf("%s = %s %s %s\n", tempVar, left, getOperatorSymbol(node.Type), right)
		writer.WriteString(line)
		return tempVar
	default:
		return ""
	}
}

var optimizedTempVarCounter int

func getOptimizedTempVar() string {
	optimizedTempVarCounter++
	return fmt.Sprintf("opt_t%d", optimizedTempVarCounter)
}

func getOperatorSymbol(nodeType string) string {
	switch nodeType {
	case "ADD":
		return "+"
	case "SUB":
		return "-"
	case "MULT":
		return "*"
	case "DIV":
		return "/"
	default:
		return ""
	}
}

var labelCounter int

func getLabel() string {
	labelCounter++
	return fmt.Sprintf("L%d", labelCounter)
}
