package main

import (
	"bufio"
	"reflect"
	"strings"
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

func Test_strip_fixup_squash(t *testing.T) {
	testcases := map[string]struct {
		in, out string
	}{
		"no-op": {"unchanged", "unchanged"},
		"simple-fixup": {"fixup! changed", "changed"},
		"simple-squash": {"squash! changed", "changed"},
		"complex-multiple-weird-whitespace": {"\tfixup!     squash!\tchanged with some more words", "changed with some more words"},
	}

	for k, v := range testcases {
		t.Run(k, func(t *testing.T) {
			out := strip_fixup_squash(v.in)

			if v.out != out {
				t.Errorf("Unexpected result: got %s, expected %s", out, v.out)
			}
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

func Test_push_commit(t *testing.T) {
	trailers := []trailer{nil, break_trailer{}, exec_trailer{cmd: "foo"}, exec_trailer{cmd: "bar"}}

	testcases := map[string]struct {
		inputs [][]string
		t []trailer
		conflicts int
	}{
		"duplicates": {[][]string{[]string{"1", "msg1"}, []string{"2", "msg2"}, []string{"3", "msg1"}}, trailers[1:4], 1},
		"several": {[][]string{[]string{"1", "msg1"}, []string{"2", "msg2"}, []string{"3", "msg3"}}, trailers[0:3], 0},
		"one": {[][]string{[]string{"1", "msg1"}}, trailers[1:2], 0},
	}

	for k, v := range testcases {
		t.Run(k, func(t *testing.T) {
			cmap := make(map[string]*output_node)

			head, tail := newList()
			for i, s := range v.inputs {
				push_commit(s[0], s[1], []trailer{v.t[i]}, head, cmap)
			}

			i := 0
			for node := tail.prev; node != head; node = node.prev {
				if i >= len(v.inputs) {
					t.Errorf("Out of range: %d (max %d)", i, len(v.inputs))
					if i > 10 { break }
				} else {
					if node.line != v.inputs[i][0] {
						t.Errorf("Unexpected value at position %d: got %s, expected %s", i, v.inputs[i][0], node.line)
					}
					if node.trailers[0] != v.t[i] {
						t.Errorf("Unexpected trailer at position %d: got %v, expected %v", i, v.t[i], node.trailers[i])
					}
				}
				i++
			}

			if i - v.conflicts != len(cmap) {
				t.Errorf("Wrong number of distinct commits: expected %d, got %d", i - v.conflicts, len(cmap))
			}

			if i != len(v.inputs) {
				t.Errorf("Wrong number of values: expected %d, got %d", len(v.inputs), i)
			}

			for k, v := range cmap {
				if k != v.msg {
					t.Errorf("Commit message map has bad entry: %s -> %s", k, v.msg)
				}
			}
		})
	}
}

func Test_relocate_commit(t *testing.T) {
	trailers := []trailer{nil, break_trailer{}, exec_trailer{cmd: "foo"}, exec_trailer{cmd: "bar"}}

	testcases := map[string]struct {
		input []output_node
		output []output_node
		expected_err string
		cmap map[string]int

		line, msg string
		trailer trailer
		write_out_index int
	}{
		"first-missing": {
			[]output_node{},
			[]output_node{output_node{}},
			"",
			map[string]int{},
			"p 123 m1", "m1", trailers[0], 0,
		},
		"second-collide": {
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]}},
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]}, output_node{}},
			"",
			map[string]int{"test": 0},
			"p 456 fixup! test", "fixup! test", trailers[1], 1,
		},
		"middle-collide": {
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]},
			              output_node{line: "p 234 test2", msg: "test2", trailers: trailers[1:2]}},
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]},
			              output_node{},
			              output_node{line: "p 234 test2", msg: "test2", trailers: trailers[1:2]}},
			"",
			map[string]int{"test": 0, "test2": 1},
			"p 456 fixup! test", "fixup! test", trailers[2], 1,
		},
		"multi-fixup": {
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]},
			              output_node{line: "p 234 fixup! test", msg: "fixup! test", trailers: trailers[2:3]},
			              output_node{line: "p 345 test2", msg: "test2", trailers: trailers[1:2]}},
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]},
			              output_node{line: "p 234 fixup! test", msg: "fixup! test", trailers: trailers[2:3]},
			              output_node{},
			              output_node{line: "p 345 test2", msg: "test2", trailers: trailers[1:2]}},
			"",
			map[string]int{"test": 1, "test2": 2},
			"p 456 fixup! test", "fixup! test", trailers[3], 2,
		},
		"nested-fixup": {
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]},
			              output_node{line: "p 234 fixup! test", msg: "fixup! test", trailers: trailers[2:3]},
			              output_node{line: "p 567 fixup! fixup! test", msg: "fixup! fixup! test", trailers: trailers[3:4]},
			              output_node{line: "p 345 test2", msg: "test2", trailers: trailers[1:2]}},
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]},
			              output_node{line: "p 234 fixup! test", msg: "fixup! test", trailers: trailers[2:3]},
			              output_node{line: "p 567 fixup! fixup! test", msg: "fixup! fixup! test", trailers: trailers[3:4]},
			              output_node{},
			              output_node{line: "p 345 test2", msg: "test2", trailers: trailers[1:2]}},
			"",
			map[string]int{"test": 2, "test2": 3},
			"p 456 fixup! test", "fixup! test", trailers[3], 3,
		},
	}

	for k, v := range testcases {
		t.Run(k, func(t *testing.T) {
			if v.write_out_index != -1 { v.output[v.write_out_index] = output_node{line: v.line, msg: v.msg, trailers: []trailer{v.trailer}} }

			head, tail := newList()
			for i := range v.input { head.insert_after(&v.input[i]) }

			cmap := make(map[string]*output_node)
			for k, vv := range v.cmap {
				cmap[k] = &v.input[vv]
			}

			oldnode := cmap[strip_fixup_squash(v.msg)]
			_, err := relocate_commit(v.line, v.msg, []trailer{v.trailer}, head, cmap)

			if err == nil && v.expected_err != "" || err != nil && (v.expected_err == "" || !strings.Contains(err.Error(), v.expected_err)) {
				t.Errorf("Unexpected error: got '%v', wanted '%s'", err, v.expected_err)
			}

			if err != nil { return }

			if oldnode == cmap[strip_fixup_squash(v.msg)] {
				t.Errorf("Commit message map for %s was not updated!", v.msg)
			}

			i := 0
			for node := tail.prev; node != head; node = node.prev {
				if node.line != v.output[i].line { t.Errorf("Unexpected line at node %d: got %s, expected %s", i, node.line, v.output[i].line) }
				if node.msg != v.output[i].msg { t.Errorf("Unexpected message at node %d: got %s, expected %s", i, node.msg, v.output[i].msg) }
				if node.trailers[0] != v.output[i].trailers[0] { t.Errorf("Unexpected message at node %d: got %v, expected %v", i, node.trailers[0], v.output[i].trailers[0]) }
				i++
			}

			if i != len(v.output) { t.Errorf("Wrong number of output nodes: got %d, expected %d", i, len(v.output)) }
		})
	}
}

