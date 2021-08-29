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

func Test_grab(t *testing.T) {
	testcases := map[string]struct{
		input, out1, out2 string
	}{
		"leading-space": {"    this is\t \na string withmanytokens\r", "this", "is\t \na string withmanytokens\r"},
		"normal": {"this is\t \na string withmanytokens\r", "this", "is\t \na string withmanytokens\r"},
		"end": {"token", "token", ""},
		"trailing-space": {"token   ", "token", ""},
	}

	for k, v := range testcases {
		t.Run(k, func(t *testing.T) {
			out1, out2 := grab(v.input)
			if out1 != v.out1 { t.Errorf("Unexpected token: got '%s', expected '%s'", out1, v.out1) }
			if out2 != v.out2 { t.Errorf("Unexpected trailer: got '%s', expected '%s'", out2, v.out2) }
		})
	}
}

func Test_push(t *testing.T) {
	testcases := map[string]struct {
		inputs []string
	}{
		"normal": {[]string{"123", "456", "789"}},
	}

	for k, v := range testcases {
		t.Run(k, func(t *testing.T) {
			head, tail := newList()
			for _, s := range v.inputs {
				push(s, head)
			}

			i := 0
			for node := tail.prev; node != head; node = node.prev {
				if i >= len(v.inputs) {
					t.Errorf("Out of range: %d (max %d)", i, len(v.inputs))
					if i > 10 { break }
				} else if node.line != v.inputs[i] {
					t.Errorf("Unexpected value at position %d: got %s, expected %s", i, v.inputs[i], node.line)
				}
				i++
			}

			if i != len(v.inputs) {
				t.Errorf("Wrong number of values: expected %d, got %d", len(v.inputs), i)
			}
		})
	}
}
