package linebuf

import (
	"bufio"
	"io"
)

/*
NewLineSanitizedReader returns a line-sanitizing reader that strips all newlines from JSON objects recieved by the underlaying io.Reader.

This allows to differntiate between multiline JSON and linebuffered JSON.
*/
func NewLineSanitizedReader(r io.Reader) io.Reader {
	var (
		bufReader       = bufio.NewReader(r)
		tmpReader, w    = io.Pipe()
		sanitizedReader = LineSanitizedReader{tmpReader, nil}
	)

	go func() {
		var (
			line []byte
		)
		defer func() {
			w.Close()
		}()
		for {
			line, _, sanitizedReader.err = bufReader.ReadLine()
			if sanitizedReader.err == nil {
				if _, sanitizedReader.err = w.Write(line); sanitizedReader.err != nil {
					return
				}
			} else {
				w.Write(line)
				return
			}
		}
	}()

	return sanitizedReader
}

/*
LastError returns the last error that occured or nil if everything worked well
*/
func (s *LineSanitizedReader) LastError() error {
	return s.err
}
