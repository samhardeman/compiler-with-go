package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
)

type ValueTable struct {
	Body []*Node
}

var Values ValueTable
var Functions ValueTable

func optimizer(root *Node) Node {
	optimizedAST := Node{
		Type:  root.Type,
		DType: root.DType,
		Value: root.Value,
		Body:  []*Node{},
	}

	for index, statement := range root.Body {
		switch statement.Type {
		case "ASSIGN":
			optimizedNode := fold(root, statement.Right, index)
			statement.Right = optimizedNode
			if optimizedNode != nil {
				if optimizedNode.Value != "{}" {
					optimizedAST.Body = append(optimizedAST.Body, statement)
				}
			}
			updateValueTable(&Values, statement)
		case "FUNCTION_CALL":
			if statement.Value == "write" {
				writeNode := statement
				for paramIndex, param := range writeNode.Params {
					writeNode.Params[paramIndex] = fold(root, param, index)
				}
				optimizedAST.Body = append(optimizedAST.Body, writeNode)
			} else {
				funcNode := getFunction(&Functions, statement.Value)
				params := statement.Params

				if len(funcNode.Params) != len(params) {
					fmt.Println("Optimizer: more parameters than accepted!")
					os.Exit(3)
				}

				var foldedParams []*Node
				for paramIndex, param := range params {
					paramNode := Node{
						DType: "OP",
						Type:  "ASSIGN",
						Value: "=",
						Right: fold(root, param, index),
						Left:  funcNode.Params[paramIndex],
					}
					foldedParams = append(foldedParams, &paramNode)
				}

				foldedFunction := foldFunction(funcNode, foldedParams, index)
				if foldedFunction != nil && foldedFunction.Type == "ASSIGN" {
					fmt.Println("ASSIGN")
					optimizedAST.Body = append(optimizedAST.Body, foldedFunction.Right)
				} else if funcNode != nil {
					optimizedAST.Body = append(optimizedAST.Body, foldedFunction)
				}
			}
		case "IF_STATEMENT":
			optimizedIfNode := optimizeIfStatement(root, statement, index)
			if optimizedIfNode.Left.Value == "FALSE" || optimizedIfNode.Left.Value == "TRUE" {
				for _, nice := range optimizedIfNode.Body {
					optimizedStatement := fold(root, nice, index)
					optimizedAST.Body = append(optimizedAST.Body, optimizedStatement)
				}
			} else {
				optimizedAST.Body = append(optimizedAST.Body, optimizedIfNode)
			}

		case "FOR_LOOP":
			unrolledForLoop := optimizeForLoop(root, statement, index)
			var optimizedForLoop Node
			for _, stmt := range unrolledForLoop.Body {
				optimizedStmt := fold(root, stmt, index)
				if optimizedStmt != nil {
					if optimizedStmt.Value == "IF_STATEMENT" {
						optimizedForLoop.Body = append(optimizedForLoop.Body, optimizedStmt.Body...)
					}
					optimizedForLoop.Body = append(optimizedForLoop.Body, optimizedStmt)
				}
			}
			optimizedAST.Body = append(optimizedAST.Body, optimizedForLoop.Body...)

		case "FUNCTION_DECL":
			addFunction(&Functions, statement)
		}
	}

	return optimizedAST
}

func optimizeIfStatement(root *Node, ifNode *Node, index int) *Node {

	newIfNode := &Node{
		Type:  "IF_STATEMENT",
		Value: "if",
		Body:  []*Node{},
	}

	// Optimize condition
	if ifNode.Left != nil {
		newIfNode.Left = fold(root, ifNode.Left, index)

		if newIfNode.Left == nil {
			newIfNode.Left = ifNode.Left
		}

	}

	condition := newIfNode.Left.Value

	// Recursive folding for both main body and else body
	if condition == "FALSE" && ifNode.Right != nil {
		// Process else branch
		for _, stmt := range ifNode.Right.Body {
			optimizedStmt := fold(root, stmt, index)
			if optimizedStmt != nil {
				// Recursively handle nested if statements
				if optimizedStmt.Type == "IF_STATEMENT" {
					recursiveOptimizedStmt := optimizeIfStatement(root, optimizedStmt, index)
					newIfNode.Body = append(newIfNode.Body, recursiveOptimizedStmt.Body...)
				} else {
					newIfNode.Body = append(newIfNode.Body, optimizedStmt)
				}
			}
		}
	} else if condition == "TRUE" {
		// Process main body
		for _, stmt := range ifNode.Body {
			optimizedStmt := fold(root, stmt, index) // fold each statement
			// if the statement isn't nil
			if optimizedStmt != nil {
				// if the statement is an if statement:
				// we want to append the body directly
				// to the body of the new if statement
				if optimizedStmt.Type == "IF_STATEMENT" {
					// append body of if to body of newifnode
					newIfNode.Body = append(newIfNode.Body, optimizedStmt.Body...)
				} else {
					// otherwise just append the node like normal
					newIfNode.Body = append(newIfNode.Body, optimizedStmt)
				}
			}
		}
	} else { // if it is false and node.right is nil (no else)
		return nil
	}

	return newIfNode
}

