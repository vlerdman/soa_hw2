package client

import (
	"bufio"
	"context"
	"fmt"
	"os"
)

type Messenger struct {
	ctx    context.Context
	cancel context.CancelFunc

	scanner *bufio.Scanner

	input  chan string
	output chan string
}

func NewMessenger(ctx context.Context) *Messenger {
	ctx, cancel := context.WithCancel(ctx)

	return &Messenger{
		ctx:     ctx,
		cancel:  cancel,
		scanner: bufio.NewScanner(os.Stdin),
		input:   make(chan string, 10),
		output:  make(chan string, 10),
	}
}

func (m *Messenger) Start() {
	go m.read()
	go m.write()
}

func (m *Messenger) read() {
	for m.scanner.Scan() {
		m.input <- m.scanner.Text()
	}
}

func (m *Messenger) write() {
	for s := range m.output {
		_, _ = fmt.Fprintf(os.Stdout, s)
	}
}
