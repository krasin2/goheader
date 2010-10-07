// Copyright 2010  The "goheader" Authors
//
// Use of this source code is governed by the Simplified BSD License
// that can be found in the LICENSE file.
//
// This software is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
// OR CONDITIONS OF ANY KIND, either express or implied. See the License
// for more details.

package main

import (
	"bufio"
	"bytes"
	"container/vector"
	"fmt"
	"regexp"
	"os"
	"strings"
)


// Translates C type declaration into Go type declaration.
func translateC(output *bytes.Buffer, file *os.File) (err os.Error) {
	var isMultipleComment, isDefineBlock, isStruct bool
	var extraType vector.StringVector // Types defined in the header file.

	// === Regular expressions
	reSkip := regexp.MustCompile(`^(\n|//)`) // Empty lines and comments.

	reType := regexp.MustCompile(`^(typedef)[ \t]+(.+)[ \t]+(.+)[;](.+)?`)

	reStruct := regexp.MustCompile(`^(struct)[ \t]+(.+)[ \t]*{`)
	reStructField := regexp.MustCompile(`^(.+)[ \t]+(.+)[;](.+)?`)
	reStructFieldName := regexp.MustCompile(`^([^_]*_)?(.+)`)

	reDefine := regexp.MustCompile(`^#[ \t]*(define|DEFINE)[ \t]+([^ \t]+)[ \t]+(.+)`)
	reDefineOnly := regexp.MustCompile(`^#[ \t]*(define|DEFINE)[ \t]+`)
	reDefineMacro := regexp.MustCompile(`^.*(\(.*\))`)

	reSingleComment := regexp.MustCompile(`^(.+)?/\*[ \t]*(.+)[ \t]*\*/`)
	reStartMultipleComment := regexp.MustCompile(`^/\*(.+)?`)
	reMiddleMultipleComment := regexp.MustCompile(`^[ \t*]*(.+)`)
	reEndMultipleComment := regexp.MustCompile(`^(.+)?\*/`)
	// ===

	if _, err = output.WriteString(goBase); err != nil {
		return
	}

	// === File to read
	fileBuf := bufio.NewReader(file)

	for {
		line, err := fileBuf.ReadString('\n')
		if err == os.EOF {
			break
		}
		line = strings.TrimSpace(line) + "\n"
		isSingleComment := false

		// === Translate comment of single line.
		if !isMultipleComment {
			if sub := reSingleComment.FindStringSubmatch(line); sub != nil {
				isSingleComment = true
				line = "// " + sub[2] + "\n"

				if sub[1] != "" {
					line = sub[1] + line
				}
			}
		}
		if !isSingleComment && !isMultipleComment && strings.HasPrefix(line, "/*") {
			isMultipleComment = true
		}

		// === Translate comments of multiple line.
		if isMultipleComment {
			if sub := reStartMultipleComment.FindStringSubmatch(line); sub != nil {
				if sub[1] != "\n" {
					line = "// " + sub[1]
					if _, err = output.WriteString(line); err != nil {
						return
					}
				}
				continue
			}
			if sub := reEndMultipleComment.FindStringSubmatch(line); sub != nil {
				if sub[1] != "" {
					line = "// " + sub[1] + "\n"
					if _, err = output.WriteString(line); err != nil {
						return
					}
				}
				isMultipleComment = false
				continue
			}
			if sub := reMiddleMultipleComment.FindStringSubmatch(line); sub != nil {
				line = "// " + sub[1]
				if _, err = output.WriteString(line); err != nil {
					return
				}
				continue
			}
		}

		// === Translate type definitions.
		if sub := reType.FindStringSubmatch(line); sub != nil {
			// Add the new type.
			extraType.Push(sub[3])

			gotype, ok := ctypeTogo(sub[2], &extraType)
			line = fmt.Sprintf("type %s %s", sub[3], gotype)

			if sub[4] != "\n" {
				line += sub[4]
			} else {
				line += "\n"
			}
			if !ok {
				line = COMMENT_LINE + line
			}

			if _, err = output.WriteString(line); err != nil {
				return
			}
			continue
		}

		// === Translate defines.
		if sub := reDefine.FindStringSubmatch(line); sub != nil {
			line = fmt.Sprintf("%s = %s", sub[2], sub[3])

			if !isDefineBlock {
				// Get characters of next line.
				startNextLine, err := fileBuf.Peek(10)
				if err != nil {
					return
				}

				// Constant in single line.
				if !reDefineOnly.Match(startNextLine) {
					line = "const " + line
				} else {
					isDefineBlock = true
					line = "const (\n" + line
				}
			}

			// Removes comment (if any) to ckeck if it is a macro.
			lastField := strings.Split(sub[3], "//", -1)[0]
			if reDefineMacro.MatchString(lastField) {
				line = COMMENT_LINE + line
			}

			if _, err = output.WriteString(line); err != nil {
				return
			}
			continue
		}

		if isDefineBlock && line == "\n" {
			if _, err = output.WriteString(")\n\n"); err != nil {
				return
			}
			isDefineBlock = false
			continue
		}

		// === Translate structs.
		if !isStruct {
			if sub := reStruct.FindStringSubmatch(line); sub != nil {
				isStruct = true

				if isDefineBlock {
					if _, err = output.WriteString(")\n"); err != nil {
						return
					}
					isDefineBlock = false
				}

				if _, err = output.WriteString(fmt.Sprintf(
					"type %s struct {\n", strings.Title(sub[2]))); err != nil {
					return
				}
				continue
			}
		} else {
			if sub := reStructField.FindStringSubmatch(line); sub != nil {
				// Translate the field type.
				gotype, ok := ctypeTogo(sub[1], &extraType)

				// === Translate the field name.
				fieldName := reStructFieldName.FindStringSubmatch(sub[2])
				_fieldName := ""

				if fieldName[1] != "" {
					_fieldName = fieldName[2]
				} else {
					_fieldName = fieldName[0]
				}
				// ===

				line = fmt.Sprintf("%s %s %s",
					strings.Title(_fieldName), gotype, sub[3])

				// C type not found.
				if !ok {
					line = COMMENT_LINE + line
				}

				if _, err = output.WriteString(line); err != nil {
					return
				}
				continue
			}
			if strings.HasPrefix(line, "}") {
				if _, err = output.WriteString(strings.Replace(line, ";", "", 1)); err != nil {
					return
				}
				isStruct = false
				continue
			}
		}

		// Comment another C lines.
		//if line != "\n" && !strings.HasPrefix(line, "//") {
		if !reSkip.MatchString(line) {
			line = COMMENT_LINE + line
		}

		if _, err = output.WriteString(line); err != nil {
			return
		}
	}

	return
}

// Translates a C type definition into Go definition. The C header could have
// defined new types so they're checked in the firs place.
func ctypeTogo(ctype string, extraCtype *vector.StringVector) (gotype string, ok bool) {
	for _, v := range *extraCtype {
		if v == ctype {
			return ctype, true
		}
	}

	switch ctype {
	case "char", "signed char", "signed short int", "short int", "short":
		return "int8", true
	case "unsigned char", "unsigned short int", "unsigned short":
		return "uint8", true
	case "int", "signed int":
		return "int16", true
	case "unsigned int", "unsigned":
		return "uint16", true
	case "signed long int", "long int", "long":
		return "int32", true
	case "unsigned long int", "unsigned long":
		return "uint32", true
	case "float":
		return "float32", true
	case "double", "long double":
		return "float64", true
	}
	return ctype, false
}
