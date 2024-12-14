package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Represents a simple TAC instruction
type TacInstruction struct {
	op     string // Operation type (=, blt, jump, call, etc.)
	arg1   string // First argument
	arg2   string // Second argument (if needed)
	result string // Result or target label
	label  bool   // Is this a label declaration?
}

// Parses TAC input lines into TacInstruction structs
func parseTAC(lines []string) []TacInstruction {
	var instructions []TacInstruction

	for _, line := range lines {
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		switch {
		case len(tokens) == 3 && tokens[1] == "=":
			instructions = append(instructions, TacInstruction{
				op:     "=",
				arg1:   tokens[2],
				result: tokens[0],
			})

		case tokens[0] == "bgt":
			instructions = append(instructions, TacInstruction{
				op:     "bgt",
				arg1:   tokens[1],
				arg2:   tokens[2],
				result: tokens[3],
			})

		case tokens[0] == "blt":
			instructions = append(instructions, TacInstruction{
				op:     "blt",
				arg1:   tokens[1],
				arg2:   tokens[2],
				result: tokens[3],
			})

		case tokens[0] == "jump":
			instructions = append(instructions, TacInstruction{
				op:   "jump",
				arg1: tokens[1],
			})

		case strings.HasSuffix(tokens[0], ":"):
			instructions = append(instructions, TacInstruction{
				op:   "label",
				arg1: strings.TrimSuffix(tokens[0], ":"),
			})

		case tokens[0] == "call":
			instructions = append(instructions, TacInstruction{
				op:   "call",
				arg1: tokens[1],
				arg2: tokens[2],
			})
		}
	}
	return instructions
}

// Update generateMIPS to handle both blt and bgt:
func generateMIPS(instructions []TacInstruction) string {
	var mipsCode strings.Builder
	declaredVariables := make(map[string]bool)

	mipsCode.WriteString(".data\n")
	for _, instr := range instructions {
		if instr.op == "=" {
			// Declare variables if not already declared
			if !declaredVariables[instr.result] {
				declaredVariables[instr.result] = true

				// Determine the type and declare appropriately
				varType := determineTypeFromVar(instr.result)
				if varType == "STRING" {
					// Declare string with placeholder or value
					mipsCode.WriteString(fmt.Sprintf("%s: .asciiz %s\n", instr.result, instr.arg1))
				} else {
					// Declare integers
					mipsCode.WriteString(fmt.Sprintf("%s: .word 0\n", instr.result))
				}
			}
		}
	}

	mipsCode.WriteString("\n.text\nmain:\n")

	for _, instr := range instructions {
		switch instr.op {
		case "=":
			// Handle assignments
			varType := determineTypeFromVar(instr.result)
			if varType == "STRING" {
				// Handle string assignment (directly store the address of the string)
				// Strings are already declared with .asciiz; no runtime assignment is needed.
			} else {
				// Handle integer assignment
				if isNumeric(instr.arg1) {
					// Immediate assignment
					mipsCode.WriteString(fmt.Sprintf("    li $t0, %s\n", instr.arg1))
					mipsCode.WriteString(fmt.Sprintf("    sw $t0, %s\n", instr.result))
				} else {
					// Assignment from another variable
					mipsCode.WriteString(fmt.Sprintf("    lw $t0, %s\n", instr.arg1))
					mipsCode.WriteString(fmt.Sprintf("    sw $t0, %s\n", instr.result))
				}
			}

		case "blt":
			mipsCode.WriteString(fmt.Sprintf("    lw $t0, %s\n", instr.arg1))
			mipsCode.WriteString(fmt.Sprintf("    lw $t1, %s\n", instr.arg2))
			mipsCode.WriteString(fmt.Sprintf("    blt $t0, $t1, %s\n", instr.result))

		case "bgt":
			mipsCode.WriteString(fmt.Sprintf("    lw $t0, %s\n", instr.arg1))
			mipsCode.WriteString(fmt.Sprintf("    lw $t1, %s\n", instr.arg2))
			mipsCode.WriteString(fmt.Sprintf("    bgt $t0, $t1, %s\n", instr.result))

		case "jump":
			mipsCode.WriteString(fmt.Sprintf("    j %s\n", instr.arg1))

		case "label":
			mipsCode.WriteString(fmt.Sprintf("%s:\n", instr.arg1))

		case "call":
			if instr.arg1 == "write" {
				// Determine the variable's type for printing
				varType := determineTypeFromVar(instr.arg2)
				if varType == "STRING" {
					// Print string
					mipsCode.WriteString("    li $v0, 4\n")                           // Print string syscall
					mipsCode.WriteString(fmt.Sprintf("    la $a0, %s\n", instr.arg2)) // Load address of string
					mipsCode.WriteString("    syscall\n")
				} else {
					// Print integer
					mipsCode.WriteString("    li $v0, 1\n")                           // Print integer syscall
					mipsCode.WriteString(fmt.Sprintf("    lw $a0, %s\n", instr.arg2)) // Load integer value
					mipsCode.WriteString("    syscall\n")
				}
			}
		}
	}

	mipsCode.WriteString("\n    li $v0, 10\n    syscall\n")
	mipsCode.WriteString("\n# End of program\n")

	return mipsCode.String()
}

// Helper function to check if a string is numeric (an integer)
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Extracts the type from a variable name, e.g., "opt_t1_STRING" -> "STRING"
func extractTypeFromVar(varName string) string {
	parts := strings.Split(varName, "_")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// Determines the type of the argument for MIPS storage based on the tempVar type
func determineTypeFromVar(tempVar string) string {
	return extractTypeFromVar(tempVar)
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
		line := scanner.Text()
		if line != "" { // Skip empty lines
			lines = append(lines, line)
		}
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
