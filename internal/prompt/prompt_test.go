package prompt

import (
	"bytes"
	"strings"
	"testing"
)

func TestSelect(t *testing.T) {
	t.Parallel()

	in := strings.NewReader("3\n2\n")
	out := &bytes.Buffer{}
	p := &IO{In: in, Out: out, ErrOut: out}

	idx, err := p.Select("Multiple MFA devices found.", []string{"one", "two", "three"})
	if err != nil {
		t.Fatal(err)
	}
	if idx != 2 {
		t.Fatalf("idx = %d, want 2", idx)
	}

	idx, err = p.Select("again", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if idx != 1 {
		t.Fatalf("idx = %d, want 1", idx)
	}
}

func TestSelectSingle(t *testing.T) {
	t.Parallel()

	p := &IO{In: strings.NewReader(""), Out: &bytes.Buffer{}}
	idx, err := p.Select("title", []string{"only"})
	if err != nil {
		t.Fatal(err)
	}
	if idx != 0 {
		t.Fatalf("idx = %d", idx)
	}
}

func TestConfirm(t *testing.T) {
	t.Parallel()

	p := &IO{In: strings.NewReader("y\n"), Out: &bytes.Buffer{}}
	ok, err := p.Confirm("create file?")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected yes")
	}

	p = &IO{In: strings.NewReader("n\n"), Out: &bytes.Buffer{}}
	ok, err = p.Confirm("create file?")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no")
	}
}
