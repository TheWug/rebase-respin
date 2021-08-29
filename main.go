package main

import (
	"bufio"
	"fmt"
	"os"
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

	config := make(map[string]reaction)

	config, err = readSettings(config, bufio.NewScanner(os.Stdin))
	if err != nil { die("%s", err) }

	head, tail, err := parseInput(config, bufio.NewScanner(rebase_todo))
	if err != nil { die("%s", err) }

	for node := tail.prev; node != head; node = node.prev {
		fmt.Println(node.line)
		for _, t := range node.trailers {
			fmt.Println(t.command())
		}
	}
}
