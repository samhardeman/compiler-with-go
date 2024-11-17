package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Represents a simple TAC instruction
type TacInstruction struct {
	op     string
	arg1   string
	arg2   string
	result string
}

// Parses TAC input lines into TacInstruction structs
func parseTAC(lines []string) []TacInstruction {
	var instructions []TacInstruction

	// Regular expression to split the line into tokens, ignoring spaces inside quotes
	re := regexp.MustCompile(`"(.*?)"|\S+`) // Match anything inside quotes or non-space characters

	for _, line := range lines {
		// Find all matches using the regex
		tokens := re.FindAllString(line, -1)
		if len(tokens) == 0 {
			continue
		}

		switch tokens[0] {
		case "if":
			// Format: if x goto L1
			// Ensure the tokens match the expected pattern
			if len(tokens) >= 4 && tokens[2] == "goto" {
				instructions = append(instructions, TacInstruction{
					op:     "if",
					arg1:   tokens[1], // Condition variable or expression
					result: tokens[3], // Label to jump to
				})
			} else {
				fmt.Printf("Invalid 'if' instruction format: %v\n", tokens)
				os.Exit(1)
			}
		case "goto":
			// Format: goto L2
			if len(tokens) >= 2 {
				instructions = append(instructions, TacInstruction{
					op:     "goto",
					result: tokens[1], // Label to jump to
				})
			} else {
				fmt.Printf("Invalid 'goto' instruction format: %v\n", tokens)
				os.Exit(1)
			}
		default:
			// Handle labels ending with ':'
			if strings.HasSuffix(tokens[0], ":") {
				// Remove the ':' from the label name
				label := strings.TrimSuffix(tokens[0], ":")
				instructions = append(instructions, TacInstruction{
					op:     "label",
					result: label,
				})
			} else if len(tokens) == 3 && tokens[1] == "=" {
				// Handle assignments: var = value
				instructions = append(instructions, TacInstruction{
					op:     "=",
					arg1:   tokens[2],
					result: tokens[0],
				})
			} else if tokens[0] == "call" {
				// Handle function calls: call function arg
				if len(tokens) >= 3 {
					instructions = append(instructions, TacInstruction{
						op:   "call",
						arg1: tokens[1], // Function name
						arg2: tokens[2], // Argument
					})
				} else {
					fmt.Printf("Invalid 'call' instruction format: %v\n", tokens)
					os.Exit(1)
				}
			}
		}
	}
	return instructions
}

// Determines the type of the argument for MIPS storage
func determineType(value string) string {
	// Check for strings
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return "string"
	} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return "char"
	} else if value == "True" || value == "False" {
		return "bool"
	} else if _, err := strconv.ParseFloat(value, 64); err == nil {
		// If the value is a valid float, it's a float type, unless it's an integer without a decimal
		if strings.Contains(value, ".") {
			return "float"
		}
		return "int" // If it's an integer, we'll consider it as int
	}
	return "unknown"
}