func Test_readSettings(t *testing.T) {
	testcases := map[string]struct{
		input, output map[string]reaction
		input_data string
		expected_err string
	}{
		"individual": {
			map[string]reaction{},
			map[string]reaction{
				"1111": reaction{mode: commands["pick"]},
				"1113": reaction{mode: commands["pick"]},
				"1141": reaction{mode: commands["reword"]},
				"1143": reaction{mode: commands["reword"]},
				"1151": reaction{mode: commands["edit"]},
				"1153": reaction{mode: commands["edit"]},
				"1121": reaction{mode: commands["fixup"]},
				"1123": reaction{mode: commands["fixup"]},
				"1131": reaction{mode: commands["squash"]},
				"1133": reaction{mode: commands["squash"]},
				"1161": reaction{mode: commands["drop"]},
				"1163": reaction{mode: commands["drop"]},
				"1171": reaction{mode: commands["override"], auxiliary: []trailer{exec_trailer{cmd: "./test.sh arg1"}}},
				"1173": reaction{mode: commands["override"], auxiliary: []trailer{exec_trailer{cmd: "./test.sh arg3"}}},
				"1181": reaction{mode: commands["override"], auxiliary: []trailer{break_trailer{}}},
				"1183": reaction{mode: commands["override"], auxiliary: []trailer{break_trailer{}}},
				"1191": reaction{mode: commands["override"]},
				"1193": reaction{mode: commands["override"]},
			},
`
pick     1111
  p      1113
reword   1141
  r      1143
edit     1151
  e      1153
fixup    1121
  f      1123
squash   1131
  s      1133
drop     1161
  d      1163
exec     1171 ./test.sh arg1
  x      1173 ./test.sh arg3
break    1181
  b      1183
override 1191
  o      1193
`, "",
		},
		"bad-command": {
			map[string]reaction{},
			map[string]reaction{},
			"missing-command 1111", "Got a junk rebase command",
		},
		"missing-hash": {
			map[string]reaction{},
			map[string]reaction{},
			"pick 1111\n  pick  \npick 1112", "Missing hash string",
		},
		"missing-exec-command": {
			map[string]reaction{},
			map[string]reaction{
				"1111": reaction{mode: commands["pick"]},
				"1112": reaction{mode: commands["pick"]},
				"2222": reaction{mode: commands["override"], auxiliary: []trailer{exec_trailer{cmd: ""}}},
			},
			"pick 1111\n  exec 2222  \npick 1112", "",
		},
		"stack-everything-up": {
			map[string]reaction{},
			map[string]reaction{
				"1111": reaction{mode: commands["squash"], auxiliary: []trailer{
					exec_trailer{cmd: "./foobar"},
					break_trailer{},
					exec_trailer{cmd: "./foobar2"},
					break_trailer{},
				}},
			},
`
pick 1111
reword 1111
edit 1111
fixup 1111
squash 1111
drop 1111
exec 1111 ./foobar
break 1111
exec 1111 ./foobar2
break 1111
squash 1111
`, "",
		},
	}

	for k, v := range testcases {
		t.Run(k, func(t *testing.T) {
			out, err := readSettings(v.input, bufio.NewScanner(strings.NewReader(v.input_data)))

			if err == nil && v.expected_err != "" || err != nil && (v.expected_err == "" || !strings.Contains(err.Error(), v.expected_err)) {
				t.Errorf("Unexpected error: got '%v', wanted '%s'", err, v.expected_err)
			}

			if err != nil { return }

			if !reflect.DeepEqual(v.output, out) { t.Errorf("Unexpected result: got:\n%v\n\n, expected:\n%v\n\n", out, v.output) }
		})
	}
}
