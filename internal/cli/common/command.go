package common

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BenasB/bx2cloud/internal/cli/exits"
	"google.golang.org/grpc"
)

type CliCommand struct {
	description    string
	argDescription string
	flagSet        *flag.FlagSet
	handler        func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error)
	subcommands    []*CliCommand
}

func NewCliCommand(
	name string,
	description string,
	argDescription string,
	handler func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error),
) *CliCommand {
	return NewCliCommandWithFlags(name, description, argDescription, handler, func(fs *flag.FlagSet) {})
}

func NewCliCommandWithFlags(
	name string,
	description string,
	argDescription string,
	handler func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error),
	flagSetUp func(*flag.FlagSet),
) *CliCommand {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)

	flagSetUp(flagSet)

	return &CliCommand{
		description:    description,
		argDescription: argDescription,
		handler:        handler,
		flagSet:        flagSet,
	}
}

func NewCliSubcommand(name string, subcommands []*CliCommand) *CliCommand {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)

	return &CliCommand{
		subcommands: subcommands,
		flagSet:     flagSet,
	}
}

func (c *CliCommand) Execute(args []string, conn *grpc.ClientConn, cmdNameChain []string) exits.ExitCode {
	cmdNameChain = append(cmdNameChain, c.flagSet.Name())

	c.flagSet.Usage = func() {
		fullName := strings.Join(cmdNameChain, " ")
		if len(c.subcommands) == 0 {
			fmt.Fprintf(c.flagSet.Output(), "%s: %s\n", fullName, c.description)
			fmt.Fprintf(c.flagSet.Output(), "usage: %s %s\n", fullName, c.argDescription)

			fmt.Fprintf(c.flagSet.Output(), "flags:")
			hasFlags := false
			c.flagSet.VisitAll(func(f *flag.Flag) { hasFlags = true })
			if hasFlags {
				fmt.Fprintf(c.flagSet.Output(), "\n")
				c.flagSet.PrintDefaults()
			} else {
				fmt.Fprintf(c.flagSet.Output(), " none\n")
			}
		} else {
			FprintSubcommands(c.flagSet.Output(), fullName, c.subcommands)
		}
	}

	if err := c.flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return exits.SUCCESS
		}

		return exits.BAD_FLAG
	}

	args = c.flagSet.Args()

	if len(c.subcommands) == 0 {
		// TODO: Pass flag data onto handler

		if c.handler == nil {
			fmt.Fprintf(os.Stderr, "This command does not have a handler attached to it, please report to the developer\n")
			return exits.MISSING_SUBCOMMAND
		}

		exitCode, err := c.handler(args, conn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}

		return exitCode
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Missing subcommand\n")
		return exits.MISSING_SUBCOMMAND
	}

	subcommand := args[0]
	args = args[1:]

	for _, sc := range c.subcommands {
		if subcommand != sc.flagSet.Name() {
			continue
		}

		return sc.Execute(args, conn, cmdNameChain)
	}

	fmt.Fprintf(os.Stderr, "Unrecognized subcommand '%s'\n", subcommand)
	return exits.UNKNOWN_SUBCOMMAND
}

func FprintSubcommands(w io.Writer, cmdName string, subcommands []*CliCommand) {
	fmt.Fprintf(w, "%s subcommands:\n", cmdName)
	for _, sc := range subcommands {
		fmt.Fprintf(w, "  %s\n", sc.flagSet.Name())
	}
}
