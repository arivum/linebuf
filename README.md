# LineBuf

This module allows JSON stream processing via line-buffered JSON objects.

See [general documenation](https://pkg.go.dev/github.com/arivum/linebuf) and [package documentation](https://pkg.go.dev/github.com/arivum/linebuf/pkg/linebuf)

## How to use

```go
package main

import (
	"fmt"
	"io"

	"github.com/arivum/linebuf"
)

type Test struct {
	ID   int64
	Text string
}

func main() {
	var array = make([]Test, 10000)
	for i := 0; i < len(array); i++ {
		array[i] = Test{
			Text: fmt.Sprintf("this is text #%d", i),
			ID:   int64(i),
		}
	}

	reader, writer := io.Pipe()
	go func() {
		// create new line buffered JSON encoder that writes to the write-end of the pipe
		encoder, _ := linebuf.NewEncoder(writer)

		// get stream channel to send stream entries onto the wire
		stream := encoder.Stream()
		for _, a := range array {
			// send each array entry to the encoding channel
			stream <- a
		}
		// don't forget to close the encoder. It signals the end to the decoder on the other end of the wire
		encoder.Close()
	}()

	// create new decoder that reads the read-end of the pipe
	decoder, _ := linebuf.NewDecoder(reader)

	// get stream channel to read each decoded entry. Specify an optional parameter to tell the decoder how to unmarshal each entry. If left empty, the chance of getting map[string]interface{} is high
	stream := decoder.Stream(&Test{})
	for entry := range stream {
		// cast to target type `Test` to access the ID field
		fmt.Println(entry.(Test).ID)
	}
}

```
