package linebuf

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sync"
)

/*
LineSanitizedReader wraps an io.Reader and strips all newlines, to allow to differentiate between multiline JSON and linebuffered JSON.
*/
type LineSanitizedReader struct {
	*io.PipeReader
	err error
}

/*
LinebufJSONConverter wraps an io.Writer and converts line-buffered writes to regular JSON
*/
type LinebufJSONConverter struct {
	*bufio.Writer
	w          io.WriteCloser
	pipeWriter *io.PipeWriter
	r          *bufio.Reader
	err        error
	isArray    bool
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
	closeOnce   sync.Once
	unbuffered  bool
	bufBytes    uint64
	mutex       sync.Mutex
}

/*
Decoder reads line-buffered JSON from the underlaying io.Reader and decodes them in single-entry modde or stream mode
*/
type Decoder struct {
	context.Context
	buf        *bufio.Reader
	r          io.Reader
	s          chan interface{}
	once       sync.Once
	err        error
	jsonDec    *json.Decoder
	unbuffered bool
	bufBytes   uint64
}
