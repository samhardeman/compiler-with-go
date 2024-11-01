package main

import (
	"fmt"
	"os"
	"strconv"
)

func optimizer(root *Node) Node {
	fmt.Println("Starting optimization...")
	optimizedAST := Node{
		Type:  "root",
		DType: "",
		Value: "",
	}

	for index, statement := range root.Body {
		fmt.Printf("Processing statement %d: %s\n", index, statement.Value)
		switch statement.Type {
		case "ASSIGN":
			fmt.Println("  Found ASSIGN statement.")
			optimizedNode := fold(root, statement.Right, index)
			statement.Right = optimizedNode
			if optimizedNode != nil {
				fmt.Printf("  ASSIGN optimized: %s = %s\n", statement.Value, optimizedNode.Value)
				optimizedAST.Body = append(optimizedAST.Body, statement)
			}
		case "FUNCTION_CALL":
			fmt.Println("  Found FUNCTION_CALL statement.")
			if statement.Value == "write" {
				fmt.Println("  Writing unoptimized FUNCTION_CALL statement.")
				optimizedAST.Body = append(optimizedAST.Body, statement)
			} else {
				funcNode := searchForFunctions(root, index, statement.Value)
				if funcNode != nil {
					fmt.Printf("  Optimized function call found: %s\n", funcNode.Value)
					optimizedAST.Body = append(optimizedAST.Body, funcNode)
				} else {
					fmt.Println("  No optimization found for FUNCTION_CALL.")
				}
			}
		default:
			fmt.Printf("  No optimization for statement of type %s\n", statement.Type)
			optimizedAST.Body = append(optimizedAST.Body, statement)
		}
		index++
	}

	fmt.Println("Optimization complete.")
	return optimizedAST
}

func fold(root *Node, node *Node, index int) *Node {
	if node == nil {
		fmt.Println("  fold: Received a nil node.")
		return nil
	}

	fmt.Printf("  Folding node %s with type %s\n", node.Value, node.Type)
	switch node.Type {
	case "ADD", "SUB", "MULT", "DIV":
		fmt.Println("  Performing arithmetic folding.")
		return handleArithmetic(root, node, index)
	case "IDENTIFIER":
		fmt.Printf("  Resolving identifier: %s\n", node.Value)
		resolvedNode := search(root, index, node.Value)
		return fold(root, resolvedNode, index)
	case "FUNCTION_CALL":
		fmt.Printf("  Resolving function call: %s\n", node.Value)
		funcNode := searchForFunctions(root, index, node.Value)
		params := node.Params
		var foldedParams []*Node
		for _, param := range params {
			foldedParams = append(foldedParams, fold(root, param, index))
		}
		fmt.Println("params " + params[0].Value)
		return foldFunction(funcNode, foldedParams, index)
	case "ASSIGN":
		fmt.Printf("  Resolving assign call: %s\n", node.Left.Value)
		return fold(root, node.Right, index)
	case "NUMBER":
		fmt.Printf("  Resolving number call: %s\n", node.Value)
		return node
	default:
		fmt.Println("  No folding applied for unhandled node type.")
		return node
	}
}

func handleArithmetic(root *Node, node *Node, index int) *Node {
	fmt.Printf("  Handling arithmetic for node %s with operation %s\n", node.Value, node.Right.Type)
	leftNode := node.Right.Left
	rightNode := node.Right.Right

	// Ensure left and right nodes are not nil
	if leftNode == nil || rightNode == nil {
		fmt.Println("  Arithmetic operation missing operands.")
		return node
	}

	// Resolve identifiers to their values, if necessary
	if leftNode.Type == "IDENTIFIER" {
		resolvedLeft := search(root, index, leftNode.Value)
		fmt.Println(leftNode.Value)
		if resolvedLeft != nil && resolvedLeft.Type == "NUMBER" {
			fmt.Printf("  Resolved left identifier %s to %s\n", leftNode.Value, resolvedLeft.Value)
			leftNode = resolvedLeft
		}
	}
	if rightNode.Type == "IDENTIFIER" {
		resolvedRight := search(root, index, rightNode.Value)
		fmt.Println(rightNode.Value)
		if resolvedRight != nil && resolvedRight.Type == "NUMBER" {
			fmt.Printf("  Resolved right identifier %s to %s\n", rightNode.Value, resolvedRight.Value)
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

	// After resolution, check if both nodes are numbers
	if leftNode.Type == "NUMBER" && rightNode.Type == "NUMBER" {
		leftVal, _ := strconv.Atoi(leftNode.Value)
		rightVal, _ := strconv.Atoi(rightNode.Value)

		switch node.Right.Type {
		case "ADD":
			node.Right.Value = strconv.Itoa(leftVal + rightVal)
			fmt.Printf("  Folded ADD result: %s\n", node.Right.Value)
		case "SUB":
			node.Right.Value = strconv.Itoa(leftVal - rightVal)
			fmt.Printf("  Folded SUB result: %s\n", node.Right.Value)
		case "MULT":
			node.Right.Value = strconv.Itoa(leftVal * rightVal)
			fmt.Printf("  Folded MULT result: %s\n", node.Right.Value)
		case "DIV":
			if rightVal == 0 {
				fmt.Println("  Error: Division by zero!")
				os.Exit(3)
			}
			node.Right.Value = strconv.Itoa(leftVal / rightVal)
			fmt.Printf("  Folded DIV result: %s\n", node.Right.Value)
		}

		// Update the node to represent a single number after folding
		node.Right.Type = "NUMBER"
		node.Right.DType = "INT"
		node.Right.Left = nil
		node.Right.Right = nil

		return node

	} else {
		fmt.Println("  Operands are not both numbers; cannot fold.")
	}
	return node
}

func search(root *Node, searchBehind int, value string) *Node {
	fmt.Println(searchBehind)
	fmt.Println(len(root.Body))
	fmt.Printf("  Searching for identifier %s in AST.\n", value)
	for i := searchBehind - 1; i >= 0; i-- {
		fmt.Println(root.Body[i].Value)
		if root.Body[i].Type == "ASSIGN" {
			if root.Body[i].Left.Value == value {
				fmt.Printf("  Identifier %s found in AST.\n", value)
				return root.Body[i].Right
			}
		}
	}
	fmt.Printf("  Identifier %s not found in AST.\n", value)
	return nil
}

func searchForFunctions(root *Node, searchBehind int, value string) *Node {
	fmt.Printf("  Searching for function %s in AST.\n", value)
	for i := searchBehind - 1; i >= 0; i-- {
		if root.Body[i].Value == value && root.Body[i].Type == "FUNCTION_DECL" {
			fmt.Printf("  Function %s found in AST.\n", value)
			return root.Body[i]
		}
	}
	fmt.Printf("  Function %s not found in AST.\n", value)
	return nil
}

func foldFunction(node *Node, params []*Node, index int) *Node {
	if node == nil {
		fmt.Println("  foldFunction: Received a nil node.")
		return nil
	}

	for _, param := range params {
		node.Body = prependNode(node.Body, fold(node, param, index))
	}

	fmt.Printf("  Folding function %s\n", node.Value)
	foldedFunction := &Node{}
	for _, statement := range node.Body {
		result := fold(node, statement, index)
		if result != nil {
			fmt.Printf("  Statement folded: %s\n", result.Value)
			foldedFunction.Body = append(foldedFunction.Body, result)
		}
	}
	return foldedFunction
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