func fold(root *Node, node *Node, index int) *Node {
	if node == nil {
		return nil
	}

	switch node.Type {
	case "ADD", "SUB", "MULT", "DIV":
		return handleArithmetic(root, node, index)
	case "IDENTIFIER":
		valueTableNode := searchValueTable(Values, node.Value)
		if valueTableNode != nil {
			return valueTableNode
		}
		return node // Return the identifier if not found
	case "ASSIGN":
		node.Right = fold(root, node.Right, index)
		updateValueTable(&Values, node)
		return node
	case "FUNCTION_CALL":
		if node.Value == "write" {
			writeNode := node
			for paramIndex, param := range writeNode.Params {
				writeNode.Params[paramIndex] = fold(root, param, index)
			}
			return writeNode
		} else {
			funcNode := getFunction(&Functions, node.Value)
			params := node.Params

			if funcNode == nil {
				fmt.Println("Optimizer: Function Search Returned Nil Results")
				return nil
			}

			if len(funcNode.Params) != len(params) {
				fmt.Println("Optimizer: More parameters than accepted!")
				os.Exit(3)
			}

			var foldedParams []*Node
			for paramIndex, param := range params {
				paramNode := Node{
					DType: "OP",
					Type:  "ASSIGN",
					Value: "=",
					Right: fold(root, param, index),
					Left:  funcNode.Params[paramIndex],
				}
				foldedParams = append(foldedParams, &paramNode)
			}

			foldedFunction := foldFunction(funcNode, foldedParams, index)
			if foldedFunction.Type == "ASSIGN" {
				return foldedFunction.Right
			}
			return foldedFunction
		}
	case "ARRAY_INDEX":
		arrayIndexNode := fold(root, node.Body[0], index)
		arrayNode := search(root, index, node.Value)

		arrayIndex, _ := strconv.Atoi(arrayIndexNode.Value)

		return arrayNode.Body[arrayIndex]
	case "RETURN":
		return fold(root, node, index)
	case "IF_STATEMENT":
		return optimizeIfStatement(root, node, index)
	case "ELSE_STATEMENT":
		// Create new else node with optimized body
		newElseNode := &Node{
			Type:  "ELSE_STATEMENT",
			Value: "else",
			Body:  []*Node{},
		}
		for _, stmt := range node.Body {
			optimizedStmt := fold(root, stmt, index)
			if optimizedStmt != nil {
				newElseNode.Body = append(newElseNode.Body, optimizedStmt)
			}
		}
		return newElseNode

	case "GREATER_THAN", "LESS_THAN", "GREATER_THAN_OR_EQUALS_TO", "LESS_THAN_OR_EQUAL_TO", "EQUALS":
		return optimizeComparison(root, node, index)

	case "FOR_LOOP":

		return optimizeForLoop(root, node, index)

	default:
		// Return node as is if no folding is applied
		return node
	}
}

