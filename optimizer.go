package main

import (
	"fmt"
	"strconv"
)

// Optimizer struct to hold optimization-related methods.
type Optimizer struct {
	constants map[string]int  // Holds propagated constants
	usedVars  map[string]bool // Tracks variable usage for dead code elimination
}

// NewOptimizer initializes the optimizer with maps for constants and variable usage.
func NewOptimizer() *Optimizer {
	return &Optimizer{
		constants: make(map[string]int),
		usedVars:  make(map[string]bool),
	}
}

// Optimize performs constant folding, propagation, and dead code elimination.
func (opt *Optimizer) Optimize(root Node) Node {
	// Perform the three optimizations
	opt.constantPropagation(&root)
	opt.constantFolding(&root)
	opt.deadCodeElimination(&root)
	return root
}

// constantPropagation replaces variables with their constant values.
func (opt *Optimizer) constantPropagation(node *Node) {
	if node == nil {
		return
	}

	// If it's an assignment, check if we can propagate constants
	if node.Type == "ASSIGN" && node.Left != nil && node.Right != nil {
		if node.Right.Type == "NUMBER" {
			// Assigning a constant value to a variable
			val, _ := strconv.Atoi(node.Right.Value)
			opt.constants[node.Left.Value] = val
		}
	} else if node.Type == "IDENTIFIER" {
		// If the node is a variable, replace it if we have a constant for it
		if val, exists := opt.constants[node.Value]; exists {
			node.Type = "NUMBER"
			node.DType = "INT"
			node.Value = strconv.Itoa(val)
		}
	}

	// Recur on left and right children
	opt.constantPropagation(node.Left)
	opt.constantPropagation(node.Right)
	for _, child := range node.Body {
		opt.constantPropagation(child)
	}
}

// constantFolding evaluates constant expressions and replaces them with a single value.
func (opt *Optimizer) constantFolding(node *Node) {
	if node == nil {
		return
	}

	// If node is an ADD, try to fold it if both operands are constants
	if node.Type == "ADD" && node.Left != nil && node.Right != nil {
		if node.Left.Type == "NUMBER" && node.Right.Type == "NUMBER" {
			// Perform the addition
			leftVal, _ := strconv.Atoi(node.Left.Value)
			rightVal, _ := strconv.Atoi(node.Right.Value)
			node.Type = "NUMBER"
			node.DType = "INT"
			node.Value = strconv.Itoa(leftVal + rightVal)
			node.Left = nil
			node.Right = nil
		}
	}

	// Recur on left and right children
	opt.constantFolding(node.Left)
	opt.constantFolding(node.Right)
	for _, child := range node.Body {
		opt.constantFolding(child)
	}
}

// deadCodeElimination removes dead assignments by checking variable usage.
func (opt *Optimizer) deadCodeElimination(node *Node) {
	if node == nil {
		return
	}

	// Mark variables in function calls or return values as used
	if node.Type == "FUNCTION_CALL" || node.Type == "RETURN" {
		for _, param := range node.Params {
			opt.usedVars[param.Value] = true
		}
	}

	// Check assignments; if a variable is never used, mark as dead
	if node.Type == "ASSIGN" && node.Left != nil && node.Left.Type == "IDENTIFIER" {
		varName := node.Left.Value
		if !opt.usedVars[varName] {
			node.Type = "NOP" // Mark as no-op to indicate dead code
		}
		// Mark right side variables as used if they are identifiers
		if node.Right != nil && node.Right.Type == "IDENTIFIER" {
			opt.usedVars[node.Right.Value] = true
		}
	}

	// Recur on children and update used variables map
	opt.deadCodeElimination(node.Left)
	opt.deadCodeElimination(node.Right)
	for _, child := range node.Body {
		opt.deadCodeElimination(child)
	}
}

// printAST prints the AST for debugging.
// Adapted printNode function for displaying the optimized AST
func printAST(node *Node, prefix string, isTail bool) {
	// Construct the current node's details (Type, DType, Value)
	nodeRepresentation := fmt.Sprintf("%s [Type: %s, DType: %s, Value: %s]", node.Value, node.Type, node.DType, node.Value)

	// Print the node, with graphical tree branches (└── or ├──)
	fmt.Printf("%s%s%s\n", prefix, getBranch(isTail), nodeRepresentation)

	// Prepare the prefix for children
	newPrefix := prefix
	if isTail {
		newPrefix += "    " // For the last child, indent
	} else {
		newPrefix += "│   " // For other children, continue the branch
	}

	// Handle Params, if any
	if len(node.Params) > 0 {
		fmt.Println(newPrefix + "Params:")
		for i := 0; i < len(node.Params); i++ {
			printNode(node.Params[i], newPrefix, i == len(node.Params)-1)
		}
	}

	// Handle Body, if any
	if len(node.Body) > 0 {
		fmt.Println(newPrefix + "Body:")
		for i := 0; i < len(node.Body); i++ {
			printNode(node.Body[i], newPrefix, i == len(node.Body)-1)
		}
	}

	// Handle Left and Right children (for operations like +, -, *, /)
	if node.Left != nil {
		fmt.Println(newPrefix + "Left:")
		printNode(node.Left, newPrefix, false)
	}
	if node.Right != nil {
		fmt.Println(newPrefix + "Right:")
		printNode(node.Right, newPrefix, true)
	}
}
