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
		// Generate TAC for assignment
		left := node.Left.Value
		right := node.Right.Value
		line := fmt.Sprintf("%s = %s\n", left, right)
		writer.WriteString(line)
	case "FUNCTION_DECL":
		// Handle function declaration
		writer.WriteString(fmt.Sprintf("func %s:\n", node.Value))
		for _, stmt := range node.Body {
			generateOptimizedTAC(stmt, writer)
		}
		writer.WriteString("end func\n")
	case "FUNCTION_CALL":
		// Handle function call with arguments
		args := []string{}
		for _, param := range node.Params {
			arg := generateOptimizedExpressionTAC(param, writer)
			args = append(args, arg)
		}
		line := fmt.Sprintf("call %s %s\n", node.Value, strings.Join(args, ", "))
		writer.WriteString(line)
	case "RETURN":
		// Handle return statement
		expr := generateOptimizedExpressionTAC(node.Right, writer)
		writer.WriteString(fmt.Sprintf("return %s\n", expr))
	case "IF_STATEMENT":
		// Generate TAC for the condition
		conditionVar := generateOptimizedExpressionTAC(node.Left, writer)

		// Create labels for branching
		trueLabel := getNewLabel()
		endLabel := getNewLabel()

		// Write TAC for the conditional jump
		writer.WriteString(fmt.Sprintf("if %s goto %s\n", conditionVar, trueLabel))

		// Generate TAC for the 'else' body if it exists
		if node.Right != nil && node.Right.Type == "ELSE_STATEMENT" {
			// Generate TAC for 'else' body
			for _, stmt := range node.Right.Body {
				generateOptimizedTAC(stmt, writer)
			}
		}

		// Jump to end after 'else' body
		writer.WriteString(fmt.Sprintf("goto %s\n", endLabel))

		// Label for the 'if' body
		writer.WriteString(fmt.Sprintf("%s:\n", trueLabel))

		// Generate TAC for 'if' body
		for _, stmt := range node.Body {
			generateOptimizedTAC(stmt, writer)
		}

		// End label
		writer.WriteString(fmt.Sprintf("%s:\n", endLabel))
	default:
		// Handle other node types if necessary
	}

	// Recursively generate TAC for child nodes
	if node.Left != nil {
		generateOptimizedTAC(node.Left, writer)
	}
	if node.Right != nil {
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
	case "NUMBER", "INT", "FLOAT", "CHAR", "STRING", "BOOL":
		return node.Value
	case "IDENTIFIER":
		return node.Value
	case "ADD", "SUB", "MULT", "DIV":
		left := generateOptimizedExpressionTAC(node.Left, writer)
		right := generateOptimizedExpressionTAC(node.Right, writer)
		tempVar := getOptimizedTempVar()
		line := fmt.Sprintf("%s = %s %s %s\n", tempVar, left, getOperatorSymbol(node.Type), right)
		writer.WriteString(line)
		return tempVar
	case "GREATER_THAN", "LESS_THAN", "GREATER_EQUAL", "LESS_EQUAL", "EQUAL_TO", "NOT_EQUAL":
		left := generateOptimizedExpressionTAC(node.Left, writer)
		right := generateOptimizedExpressionTAC(node.Right, writer)
		tempVar := getOptimizedTempVar()
		operator := getOperatorSymbol(node.Type)
		// Generate TAC for the comparison operation
		line := fmt.Sprintf("%s = %s %s %s\n", tempVar, left, operator, right)
		writer.WriteString(line)
		return tempVar
	default:
		fmt.Printf("Unhandled node type in expression: %s\n", node.Type)
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
	case "GREATER_THAN":
		return ">"
	case "LESS_THAN":
		return "<"
	case "GREATER_EQUAL":
		return ">="
	case "LESS_EQUAL":
		return "<="
	case "EQUAL_TO":
		return "=="
	case "NOT_EQUAL":
		return "!="
	default:
		return ""
	}
}

var labelCounter int

func getNewLabel() string {
	label := fmt.Sprintf("L%d", labelCounter)
	labelCounter++
	return label
}