func optimizeComparison(root *Node, node *Node, index int) *Node {
	// Debugging print

	leftNode := node.Left
	rightNode := node.Right

	boolTrue := Node{
		Type:  "BOOL",
		DType: "BOOL",
		Value: "TRUE",
	}

	boolFalse := Node{
		Type:  "BOOL",
		DType: "BOOL",
		Value: "FALSE",
	}

	// Ensure left and right nodes are not nil
	if leftNode == nil || rightNode == nil {
		fmt.Println("Optimizer: Comparison of Nil Nodes")
		return node
	}

	// Resolve identifiers to their most recent values
	if leftNode.Type == "IDENTIFIER" {
		resolvedLeft := searchValueTable(Values, leftNode.Value)
		if resolvedLeft != nil {
			leftNode = resolvedLeft
		}
	}
	if rightNode.Type == "IDENTIFIER" {
		resolvedRight := searchValueTable(Values, rightNode.Value)
		if resolvedRight != nil {
			rightNode = resolvedRight
		}
	}

	// Resolve ARRAY_INDEX nodes
	if leftNode.Type == "ARRAY_INDEX" {
		resolvedLeft := fold(root, leftNode, index)
		if resolvedLeft != nil {
			leftNode = resolvedLeft
		}
	}
	if rightNode.Type == "ARRAY_INDEX" {
		resolvedRight := fold(root, rightNode, index)
		if resolvedRight != nil {
			rightNode = resolvedRight
		}
	}

	// Validate node types
	if leftNode.Type != "INT" || rightNode.Type != "INT" {
		fmt.Printf("Comparison: Invalid node types. Left: %s, Right: %s\n", leftNode.Type, rightNode.Type)
		os.Exit(3)
		return node
	}

	// Convert to integers
	leftVal := atoi(leftNode.Value)
	rightVal := atoi(rightNode.Value)

	// Perform comparison
	var result *Node
	switch node.Type {
	case "GREATER_THAN":
		if leftVal > rightVal {
			result = &boolTrue
		} else {
			result = &boolFalse
		}
	case "LESS_THAN":
		if leftVal < rightVal {
			result = &boolTrue
		} else {
			result = &boolFalse
		}
	case "GREATER_THAN_OR_EQUAL_TO":
		if leftVal >= rightVal {
			result = &boolTrue
		} else {
			result = &boolFalse
		}
	case "LESS_THAN_OR_EQUAL_TO":
		if leftVal <= rightVal {
			result = &boolTrue
		} else {
			result = &boolFalse
		}
	case "EQUALS":
		if leftVal == rightVal {
			result = &boolTrue
		} else {
			result = &boolFalse
		}
	default:
		fmt.Println("Unknown comparison type")
		return node
	}

	return result
}

func handleArithmetic(root *Node, node *Node, index int) *Node {
	leftNode := node.Left
	rightNode := node.Right

	// Ensure left and right nodes are not nil
	if leftNode == nil || rightNode == nil {
		return node
	}

	// Resolve identifiers to their values, if necessary
	if leftNode.Type == "IDENTIFIER" {
		resolvedLeft := searchValueTable(Values, leftNode.Value)
		if resolvedLeft != nil {
			leftNode = resolvedLeft
		}
	}
	if rightNode.Type == "IDENTIFIER" {
		resolvedRight := searchValueTable(Values, rightNode.Value)
		if resolvedRight != nil {
			rightNode = resolvedRight
		}
	}

	// Resolve ARRAY_INDEX nodes
	if leftNode.Type == "ARRAY_INDEX" {
		resolvedLeft := fold(root, leftNode, index)
		if resolvedLeft != nil {
			leftNode = resolvedLeft
		}
	}
	if rightNode.Type == "ARRAY_INDEX" {
		resolvedRight := fold(root, rightNode, index)
		if resolvedRight != nil {
			rightNode = resolvedRight
		}
	}

	// Resolve subtrees that are arithmetic expressions
	if leftNode.Type == "ADD" || leftNode.Type == "SUB" || leftNode.Type == "MULT" || leftNode.Type == "DIV" {
		leftNode = fold(root, leftNode, index)
	}
	if rightNode.Type == "ADD" || rightNode.Type == "SUB" || rightNode.Type == "MULT" || rightNode.Type == "DIV" {
		rightNode = fold(root, rightNode, index)
	}

	// Perform arithmetic operation if types are compatible
	if (leftNode.DType == "INT" || leftNode.DType == "FLOAT") && (rightNode.DType == "INT" || rightNode.DType == "FLOAT") {
		var leftVal, rightVal float64
		var err error

		// Convert left value
		if leftNode.DType == "INT" {
			intVal, _ := strconv.Atoi(leftNode.Value)
			leftVal = float64(intVal)
		} else {
			leftVal, err = strconv.ParseFloat(leftNode.Value, 64)
			if err != nil {
				fmt.Println("Error parsing left float:", err)
				os.Exit(3)
			}
		}

		// Convert right value
		if rightNode.DType == "INT" {
			intVal, _ := strconv.Atoi(rightNode.Value)
			rightVal = float64(intVal)
		} else {
			rightVal, err = strconv.ParseFloat(rightNode.Value, 64)
			if err != nil {
				fmt.Println("Error parsing right float:", err)
				os.Exit(3)
			}
		}

		// Perform the operation based on the node type
		switch node.Type {
		case "ADD":
			node.Value = strconv.FormatFloat(leftVal+rightVal, 'f', -1, 64)
		case "SUB":
			node.Value = strconv.FormatFloat(leftVal-rightVal, 'f', -1, 64)
		case "MULT":
			node.Value = strconv.FormatFloat(leftVal*rightVal, 'f', -1, 64)
		case "DIV":
			if rightVal == 0 {
				fmt.Println("Error: Division by zero!")
				os.Exit(3)
			}
			node.Value = strconv.FormatFloat(leftVal/rightVal, 'f', -1, 64)
		default:
			fmt.Println("Unknown operation")
			return node
		}

		// Determine the resulting type
		if leftNode.DType == "FLOAT" || rightNode.DType == "FLOAT" {
			node.DType = "FLOAT"
		} else {
			// Check if the result can remain an integer
			intResult, err := strconv.Atoi(node.Value)
			if err == nil && strconv.FormatInt(int64(intResult), 10) == node.Value {
				node.DType = "INT"
				node.Value = strconv.Itoa(intResult)
			} else {
				node.DType = "FLOAT"
			}
		}

		// Cleanup the current node
		node.Left = nil
		node.Right = nil
		node.Type = "INT"
	}

	// After resolution, check if both nodes are numbers
	if leftNode.DType == "STRING" && rightNode.DType == "STRING" {
		// Regular expression to match non-escaped quotes
		re := regexp.MustCompile(`(^|[^\\])"`)

		// Remove non-escaped quotes from each string
		cleanLeft := re.ReplaceAllString(leftNode.Value, `$1`)
		cleanRight := re.ReplaceAllString(rightNode.Value, `$1`)

		switch node.Type {
		case "ADD":
			node.Value = fmt.Sprintf("\"%s%s\"", cleanLeft, cleanRight)
			node.Type = leftNode.Type
			node.DType = leftNode.DType
			node.Left = nil
			node.Right = nil
		default:
			fmt.Println("Unsupported string operation between", leftNode.Value, "and", rightNode.Value)
			os.Exit(3)
		}
	}

	return node
}

