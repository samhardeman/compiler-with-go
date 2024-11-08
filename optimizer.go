package main

import (
	"fmt"
	"os"
	"strconv"
)

func optimizer(root *Node) Node {
	fmt.Println("Starting optimization...")
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
				optimizedAST.Body = append(optimizedAST.Body, statement)
			} else {
				funcNode := searchForFunctions(root, index, statement.Value)
				if funcNode != nil {
					optimizedAST.Body = append(optimizedAST.Body, funcNode)
				}
			}
		case "IF_STATEMENT":
			optimizedIfNode := optimizeIfStatement(root, statement, index)
			optimizedAST.Body = append(optimizedAST.Body, optimizedIfNode)
		default:
			fmt.Printf("  No optimization for statement of type %s\n", statement.Type)
		}
	}

	fmt.Println("Optimization complete.")
	return optimizedAST
}

func optimizeIfStatement(root *Node, ifNode *Node, index int) *Node {
	// Optimize the condition
	ifNode.Left = fold(root, ifNode.Left, index)

	// Optimize the 'if' body
	if len(ifNode.Body) > 0 {
		optimizedBody := []*Node{}
		for idx, stmt := range ifNode.Body {
			optimizedStmt := fold(root, stmt, idx)
			if optimizedStmt != nil {
				optimizedBody = append(optimizedBody, optimizedStmt)
			}
		}
		ifNode.Body = optimizedBody
	}

	// Optimize the 'else' body, if present
	if ifNode.Right != nil && ifNode.Right.Type == "ELSE_STATEMENT" {
		elseNode := ifNode.Right
		if len(elseNode.Body) > 0 {
			optimizedElseBody := []*Node{}
			for idx, stmt := range elseNode.Body {
				optimizedStmt := fold(root, stmt, idx)
				if optimizedStmt != nil {
					optimizedElseBody = append(optimizedElseBody, optimizedStmt)
				}
			}
			elseNode.Body = optimizedElseBody
		}
	}

	return ifNode
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
	case "IF_STATEMENT":
		// Optimize the condition
		if node.Left != nil {
			node.Left = fold(root, node.Left, index)
		}

		// Optimize the 'if' body
		if node.Body != nil && len(node.Body) > 0 {
			for i, stmt := range node.Body {
				if stmt != nil {
					node.Body[i] = fold(node, stmt, i)
				}
			}
		}

		// Optimize the 'else' body, if present
		if node.Right != nil && node.Right.Type == "ELSE_STATEMENT" {
			elseNode := node.Right
			if elseNode.Body != nil && len(elseNode.Body) > 0 {
				for i, stmt := range elseNode.Body {
					if stmt != nil {
						elseNode.Body[i] = fold(elseNode, stmt, i)
					}
				}
			}
		}

		return node
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
		if resolvedLeft != nil && resolvedLeft.Type == "NUMBER" {
			leftNode = resolvedLeft
		}
	}
	if rightNode.Type == "IDENTIFIER" {
		resolvedRight := fold(root, search(root, index, rightNode.Value), index)
		if resolvedRight != nil && resolvedRight.Type == "NUMBER" {
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

		switch node.Type {
		case "ADD":
			node.Value = strconv.Itoa(leftVal + rightVal)
			node.Type = leftNode.Type
			node.DType = leftNode.DType
			node.Left = nil
			node.Right = nil
		case "SUB":
			node.Right.Value = strconv.Itoa(leftVal - rightVal)
			node.Type = leftNode.Type
			node.DType = leftNode.DType
			node.Left = nil
			node.Right = nil
		case "MULT":
			node.Right.Value = strconv.Itoa(leftVal * rightVal)
			node.Type = leftNode.Type
			node.DType = leftNode.DType
			node.Left = nil
			node.Right = nil
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
		default:
			fmt.Println("nothing")
		}
	} else {
		fmt.Println("  Operands are not both numbers; cannot fold.")
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
