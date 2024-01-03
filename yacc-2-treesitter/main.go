package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Please invoke with following arguments: languageName inputFile outputFile")
		os.Exit(0)
	}

	languageName := os.Args[1]
	inputFile := os.Args[2]
	outputFile := os.Args[3]

	converted, err := convert(languageName, inputFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(outputFile, []byte(converted), 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		os.Exit(1)
	}
}

var optionals []string

func convert(languageName, inputFile string) (string, error) {
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return "", err
	}

	contentString := string(content)
	rulesSectionStart := strings.Index(contentString, "%%")
	rulesSectionEnd := strings.LastIndex(contentString, "%%")

	if rulesSectionStart == -1 || rulesSectionEnd == -1 {
		return "", fmt.Errorf("Rules section not found")
	}

	content2 := contentString[rulesSectionStart+2 : rulesSectionEnd]
	content3 := removeSemanticActions(strings.TrimSpace(content2))

	var builder strings.Builder
	builder.WriteString(c3PrecedenceRules())
	builder.WriteString(
		fmt.Sprintf(
			"module.exports = grammar({\n"+
				"    name: '%s',\n"+
				"\n"+
				"    rules: {",
			languageName,
		),
	)

	rules := splitByDelimiter(content3, ";")
	for _, rule := range rules {
		if strings.TrimSpace(rule) == "" {
			continue
		}

		has := strings.HasPrefix(strings.TrimSpace(rule), "range_expr")
		if has {
			println("lele")
		}
		splitRule := splitByDelimiter(strings.TrimSpace(rule), ":")

		ruleName := strings.TrimSpace(splitRule[0])
		ruleBranches := splitByDelimiter(splitRule[1], "|")

		if len(ruleBranches) == 0 {
			fmt.Printf("Rule %s has no branches\n", ruleName)
		}

		var formedRule string
		if len(ruleBranches) == 1 {
			formedRule = formOneBranchRule(ruleName, ruleBranches)
		} else {
			formedRule = formManyBranchRule(ruleName, ruleBranches)
		}

		println(formedRule)
		builder.WriteString(formedRule)
	}

	builder.WriteString("}\n});")

	return postProcess(builder.String()), nil
}

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
	semanticActionsRegex := regexp.MustCompile(`\{(.|\n)+?\}`)
	commentsRegex := regexp.MustCompile(`(//.*?\n|/\*(.|\n)*?\*/)`)

	result := semanticActionsRegex.ReplaceAllString(rule, "")
	result = commentsRegex.ReplaceAllString(result, "")

	return result
}

func formOneBranchRule(ruleName string, ruleBranches []string) string {
	var builder strings.Builder
	builder.WriteString(makeHeader(ruleName))

	branch := strings.Fields(ruleBranches[0])
	if len(branch) == 1 {
		builder.WriteString("$.")
		builder.WriteString(branch[0])
		builder.WriteString(",")
	} else {
		builder.WriteString(processBranch(branch))
	}
	builder.WriteString("\n")
	return builder.String()
}

func formManyBranchRule(ruleName string, ruleBranches []string) string {
	var builder strings.Builder
	builder.WriteString(makeHeader(ruleName))

	actuallyMoreThanOneBranch := countNonEmptyStrings(ruleBranches) > 1
	if actuallyMoreThanOneBranch {
		builder.WriteString("choice(\n")
	}

	for _, branch := range ruleBranches {
		if strings.TrimSpace(branch) == "" {
			optionals = append(optionals, ruleName)
		} else {
			builder.WriteString(processBranch(strings.Fields(branch)))
		}
	}

	if actuallyMoreThanOneBranch {
		builder.WriteString("),\n\n")
	}

	return builder.String()
}

func makeHeader(ruleName string) string {
	return fmt.Sprintf("%s: $ => ", ruleName)
}

func processBranch(branch []string) string {
	var builder strings.Builder
	if len(branch) > 1 {
		builder.WriteString("seq(\n")
	}
	for _, element := range branch {
		if !strings.HasPrefix(element, "'") || !strings.HasSuffix(element, "'") {
			builder.WriteString("$.")
		}
		builder.WriteString(element)
		builder.WriteString(",\n")
	}
	if len(branch) > 1 {
		builder.WriteString("),\n")
	}
	return builder.String()
}

func postProcess(output string) string {
	newOutput := output
	for _, optionalRule := range optionals {
		newOutput = strings.Replace(newOutput, "$."+optionalRule+",", "optional($."+optionalRule+"),", -1)
	}

	return newOutput
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

func c3PrecedenceRules() string {
	prec := `
const PREC = {
  // () [] . ++ --
  postfix: 10,
  // @ * & && ~ ! + - ++ --
  prefix: 9,
  // * / % *%
  multiplicative: 8,
  // << >>
  shift: 7,
  // ^ | &
  bitwise: 6,
  // + -
  additive: 5,
  // == != >= <= > <
  comparative: 4,
  // &&
  and: 3,
  // ||
  or: 2,
  // ?:
  ternary: 1,
  // == *= /= %= *%= += -= <<= >>= &= ^= |=
  assign: 0,
};
`
	return prec
}