func search(root *Node, searchBehind int, value string) *Node {
	for i := searchBehind - 1; i >= 0; i-- {
		if root.Body[i].Type == "ASSIGN" {
			if root.Body[i].Left.Value == value {
				return root.Body[i].Right
			}
		}
	}
	return nil
}

func searchForFunctions(root *Node, searchBehind int, value string) *Node {
	// First, search in the current context
	for i := searchBehind - 1; i >= 0; i-- {
		if root.Body[i].Value == value && root.Body[i].Type == "FUNCTION_DECL" {
			return root.Body[i]
		}
	}

	// If not found, try a broader search or add a global function table
	fmt.Printf("Warning: Function %s not found\n", value)
	return nil
}

func foldFunction(funcNode *Node, params []*Node, index int) *Node {
	// Deep copy the function node to prevent parameter persistence
	foldedFunction := deepCopyNode(funcNode)

	// Add parameters to the beginning of the copied function body
	for _, param := range params {
		foldedFunction.Body = prependNode(foldedFunction.Body, param)
	}

	// Create a new node to store folded statements
	resultNode := &Node{}

	for funcIndex, statement := range foldedFunction.Body {
		result := fold(foldedFunction, statement, funcIndex)
		if result != nil {
			if result.Type == "IF_STATEMENT" {
				if result.Left.Value == "FALSE" || result.Left.Value == "TRUE" {
					resultNode.Body = append(resultNode.Body, result.Body...)
				} else {
					resultNode.Body = append(resultNode.Body, result)
				}
			} else {
				resultNode.Body = append(resultNode.Body, result)
			}
		}
	}

	// Return the last statement of the function
	if len(resultNode.Body) > 0 {
		return resultNode.Body[len(resultNode.Body)-1]
	}
	return nil
}

// deepCopyNode creates a complete recursive copy of a node
func deepCopyNode(node *Node) *Node {
	if node == nil {
		return nil
	}

	newNode := &Node{
		Type:  node.Type,
		DType: node.DType,
		Value: node.Value,
	}

	// Deep copy Left and Right
	if node.Left != nil {
		newNode.Left = deepCopyNode(node.Left)
	}
	if node.Right != nil {
		newNode.Right = deepCopyNode(node.Right)
	}

	// Deep copy Params
	for _, param := range node.Params {
		newNode.Params = append(newNode.Params, deepCopyNode(param))
	}

	// Deep copy Body
	for _, bodyNode := range node.Body {
		newNode.Body = append(newNode.Body, deepCopyNode(bodyNode))
	}

	return newNode
}

