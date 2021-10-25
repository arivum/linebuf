package linebuf

import (
	"bufio"
	"io"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/segmentio/encoding/json"
)

/*
NewEncoder returns a linebuf encoder from an io.Writer. The buffersize will be 4kB
*/
func NewEncoder(w io.Writer) *Encoder {
	e, _ := NewEncoderWithBuffersize(w, "4k")
	return e
}

/*
NewEncoderWithBuffersize returns a linebuf encoder from an io.Writer while allowing to specify a custom buffersize.

An error will be returned if parsing of the bufsize fails
*/
func NewEncoderWithBuffersize(w io.Writer, bufSize string) (*Encoder, error) {
	var (
		err      error
		bufBytes uint64
		buf      *bufio.Writer
	)

	if bufBytes, err = bytefmt.ToBytes(bufSize); err != nil {
		return nil, err
	}

	buf = bufio.NewWriterSize(w, int(bufBytes))
	return &Encoder{
		buf:         buf,
		w:           w,
		s:           make(chan interface{}, 10),
		err:         nil,
		jsonEncoder: json.NewEncoder(buf),
	}, nil
}

/*
LastError returns the last error that occured or nil if everything worked well
*/
func (enc *Encoder) LastError() error {
	return enc.err
}

func (enc *Encoder) write(v interface{}) error {
	var (
		err error
	)

	if err = enc.jsonEncoder.Encode(v); err != nil {
		return err
	}

	return nil
}

/*
Decode encodes a single entry to JSON and write it to the underlaying io.Writer.

If some errors occure, they will mainly be I/O errors
*/
func (enc *Encoder) Encode(v interface{}) error {
	defer enc.Close()
	return enc.write(v)
}

/*
Stream creates and returns a channel that's used to send consecutive entries that will be encoded and written to the underlaying io.Writer.

A possible snippet:
	for entry := range array {
		enc.Stream() <- entry
	}

To retrieve the last error that occured, use
	enc.LastError()
*/
func (enc *Encoder) Stream() chan<- interface{} {
	enc.once.Do(func() {
		go func() {
			var (
				value interface{}
			)

			for value = range enc.s {
				if enc.err = enc.write(value); enc.err != nil {
					continue
				}
			}
		}()
	})

	return enc.s
}

/*
Close closes the encoder and the underlaying io.Writer. This signals the opposite read-end that the transmission has ended.
*/
func (enc *Encoder) Close() {
	var (
		ok bool
	)

	for len(enc.s) > 0 {
		time.Sleep(100 * time.Microsecond)
	}

	enc.buf.Flush()

	if _, ok = enc.w.(io.Closer); ok {
		enc.w.(io.Closer).Close()
	}

	enc.closeChan()
}

func (enc *Encoder) closeChan() {
	enc.closeOnce.Do(func() {
		close(enc.s)
	})
}
