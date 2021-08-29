package main

import (
	"fmt"
	"strings"
)

func push(s string, head *output_node) {
	head.insert_after(&output_node{line: s})
}

func push_commit(s, msg string, t []trailer, head *output_node, commits_by_message map[string]*output_node) {
	node := &output_node{line: s, msg: msg, trailers: t}
	commits_by_message[msg] = node
	head.insert_after(node)
}

func relocate_commit(s, msg string, t []trailer, head *output_node, commits_by_message map[string]*output_node) error {
	node := &output_node{line: s, msg: msg, trailers: t}
	commits_by_message[msg] = node
	for {
		token, new_msg := grab(msg)
		if token != "fixup!" && token != "squash!" {
			return fmt.Errorf("Couldn't figure out where to place commit: %s", s)
		}

		old_node, ok := commits_by_message[new_msg]
		if ok {
			for strings.Contains(old_node.msg, new_msg) && old_node.prev != nil {
				old_node = old_node.prev
			}
			old_node.insert_after(node)
			return nil
		}

		msg = new_msg
	}
	panic("not reached")
}

// typical implementer is bufio.Scanner
type myscanner interface {
	Scan() bool
	Text() string
}

func readSettings(input map[string]reaction, scanner myscanner) (map[string]reaction, error) {
	var n int
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// discard blank lines
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// grab a command, and barf if we don't recognize it
		token, line := grab(line)
		mode, ok := commands[token]
		if !ok { return nil, fmt.Errorf("Got a junk rebase command: %s (line %d)", token, n) }

		// grab a hash and barf if its empty
		hash, line := grab(line)
		if len(hash) == 0 { return nil, fmt.Errorf("Missing hash string (line %d)", n) }

		// look up the reaction for this hash and modify it.
		r := input[hash]
		if mode == commands["break"] {
			r.auxiliary = append(r.auxiliary, break_trailer{})
		} else if mode == commands["exec"] {
			r.auxiliary = append(r.auxiliary, exec_trailer{cmd: line})
		} else {
			r.mode = mode
		}
		input[hash] = r
		n++
	}

	return input, nil
}
