package main

import (
	"bufio"
	"bytes"
	"fmt"
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
		"duplicates": {[][]string{[]string{"1", "msg1", "111"}, []string{"2", "msg2", "222"}, []string{"3", "msg1", "333"}}, trailers[1:4], 1},
		"several": {[][]string{[]string{"1", "msg1", "111"}, []string{"2", "msg2", "222"}, []string{"3", "msg3", "333"}}, trailers[0:3], 0},
		"one": {[][]string{[]string{"1", "msg1", "111"}}, trailers[1:2], 0},
	}

	for k, v := range testcases {
		t.Run(k, func(t *testing.T) {
			cmap := make(map[string]*output_node)
			hmap := make(map[string]*output_node)

			head, tail := newList()
			for i, s := range v.inputs {
				push_commit(s[0], s[1], s[2], []trailer{v.t[i]}, head, cmap, hmap)
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
			if i != len(hmap) {
				t.Errorf("Wrong number of distinct hashes: expected %d, got %d", i, len(hmap))
			}

			if i != len(v.inputs) {
				t.Errorf("Wrong number of values: expected %d, got %d", len(v.inputs), i)
			}

			for _, x := range v.inputs {
				if _, ok := hmap[x[2]]; !ok {
					t.Errorf("Hash map has missing entry: %s", x[2])
				}
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
		hmap map[string]int

		line, msg, hash, after string
		trailer trailer
		write_out_index int
	}{
		"first-missing": {
			[]output_node{},
			[]output_node{output_node{}},
			"",
			map[string]int{}, map[string]int{},
			"p 123 m1", "m1", "123", "", trailers[0], 0,
		},
		"second-collide": {
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]}},
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]}, output_node{}},
			"",
			map[string]int{"test": 0}, map[string]int{"123": 0},
			"p 456 fixup! test", "fixup! test", "456", "", trailers[1], 1,
		},
		"middle-collide": {
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]},
			              output_node{line: "p 234 test2", msg: "test2", trailers: trailers[1:2]}},
			[]output_node{output_node{line: "p 123 test", msg: "test", trailers: trailers[0:1]},
			              output_node{},
			              output_node{line: "p 234 test2", msg: "test2", trailers: trailers[1:2]}},
			"",
			map[string]int{"test": 0, "test2": 1}, map[string]int{"123": 0, "234": 1},
			"p 456 fixup! test", "fixup! test", "456", "", trailers[2], 1,
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
			map[string]int{"test": 1, "test2": 2}, map[string]int{"123": 0, "234": 1, "345": 2},
			"p 456 fixup! test", "fixup! test", "456", "", trailers[3], 2,
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
			map[string]int{"test": 2, "test2": 3}, map[string]int{"123": 0, "234": 1, "567": 2, "345": 3},
			"p 456 fixup! test", "fixup! test", "456", "", trailers[3], 3,
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
			hmap := make(map[string]*output_node)
			for k, vv := range v.hmap {
				hmap[k] = &v.input[vv]
			}

			oldnode := cmap[strip_fixup_squash(v.msg)]
			if _, ok := hmap[v.hash]; ok { t.Errorf("Hash present in map before starting! %s", v.hash) }
			_, err := relocate_commit(v.line, v.msg, v.hash, v.after, []trailer{v.trailer}, head, cmap, hmap)

			if err == nil && v.expected_err != "" || err != nil && (v.expected_err == "" || !strings.Contains(err.Error(), v.expected_err)) {
				t.Errorf("Unexpected error: got '%v', wanted '%s'", err, v.expected_err)
			}

			if err != nil { return }

			if oldnode == cmap[strip_fixup_squash(v.msg)] {
				t.Errorf("Commit message map for %s was not updated!", v.msg)
			}
			if _, ok := hmap[v.hash]; !ok { t.Errorf("Hash not present in map after starting! %s", v.hash) }

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
				"1125": reaction{mode: commands["fixup"], extra: "5555"},
				"1131": reaction{mode: commands["squash"]},
				"1133": reaction{mode: commands["squash"]},
				"1135": reaction{mode: commands["squash"], extra: "This is a string"},
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
  f      1125 5555
squash   1131
  s      1133
  s      1135 This is a string
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
squash 1111 extraaaa
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

func str(h, t *output_node) string {
	var b bytes.Buffer
	for n := h.next; n != t; n = n.next { b.WriteString(fmt.Sprintf("%+v\n", n)) }
	return b.String()
}

