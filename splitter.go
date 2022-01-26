package lsv

import (
	"bufio"
	"github.com/pkg/errors"
	"io"
	"strings"
	"unicode"
)

/*
Rules:
 - Each value must be seperated by a newline \n (\r are stripped)
 - Text proceeded by # are comments are discarded on reading
 - Whitespace that is leading or trailing the line or preceding a comment are stripped
 - All text between " and " is a raw string literal and considered a single value
    - Whitespace is preserved
    - Comments are not allowed
    - Quotes (" ") are stripped from value
 - Quotes and comments can be escaped with \
 - Escape characters preceding quotes and comments can also be escaped
*/

func splitter(str string) ([]string, error) {
	var list []string

	reader := bufio.NewReader(strings.NewReader(str))
	var inRaw bool
	var line, rawString string
	var err error

	for {
		line, err = reader.ReadString('\n')

		line = strings.TrimLeftFunc(line, unicode.IsSpace)
		if line == "" {
			continue
		}
		if line[0] == '"' {
			inRaw = true
			line = line[1:]
		}

		trimmedLine := trimComment(line, inRaw)
		if trimmedLine != "" {
			if inRaw {
				if isRaw(len(trimmedLine)-1, trimmedLine) {
					line = rawString + trimmedLine[:len(trimmedLine)-1]
					list = append(list, line)
					inRaw = false
					rawString = ""
				} else {
					if trimmedLine[len(trimmedLine)-1] == '"' {
						i := strings.LastIndex(line, "\\\"")
						if i > -1 {
							line = line[:i] + line[i+1:]
						}
					}
					rawString += line
				}
			} else {
				list = append(list, strings.NewReplacer("\\#", "#").Replace(trimmedLine))
			}
		}

		if err != nil {
			break
		}
	}

	if inRaw {
		err = errors.Errorf("missing closing quotes")
	}

	if err != io.EOF {
		return nil, err
	}

	return list, nil
}

func trimComment(line string, inRaw bool) string {
	for j := range line {
		if isComment(j, line) && !inRaw {
			line = line[:j]
			break
		} else if isRaw(j, line) && inRaw {
			inRaw = false
		}
	}
	return strings.TrimRightFunc(line, unicode.IsSpace)
}

func isComment(i int, str string) bool {
	isChar := str[i] == '#'
	isEsc1 := i > 0 && str[i-1] == '\\'
	isEsc2 := i > 1 && str[i-2] == '\\'

	return isChar && !isEsc1 && !(isEsc1 && isEsc2)
}

func isRaw(i int, str string) bool {
	isChar := str[i] == '"'
	isEsc1 := i > 0 && str[i-1] == '\\'
	isEsc2 := i > 1 && str[i-2] == '\\'

	return isChar && !isEsc1 && !(isEsc1 && isEsc2)
}

func isChar(char rune, i int, str string) bool {
	match := str[i] == byte(char)
	isEsc1 := i > 0 && str[i-1] == '\\'
	isEsc2 := i > 1 && str[i-2] == '\\'

	return match && !isEsc1 && !(isEsc1 && isEsc2)
}
