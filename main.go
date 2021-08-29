package main

import (
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
		fmt.Printf("USAGE: %s [rebase-todo-file]\n", os.Args[0])
		fmt.Printf("    Expects a list of instructions to be supplied on standard input.\n")
		fmt.Printf("    Writes a remastered rebase todo file to standard output.\n")
		fmt.Printf("    Those instructions must be of the form:\n")
		fmt.Printf("        [COMMAND] [COMMIT-ID] [ARGS]\n")
		fmt.Printf("    COMMAND must be a valid rebase command, or its abbreviation,\n")
		fmt.Printf("            or the special command 'override', or its abbreviation 'o'.\n")
		fmt.Printf("    COMMIT-ID must be an exactly matching abbreviated commit hash,\n")
		fmt.Printf("              or the special keyword 'default'.\n")
		fmt.Printf("    ARGS is only specified if COMMAND = {x, exec}, and is the command to run.\n")
		fmt.Printf("\n")
		fmt.Printf("    If COMMAND = {x, exec, b, break}, then rather than changing the existing\n")
		fmt.Printf("    line within the rebase message, a break or exec command will be inserted\n")
		fmt.Printf("    after it. Such commands will evaluate in the order they are specified in\n")
		fmt.Printf("    the control instructions.\n")
		fmt.Printf("\n")
		fmt.Printf("    The special keyword 'default' declares behavior for any commit not\n")
		fmt.Printf("    explicitly mentioned. For most commands, the specific command takes\n")
		fmt.Printf("    precedence. For break and exec, both specific and default statements\n")
		fmt.Printf("    are included, default ones first.\n")
		fmt.Printf("\n")
		fmt.Printf("    The special command 'override' and its abbreviation 'o' force\n")
		fmt.Printf("    the line from the rebase todo list to be echoed verbatim.  It is useful\n")
		fmt.Printf("    for overriding the behavior specified for the 'default' keyword, and\n")
		fmt.Printf("    choosing to perform the action prescribed initially by rebase.\n")
		fmt.Printf("    break and exec options are still honored since they are placed after\n")
		fmt.Printf("    this line.\n")
		fmt.Printf("\n")
		fmt.Printf("    The default behavior is to use the command specified by rebase, i.e.\n")
		fmt.Printf("    to behave as if 'override default' was specified.  Additionally, comments\n")
		fmt.Printf("    (starting with #) and blank lines in the rebase todo file are passed\n")
		fmt.Printf("    through verbatim. Quick note: you must write-new and swap, not\n")
		fmt.Printf("    overwrite the initial rebase todo file (for now).\n")
		os.Exit(0)
	}
}
