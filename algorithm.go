package main

import (
	"fmt"
	"strings"
)

func push(s string, head *output_node) {
	head.insert_after(&output_node{line: s})
}

func strip_fixup_squash(msg string) string {
	for {
		token, new_msg := grab(msg)
		if token != "fixup!" && token != "squash!" { break }
		msg = new_msg
	}
	return msg
}

func push_commit(s, msg string, t []trailer, head *output_node, commits_by_message map[string]*output_node) *output_node {
	orig_msg := msg
	msg = strip_fixup_squash(msg)

	node := &output_node{line: s, msg: orig_msg, trailers: t}
	commits_by_message[msg] = node
	head.insert_after(node)
	return head
}

func relocate_commit(s, msg string, t []trailer, head *output_node, commits_by_message map[string]*output_node) (*output_node, error) {
	orig_msg := msg
	msg = strip_fixup_squash(msg)

	if orig_msg != msg {
		// if it is a fixup commit, look up the commit to apply it to by commit message.
		// it is an error to try to process a fixup which attaches to a commit outside the scope of the rebase.
		var ok bool
		head, ok = commits_by_message[msg]
		if !ok { return nil, fmt.Errorf("Can't apply fixup (subject commit is missing: %s)", msg) }
		head = head.prev
	} else {
		// this isn't a designated fixup commit, so just apply it to the head.
		// in this case, head is already correct (as passed by the caller).
	}

	node := &output_node{line: s, msg: orig_msg, trailers: t}
	head.insert_after(node)

	commits_by_message[msg] = node

	return head, nil
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


func parseInput(config map[string]reaction, scanner myscanner) (*output_node, *output_node, error) {
	head, tail := newList()
	var last *output_node

	commits_by_message := make(map[string]*output_node)

	// grab the default hash so we don't have to look it up a million times
	default_reaction := config["default"]

	for scanner.Scan() {
		raw_line := scanner.Text()
		line := strings.TrimSpace(raw_line)

		// blank lines and comments get passed through verbatim
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			push(raw_line, head)
			continue
		}

		// grab 2 tokens from the input
		token, remainder := grab(line)
		hash, remainder := grab(remainder)

		// if we don't recognize the command, just repeat it verbatim and proceed to the next.
		mode, ok := commands[token]
		if !ok {
			push(raw_line, head)
			continue
		}

		// look up a specific reaction to this hash, if one exists
		specific_reaction, ok := config[hash]

		// start with the default settings, and override them if necessary
		r := default_reaction
		if ok {
			r.mode = specific_reaction.mode
			r.auxiliary = append(r.auxiliary, specific_reaction.auxiliary...)
		}

		// override is special, it means "keep the line verbatim", so grab the command from the line
		if r.mode == commands["override"] {
			r.mode = command(mode)
		}

		if r.mode == commands["fixup"] || r.mode == commands["squash"] {
			if mode == commands["fixup"] || mode == commands["squash"] {
				// if the command came in as a fixup/squash, and is configured to remain a fixup/squash, then
				// it should remain bound to the commit it was originally attached to if that commit moves.
				last = push_commit(fmt.Sprintf("%s %s %s", r.mode, hash, remainder), remainder, r.auxiliary, last, commits_by_message)
			} else {
				// if we are converting it into a fixup/squash, then relocate it.
				var e error
				last, e = relocate_commit(fmt.Sprintf("%s %s %s", r.mode, hash, remainder), remainder, r.auxiliary, last, commits_by_message)
				if e != nil { return nil, nil, e }
			}
		} else {
			last = push_commit(fmt.Sprintf("%s %s %s", r.mode, hash, remainder), remainder, r.auxiliary, head, commits_by_message)
		}
	}

	return head, tail, nil
}
