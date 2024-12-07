package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
)

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
		case "FUNCTION_CALL":
			if statement.Value == "write" {
				writeNode := statement
				for paramIndex, param := range writeNode.Params {
					writeNode.Params[paramIndex] = fold(root, param, index)
				}
				optimizedAST.Body = append(optimizedAST.Body, writeNode)
			} else {
				funcNode := searchForFunctions(root, index, statement.Value)
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
				if funcNode != nil {
					optimizedAST.Body = append(optimizedAST.Body, foldedFunction)
				}
			}
		case "IF_STATEMENT":
			optimizedIfNode := optimizeIfStatement(root, statement, index)
			// Debug print to verify if statement structure
			fmt.Printf("Debug: Optimized if node structure:\n")
			fmt.Printf("  Has condition: %v\n", optimizedIfNode.Left != nil)
			fmt.Printf("  Has if body: %v\n", len(optimizedIfNode.Body) > 0)
			fmt.Printf("  Has else: %v\n", optimizedIfNode.Right != nil)
			optimizedAST.Body = append(optimizedAST.Body, optimizedIfNode)

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
	}

	// Optimize if body
	for _, stmt := range ifNode.Body {
		optimizedStmt := fold(root, stmt, index)
		if optimizedStmt != nil {
			newIfNode.Body = append(newIfNode.Body, optimizedStmt)
		}
	}

	// Optimize else body if it exists
	if ifNode.Right != nil {
		newElseNode := &Node{
			Type:  "ELSE_STATEMENT",
			Value: "else",
			Body:  []*Node{},
		}

		for _, stmt := range ifNode.Right.Body {
			optimizedStmt := fold(root, stmt, index)
			if optimizedStmt != nil {
				newElseNode.Body = append(newElseNode.Body, optimizedStmt)
			}
		}

		// Only set the else node if it has statements
		if len(newElseNode.Body) > 0 {
			newIfNode.Right = newElseNode
		}
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
		resolvedNode := search(root, index, node.Value)
		if resolvedNode != nil {
			return fold(root, resolvedNode, index)
		}
		return node // Return the identifier if not found
	case "ASSIGN":
		node.Right = fold(root, node.Right, index)
		return node
	case "FUNCTION_CALL":
		if node.Value == "write" {
			writeNode := node
			for paramIndex, param := range writeNode.Params {
				writeNode.Params[paramIndex] = fold(root, param, index)
			}
			return writeNode
		} else {
			funcNode := searchForFunctions(root, index, node.Value)
			params := node.Params

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
	default:
		// Return node as is if no folding is applied
		return node
	}
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
		resolvedLeft := fold(root, search(root, index, leftNode.Value), index)
		if resolvedLeft != nil {
			leftNode = resolvedLeft
		}
	}
	if rightNode.Type == "IDENTIFIER" {
		resolvedRight := fold(root, search(root, index, rightNode.Value), index)
		if resolvedRight != nil {
			rightNode = resolvedRight
		}
	}

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

	// Resolve identifiers to their values, if necessary
	if leftNode.Type == "ADD" || leftNode.Type == "SUB" || leftNode.Type == "MULT" || leftNode.Type == "DIV" {
		leftNode = fold(root, leftNode, index)
	}
	if rightNode.Type == "ADD" || rightNode.Type == "SUB" || rightNode.Type == "MULT" || rightNode.Type == "DIV" {
		rightNode = fold(root, rightNode, index)
	}

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

		// Perform operation based on node type
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

		// Set the result type and cleanup nodes
		node.Type = leftNode.Type
		node.DType = "FLOAT" // Result is float if any operand was float
		node.Left = nil
		node.Right = nil

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
	for i := searchBehind - 1; i >= 0; i-- {
		if root.Body[i].Value == value && root.Body[i].Type == "FUNCTION_DECL" {
			funcNode := root.Body[i]
			return funcNode // Return just the function body
		}
	}
	return nil
}

func foldFunction(funcNode *Node, params []*Node, index int) *Node {
	foldedFunction := &Node{}
	for _, param := range params {
		funcNode.Body = prependNode(funcNode.Body, param)
	}

	for funcIndex, statement := range funcNode.Body {
		result := fold(funcNode, statement, funcIndex) // Pass `nil` if root isn't needed
		if result != nil {
			foldedFunction.Body = append(foldedFunction.Body, result)
		}
	}
	return foldedFunction.Body[len(foldedFunction.Body)-1]
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
