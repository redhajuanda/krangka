package stringx

import (
	"regexp"
	"strings"

	"github.com/gobeam/stringy"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func PascalCase(str string) string {
	specialCharacters := []string{
		"!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "-", "_", "=", "+", "", "|", "[", "]", "{", "}", ";", ":", "/", "?", ".", ">",
	}

	var replaceSpecialCharacters []string
	for _, sp := range specialCharacters {
		replaceSpecialCharacters = append(replaceSpecialCharacters, sp, " ")
	}

	rep := strings.NewReplacer(replaceSpecialCharacters...)
	str = rep.Replace(str)

	// strs := strings.Split(str, " ")
	//
	// var result string
	// for _, s := range strs {
	// 	if s == "" {
	// 		continue
	// 	}
	//
	// 	fmt.Println(s)
	// 	h := stringy.New(s).PascalCase().Get()
	// 	result += h
	// }

	return stringy.New(str).PascalCase(" ", "").Get()
	// return stringy.New(str).Title()
	// return result
}

func LowerCamelCase(input string) string {
	if len(input) == 0 {
		return input
	}

	// Convert the first character to lowercase
	result := strings.ToLower(string(input[0])) + input[1:]
	return result
}

func SnakeToCamelCase(snake string) string {
	words := strings.Split(snake, "_")
	for i, word := range words {
		words[i] = cases.Title(language.English).String(word)
	}
	return strings.Join(words, "")
}

func AddSpaceBeforeCaps(input string) string {
	// Define the regex pattern to match each capital letter preceded by any character (except at the beginning)
	re := regexp.MustCompile(`([a-z])([A-Z])`)

	// Replace matches with the first character followed by a space and the second character
	output := re.ReplaceAllString(input, `$1 $2`)

	return output
}

func StrExistsInArr(s string, arr []string) bool {
	for _, a := range arr {
		if a == s {
			return true
		}
	}
	return false
}
