package linebuf

import (
	"bufio"
	"io"
)

/*
NewLinebufJSONConverter returns a writer that converts line-buffered JSON back to valid JSON and writes it to the underlaying io.WriteCloser.
*/
func NewLinebufJSONConverter(w io.WriteCloser) *LinebufJSONConverter {
	var (
		r, pipeWriter   = io.Pipe()
		bufReader       = bufio.NewReaderSize(r, 16)
		bufWriter       = bufio.NewWriterSize(pipeWriter, 16)
		sanitizedWriter = &LinebufJSONConverter{
			Writer:     bufWriter,
			w:          w,
			pipeWriter: pipeWriter,
			r:          bufReader,
			err:        nil,
		}
	)

	go func() {
		var (
			firstline []byte
			line      []byte
		)

		defer func() {
			if sanitizedWriter.isArray {
				_, sanitizedWriter.err = sanitizedWriter.w.Write([]byte("]\n"))
			}
			sanitizedWriter.w.Close()
		}()

		if firstline, sanitizedWriter.err = sanitizedWriter.r.ReadBytes('\n'); sanitizedWriter.err != nil {
			_, sanitizedWriter.err = sanitizedWriter.w.Write(firstline)
			return
		}

		for {
			if line, sanitizedWriter.err = sanitizedWriter.r.ReadBytes('\n'); sanitizedWriter.err != nil {
				_, sanitizedWriter.err = sanitizedWriter.w.Write(line)
				return
			} else {
				if len(line) > 0 && firstline != nil {
					if sanitizedWriter.err = write(sanitizedWriter.w, []byte("["), firstline[:len(firstline)-1], []byte(","), line[:len(line)-1]); sanitizedWriter.err != nil {
						return
					}
					firstline = nil
					sanitizedWriter.isArray = true
				} else if firstline != nil {
					_, sanitizedWriter.err = w.Write(firstline[:])
					return
				} else if len(line) > 0 {
					if sanitizedWriter.err = write(sanitizedWriter.w, []byte(","), []byte(line[:len(line)-1])); sanitizedWriter.err != nil {
						return
					}
				} else {
					return
				}
			}
		}
	}()

	return sanitizedWriter
}

/*
Close waits until the last writes have finished and gracefully closes the underlaying writers
*/
func (l *LinebufJSONConverter) Close() error {
	var (
		err error
	)

	if err = l.Writer.Flush(); err != nil {
		return err
	}
	if err = l.pipeWriter.Close(); err != nil {
		return err
	}
	return nil
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
