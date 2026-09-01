package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

func newLogger(level string, w io.Writer) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(&plainHandler{w: w, level: lvl})
}

type plainHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
}

func (h *plainHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *plainHandler) Handle(_ context.Context, r slog.Record) error {
	msg := r.Message
	write := func(a slog.Attr) {
		if a.Equal(slog.Attr{}) {
			return
		}
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
	}
	for _, a := range h.attrs {
		write(a)
	}
	r.Attrs(func(a slog.Attr) bool {
		write(a)
		return true
	})
	_, err := fmt.Fprintf(h.w, "%s - %s\n", strings.ToUpper(r.Level.String()), msg)
	return err
}

func (h *plainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &cp
}

func (h *plainHandler) WithGroup(string) slog.Handler { return h }
