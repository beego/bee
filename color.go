// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import "io"

type outputMode int

// DiscardNonColorEscSeq supports the divided color escape sequence.
// But non-color escape sequence is not output.
// Please use the OutputNonColorEscSeq If you want to output a non-color
// escape sequences such as ncurses. However, it does not support the divided
// color escape sequence.
const (
	_ outputMode = iota
	DiscardNonColorEscSeq
	OutputNonColorEscSeq
)

// NewColorWriter creates and initializes a new ansiColorWriter
// using io.Writer w as its initial contents.
// In the console of Windows, which change the foreground and background
// colors of the text by the escape sequence.
// In the console of other systems, which writes to w all text.
func NewColorWriter(w io.Writer) io.Writer {
	return NewModeColorWriter(w, DiscardNonColorEscSeq)
}

// NewModeColorWriter create and initializes a new ansiColorWriter
// by specifying the outputMode.
func NewModeColorWriter(w io.Writer, mode outputMode) io.Writer {
	if _, ok := w.(*colorWriter); !ok {
		return &colorWriter{
			w:    w,
			mode: mode,
		}
	}
	return w
}
