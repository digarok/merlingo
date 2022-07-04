package merlingo

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strings"
	"unicode"
)

//  params/defaults
var mnemonic_col_x = 18
var operand_col_x = 24
var comment_col_x = 48
var min_space = 1
var bump_space = 2
var indent_semi = true
var indent_ast = false

func fmtLine(code string) (string, error) {

	// state machine - resets each line
	in_quote := false
	in_comment := false
	label_done := false
	in_label := false
	opcode_done := false
	in_opcode := false
	operand_done := false
	in_operand := false
	var quote_char string
	x := 0

	buf := "" // line buffer, starts empty each line and is appended to output_buf
	label_done = true

	// scan by char
	for i, c := range code {
		// first de-tabify as we're in spaces land now!
		code = strings.ReplaceAll(code, "\t", " ")

		// starts with whitespace? do an indent
		if i == 0 && unicode.IsSpace(c) {
			buf += strings.Repeat(" ", mnemonic_col_x)
			x += mnemonic_col_x
			label_done = true
			continue // SHORT CIRCUIT
		}

		// are we in a comment? just print the char
		if in_comment {
			buf += string(c)
			x += 1
			continue // SHORT CIRCUIT
		}

		// are we in a quote? print, but also look for matching end quote
		if in_quote {
			buf += string(c)
			x += 1
			if string(c) == quote_char { // did we find closing quotes?
				in_quote = false
			}
			continue // SHORT CIRCUIT
		}

		// not already in comment or quote
		if unicode.IsSpace(c) {
			// ignore
			if in_label {
				in_label = false
				label_done = true
				// do we need to bump out space ?
				if x > mnemonic_col_x-min_space {
					buf += strings.Repeat(" ", min_space)
					x += min_space
				} else {
					buf += strings.Repeat(" ", mnemonic_col_x-x)
					x += mnemonic_col_x - x
				}
			} else if in_opcode {
				in_opcode = false
				opcode_done = true
				// do we need to bump out space ?
				if x > operand_col_x-min_space {
					buf += strings.Repeat(" ", min_space)
					x += min_space
				} else {
					buf += strings.Repeat(" ", operand_col_x-x)
					x += operand_col_x - x
				}
			} else if in_operand {
				in_operand = false
				operand_done = true
				// do we need to bump out space ?
				if x > comment_col_x-min_space {
					buf += strings.Repeat(" ", min_space)
					x += min_space
				} else {
					buf += strings.Repeat(" ", comment_col_x-x)
					x += comment_col_x - x
				}
			}

			continue
		} else {
			// see if we are starting a quote
			if string(c) == "\"" || string(c) == "'" {
				quote_char = string(c)
				in_quote = true
				in_operand = true
				buf += string(c)

				// see if we are starting a line with a comment
			} else if (c == ';' || c == '*') && i == 0 {
				in_comment = true
				buf += string(c)
				x += 1

				// found a semi-colon not in an operand (macro!danger)
				//  (and not in quote or comment)
			} else if c == ';' && !in_operand {
				in_comment = true
				// protect against "negative" spacing
				var spaces = 0
				if 0 <= (comment_col_x - x) {
					spaces = comment_col_x - x
				}
				buf += strings.Repeat(" ", spaces)

				x += comment_col_x - x
				buf += string(c)
				x += 1

				// found asterisk preceded only by whitespace
			} else if c == '*' && strings.Replace(string(code[0:i-1]), " ", "", -1) == "" {
				in_comment = true
				buf += string(c)
				x += 1

				// real label!
			} else if i == 0 {
				in_label = true
				buf += string(c)
				x += 1

				// already in label?
			} else if in_label {
				buf += string(c)
				x += 1

				// real opcode!
			} else if label_done && !opcode_done {
				in_opcode = true
				buf += string(c)
				x += 1

				// already in opcode
			} else if in_opcode {
				buf += string(c)
				x += 1

				// real operand!
			} else if opcode_done && !operand_done {
				in_operand = true
				buf += string(c)
				x += 1

				// already in operand
			} else if in_operand {
				buf += string(c)
				x += 1

				// if they have unhandled weirdness, just pass them through minus whitespace
			} else {
				if !unicode.IsSpace(c) {
					buf += string(c)
					x += 1
				}
			}
		}
	}

	// check for lossiness
	c1 := strings.ReplaceAll(string(code), " ", "")
	c2 := strings.ReplaceAll(buf, " ", "")
	if c1 != c2 {
		return buf, errors.New("error on line")
	}

	// this strips excess whitespace (@todo: locic issue?)
	//   .. adds a newline, not sure about cross-platformness
	return strings.TrimRight(buf, " ") + "\n", nil
}

func FmtFile(filename string) {
	readFile, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	buf := ""
	i := 1 // line counter starts at 1
	for fileScanner.Scan() {
		original_line := fileScanner.Text()
		formatted_line, err := fmtLine(original_line)
		if err != nil {
			log.Fatal(err, i+1) // error + line
		}
		buf += formatted_line
		i += 1
	}
	readFile.Close()
	if err := os.WriteFile(filename, []byte(buf), 0666); err != nil {
		log.Fatal(err)
	}
}
