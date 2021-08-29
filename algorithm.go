package main

import (
	"fmt"
	"strings"
)

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
