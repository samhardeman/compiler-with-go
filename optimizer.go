package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func optimizer(root *Node) *Node {
	fmt.Println("Starting optimization...")
	optimizedAST := *root

	// Create a map to keep track of constants for propagation
	constants := make(map[string]string)

	// Perform optimization on the AST
	optimizedAST.Body = optimizeStatements(root.Body, constants)

	fmt.Println("Optimization complete.")
	return &optimizedAST
}

func optimizeStatements(statements []*Node, constants map[string]string) []*Node {
	var optimized []*Node
	for _, stmt := range statements {
		switch stmt.Type {
		case "ASSIGN":
			optimizedStmt := foldAndPropagate(stmt, constants)
			optimized = append(optimized, optimizedStmt)
		case "FUNCTION_DECL":
			// Create a new scope for constants in functions
			funcConstants := make(map[string]string)
			stmt.Body = optimizeStatements(stmt.Body, funcConstants)
			optimized = append(optimized, stmt)
		case "FUNCTION_CALL":
			// Optimize function call arguments
			for i, arg := range stmt.Params {
				stmt.Params[i] = evaluateExpression(arg, constants)
			}
			optimized = append(optimized, stmt)
		default:
			optimized = append(optimized, stmt)
		}
	}
	return optimized
}

func foldAndPropagate(node *Node, constants map[string]string) *Node {
	if node.Type == "ASSIGN" && node.Right != nil {
		// Evaluate the right side of the assignment
		node.Right = evaluateExpression(node.Right, constants)

		// If the right side is a NUMBER, add it to constants for propagation
		if node.Right.Type == "NUMBER" {
			constants[node.Left.Value] = node.Right.Value
		} else {
			// Remove from constants if it's reassigned to a non-constant
			delete(constants, node.Left.Value)
		}
	}
	return node
}

func evaluateExpression(node *Node, constants map[string]string) *Node {
	if node == nil {
		return nil
	}

	switch node.Type {
	case "NUMBER":
		// Return number as is
		return node
	case "IDENTIFIER":
		// Replace identifier with constant value if available
		if val, found := constants[node.Value]; found {
			return &Node{
				Type:  "NUMBER",
				DType: node.DType,
				Value: val,
			}
		}
		return node
	}

	// Recursively evaluate left and right children
	if node.Left != nil {
		node.Left = evaluateExpression(node.Left, constants)
	}
	if node.Right != nil {
		node.Right = evaluateExpression(node.Right, constants)
	}

	// If both children are numbers, perform the operation
	if node.Left != nil && node.Right != nil &&
		node.Left.Type == "NUMBER" && node.Right.Type == "NUMBER" {
		leftVal, _ := strconv.Atoi(node.Left.Value)
		rightVal, _ := strconv.Atoi(node.Right.Value)
		var result int
		switch node.Type {
		case "ADD":
			result = leftVal + rightVal
		case "SUB":
			result = leftVal - rightVal
		case "MULT":
			result = leftVal * rightVal
		case "DIV":
			if rightVal == 0 {
				fmt.Println("Error: Division by zero!")
				os.Exit(3)
			}
			result = leftVal / rightVal
		}
		return &Node{
			Type:  "NUMBER",
			DType: "INT",
			Value: strconv.Itoa(result),
		}
	}

	return node
}

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
		right := generateOptimizedExpressionTAC(node.Right, writer)
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