func Test_parseInput(t *testing.T) {
	testcases := map[string]struct {
		input map[string]reaction
		input_data string

		expected_err string
		output []output_node
	}{
		"simple": {
			map[string]reaction{
				"1111": reaction{mode: commands["drop"]},
			}, "pick 2222 m1\npick 1111 m2\npick 3333 m3\n", "", []output_node{
				output_node{line: "pick 2222 m1", msg: "m1"},
				output_node{line: "drop 1111 m2", msg: "m2"},
				output_node{line: "pick 3333 m3", msg: "m3"},
			},
		},
		"default": {
			map[string]reaction{
				"default": reaction{mode:commands["drop"]},
				"1111": reaction{mode: commands["pick"]},
			}, "pick 2222 m1\npick 1111 m2\npick 3333 m3\n", "", []output_node{
				output_node{line: "drop 2222 m1", msg: "m1"},
				output_node{line: "pick 1111 m2", msg: "m2"},
				output_node{line: "drop 3333 m3", msg: "m3"},
			},
		},
		"default-exec": {
			map[string]reaction{
				"default": reaction{mode:commands["override"], auxiliary: []trailer{exec_trailer{cmd: "./foobar.sh"}}},
				"1111": reaction{mode: commands["edit"]},
			}, "pick 2222 m1\npick 1111 m2\npick 3333 m3\n", "", []output_node{
				output_node{line: "pick 2222 m1", msg: "m1", trailers: []trailer{exec_trailer{cmd: "./foobar.sh"}}},
				output_node{line: "edit 1111 m2", msg: "m2", trailers: []trailer{exec_trailer{cmd: "./foobar.sh"}}},
				output_node{line: "pick 3333 m3", msg: "m3", trailers: []trailer{exec_trailer{cmd: "./foobar.sh"}}},
			},
		},
		"fixup-squash": {
			map[string]reaction{
				"7777": reaction{mode: commands["fixup"]},
				"8888": reaction{mode: commands["squash"]},
			}, "pick 2222 m1\npick 1111 m2\npick 3333 m3\npick 7777 fixup! m1\npick 8888 squash! m2", "", []output_node{
				output_node{line: "pick 2222 m1", msg: "m1"},
				output_node{line: "fixup 7777 fixup! m1", msg: "fixup! m1"},
				output_node{line: "pick 1111 m2", msg: "m2"},
				output_node{line: "squash 8888 squash! m2", msg: "squash! m2"},
				output_node{line: "pick 3333 m3", msg: "m3"},
			},
		},
		"multi-fixup-squash": {
			map[string]reaction{
				"444": reaction{mode: commands["fixup"]},
				"555": reaction{mode: commands["squash"]},
				"666": reaction{mode: commands["fixup"]},
				"777": reaction{mode: commands["fixup"]},
				"888": reaction{mode: commands["squash"]},
			}, "pick 111 m1\npick 222 m2\npick 333 m3\npick 444 fixup! m1\npick 555 squash! m1\npick 666 fixup! fixup! m1\npick 777 fixup! m1\npick 888 squash! squash! m1", "", []output_node{
				output_node{line: "pick 111 m1", msg: "m1"},
				output_node{line: "fixup 444 fixup! m1", msg: "fixup! m1"},
				output_node{line: "squash 555 squash! m1", msg: "squash! m1"},
				output_node{line: "fixup 666 fixup! fixup! m1", msg: "fixup! fixup! m1"},
				output_node{line: "fixup 777 fixup! m1", msg: "fixup! m1"},
				output_node{line: "squash 888 squash! squash! m1", msg: "squash! squash! m1"},
				output_node{line: "pick 222 m2", msg: "m2"},
				output_node{line: "pick 333 m3", msg: "m3"},
			},
		},
		"incoming-no-relocate": {
			map[string]reaction{
			}, "pick 111 m1\npick 222 m2\nfixup 333 fixup! m1\npick 444 m4", "", []output_node{
				output_node{line: "pick 111 m1", msg: "m1"},
				output_node{line: "pick 222 m2", msg: "m2"},
				output_node{line: "fixup 333 fixup! m1", msg: "fixup! m1"},
				output_node{line: "pick 444 m4", msg: "m4"},
			},
		},
		"incoming-yes-relocate": {
			map[string]reaction{
				"333": reaction{mode: commands["fixup"]},
			}, "pick 111 m1\npick 222 m2\npick 333 fixup! m1\npick 444 m4", "", []output_node{
				output_node{line: "pick 111 m1", msg: "m1"},
				output_node{line: "fixup 333 fixup! m1", msg: "fixup! m1"},
				output_node{line: "pick 222 m2", msg: "m2"},
				output_node{line: "pick 444 m4", msg: "m4"},
			},
		},
		"incoming-follow-relocate": {
			map[string]reaction{
				"333": reaction{mode: commands["fixup"]},
			}, "pick 111 m1\npick 222 m2\npick 333 fixup! m1\nfixup 444 m4", "", []output_node{
				output_node{line: "pick 111 m1", msg: "m1"},
				output_node{line: "fixup 333 fixup! m1", msg: "fixup! m1"},
				output_node{line: "fixup 444 m4", msg: "m4"},
				output_node{line: "pick 222 m2", msg: "m2"},
			},
		},
		"directed-relocate": {
			map[string]reaction{
				"333": reaction{mode: commands["fixup"], extra: "m1"},
				"444": reaction{mode: commands["fixup"], extra: "111"},
				"666": reaction{mode: commands["fixup"], extra: "111"},
			}, "pick 111 m1\npick 222 m2\npick 333 m3\npick 444 m4\npick 555 m5\nfixup 666 m6", "", []output_node{
				output_node{line: "pick 111 m1", msg: "m1"},
				output_node{line: "fixup 666 m6", msg: "m6"},
				output_node{line: "fixup 444 m4", msg: "m4"},
				output_node{line: "fixup 333 m3", msg: "m3"},
				output_node{line: "pick 222 m2", msg: "m2"},
				output_node{line: "pick 555 m5", msg: "m5"},
			},
		},
	}

	for k, v := range testcases {
		t.Run(k, func(t *testing.T) {
			expected_head, expected_tail := newList()
			for i := range v.output { expected_head.insert_after(&v.output[i]) }

			head, tail, err := parseInput(v.input, bufio.NewScanner(strings.NewReader(v.input_data)))

			if err == nil && v.expected_err != "" || err != nil && (v.expected_err == "" || !strings.Contains(err.Error(), v.expected_err)) {
				t.Errorf("Unexpected error: got '%v', wanted '%s'", err, v.expected_err)
			}

			if err != nil { return }

			if !reflect.DeepEqual(head, expected_head) { t.Errorf("Unexpected result: got:\n%s\n, expected:\n%v\n", str(head, tail), str(expected_head, expected_tail)) }
		})
	}
}
