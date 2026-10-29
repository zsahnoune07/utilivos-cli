package logutil

import (
	"fmt"
	"io"
)

type Logger struct {
	out     io.Writer
	verbose bool
}

func New(out io.Writer, verbose bool) *Logger {
	return &Logger{out: out, verbose: verbose}
}

func (l *Logger) Infof(format string, args ...any) {
	fmt.Fprintf(l.out, "INFO: "+format+"\n", args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	fmt.Fprintf(l.out, "WARN: "+format+"\n", args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	fmt.Fprintf(l.out, "ERROR: "+format+"\n", args...)
}

func (l *Logger) Debugf(format string, args ...any) {
	if !l.verbose {
		return
	}
	fmt.Fprintf(l.out, "DEBUG: "+format+"\n", args...)
}
