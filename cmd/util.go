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
	"os"
	"strings"
	"path/filepath"
)


func isHeader(f *os.FileInfo) bool {
	return f.IsRegular() && !strings.HasPrefix(f.Name, ".") &&
		strings.HasSuffix(f.Name, ".h")
}

// === Walk into a directory
// ===

type fileVisitor chan os.Error

func (v fileVisitor) VisitDir(path string, f *os.FileInfo) bool {
	return true
}

func (v fileVisitor) VisitFile(path string, f *os.FileInfo) {
	if isHeader(f) {
		v <- nil // Synchronize error handler.
		if err := processFile(path); err != nil {
			v <- err
		}
	}
}


func walkDir(path string) {
	// === Start an error handler
	v := make(fileVisitor)
	done := make(chan bool)

	go func() {
		for err := range v {
			if err != nil {
				reportError(err)
			}
		}
		done <- true
	}()
	// ===

	filepath.Walk(path, v, v) // Walk the tree.
	close(v)                  // Terminate error handler loop.
	<-done                    // Wait for all errors to be reported.
}

