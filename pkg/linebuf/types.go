package linebuf

import (
	"bufio"
	"context"
	"io"
	"sync"

	"github.com/segmentio/encoding/json"
)

/*
LineSanitizedReader wraps an io.Reader and strips all newlines, to allow to differentiate between multiline JSON and linebuffered JSON.
*/
type LineSanitizedReader struct {
	*io.PipeReader
	err error
}

/*
Encoder encodes objects and writes the resulting JSON in line-buffered-mode to the underlaying io.Writer
*/
type Encoder struct {
	context.Context
	w           io.Writer
	buf         *bufio.Writer
	s           chan interface{}
	err         error
	once        sync.Once
	jsonEncoder *json.Encoder
}

/*
Decoder reads line-buffered JSON from the underlaying io.Reader and decodes them in single-entry modde or stream mode
*/
type Decoder struct {
	context.Context
	buf     *bufio.Reader
	r       io.Reader
	s       chan interface{}
	once    sync.Once
	err     error
	jsonDec *json.Decoder
}
