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
			isArray   = false
		)

		sanitizedWriter.mutex.Lock()
		defer func() {
			sanitizedWriter.mutex.Unlock()

			if isArray {
				_, sanitizedWriter.err = sanitizedWriter.w.Write([]byte("]\n"))
			}
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
	l.mutex.Lock()
	err = l.w.Close()
	l.mutex.Unlock()
	return err
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