func optimizeForLoop(root *Node, forLoopNode *Node, index int) *Node {
	if forLoopNode.Type != "FOR_LOOP" {
		panic("Node is not a for loop")
	}

	// Extract key components from the for loop
	condition := forLoopNode.Params[0]
	init := forLoopNode.Body[0]
	updation := forLoopNode.Body[len(forLoopNode.Body)-1]

	// Analyze the initialization, condition, and updation
	loopVar := init.Left.Value
	start := atoi(init.Right.Value)
	end := atoi(condition.Right.Value)
	step := 1

	if updation.Right.Type == "ADD" {
		step = atoi(updation.Right.Right.Value)
	}

	// Generate the optimized body by simulating the loop execution
	unrolledLoop := Node{}
	unrolledLoop.Body = append(unrolledLoop.Body, init)

	for i := start; i < end; i += step {
		for _, stmt := range forLoopNode.Body[1 : len(forLoopNode.Body)-1] {
			// Replace loop variable with current iteration value
			replacedStmt := replaceLoopVar(stmt, loopVar, strconv.Itoa(i))

			// Handle if statement body separately
			if stmt.Type == "IF_STATEMENT" {
				for _, bodyStmt := range stmt.Body {
					replacedBodyStmt := replaceLoopVar(bodyStmt, loopVar, strconv.Itoa(i))
					unrolledLoop.Body = append(unrolledLoop.Body, replacedBodyStmt)
				}
			} else {
				unrolledLoop.Body = append(unrolledLoop.Body, replacedStmt)
			}
		}
	}

	return &unrolledLoop
}

// replaceLoopVar replaces occurrences of the loop variable in a node with a given value.
func replaceLoopVar(node *Node, loopVar string, value string) *Node {
	if node == nil {
		return nil
	}

	// Clone the node
	newNode := &Node{
		Type:   node.Type,
		DType:  node.DType,
		Value:  node.Value,
		Params: []*Node{},
		Left:   replaceLoopVar(node.Left, loopVar, value),
		Right:  replaceLoopVar(node.Right, loopVar, value),
		Body:   []*Node{},
	}

	// Replace loop variable if found
	if node.Type == "IDENTIFIER" && node.Value == loopVar {
		newNode = &Node{
			Type:  "INT",
			DType: "INT",
			Value: value,
		}
	}

	// Recursively process parameters and body
	for _, param := range node.Params {
		newNode.Params = append(newNode.Params, replaceLoopVar(param, loopVar, value))
	}
	for _, bodyNode := range node.Body {
		newNode.Body = append(newNode.Body, replaceLoopVar(bodyNode, loopVar, value))
	}

	return newNode
}

// atoi is a helper function to convert string to integer.
func atoi(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func prependNode(body []*Node, node *Node) []*Node {
	var emptyNode *Node
	body = append(body, emptyNode)
	copy(body[1:], body)
	body[0] = node
	return body
}

// Print function to display the optimized AST
func printAST(root *Node) {
	fmt.Println("Printing AST...")
	if root != nil {
		printNode(root, "", true)
	}
}

func updateValueTable(Values *ValueTable, node *Node) {
	ident := node.Left.Value
	newValue := node.Right.Value

	// Instead of replacing, always append new assignments
	newAssignment := &Node{
		Type: "ASSIGN",
		Left: &Node{
			Type:  "IDENTIFIER",
			Value: ident,
			DType: node.Left.DType,
		},
		Right: &Node{
			Type:  node.Right.Type,
			Value: newValue,
			DType: node.Right.DType,
		},
	}

	Values.Body = append(Values.Body, newAssignment)
}

func addFunction(Functions *ValueTable, node *Node) {
	Functions.Body = append(Functions.Body, node)
}

func getFunction(Functions *ValueTable, name string) *Node {
	for _, function := range Functions.Body {
		if function.Value == name {
			return function
		}
	}
	fmt.Println("getFunction: Function Not Found!")
	return nil
}

func searchValueTable(Values ValueTable, ident string) *Node {
	// Search from the end to get the most recent value
	for i := len(Values.Body) - 1; i >= 0; i-- {
		stmt := Values.Body[i]
		if stmt.Left.Value == ident {
			return stmt.Right
		}
	}

	return nil
}

func finalRound(root *Node) {
	if root == nil {
		return
	}

	// Process the body slice if it exists
	if len(root.Body) > 0 {
		var newBody []*Node
		for _, child := range root.Body {
			if child.Type == "ASSIGN" {
				// Skip nodes with Type == "ASSIGN"
				continue
			} else if child.Type == "IF_STATEMENT" {
				// Replace "IF_STATEMENT" node with its Body
				newBody = append(newBody, child.Body...)
			} else {
				// Keep other nodes as they are
				newBody = append(newBody, child)
			}
		}
		root.Body = newBody
	}
}
