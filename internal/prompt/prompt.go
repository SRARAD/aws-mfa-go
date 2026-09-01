package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// IO is the interactive prompt surface. Tests can replace Reader/Writer.
type IO struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
	// ReadPassword reads a secret from a file descriptor. Defaults to term.ReadPassword.
	ReadPassword func(fd int) ([]byte, error)
	// IsTerminal reports whether stdout is a TTY. Defaults to term.IsTerminal.
	IsTerminal func(fd int) bool
	br         *bufio.Reader
}

func Default() *IO {
	return &IO{
		In:           os.Stdin,
		Out:          os.Stdout,
		ErrOut:       os.Stderr,
		ReadPassword: term.ReadPassword,
		IsTerminal:   term.IsTerminal,
	}
}

// Confirm asks a yes/no question. Empty input is no.
func (p *IO) Confirm(question string) (bool, error) {
	answer, err := p.Line("%s [y/n]: ", question)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// Line prints a prompt and reads a single line.
func (p *IO) Line(format string, args ...any) (string, error) {
	if _, err := fmt.Fprintf(p.Out, format, args...); err != nil {
		return "", err
	}
	reader := p.lineReader()
	line, err := reader.ReadString('\n')
	if err != nil && (err != io.EOF || line == "") {
		if err == io.EOF {
			return "", errors.New("no input")
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (p *IO) lineReader() *bufio.Reader {
	if p.br == nil {
		p.br = bufio.NewReader(p.In)
	}
	return p.br
}

// Secret prompts without echoing input when stdin is a terminal.
func (p *IO) Secret(label string) (string, error) {
	if _, err := fmt.Fprintf(p.Out, "%s: ", label); err != nil {
		return "", err
	}
	if p.canReadPassword() {
		b, err := p.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(p.Out)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	return p.Line("")
}

func (p *IO) canReadPassword() bool {
	if p.ReadPassword == nil {
		return false
	}
	check := p.IsTerminal
	if check == nil {
		check = term.IsTerminal
	}
	return check(int(os.Stdin.Fd()))
}

// Select presents a numbered list and returns the chosen index.
func (p *IO) Select(title string, labels []string) (int, error) {
	if len(labels) == 0 {
		return 0, errors.New("nothing to select")
	}
	if len(labels) == 1 {
		return 0, nil
	}
	if _, err := fmt.Fprintf(p.Out, "%s\n\n", title); err != nil {
		return 0, err
	}
	for i, label := range labels {
		if _, err := fmt.Fprintf(p.Out, "  %d) %s\n", i+1, label); err != nil {
			return 0, err
		}
	}
	for {
		line, err := p.Line("\nSelect a device [1-%d]: ", len(labels))
		if err != nil {
			return 0, err
		}
		n, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr != nil || n < 1 || n > len(labels) {
			if _, err := fmt.Fprintf(p.Out, "Please enter a number between 1 and %d.\n", len(labels)); err != nil {
				return 0, err
			}
			continue
		}
		return n - 1, nil
	}
}
