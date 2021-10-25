package linebuf

import (
	"bufio"
	"io"
	"time"
)

/*
NewLinebufJSONConverter returns a writer that converts line-buffered JSON back to valid JSON and writes it to the underlaying io.WriteCloser.
*/
func NewLinebufJSONConverter(w io.WriteCloser) *LinebufJSONConverter {
	var (
		r, pipeWriter   = io.Pipe()
		bufReader       = bufio.NewReaderSize(r, 4096)
		bufWriter       = bufio.NewWriterSize(pipeWriter, 4096)
		sanitizedWriter = &LinebufJSONConverter{bufWriter, w, pipeWriter, bufReader, nil, false}
	)

	go func() {
		var (
			firstline []byte
			line      []byte
			isArray   = false
		)

		defer func() {
			sanitizedWriter.finished = true
		}()

		if firstline, sanitizedWriter.err = sanitizedWriter.r.ReadBytes('\n'); sanitizedWriter.err != nil {
			_, sanitizedWriter.err = sanitizedWriter.w.Write(firstline)
			return
		}

		for {
			if line, sanitizedWriter.err = sanitizedWriter.r.ReadBytes('\n'); sanitizedWriter.err != nil {
				_, sanitizedWriter.err = sanitizedWriter.w.Write(line)
				break
			} else {
				if len(line) > 0 && firstline != nil {
					if sanitizedWriter.err = write(sanitizedWriter.w, []byte("["), firstline[:len(firstline)-1], []byte(","), line[:len(line)-1]); sanitizedWriter.err != nil {
						break
					}
					firstline = nil
					isArray = true
				} else if firstline != nil {
					_, sanitizedWriter.err = w.Write(firstline[:])
					break
				} else if len(line) > 0 {
					if sanitizedWriter.err = write(sanitizedWriter.w, []byte(","), []byte(line[:len(line)-1])); sanitizedWriter.err != nil {
						break
					}
				} else {
					break
				}
			}
		}
		if isArray {
			_, sanitizedWriter.err = sanitizedWriter.w.Write([]byte("]\n"))
		}
	}()

	return sanitizedWriter
}

/*
Close waits until the last writes have finished and gracefully closes the underlaying writers
*/
func (l *LinebufJSONConverter) Close() error {
	l.Writer.Flush()
	l.pipeWriter.Close()
	for !l.finished {
		time.Sleep(100 * time.Microsecond)
	}
	return l.w.Close()
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
