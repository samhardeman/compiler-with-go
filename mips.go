package main

import (
	"bufio"
	"fmt"
	"os"
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
	for _, line := range lines {
		tokens := strings.Fields(line)

		// Handle TAC format: var = value or call function arg
		if len(tokens) == 3 && tokens[1] == "=" {
			instructions = append(instructions, TacInstruction{
				op:     "=",
				arg1:   tokens[2],
				result: tokens[0],
			})
		} else if len(tokens) == 4 && tokens[0] == "call" {
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
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return "string"
	} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return "char"
	} else if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "float"
	} else if value == "True" || value == "False" {
		return "bool"
	} else if _, err := strconv.Atoi(value); err == nil {
		return "int"
	}
	return "unknown"
}

// Generates MIPS assembly code from TAC instructions
func generateMIPS(tac []TacInstruction) string {
	var mipsCode strings.Builder

	// Declare a data section for variables
	mipsCode.WriteString(".data\n")
	variableTypes := make(map[string]string)

	// Process TAC instructions
	for _, instr := range tac {
		switch instr.op {
		case "=":
			// Detect type and create variable in MIPS data section
			valueType := determineType(instr.arg1)
			variableTypes[instr.result] = valueType

			switch valueType {
			case "string":
				value := instr.arg1[1 : len(instr.arg1)-1] // Remove quotes
				mipsCode.WriteString(fmt.Sprintf("%s: .asciiz \"%s\"\n", instr.result, value))

			case "char":
				charValue := instr.arg1[1 : len(instr.arg1)-1] // Remove single quotes
				mipsCode.WriteString(fmt.Sprintf("%s: .byte '%s'\n", instr.result, charValue))

			case "int":
				mipsCode.WriteString(fmt.Sprintf("%s: .word %s\n", instr.result, instr.arg1))

			case "float":
				mipsCode.WriteString(fmt.Sprintf("%s: .float %s\n", instr.result, instr.arg1))

			case "bool":
				boolValue := "0"
				if instr.arg1 == "True" {
					boolValue = "1"
				}
				mipsCode.WriteString(fmt.Sprintf("%s: .word %s\n", instr.result, boolValue))
			}

		case "call":
			// Handle write calls based on type
			mipsCode.WriteString("\n.text\n")

			if instr.arg1 == "write" {
				varType, exists := variableTypes[instr.arg2]
				if !exists {
					mipsCode.WriteString(fmt.Sprintf("# Error: Variable %s not defined\n", instr.arg2))
					continue
				}

				switch varType {
				case "string":
					mipsCode.WriteString("li $v0, 4\n")                           // syscall for printing string
					mipsCode.WriteString(fmt.Sprintf("la $a0, %s\n", instr.arg2)) // load address of string

				case "char":
					mipsCode.WriteString("li $v0, 11\n")                          // syscall for printing char
					mipsCode.WriteString(fmt.Sprintf("lb $a0, %s\n", instr.arg2)) // load byte (char)

				case "int":
					mipsCode.WriteString("li $v0, 1\n")                           // syscall for printing integer
					mipsCode.WriteString(fmt.Sprintf("lw $a0, %s\n", instr.arg2)) // load word (int)

				case "float":
					mipsCode.WriteString("li $v0, 2\n")                             // syscall for printing float
					mipsCode.WriteString(fmt.Sprintf("l.s $f12, %s\n", instr.arg2)) // load float

				case "bool":
					mipsCode.WriteString("li $v0, 1\n")                           // syscall for printing integer
					mipsCode.WriteString(fmt.Sprintf("lw $a0, %s\n", instr.arg2)) // load word (bool as int)
				}
				mipsCode.WriteString("syscall\n") // make syscall
			}
		}
	}
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
