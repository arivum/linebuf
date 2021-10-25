package linebuf

import (
	"bufio"
	"io"
	"math"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/segmentio/encoding/json"
)

type EncoderOption (func(*Encoder) error)

/*
WithUnbuffered can be provided as option to NewEncoder, thus turning off buffer mode
*/
func WithEncoderUnbuffered() EncoderOption {
	return func(enc *Encoder) error {
		enc.unbuffered = true
		return nil
	}
}

/*
WithBuffersize can be used as option to NewEncoder, thus providing another buffersize than the default 4k
*/
func WithEncoderBuffersize(bufSize string) EncoderOption {
	return func(enc *Encoder) error {
		var (
			bufBytes uint64
			err      error
		)
		if bufBytes, err = bytefmt.ToBytes(bufSize); err != nil {
			return err
		}
		enc.bufBytes = bufBytes
		return nil
	}
}

/*
NewEncoder returns a linebuf encoder from an io.Writer while allowing to sspecify custom options like unbuffered mode and/or setting the buffer size.

The default buffer size is 4k

An error will be returned e.g. if parsing of the bufsize fails
*/
func NewEncoder(w io.Writer, options ...EncoderOption) (*Encoder, error) {
	var (
		err    error
		option EncoderOption
		buf    *bufio.Writer
		enc    = &Encoder{
			w:   w,
			s:   make(chan interface{}),
			err: nil,
		}
	)

	for _, option = range options {
		if err = option(enc); err != nil {
			return nil, err
		}
	}

	buf = bufio.NewWriterSize(w, int(math.Max(float64(enc.bufBytes), 4<<10)))
	enc.buf = buf
	enc.jsonEncoder = json.NewEncoder(enc.buf)
	return enc, nil
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

			enc.mutex.Lock()
			for value = range enc.s {
				if enc.err = enc.write(value); enc.err != nil {
					continue
				}
			}
			enc.mutex.Unlock()
		}()
	})

	return enc.s
}

/*
Close closes the encoder and the underlaying io.Writer. This signals the opposite read-end that the transmission has ended.
*/
func (enc *Encoder) Close() {
	enc.closeOnce.Do(func() {
		var (
			ok bool
		)

		for len(enc.s) > 0 {
			time.Sleep(100 * time.Microsecond)
		}
		close(enc.s)
		enc.mutex.Lock()

		enc.buf.Flush()

		if _, ok = enc.w.(io.Closer); ok {
			enc.w.(io.Closer).Close()
		}
		enc.mutex.Unlock()
	})

}
