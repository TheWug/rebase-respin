package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

type command string
type trailer interface {
	command() string
}

type break_trailer struct {}
func (t break_trailer) command() string { return "break" }

type exec_trailer struct {
	cmd string
}
func (t exec_trailer) command() string { return fmt.Sprintf("exec %s", t.cmd) }

type reaction struct {
	mode command
	auxiliary []trailer
}

type output_node struct {
	next, prev *output_node
	line string
	msg string
	trailers []trailer
}

func (this *output_node) insert_after(n *output_node) {
	next := this.next
	next.prev, this.next = n, n
	n.prev, n.next = this, next
}

var commands = map[string]command{
	"pick":   "pick",
	"p":      "pick",
	"reword": "reword",
	"r":      "reword",
	"edit":   "edit",
	"e":      "edit",
	"squash": "squash",
	"s":      "squash",
	"fixup":  "fixup",
	"f":      "fixup",
	"drop":   "drop",
	"d":      "drop",
	"exec":   "exec",
	"x":      "exec",
	"break":  "break",
	"b":      "break",
	"override": "",
	"o":        "",
	"label":    "",
	"l":        "",
	"reset":    "",
	"t":        "",
	"merge":    "",
	"m":        "",
	"":         "",
}

// grab one token off the front of a string.
// token boundaries are whitespace.
// whitespace is trimmed from the remainder string.
// if there is no suitable token, an empty string is returned ad infinitum.
func grab(s string) (string, string) {
	var i int
	var r rune
	var end bool
	end = true
	for i, r = range s {
		if !unicode.IsSpace(r) {
			end = false
			break
		}
	}
	if end { i = len(s) }
	s = s[i:]

	end = true
	for i, r = range s {
		if unicode.IsSpace(r) {
			end = false
			break
		}
	}
	if end { i = len(s) }
	out := s[:i]
	s = s[i:]

	end = true
	for i, r = range s {
		if !unicode.IsSpace(r) {
			end = false
			break
		}
	}
	if end { i = len(s) }
	s = s[i:]

	return out, s
}

// fail_to_parse_args complains and exits the program if called.
func die(format string, objs ...interface{}) {
	println(fmt.Sprintf(format, objs...))
	os.Exit(1)
}

func main() {
	if len(os.Args) != 2 {
		die("Unexpected arguments: only wanted 1, got %d", len(os.Args) - 1)
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		showUsage()
		os.Exit(0)
	}

	rebase_todo, err := os.Open(os.Args[1])
	if err != nil { die("Error opening \"%s\" for read: %s", os.Args[1], err) }

	input := make(map[string]reaction)
	n := 1

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// discard blank lines
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// grab a command, and barf if we don't recognize it
		token, line := grab(line)
		mode, ok := commands[token]
		if !ok { die("Got a junk rebase command: %s (line %d)", token, n) }

		// grab a hash and barf if its empty
		hash, line := grab(line)
		if len(hash) == 0 { die("Missing hash string (line %d)", n) }

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

	// grab the default hash so we don't have to look it up a million times
	default_reaction := input["default"]

	commits_by_message := make(map[string]*output_node)
	var head, tail output_node
	head.next, tail.prev = &tail, &head

	push := func(s string) {
		head.insert_after(&output_node{line: s})
	}

	push_commit := func(s, msg string, t []trailer) {
		node := &output_node{line: s, msg: msg, trailers: t}
		commits_by_message[msg] = node
		head.insert_after(node)
	}

	relocate_commit := func(s, msg string, t []trailer) {
		node := &output_node{line: s, msg: msg, trailers: t}
		commits_by_message[msg] = node
		for {
			token, new_msg := grab(msg)
			if token != "fixup!" && token != "squash!" {
				die("Couldn't figure out where to place commit: %s", s)
			}

			old_node, ok := commits_by_message[new_msg]
			if ok {
				for strings.Contains(old_node.msg, new_msg) && old_node.prev != nil {
					old_node = old_node.prev
				}
				old_node.insert_after(node)
				return
			}

			msg = new_msg
		}
	}

	scanner = bufio.NewScanner(rebase_todo)
	for scanner.Scan() {
		raw_line := scanner.Text()
		line := strings.TrimSpace(raw_line)

		// blank lines and comments get passed through verbatim
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			push(raw_line)
			continue
		}

		// grab 2 tokens from the input
		token, remainder := grab(line)
		hash, remainder := grab(remainder)

		// if we don't recognize the command, just repeat it verbatim and proceed to the next.
		mode, ok := commands[token]
		if !ok {
			push(raw_line)
			continue
		}

		// look up a specific reaction to this hash, if one exists
		specific_reaction, ok := input[hash]

		// start with the default settings, and override them if necessary
		r := default_reaction
		if ok {
			r.mode = specific_reaction.mode
			r.auxiliary = append(r.auxiliary, specific_reaction.auxiliary...)
		}

		// override is special, it means "keep the line verbatim", but we might
		// still want to process trailers
		if r.mode == commands["override"] {
			push_commit(raw_line, remainder, r.auxiliary)
		} else if r.mode == commands["fixup"] && mode != commands["fixup"] {
			relocate_commit(fmt.Sprintf("%s %s %s", mode, hash, remainder), remainder, r.auxiliary)
		} else if r.mode == commands["squash"] && mode != commands["squash"] {
			relocate_commit(fmt.Sprintf("%s %s %s", mode, hash, remainder), remainder, r.auxiliary)
		} else {
			push_commit(fmt.Sprintf("%s %s %s", mode, hash, remainder), remainder, r.auxiliary)
		}
	}

	for node := tail.prev; node != &head; node = node.prev {
		fmt.Println(node.line)
		for _, t := range node.trailers {
			fmt.Println(t.command())
		}
	}
}
