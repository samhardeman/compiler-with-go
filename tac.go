package main

import (
	"bufio"
	"fmt"
	"os"
)

// Symbol table to store values and their corresponding tempVars
var symbolTable = make(map[string]string)

func generateOptimizedTAC(node *Node, writer *bufio.Writer) {
	if node == nil {
		return
	}

	switch node.Type {
	case "IF_STATEMENT":
		labelCounter := optimizedTempVarCounter
		optimizedTempVarCounter++

		if node.Left.Type == "LESS_THAN" || node.Left.Type == "GREATER_THAN" {
			leftTemp := handleValue(node.Left.Left, writer)
			rightTemp := handleValue(node.Left.Right, writer)

			writer.WriteString(fmt.Sprintf("if_start_%d:\n", labelCounter))

			if node.Left.Type == "LESS_THAN" {
				writer.WriteString(fmt.Sprintf("blt %s %s true_%d\n", leftTemp, rightTemp, labelCounter))
			} else if node.Left.Type == "GREATER_THAN" {
				writer.WriteString(fmt.Sprintf("bgt %s %s true_%d\n", leftTemp, rightTemp, labelCounter))
			}
			writer.WriteString(fmt.Sprintf("jump false_%d\n", labelCounter))

			// True branch
			writer.WriteString(fmt.Sprintf("true_%d:\n", labelCounter))
			for _, stmt := range node.Body {
				generateOptimizedTAC(stmt, writer)
			}
			writer.WriteString(fmt.Sprintf("jump end_if_%d\n", labelCounter))

			// False branch (else)
			writer.WriteString(fmt.Sprintf("false_%d:\n", labelCounter))
			if node.Right != nil && node.Right.Type == "ELSE_STATEMENT" {
				// Directly process the else block statements
				for _, stmt := range node.Right.Body {
					generateOptimizedTAC(stmt, writer)
				}
			}
			writer.WriteString(fmt.Sprintf("end_if_%d:\n", labelCounter))
		}

	case "FUNCTION_CALL":
		if node.Value == "write" {
			for _, param := range node.Params {
				arg := handleValue(param, writer)
				writer.WriteString(fmt.Sprintf("call write %s\n", arg))
			}
		}

	case "ASSIGN":
		leftTemp := handleValue(node.Left, writer)
		var rightValue string
		if node.Right.Type == "INT" || node.Right.Type == "STRING" ||
			node.Right.Type == "FLOAT" || node.Right.Type == "BOOL" {
			rightValue = node.Right.Value
		} else {
			rightValue = handleValue(node.Right, writer)
		}
		writer.WriteString(fmt.Sprintf("%s = %s\n", leftTemp, rightValue))
	}
}

// Also update handleValue to better handle identifier nodes
func handleValue(node *Node, writer *bufio.Writer) string {
	if node == nil {
		return ""
	}

	switch node.Type {
	case "IDENTIFIER":
		if tempVar, exists := symbolTable[node.Value]; exists {
			return tempVar
		}
		tempVar := getOptimizedTempVar(node.DType)
		symbolTable[node.Value] = tempVar
		return tempVar

	case "INT", "STRING", "FLOAT", "BOOL":
		tempVar := getOptimizedTempVar(node.Type)
		writer.WriteString(fmt.Sprintf("%s = %s\n", tempVar, node.Value))
		return tempVar

	default:
		return node.Value
	}
}
func optimize_tac(root *Node, filename string) {
	//debugPrintAST(root, "")

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating optimized TAC file: %v\n", err)
		os.Exit(3)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Reset the symbol table and counter
	symbolTable = make(map[string]string)
	optimizedTempVarCounter = 0

	// Generate TAC for each node in the root's body
	for _, node := range root.Body {
		generateOptimizedTAC(node, writer)
	}

	writer.Flush()
}

var optimizedTempVarCounter int

// Function to generate a tempVar with type
func getOptimizedTempVar(varType string) string {
	optimizedTempVarCounter++
	tempVar := fmt.Sprintf("opt_t%d_%s", optimizedTempVarCounter, varType)
	return tempVar
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
func debugPrintAST(node *Node, prefix string) {
	fmt.Println("Debug: AST before TAC generation:")
	if node == nil {
		return
	}
	fmt.Printf("%sType: %s, Value: %s\n", prefix, node.Type, node.Value)
	if node.Left != nil {
		fmt.Printf("%sLeft:\n", prefix)
		debugPrintAST(node.Left, prefix+"  ")
	}
	if node.Right != nil {
		fmt.Printf("%sRight:\n", prefix)
		debugPrintAST(node.Right, prefix+"  ")
	}
	if len(node.Body) > 0 {
		fmt.Printf("%sBody (%d items):\n", prefix, len(node.Body))
		for _, child := range node.Body {
			debugPrintAST(child, prefix+"  ")
		}
	}
}
