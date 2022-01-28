package lsv

import (
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

	var inRaw bool
	var err error
	var values []string
	var rawString strings.Builder

	for _, line := range strings.SplitAfter(str, "\n") {

		if !inRaw {
			line = strings.TrimLeftFunc(line, unicode.IsSpace)
			if line == "" {
				continue
			}
			if line[0] == '"' {
				inRaw = true
				line = line[1:]
			}
		}

		line = trimComment(line, inRaw)
		if line != "" {
			if inRaw {
				i := strings.LastIndexFunc(line, func(r rune) bool {
					return !unicode.IsSpace(r)
				})
				if i > -1 {
					if isRaw(i, line) {
						_, err = rawString.WriteString(line[:i])
						if err != nil {
							return nil, err
						}

						values = append(values, rawString.String())
						inRaw = false
						rawString.Reset()
						continue
					} else if line[i] == '"' && (i > 0 && line[i-1] == '\\') {
						line = line[:i-1] + line[i:]
					}
				}
				_, err = rawString.WriteString(line)
				if err != nil {
					return nil, err
				}
			} else {
				line = strings.TrimRightFunc(line, unicode.IsSpace)
				line = strings.ReplaceAll(line, "\\#", "#")
				if line != "" {
					values = append(values, line)
				}
			}
		}
	}

	if inRaw {
		err = ErrNoClosingRaw
	}

	return values, err
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

	return line
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
