# Line-Seperated Values (LSV)
This go library is used to read line-separated values as a single-dimensional
arrays with support for comments.

All values must be seperated by a new-line (\n) with all leading and trailing
whitespace trimmed. Comments begin with # and end at the end of the line.

Values can be escaped if they start and end with quotes (" "). The starting
quote must be the first non-whitespace character on a line and the end quote
must be the last non-whitespace or non comment character on the line. All
whitespace and comments in escaped values are preserved.

Empty lines are ignored. Lines with only "" are considered empty values.

Comments and quotes can be escaped using \.
