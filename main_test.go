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

func Test_output_node(t *testing.T) {
	var f func()
	defer func(){
		r := recover()
		if r != nil { f() }
	}()

	head, tail := newList()
	f = func() { t.Errorf("linked list creator helper is not properly initializing its contents! %p: %+v, %p: %+v", head, head, tail, tail) }

	if head.next.prev.next != tail {
		t.Errorf("linked list creator helper is not properly initializing its contents! %p: %+v, %p: %+v", head, head, tail, tail)
	}

	node := &output_node{}
	head.insert_after(node)
	f = func() { t.Errorf("node insertion is behaving strangely! %p: %+v, %p: %+v, %p: %+v", head, head, node, node, tail, tail) }

	if head.next.next.prev.prev.next.next != tail {
		t.Errorf("node insertion is behaving strangely! %p: %+v, %p: %+v, %p: %+v", head, head, node, node, tail, tail)
	}
}
