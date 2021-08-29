package main

import (
	"testing"
)

func Test_trailers(t *testing.T) {
	msg := break_trailer{}.command()
	if msg != "break" {
		t.Errorf("Unexpected value for break_trailer.command(): got %s, expected break", msg)
	}
	
	msg = exec_trailer{cmd: "ls -l"}.command()
	if msg != "exec ls -l" {
		t.Errorf("Unexpected value for break_trailer.command(): got '%s', expected 'exec ls -l'", msg)
	}
}
