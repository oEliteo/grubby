package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

type PrettyLogsHandler struct {
	out   io.Writer
	level slog.Level
}

func NewPrettyLogsHandler(w io.Writer, l slog.Level) *PrettyLogsHandler {
	h := &PrettyLogsHandler{
		out:   w,
		level: l,
	}
	return h
}

func (h *PrettyLogsHandler) Handle(_ context.Context, r slog.Record) error {
	timeStr := r.Time.Format(time.TimeOnly)

	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	switch r.Message {
	case "HTTP REQUEST":
		header := fmt.Sprintf("%-10s %-15s %-8s %-50s %-8s %-12s %s", "TIME", "MESSAGE", "METHOD", "PATH", "STATUS", "DURATION", "IP")
		values := fmt.Sprintf("%-10s %-15v %-8v %-50v %-8v %-12v %v",
			timeStr,
			r.Message,
			attrs["method"],
			attrs["path"],
			attrs["status"],
			attrs["duration"],
			attrs["ip"],
		)
		_, err := fmt.Fprintf(h.out, "\n%s\n%s\n", header, values)
		return err

	case "AMQP TASK":
		header := fmt.Sprintf("%-10s %-15s %-8s %-25s %-8s %s", "TIME", "MESSAGE", "TASK ID", "QUEUE", "DURATION", "STATUS")
		values := fmt.Sprintf("%-10s %-15v %-8v %-25v %-8v %v",
			timeStr,
			r.Message,
			attrs["task_id"],
			attrs["queue"],
			attrs["duration"],
			attrs["status"],
		)
		_, err := fmt.Fprintf(h.out, "\n%s\n%s\n", header, values)
		return err

	default:
		line := fmt.Sprintf("%s [%s] %s", timeStr, r.Level, r.Message)
		for k, v := range attrs {
			line += fmt.Sprintf(" | %s = %v", k, v)
		}
		_, err := fmt.Fprintln(h.out, line)
		return err
	}
}

func (h *PrettyLogsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PrettyLogsHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *PrettyLogsHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}
