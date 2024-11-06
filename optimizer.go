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
			fmt.Println("Found ASSIGN statement.")
			optimizedNode := fold(root, statement.Right, index)
			statement.Right = optimizedNode
			if optimizedNode != nil {
				fmt.Printf("ASSIGN optimized: %s = %s\n", statement.Value, optimizedNode.Value)
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
		fmt.Println("fold: Performing arithmetic folding.")
		return handleArithmetic(root, node, index)
	case "IDENTIFIER":
		fmt.Printf("fold: Resolving identifier: %s\n", node.Value)
		resolvedNode := search(root, index, node.Value)
		fmt.Println("fold: resolveNode.value: ", resolvedNode.Value)
		return fold(root, resolvedNode, index)
	case "FUNCTION_CALL":
		fmt.Printf("fold: Resolving function call: %s\n", node.Value)
		funcNode := searchForFunctions(root, index, node.Value)
		params := node.Params
		for _, param := range funcNode.Params {
			fmt.Println(param.DType)
		}

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
				Left:  params[paramIndex],
			}
			paramNode.Left.Value = funcNode.Params[paramIndex].Value
			foldedParams = append(foldedParams, &paramNode)
			fmt.Println(len(foldedParams))
		}
		return foldFunction(funcNode, foldedParams, index)
	case "ASSIGN":
		fmt.Printf("fold: assign: Resolving assign call: %s\n", node.Left.Value)
		return fold(root, node.Right, index)
	case "NUMBER":
		fmt.Printf("fold: number: Resolving number call: %s\n", node.Value)
		return node
	case "RETURN":
		fmt.Printf("fold: return: Resolving return call: %s\n", node.Value)
		return fold(root, node, index)
	default:
		fmt.Println("  No folding applied for unhandled node type.")
		return node
	}
}

func handleArithmetic(root *Node, node *Node, index int) *Node {
	fmt.Printf("handleArithmetic: Handling arithmetic for node %s with operation %s\n", node.Value, node.Right.Type)
	leftNode := node.Left
	rightNode := node.Right

	// Ensure left and right nodes are not nil
	if leftNode == nil || rightNode == nil {
		fmt.Println("handleArithmetic: Arithmetic operation missing operands.")
		return node
	}

	// Resolve identifiers to their values, if necessary
	if leftNode.Type == "IDENTIFIER" {
		resolvedLeft := fold(root, search(root, index, leftNode.Value), index)
		fmt.Println("leftNode.Type = IDENTIFIER. Value: ", leftNode.Value)
		if resolvedLeft != nil && resolvedLeft.Type == "NUMBER" {
			fmt.Printf("  Resolved left identifier %s to %s\n", leftNode.Value, resolvedLeft.Value)
			leftNode = resolvedLeft
		}
	}
	if rightNode.Type == "IDENTIFIER" {
		resolvedRight := fold(root, search(root, index, rightNode.Value), index)
		fmt.Println("rightNode.Type = IDENTIFIER. Value: ", resolvedRight.Value)
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

		fmt.Println("handleArithmetic: :node.Type: ", node.Type)
		switch node.Type {
		case "ADD":
			node.Value = strconv.Itoa(leftVal + rightVal)
			node.Type = leftNode.Type
			node.DType = leftNode.DType
			node.Left = nil
			node.Right = nil
			fmt.Printf("  Folded ADD result: %s\n", node.Value)
		case "SUB":
			node.Right.Value = strconv.Itoa(leftVal - rightVal)
			node.Type = leftNode.Type
			node.DType = leftNode.DType
			node.Left = nil
			node.Right = nil
			fmt.Printf("  Folded SUB result: %s\n", node.Value)
		case "MULT":
			node.Right.Value = strconv.Itoa(leftVal * rightVal)
			node.Type = leftNode.Type
			node.DType = leftNode.DType
			node.Left = nil
			node.Right = nil
			fmt.Printf("  Folded MULT result: %s\n", node.Value)
		case "DIV":
			if rightVal == 0 {
				fmt.Println("  Error: Division by zero!")
				os.Exit(3)
			}
			node.Right.Value = strconv.Itoa(leftVal / rightVal)
			node.Type = leftNode.Type
			node.DType = leftNode.DType
			node.Left = nil
			node.Right = nil
			fmt.Printf("  Folded DIV result: %s\n", node.Value)
		default:
			fmt.Println("nothing")
		}
	} else {
		fmt.Println("  Operands are not both numbers; cannot fold.")
	}

	return node
}

func search(root *Node, searchBehind int, value string) *Node {
	fmt.Println("search:", root.Value)
	fmt.Printf("search: Searching for identifier %s in AST.\n", value)
	for i := searchBehind - 1; i >= 0; i-- {
		fmt.Println("search: index: ", i)
		fmt.Println("search: len: ", len(root.Body))
		if root.Body[i].Type == "ASSIGN" {
			if root.Body[i].Left.Value == value {
				fmt.Printf("search: Identifier %s found in AST.\n", value)
				return root.Body[i].Right
			}
		}
	}
	fmt.Printf("search: Identifier %s not found in AST.\n", value)
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
		fmt.Println("param value", param.Left.Value)
		funcNode.Body = prependNode(funcNode.Body, param)
	}
	for _, bodyStatement := range funcNode.Body {
		fmt.Println("foldFunction: ", bodyStatement.Left.Value)
	}
	fmt.Println("foldFunction: len:", len(funcNode.Body))
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
