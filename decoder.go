package linebuf

import (
	"bufio"
	"encoding/json"
	"io"
	"math"
	"reflect"

	"code.cloudfoundry.org/bytefmt"
)

type DecoderOption (func(*Decoder) error)

/*
WithDecoderUnbuffered can be provided as option to NewDecoder, thus turning off buffer mode
*/
func WithDecoderUnbuffered() DecoderOption {
	return func(dec *Decoder) error {
		dec.unbuffered = true
		return nil
	}
}

/*
WithDecoderBuffersize can be used as option to NewDecoder, thus providing another buffersize than the default 4k
*/
func WithDecoderBuffersize(bufSize string) DecoderOption {
	return func(dec *Decoder) error {
		var (
			bufBytes uint64
			err      error
		)
		if bufBytes, err = bytefmt.ToBytes(bufSize); err != nil {
			return err
		}
		dec.bufBytes = bufBytes
		return nil
	}
}

/*
NewDecoder returns a linebuf decoder from an io.Reader while allowing to specify custom options like unbuffered mode and/or setting the buffer size.

The default buffer size is 4k

An error will be returned e.g. if parsing of the bufsize fails
*/
func NewDecoder(r io.Reader, options ...DecoderOption) (*Decoder, error) {
	var (
		err     error
		jsonDec *json.Decoder
		option  DecoderOption
		dec     = &Decoder{
			err: nil,
			r:   r,
			s:   make(chan interface{}),
		}
		buf *bufio.Reader
	)

	for _, option = range options {
		if err = option(dec); err != nil {
			return nil, err
		}
	}

	buf = bufio.NewReaderSize(r, int(math.Max(float64(dec.bufBytes), 4<<10)))
	jsonDec = json.NewDecoder(buf)
	jsonDec.UseNumber()

	dec.buf = buf
	dec.jsonDec = jsonDec
	if !dec.unbuffered {
		dec.s = make(chan interface{}, 10)
	}

	return dec, nil
}

/*
LastError returns the last error that occured or nil if everything worked well
*/
func (dec *Decoder) LastError() error {
	return dec.err
}

func (dec *Decoder) read(v interface{}) error {
	var (
		err error
	)

	if err = dec.jsonDec.Decode(v); err != nil {
		return err
	}

	return nil
}

/*
Decode decodes a single entry that is received by the underlaying io.Reader into a pointer to an object given via parameter "v".

If some errors occure, they will mainly be I/O errors
*/
func (dec *Decoder) Decode(v interface{}) error {
	return dec.read(v)
}

/*
Stream starts decoding entries that are received by the underlaying io.Reader. Specify an optional pointer parameter `entry` that's used to decode JSON into a specific structure.

It returns a channel that can be used to receive the encoded entries inside a loop like:
	for entry := range dec.Stream() {
		// do sth.
	}

To retrieve the last error that occured, use
	dec.LastError()
*/
func (dec *Decoder) Stream(entry ...interface{}) <-chan interface{} {
	var (
		v     interface{}
		isPtr = false
	)

	if len(entry) > 0 {
		v = entry[0]
		isPtr = (reflect.ValueOf(v).Kind() == reflect.Ptr)
	}

	dec.once.Do(func() {
		go func() {
			for {
				if dec.err = dec.read(&v); dec.err == io.EOF {
					close(dec.s)
					dec.err = nil
					break
				} else if dec.err != nil {
					break
				}
				if isPtr {
					dec.s <- reflect.ValueOf(v).Elem().Interface()
				} else {
					dec.s <- v
				}
			}
		}()
	})

	return dec.s
}
