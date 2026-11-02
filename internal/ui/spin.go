package ui

import (
	"fmt"
	"sync"
	"time"
)

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner is a line on stderr, a turning mark and what is going on, while
// something slow runs; nothing when stderr is not a terminal. It shows only
// after 150 ms, so quick work never flickers.
type Spinner struct {
	mu   sync.Mutex
	msg  string
	stop chan struct{}
	done chan struct{}
}

// Spin starts one with msg.
func Spin(msg string) *Spinner {
	s := &Spinner{msg: msg}
	if !IsTTY(Stderr) {
		return s
	}
	s.stop, s.done = make(chan struct{}), make(chan struct{})
	go s.loop()
	return s
}

func (s *Spinner) loop() {
	defer close(s.done)
	t := time.NewTicker(80 * time.Millisecond)
	defer t.Stop()
	start := time.Now()
	drawn := false
	for i := 0; ; i++ {
		select {
		case <-s.stop:
			if drawn {
				fmt.Fprint(Stderr, "\r\x1b[K")
			}
			return
		case <-t.C:
			if time.Since(start) < 150*time.Millisecond {
				continue
			}
			s.mu.Lock()
			msg := s.msg
			s.mu.Unlock()
			lineErr(Accent.Render(frames[i%len(frames)]) + " " + Dim.Render(msg+"…"))
			drawn = true
		}
	}
}

func lineErr(s string) {
	fmt.Fprint(Stderr, "\r\x1b[K")
	Eprint(Cut(s, Width()))
}

// Eprint prints to stderr without a newline, downsampling colours.
func Eprint(a ...any) {
	fmt.Fprint(newErrWriter(), a...)
}

// Set changes what it says.
func (s *Spinner) Set(msg string) {
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

// Stop erases it. Safe to call more than once, and on a nil one.
func (s *Spinner) Stop() {
	if s == nil || s.stop == nil {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	<-s.done
}
