// Copyright 2013-2017 Docker, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

// CHANGES:
// - update package

package gottyclient

import (
	"io"
)

// EscapeError is special error which returned by a TTY proxy reader's Read()
// method in case its detach escape sequence is read.
type EscapeError struct{}

// Error is a function.
func (EscapeError) Error() string {
	return "read escape sequence"
}

// escapeProxy is used only for attaches with a TTY. It is used to proxy
// stdin keypresses from the underlying reader and look for the passed in
// escape key sequence to signal a detach.
type escapeProxy struct {
	escapeKeys   []byte
	escapeKeyPos int
	r            io.Reader
}

// NewEscapeProxy returns a new TTY proxy reader which wraps the given reader
// and detects when the specified escape keys are read, in which case the Read
// method will return an error of type EscapeError.
func NewEscapeProxy(r io.Reader, escapeKeys []byte) io.Reader {
	return &escapeProxy{
		escapeKeys: escapeKeys,
		r:          r,
	}
}

// Read is a function.
func (r *escapeProxy) Read(buf []byte) (int, error) {
	nr, err := r.r.Read(buf)

	if len(r.escapeKeys) == 0 {
		return nr, err
	}

	preserve := func() {
		// this preserves the original key presses in the passed in buffer
		nr += r.escapeKeyPos
		preserve := make([]byte, 0, r.escapeKeyPos+len(buf))
		preserve = append(preserve, r.escapeKeys[:r.escapeKeyPos]...)
		preserve = append(preserve, buf...)
		r.escapeKeyPos = 0
		copy(buf[0:nr], preserve)
	}

	if nr != 1 || err != nil {
		if r.escapeKeyPos > 0 {
			preserve()
		}
		return nr, err
	}

	if buf[0] != r.escapeKeys[r.escapeKeyPos] {
		if r.escapeKeyPos > 0 {
			preserve()
		}
		return nr, nil
	}

	if r.escapeKeyPos == len(r.escapeKeys)-1 {
		return 0, EscapeError{}
	}

	// Looks like we've got an escape key, but we need to match again on the next
	// read.
	// Store the current escape key we found so we can look for the next one on
	// the next read.
	// Since this is an escape key, make sure we don't let the caller read it
	// If later on we find that this is not the escape sequence, we'll add the
	// keys back
	r.escapeKeyPos++
	return nr - r.escapeKeyPos, nil
}
