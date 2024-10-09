package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
)

type Node struct {
	class string
	dtype string
	value string
	right *Node
	left  *Node
}

func main() {
	var inputFile string = getFlags()
	readLines(inputFile)
}

func getFlags() string {
	inputFile := flag.String("file", "", "")
	flag.Parse()
	if string(*inputFile) == "" {
		fmt.Printf("no file to compile provided")
		os.Exit(3)
	}
	return string(*inputFile)
}

func readLines(inputFile string) {
	// open file
	f, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	// remember to close the file at the end of the program
	defer f.Close()

	// read the file line by line using scanner
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		parser(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func parser(line string) {
	root := []*Node{}

	tokens := strings.Fields(line)

	types := []string{"int", "string", "float", "char"}
	equals := []string{"="}
	operators := []string{"+", "-", "*", "/"}
	numbers := strings.Split("1234567890", "")
	letters := strings.Split("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "")

	format := ""

	for i := 0; i < len(tokens); i++ {
		if slices.Contains(types, tokens[i]) {
			format = format + "TYPE "
		} else if slices.Contains(equals, tokens[i]) {
			format = format + "ASSIGN "
		} else if slices.Contains(numbers, tokens[i]) {
			format = format + "NUMBER "
		} else if slices.Contains(letters, tokens[i]) {
			format = format + "IDENTIFIER "
		} else if slices.Contains(operators, tokens[i]) {
			switch tokens[i] {
			case "+":
				format = format + "PLUS "
			case "-":
				format = format + "MINUS "
			case "*":
				format = format + "MULT "
			case "/":
				format = format + "DIV "
			}
		} else {
			fmt.Println("Unrecognized character please kys")
			os.Exit(3)
		}
	}
	format = strings.TrimSpace(format)
	root = append(root, createNode(tokens, format))

	if root[0].class == "ASSIGN" {
		fmt.Println(root[0].left.value + " = " + root[0].right.value)
	}
}

func createNode(expression []string, format string) *Node {
	var newNode Node
	switch format {
	case "TYPE IDENTIFIER":
		fmt.Println(format)
		newNode = Node{
			class: "IDENTIFIER",
			dtype: "CHAR",
			value: expression[1],
		}

	case "IDENTIFIER ASSIGN NUMBER":
		fmt.Println(format)
		newNode = Node{
			class: "ASSIGN",
			dtype: "CHAR",
			value: expression[1],
			right: createNode([]string{expression[2]}, "NUMBER"),
			left:  createNode([]string{expression[0]}, "IDENTIFIER"),
		}

	case "IDENTIFIER":
		fmt.Println(format)
		newNode = Node{
			class: "IDENTIFIER",
			dtype: "CHAR",
			value: expression[0],
		}

	case "NUMBER":
		fmt.Println(format)
		newNode = Node{
			class: "NUMBER",
			dtype: "INT",
			value: expression[0],
		}
	}

	return &newNode
}