// Generate MIPS code from parsed TAC instructions
func generateMIPS(instructions []TacInstruction) string {
	var mipsCode strings.Builder

	// Maps to keep track of variables and string literals
	variables := make(map[string]bool)
	stringLiterals := make(map[string]string)
	labelCounter := 0

	// First pass: Collect variables and string literals
	for _, instr := range instructions {
		switch instr.op {
		case "=", "+", "-", "*", "/":
			// Collect variables used in assignments and arithmetic operations
			if isIdentifier(instr.result) {
				variables[instr.result] = true
			}
			if isIdentifier(instr.arg1) {
				variables[instr.arg1] = true
			}
			if isIdentifier(instr.arg2) {
				variables[instr.arg2] = true
			}
		case "call":
			// Collect variables and string literals in function calls
			if instr.arg1 == "write" {
				arg := instr.arg2
				if len(arg) > 0 && arg[0] == '"' {
					// String literal
					if _, exists := stringLiterals[arg]; !exists {
						label := fmt.Sprintf("str_%d", labelCounter)
						labelCounter++
						stringLiterals[arg] = label
					}
				} else if isIdentifier(arg) {
					variables[arg] = true
				}
			}
		case "if":
			// Collect variables used in conditions
			if isIdentifier(instr.arg1) {
				variables[instr.arg1] = true
			}
		}
	}

	// Start .data section
	mipsCode.WriteString(".data\n")

	// Declare variables
	for varName := range variables {
		mipsCode.WriteString(fmt.Sprintf("%s: .word 0\n", varName))
	}

	// Declare string literals
	for strVal, label := range stringLiterals {
		mipsCode.WriteString(fmt.Sprintf("%s: .asciiz %s\n", label, strVal))
	}

	// Start .text section
	mipsCode.WriteString("\n.text\n")
	mipsCode.WriteString("main:\n")

	// Generate MIPS code for each instruction
	for _, instr := range instructions {
		switch instr.op {
		case "=":
			// Assignment operation
			if len(instr.arg1) > 0 && instr.arg1[0] == '"' {
				// String assignment
				label := stringLiterals[instr.arg1]
				mipsCode.WriteString(fmt.Sprintf("la $t0, %s\n", label))
				mipsCode.WriteString(fmt.Sprintf("sw $t0, %s\n", instr.result))
			} else if isNumeric(instr.arg1) {
				// Immediate value assignment
				mipsCode.WriteString(fmt.Sprintf("li $t0, %s\n", instr.arg1))
				mipsCode.WriteString(fmt.Sprintf("sw $t0, %s\n", instr.result))
			} else {
				// Assignment from another variable
				mipsCode.WriteString(fmt.Sprintf("lw $t0, %s\n", instr.arg1))
				mipsCode.WriteString(fmt.Sprintf("sw $t0, %s\n", instr.result))
			}
		case "+", "-", "*", "/":
			// Arithmetic operations
			// Load operands
			loadOperand(instr.arg1, "$t0", &mipsCode)
			loadOperand(instr.arg2, "$t1", &mipsCode)
			// Perform operation
			switch instr.op {
			case "+":
				mipsCode.WriteString("add $t2, $t0, $t1\n")
			case "-":
				mipsCode.WriteString("sub $t2, $t0, $t1\n")
			case "*":
				mipsCode.WriteString("mul $t2, $t0, $t1\n")
			case "/":
				mipsCode.WriteString("div $t0, $t1\n")
				mipsCode.WriteString("mflo $t2\n")
			}
			// Store result
			mipsCode.WriteString(fmt.Sprintf("sw $t2, %s\n", instr.result))
		case "call":
			if instr.arg1 == "write" {
				// Handle 'write' function call
				arg := instr.arg2
				if len(arg) > 0 && arg[0] == '"' {
					// String literal
					label := stringLiterals[arg]
					mipsCode.WriteString("li $v0, 4\n")
					mipsCode.WriteString(fmt.Sprintf("la $a0, %s\n", label))
					mipsCode.WriteString("syscall\n")
				} else if isNumeric(arg) {
					// Immediate integer
					mipsCode.WriteString("li $v0, 1\n")
					mipsCode.WriteString(fmt.Sprintf("li $a0, %s\n", arg))
					mipsCode.WriteString("syscall\n")
				} else {
					// Variable
					mipsCode.WriteString(fmt.Sprintf("lw $a0, %s\n", arg))
					mipsCode.WriteString("li $v0, 1\n")
					mipsCode.WriteString("syscall\n")
				}
			}
		case "if":
			// Conditional branch
			loadOperand(instr.arg1, "$t0", &mipsCode)
			mipsCode.WriteString(fmt.Sprintf("bne $t0, $zero, %s\n", instr.result))
		case "goto":
			// Unconditional jump
			mipsCode.WriteString(fmt.Sprintf("j %s\n", instr.result))
		case "label":
			// Label
			mipsCode.WriteString(fmt.Sprintf("%s:\n", instr.result))
		}
	}

	// Add program termination
	mipsCode.WriteString("\nli $v0, 10\nsyscall\n") // Exit program

	// End of program
	mipsCode.WriteString("\n# End of program\n")

	return mipsCode.String()
}

// Reads lines from a file and returns them as a slice of strings
func readTac(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// Writes the generated MIPS code to a .mips file
func writeMIPSCodeToFile(filename, code string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(code)
	return err
}

func tac2Mips(filename string) {
	// Read TAC instructions from a file called output.tac
	tacFile := filename
	lines, err := readTac(tacFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse TAC and generate MIPS code
	tacInstructions := parseTAC(lines)
	mipsCode := generateMIPS(tacInstructions)

	// Output MIPS code to a .mips file
	outputFile := "output.mips"
	err = writeMIPSCodeToFile(outputFile, mipsCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing MIPS code to file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("MIPS code has been written to %s\n", outputFile)
}

// Checks if a string represents a numeric constant
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// Loads an operand (variable or immediate value) into a register
func loadOperand(operand string, register string, mipsCode *strings.Builder) {
	if isNumeric(operand) {
		// Immediate value
		mipsCode.WriteString(fmt.Sprintf("li %s, %s\n", register, operand))
	} else {
		// Variable
		mipsCode.WriteString(fmt.Sprintf("lw %s, %s\n", register, operand))
	}
}
