package command

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/utilivos/utilivos-cli/internal/logutil"
)

const (
	ExitOK     = 0
	ExitUsage  = 1
	ExitConfig = 2
	ExitApply  = 3
)

type Context struct {
	Stdout io.Writer
	Stderr io.Writer
	Logger *logutil.Logger
	Cwd    string
}

type Handler func(ctx *Context, args []string) (int, error)

type Node struct {
	Name        string
	Summary     string
	Description string
	Run         Handler
	Subcommands map[string]*Node
}

func (n *Node) Execute(ctx *Context, args []string) (int, error) {
	if len(args) == 0 {
		if n.Run != nil {
			return n.Run(ctx, args)
		}
		printHelp(ctx.Stdout, n)
		return ExitUsage, nil
	}

	next := strings.TrimSpace(args[0])
	if next == "" {
		printHelp(ctx.Stdout, n)
		return ExitUsage, nil
	}
	if next == "help" || next == "-h" || next == "--help" {
		printHelp(ctx.Stdout, n)
		return ExitOK, nil
	}
	if child, ok := n.Subcommands[next]; ok {
		return child.Execute(ctx, args[1:])
	}
	if n.Run != nil {
		return n.Run(ctx, args)
	}
	printHelp(ctx.Stderr, n)
	return ExitUsage, fmt.Errorf("unknown command: %s", next)
}

func printHelp(w io.Writer, n *Node) {
	if n.Description != "" {
		fmt.Fprintln(w, n.Description)
	}
	if len(n.Subcommands) == 0 {
		return
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	names := make([]string, 0, len(n.Subcommands))
	for name := range n.Subcommands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		child := n.Subcommands[name]
		fmt.Fprintf(w, "  %-14s %s\n", child.Name, child.Summary)
	}
}
