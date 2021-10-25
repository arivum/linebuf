package linebuf

import (
	"bufio"
	"io"
)

/*
NewLinebufJSONConverter returns a writer that converts line-buffered JSON back to valid JSON and writes it to the underlaying io.WriteCloser.
*/
func NewLinebufJSONConverter(w io.WriteCloser) LinebufJSONConverter {
	var (
		r, tmpWriter    = io.Pipe()
		bufReader       = bufio.NewReader(r)
		sanitizedWriter = LinebufJSONConverter{tmpWriter, nil}
	)

	go func() {
		var (
			firstline []byte
			line      []byte
			isArray   = false
		)
		defer func() {
			w.Close()
			sanitizedWriter.Close()
		}()

		if firstline, sanitizedWriter.err = bufReader.ReadBytes(byte('\n')); sanitizedWriter.err != nil {
			_, sanitizedWriter.err = w.Write(firstline)
			return
		}

		for {
			line, sanitizedWriter.err = bufReader.ReadBytes(byte('\n'))

			if len(line) > 0 && len(firstline) > 0 {
				if sanitizedWriter.err = write(w, []byte("["), firstline[:len(firstline)-1], []byte(","), line[:len(line)-1]); sanitizedWriter.err != nil {
					return
				}
				firstline = nil
				isArray = true
			} else if len(firstline) > 0 {
				_, sanitizedWriter.err = w.Write(firstline[:])
				break
			} else if len(line) > 0 {
				if sanitizedWriter.err = write(w, []byte(","), []byte(line[:len(line)-1])); sanitizedWriter.err != nil {
					return
				}
			} else {
				break
			}
		}
		if isArray {
			_, sanitizedWriter.err = w.Write([]byte("]\n"))
		}
	}()

	return sanitizedWriter
}

/*
LastError returns the last error that occured or nil if everything worked well
*/
func (l *LinebufJSONConverter) LastError() error {
	return l.err
}

func write(w io.Writer, raws ...[]byte) error {
	var (
		err error
		i   int
	)

	for i = range raws {
		if _, err = w.Write(raws[i]); err != nil {
			return err
		}
	}
	return nil
}
