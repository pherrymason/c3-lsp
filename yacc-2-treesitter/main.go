package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Please invoke with following arguments: yaccInputFile lexxInputFile outputFile")
		os.Exit(0)
	}

	yaccInputFile := os.Args[1]
	lexxInputFile := os.Args[2]
	outputFile := os.Args[3]

	lexxConverted, err := transformLexx(lexxInputFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	converted, err := transformYacc(yaccInputFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	writeTreeSitterGrammar(lexxConverted, converted, err, outputFile)
}

func writeTreeSitterGrammar(lexxConverted string, converted string, err error, outputFile string) {
	var builder strings.Builder
	builder.WriteString(grammarHeaderContent())
	builder.WriteString(lexxConverted)
	builder.WriteString(converted)
	builder.WriteString("\t}\n});")

	err = ioutil.WriteFile(outputFile, []byte(builder.String()), 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		os.Exit(1)
	}
}

var optionals []string

func splitByDelimiter(input string, delimiter string) []string {
	var result []string
	var currentRule string
	var isInQuotes bool

	for i := 0; i < len(input); i++ {
		if strings.HasPrefix(input[i:], delimiter) && !isInQuotes {
			result = append(result, strings.TrimSpace(currentRule))
			currentRule = ""
			i += len(delimiter) - 1 // skip delimiter
		} else if input[i] == '\'' {
			// Changes state of isInQuotes when finding quotes
			isInQuotes = !isInQuotes
			currentRule += string(input[i])
		} else {
			currentRule += string(input[i])
		}
	}

	result = append(result, strings.TrimSpace(currentRule))

	return result
}

func removeSemanticActions(rule string) string {
	semanticActionsRegex := regexp.MustCompile(`(?:^|[^'"])(\{(?:.|\n)+?\})(?:[^'"]|$)`)
	commentsRegex := regexp.MustCompile(`(//.*?\n|/\*(.|\n)*?\*/)`)

	result := semanticActionsRegex.ReplaceAllString(rule, "")
	result = commentsRegex.ReplaceAllString(result, "")

	return result
}

func makeHeader(ruleName string) string {
	return fmt.Sprintf("%s: $ => ", ruleName)
}

func countNonEmptyStrings(text []string) int {
	count := 0
	for _, str := range text {
		if strings.TrimSpace(str) != "" {
			count++
		}
	}
	return count
}

func grammarHeaderContent() string {
	content, err := ioutil.ReadFile("./grammar.header.js")
	if err != nil {
		panic("Could not read grammar.header.js file")
	}

	return string(content)
}
