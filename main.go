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

	scanner = bufio.NewScanner(rebase_todo)
	for scanner.Scan() {
		raw_line := scanner.Text()
		line := strings.TrimSpace(raw_line)

		// blank lines and comments get passed through verbatim
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			fmt.Println(raw_line)
			continue
		}

		// grab 2 tokens from the input
		token, remainder := grab(line)
		hash, remainder := grab(remainder)

		// if we don't recognize the command, just repeat it verbatim and proceed to the next.
		_, ok := commands[token]
		if !ok {
			fmt.Println(raw_line)
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
			fmt.Println(raw_line)
		} else {
			fmt.Println(r.mode, hash, remainder)
		}

		for _, aux := range r.auxiliary {
			fmt.Println(aux.command())
		}
	}
}
