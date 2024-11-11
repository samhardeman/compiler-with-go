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

		// Handle TAC format: var = value or call function arg
		if len(tokens) == 3 && tokens[1] == "=" {
			instructions = append(instructions, TacInstruction{
				op:     "=",
				arg1:   tokens[2],
				result: tokens[0],
			})
		} else if tokens[0] == "call" {
			instructions = append(instructions, TacInstruction{
				op:   "call",
				arg1: tokens[1],
				arg2: tokens[2],
			})
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

	// Start .data section
	mipsCode.WriteString(".data\n")

	// Store variables in .data section
	for _, instr := range instructions {
		switch {
		case instr.arg1[0] == '"' || instr.arg1[0] == '"':
			// Handle string literals
			mipsCode.WriteString(fmt.Sprintf("%s: .asciiz %s\n", instr.result, instr.arg1))
		case instr.arg1[0] == '\'':
			// Handle char literals (single quotes)
			mipsCode.WriteString(fmt.Sprintf("%s: .byte %s\n", instr.result, instr.arg1))
		case instr.arg1 == "True" || instr.arg1 == "False":
			// Handle boolean values (1 for true, 0 for false)
			boolVal := 0
			if instr.arg1 == "True" {
				boolVal = 1
			}
			mipsCode.WriteString(fmt.Sprintf("%s: .word %d\n", instr.result, boolVal))
		case instr.arg1[0] >= '0' && instr.arg1[0] <= '9' || instr.arg1[0] == '-':
			// Handle integer or float numbers
			if strings.Contains(instr.arg1, ".") {
				// Handle float numbers
				mipsCode.WriteString(fmt.Sprintf("%s: .float %s\n", instr.result, instr.arg1))
			} else {
				// Handle integer numbers
				mipsCode.WriteString(fmt.Sprintf("%s: .word %s\n", instr.result, instr.arg1))
			}
		}
	}

	// Start .text section
	mipsCode.WriteString("\n.text\n")
	mipsCode.WriteString("main:\n")

	// Handle syscalls (call write)
	for _, instr := range instructions {
		if instr.op == "call" && instr.arg1 == "write" {
			// Handle writing the correct variable
			if instr.arg2[0] == '"' || instr.arg2[0] == '"' {
				// Write a string
				mipsCode.WriteString(fmt.Sprintf("li $v0, 4\nla $a0, %s\nsyscall\n", instr.arg2))
			} else if instr.arg2[0] == '\'' {
				// Write a character
				mipsCode.WriteString(fmt.Sprintf("li $v0, 11\nlb $a0, %s\nsyscall\n", instr.arg2))
			} else if instr.arg2 == "True" || instr.arg2 == "False" {
				// Print boolean (int 1 or 0)
				mipsCode.WriteString(fmt.Sprintf("li $v0, 1\nlw $a0, %s\nsyscall\n", instr.arg2))
			} else if strings.Contains(instr.arg2, ".") {
				// Print float
				mipsCode.WriteString(fmt.Sprintf("li $v0, 2\nl.s $f12, %s\nsyscall\n", instr.arg2))
			} else {
				// Print integer
				mipsCode.WriteString(fmt.Sprintf("li $v0, 1\nlw $a0, %s\nsyscall\n", instr.arg2))
			}
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
