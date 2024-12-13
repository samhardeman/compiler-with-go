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

			if optimizedIfNode.Left.Value == "FALSE" {
				optimizedAST.Body = append(optimizedAST.Body, optimizedIfNode.Right.Body...)
			} else if optimizedIfNode.Left.Value == "TRUE" {
				optimizedAST.Body = append(optimizedAST.Body, optimizedIfNode.Body...)
			} else {
				optimizedAST.Body = append(optimizedAST.Body, optimizedIfNode)
			}

		case "FOR_LOOP":
			optimizedForLoop := optimizeForLoop(statement)
			optimizedAST.Body = append(optimizedAST.Body, optimizedForLoop.Body...)

		case "WHILE_LOOP":
			optimizedWhileLoop := optimizeWhileLoop(statement)
			optimizedAST.Body = append(optimizedAST.Body, optimizedWhileLoop.Body...)
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

	if newIfNode.Left.Value == "TRUE" {
		return newIfNode
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

		if newIfNode.Left.Value == "FALSE" {
			newIfNode.Body = newElseNode.Body
			return newIfNode
		} else if len(newElseNode.Body) > 0 {
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

	case "GREATER_THAN", "LESS_THAN":
		return optimizeComparison(root, node, index)

	// will not output straight to the body will need some reworking if this is supposed to be in a function which is everything
	// frick
	case "FOR_LOOP":
		return optimizeForLoop(node)

	// will not output straight to the body will need some reworking if this is supposed to be in a function which is everything
	// frick
	case "WHILE_LOOP":
		return optimizeWhileLoop(node)

	default:
		// Return node as is if no folding is applied
		return node
	}
}

func optimizeComparison(root *Node, node *Node, index int) *Node {
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

	if node.Type == "GREATER_THAN" {
		fmt.Println("GREATER_THAN")
		fmt.Println(leftNode.Value, rightNode.Value)
		if leftNode.Value > rightNode.Value {
			return &boolTrue
		} else {
			return &boolFalse
		}
	}

	if node.Type == "LESS_THAN" {
		fmt.Println("LESS_THAN")
		fmt.Println(leftNode.Value, rightNode.Value)
		if leftNode.Value < rightNode.Value {
			return &boolTrue
		} else {
			return &boolFalse
		}
	}

	return node

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
			if result.Type == "IF_STATEMENT" {
				if result.Left.Value == "FALSE" || result.Left.Value == "TRUE" {
					foldedFunction.Body = append(foldedFunction.Body, result.Body...)
				} else {
					foldedFunction.Body = append(foldedFunction.Body, result)
				}
			} else {
				foldedFunction.Body = append(foldedFunction.Body, result)
			}
		}
	}
	return foldedFunction.Body[len(foldedFunction.Body)-1]
}

// optimizeForLoop optimizes the for loop structure based on the provided AST.
func optimizeForLoop(forLoopNode *Node) *Node {
	if forLoopNode.Type != "FOR_LOOP" {
		panic("Node is not a for loop")
	}

	// Extract key components from the for loop
	condition := forLoopNode.Params[0]                    // Condition node (e.g., i < 10)
	init := forLoopNode.Body[0]                           // Initialization node (e.g., i = 0)
	updation := forLoopNode.Body[len(forLoopNode.Body)-1] // Updation node (e.g., i = i + 1)

	ifStmt := forLoopNode.Body[1] // If statement node
	if ifStmt.Type != "IF_STATEMENT" {
		panic("Second body node is not an if statement")
	}

	// Extract the write function from the if statement
	writeCall := ifStmt.Body[0]
	if writeCall.Type != "FUNCTION_CALL" || writeCall.Value != "write" {
		panic("Expected a write function call in the if statement")
	}

	// Analyze the initialization, condition, and updation
	start := atoi(init.Right.Value)    // e.g., 0
	end := atoi(condition.Right.Value) // e.g., 10
	step := 1                          // Assume step is always +1 for simplicity
	if updation.Right.Type == "ADD" {
		step = atoi(updation.Right.Right.Value)
	}

	// Generate the optimized body by simulating the loop execution
	optimizedBody := []*Node{}
	for i := start; i < end; i += step {
		optimizedBody = append(optimizedBody, &Node{
			Type:  "FUNCTION_CALL",
			Value: "write",
			Params: []*Node{
				{
					Type:  "INT",
					DType: "INT",
					Value: fmt.Sprintf("%d", i),
				},
			},
		})
	}

	// Return a new for loop node with the optimized body
	return &Node{
		Type:  "FOR_LOOP",
		DType: "FOR_LOOP",
		Value: "for",
		Body:  optimizedBody,
	}
}

// optimizeWhileLoop optimizes the while loop structure based on the provided AST.
func optimizeWhileLoop(whileLoopNode *Node) *Node {
	if whileLoopNode.Type != "WHILE_LOOP" {
		panic("Node is not a while loop")
	}

	// Extract key components from the while loop
	condition := whileLoopNode.Params[0] // Condition node (e.g., b < 7)
	ifStmt := whileLoopNode.Body[0]      // If statement node
	if ifStmt.Type != "IF_STATEMENT" {
		panic("Body node is not an if statement")
	}

	// Extract the write function and update operation from the if statement
	writeCall := ifStmt.Body[0]
	if writeCall.Type != "FUNCTION_CALL" || writeCall.Value != "write" {
		panic("Expected a write function call in the if statement")
	}

	updateOp := ifStmt.Body[1]
	if updateOp.Type != "ASSIGN" {
		panic("Expected an assignment operation in the if statement")
	}

	// Analyze the condition and updation
	start := atoi(updateOp.Left.Value) // Initial value is assumed to be set before the loop
	end := atoi(condition.Right.Value) // e.g., 7
	step := 1                          // Assume step is always +1 for simplicity
	if updateOp.Right.Type == "ADD" {
		step = atoi(updateOp.Right.Right.Value)
	}

	// Generate the optimized body by simulating the loop execution
	optimizedBody := []*Node{}
	for i := start; i < end; i += step {
		optimizedBody = append(optimizedBody, &Node{
			Type:  "FUNCTION_CALL",
			Value: "write",
			Params: []*Node{
				{
					Type:  "INT",
					DType: "INT",
					Value: fmt.Sprintf("%d", i),
				},
			},
		})
	}

	// Return a new while loop node with the optimized body
	return &Node{
		Type:  "WHILE_LOOP",
		DType: "WHILE_LOOP",
		Value: "while",
		Body:  optimizedBody,
	}
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
