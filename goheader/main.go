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
	//"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
)


const C_LINE_COMMENT = "//!!! "

var header = `// {cmd}
// MACHINE GENERATED.

package {pkg}
`

var (
	fOS      = flag.String("s", "", "The operating system")
	fPackage = flag.String("p", "", "The name of the package")
	fListOS  = flag.Bool("ls", false, "List of valid systems")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: goheader -s system -g package [defs.h...]\n")
	flag.PrintDefaults()
	os.Exit(1)
}


func turn(fname string) os.Error {
	reSkipLine := regexp.MustCompile(`^#(ifdef|ifndef|else|undef|endif|include|define)[ \t\n]`)
	reDefine := regexp.MustCompile(`^(#[ \t]*define)[ \t]+([^ \t]+)[ \t]+(.+)`)
	reMacroInDefine := regexp.MustCompile(`^.*(\(.*\))`)

	reSingleComment := regexp.MustCompile(`^(.+)?/\*[ \t]*(.+)[ \t]*\*/`)
	reStartMultipleComment := regexp.MustCompile(`^/\*(.+)?`)
	reMiddleMultipleComment := regexp.MustCompile(`^[ \t*]*(.+)`)
	reEndMultipleComment := regexp.MustCompile(`^(.+)?\*/`)

	file, err := os.Open(fname, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	r := bufio.NewReader(file)

	fmt.Println(header)
	var isMultipleComment, isDefine bool

	for {
		line, err := r.ReadString('\n')
		if err == os.EOF {
			break
		}
		line = strings.TrimSpace(line) + "\n"
		isSingleComment := false

		// === Convert comment of single line.
		if !isMultipleComment {
			if fields := reSingleComment.FindStringSubmatch(line); fields != nil {
				isSingleComment = true
				line = "// " + fields[2] + "\n"

				if fields[1] != "" {
					line = fields[1] + line
				}
			}
		}

		if !isSingleComment && !isMultipleComment && strings.HasPrefix(line, "/*") {
			isMultipleComment = true
		}

		// === Convert comments of multiple line.
		if isMultipleComment {
			if fields := reStartMultipleComment.FindStringSubmatch(line); fields != nil {
				if fields[1] != "\n" {
					line = "// " + fields[1]
					fmt.Print(line)
				}
				continue
			}

			if fields := reEndMultipleComment.FindStringSubmatch(line); fields != nil {
				if fields[1] != "" {
					line = "// " + fields[1] + "\n"
					fmt.Print(line)
				}
				isMultipleComment = false
				continue
			}

			if fields := reMiddleMultipleComment.FindStringSubmatch(line); fields != nil {
				line = "// " + fields[1]
				fmt.Print(line)
				continue
			}
		}

		// === Turn defines.
		if fields := reDefine.FindStringSubmatch(line); fields != nil {
			if !isDefine {
				isDefine = true
				fmt.Print("const (\n")
			}
			line = fmt.Sprintf("%s = %s", fields[2], fields[3])

			// Removes comment (if any) to ckeck if it is a macro.
			lastField := strings.Split(fields[3], "//", -1)[0]
			if reMacroInDefine.MatchString(lastField) {
				line = C_LINE_COMMENT + line
			}

			fmt.Print(line)
			continue
		}

		if isDefine && line == "\n" {
			fmt.Print(")\n\n")
			isDefine = false
			continue
		}

		// === Comment another lines.
		if reSkipLine.MatchString(line) {
			line = C_LINE_COMMENT + line
			fmt.Print(line)
			continue
		}

		fmt.Print(line)
	}

	return nil
}


func main() {
	var isValidOS bool
	validOS := []string{"darwin", "freebsd", "linux", "windows"}

	// === Parse the flags
	flag.Usage = usage
	flag.Parse()

	if *fListOS {
		fmt.Print("  = Systems\n\n  ")
		fmt.Println(validOS)
		os.Exit(0)
	}

	if len(os.Args) == 1 || *fOS == "" || *fPackage == "" {
		usage()
	}

	*fOS = strings.ToLower(*fOS)

	for _, v := range validOS {
		if v == *fOS {
			isValidOS = true
			break
		}
	}

	if !isValidOS {
		fmt.Fprintf(os.Stderr, "ERROR: System passed in flag 's' is invalid\n")
		os.Exit(1)
	}

	// === Update header
	cmd := strings.Join(os.Args, " ")
	header = strings.Replace(header, "{cmd}", path.Base(cmd), 1)
	header = strings.Replace(header, "{pkg}", *fPackage, 1)

//File := "/usr/include/asm-generic/ioctls.h"
File := "/usr/include/asm-generic/termbits.h"

	if err := turn(File); err != nil {
		fmt.Fprintf(os.Stderr, err.String())
		os.Exit(1)
	}
}

