package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"
)

type Node struct {
	class string
	dtype string
	value string
	left  *Node
	right *Node
}

type Symbol struct {
	dtype string
	value string
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
		doohickey(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func doohickey(line string) {
	root := []*Node{}

	tokens := splitString(line)

	format := ""

	fmt.Println(tokens)

	format = strings.TrimSpace(format)
	root = append(root, parser(tokens))

	if root[0].class == "ASSIGN" {
		fmt.Println(root[0].left.value + " = " + root[0].right.value)
	}
}

func HelperPreOrder(node *Node, processFunc func(v string)) {
	if node != nil {
		processFunc(node.value)
		HelperPreOrder(node.left, processFunc)
		HelperPreOrder(node.right, processFunc)
	}
}

func traverseAST(root []*Node) []string {
	var res []string
	for i := 0; i < len(root); i++ {
		processFunc := func(v string) {
			res = append(res, v)
		}
		HelperPreOrder(root[i], processFunc)
	}
	return res
}

func bisect(expression []string, character string, direction string) []string {
	index := slices.Index(expression, character)

	// Check if character is not found
	if index == -1 {
		fmt.Println("Character not found in expression.")
		os.Exit(3)
	}

	tokens := []string{}
	if direction == "right" {
		for i := index + 1; i < len(expression); i++ {
			tokens = append(tokens, expression[i])
		}
	} else if direction == "left" {
		for i := 0; i < index; i++ {
			tokens = append(tokens, expression[i])
		}
	} else {
		fmt.Println("Did not recognize direction: " + direction)
		os.Exit(3)
	}

	return tokens
}

func parser(tokens []string) *Node {

	var newNode Node

	numbers := strings.Split("1234567890", "")
	letters := strings.Split("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "")

	if slices.Contains(tokens, "int") {
		newNode = Node{
			class: "INITIALIZE",
			dtype: "STRING",
			value: "int",
		}

	} else if slices.Contains(tokens, "=") {
		newNode = Node{
			class: "ASSIGN",
			dtype: "CHAR",
			value: "=",
			left:  parser(bisect(tokens, "=", "left")),
			right: parser(bisect(tokens, "=", "right")),
		}
	} else if slices.Contains(tokens, "*") {
		newNode = Node{
			class: "MULT",
			dtype: "CHAR",
			value: "*",
			left:  parser(bisect(tokens, "*", "left")),
			right: parser(bisect(tokens, "*", "right")),
		}

	} else if slices.Contains(tokens, "/") {
		newNode = Node{
			class: "DIV",
			dtype: "CHAR",
			value: "/",
			left:  parser(bisect(tokens, "/", "left")),
			right: parser(bisect(tokens, "/", "right")),
		}

	} else if slices.Contains(tokens, "+") {
		newNode = Node{
			class: "ADD",
			dtype: "CHAR",
			value: "+",
			left:  parser(bisect(tokens, "+", "left")),
			right: parser(bisect(tokens, "+", "right")),
		}

	} else if slices.Contains(tokens, "-") {
		newNode = Node{
			class: "SUB",
			dtype: "CHAR",
			value: "-",
			left:  parser(bisect(tokens, "-", "left")),
			right: parser(bisect(tokens, "-", "right")),
		}

	} else if slices.Contains(numbers, tokens[0]) {
		newNode = Node{
			class: "NUMBER",
			dtype: "INT",
			value: tokens[0],
		}
	} else if slices.Contains(letters, tokens[0]) {
		newNode = Node{
			class: "IDENTIFIER",
			dtype: "CHAR",
			value: tokens[0],
		}

	} else if slices.Contains(tokens, ";") {

	} else {
		fmt.Println("Unrecognized character")
		os.Exit(3)

	}

	return &newNode
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
			left:  createNode([]string{expression[0]}, "IDENTIFIER"),
			right: createNode([]string{expression[2]}, "NUMBER"),
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

func splitString(input string) []string {
	pattern := regexp.MustCompile(`[a-zA-Z0-9]+|[(){}[\];,+\-*/%=<>!]`)
	return pattern.FindAllString(input, -1)
}
