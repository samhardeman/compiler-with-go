package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Symbol table to store values and their corresponding tempVars
var symbolTable = make(map[string]string)

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

	// Output the symbol table for debugging purposes
	fmt.Println("Symbol Table:")
	for value, tempVar := range symbolTable {
		fmt.Printf("%s -> %s\n", value, tempVar)
	}
}

func generateOptimizedTAC(node *Node, writer *bufio.Writer) {
	if node == nil {
		return
	}

	switch node.Type {
	case "ASSIGN":
		// Generate TAC for assignment
		handleValue(node.Right, writer)
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
			arg := handleValue(param, writer)
			args = append(args, arg)
		}
		line := fmt.Sprintf("call %s %s\n", node.Value, strings.Join(args, ", "))
		writer.WriteString(line)
	case "RETURN":
		// Handle return statement
		expr := handleValue(node.Right, writer)
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

// Function to handle values (check symbol table or create a new tempVar)
func handleValue(node *Node, writer *bufio.Writer) string {
	if node == nil {
		return ""
	}

	value := node.Value
	nodeType := node.Type

	// Check if the value already has an associated tempVar
	if existingTempVar, exists := symbolTable[value]; exists {
		// Value already has a tempVar, return it
		return existingTempVar
	}

	// If not, create a new tempVar
	tempVar := getOptimizedTempVar(nodeType)
	line := fmt.Sprintf("%s = %s\n", tempVar, value)
	writer.WriteString(line)

	// Store the new tempVar in the symbol table
	symbolTable[value] = tempVar

	return tempVar
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
