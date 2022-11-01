# Line-Seperated Values (LSV)
This Go library reads and writes line-separated values (LSV) files.

An LSV is defined as a text file containing zero or more values seperated by the
newline character (`\n`). LSV also support comments, which are stripped when the
file is read.

```text
value1
value2
value3
```

Block and inline comments are both allowed. All comments are denoted using the
hash character (`#`); any values after a hash character until the next new line
are considered a comments. Each line of a block comment must start with the hash
character. All whitespace between a value and an inline comment is stripped. A
hash character can be escaped using the escape character (`\`).

The source:

```text
value1  # A comment
value2  \# Not a comment
```

results in the values

```text
{`value1`, `value2  \# Not a comment`}
```

Lines that start and stop with a quote character (`"`) (excluding comments and
whitespace) are quoted values and the beginning and ending quotes are not part
of the value. All whitespace and comments inside quoted values are preserved and
included as part of the value. A quote character can be included by using the
escape character (`\`) before the quotation character (`\"`).

The source:

```text
# A comment
  value1  # A comment
"  value2  # Not a comment" # A comment
```

results in the values

```text
{`value1`, `  value2  # Not a comment`}
```

Blank lines are ignored. Lines with only whitespace are considered blank lines
unless they are quoted. Carriage returns (`\r`) before newlines are removed.
Carriage returns in quoted values are untouched. An empty value can be
represented using quotes `""`.

The source:

```text
value1

value2

""
```

results in the values

```text
{`value1`, `value2`, ``}
```


To do:
 * Figure out why benchmarks are worse from read than splitter
 * Figure out why benchmarks are worse for lsv than csv
 * Add real file and large file testing and benchmarking