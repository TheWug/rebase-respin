package main

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

func main() {
}
