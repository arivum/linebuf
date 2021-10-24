package linebuf

import (
	"bufio"
	"io"
	"reflect"

	"code.cloudfoundry.org/bytefmt"
	"github.com/segmentio/encoding/json"
)

/*
NewDecoder returns a linebuf decoder from an io.Reader. The buffersize will be 4kB
*/
func NewDecoder(r io.Reader) (*Decoder, error) {
	return NewDecoderWithBuffersize(r, "4k")
}

/*
NewDecoderWithBuffersize returns a linebuf decoder from an io.Reader while allowing to specify a custom buffersize.

An error will be returned if parsing of the bufsize fails
*/
func NewDecoderWithBuffersize(r io.Reader, bufSize string) (*Decoder, error) {
	var (
		err      error
		bufBytes uint64
		jsonDec  *json.Decoder
		buf      *bufio.Reader
	)

	if bufBytes, err = bytefmt.ToBytes(bufSize); err != nil {
		return nil, err
	}

	buf = bufio.NewReaderSize(r, int(bufBytes))
	jsonDec = json.NewDecoder(buf)
	jsonDec.ZeroCopy()
	jsonDec.UseNumber()
	jsonDec.DontMatchCaseInsensitiveStructFields()
	return &Decoder{
		buf:     buf,
		err:     nil,
		r:       r,
		s:       make(chan interface{}, 10),
		jsonDec: jsonDec,
	}, nil
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
