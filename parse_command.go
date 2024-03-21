package snaparser

import (
	"errors"
	"io"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/vanillaiice/snaparser/parser"
)

// Empty represents an empty string
const Empty = ""

// Stdin represents receiving input from standard input
const Stdin = "-"

// ErrNoInput means that no snapchat history file was provided
var ErrNoInput = errors.New("no input file provided")

// flags represent the flags used by the cli
var flags = []cli.Flag{
	&cli.StringFlag{
		Name:     "file",
		Aliases:  []string{"f"},
		Usage:    "read chats from `FILE` (use '-' for stdin)",
		Required: true,
	},
	&cli.StringFlag{
		Name:    "user",
		Aliases: []string{"u"},
		Usage:   "only extract chat with user",
	},
	&cli.BoolFlag{
		Name:    "write",
		Aliases: []string{"w"},
		Usage:   "write parsed chats to disk",
	},
	&cli.PathFlag{
		Name:    "directory",
		Aliases: []string{"d", "dir"},
		Usage:   "write parsed chats to `DIRECTORY`",
	},
	&cli.BoolFlag{
		Name:    "create",
		Aliases: []string{"c"},
		Usage:   "create directory if it does not exist",
	},
}

// parseFlags is a struct containing flags provided by the cli
type parseFlags struct {
	file, dir, user string
	write, create   bool
}

// parseFunc parses snapchat chat history according to the provided flags
var parseFunc = func(parseFlags *parseFlags) error {
	var r io.Reader
	var w io.Writer

	if parseFlags.file == Stdin {
		fi, err := os.Stdin.Stat()
		if err != nil {
			return err
		}
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			r = os.Stdin
		} else {
			return ErrNoInput
		}
	} else {
		f, err := os.Open(parseFlags.file)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}

	if parseFlags.write {
		if parseFlags.dir != Empty {
			if _, err := os.Stat(parseFlags.dir); errors.Is(err, os.ErrNotExist) {
				if parseFlags.create {
					if err = os.Mkdir(parseFlags.dir, os.ModePerm); err != nil {
						return err
					}
				}
			}
			if err := os.Chdir(parseFlags.dir); err != nil {
				return err
			}
		}
	} else {
		w = os.Stdout
	}

	if parseFlags.user == Empty {
		data, err := parser.ParseAll(r)
		if err != nil {
			return err
		}

		for user, content := range data {
			if parseFlags.write {
				f, err := createUserFile(&user)
				if err != nil {
					return err
				}
				defer f.Close()
				w = f
			}

			for j := len(content) - 1; j >= 0; j-- {
				if err = writeData(&content[j], w); err != nil {
					return err
				}
			}
		}
	} else {
		data, err := parser.ParseUser(r, parseFlags.user)
		if err != nil {
			return err
		}

		if parseFlags.write {
			f, err := createUserFile(&parseFlags.user)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
		}

		for i := len(data) - 1; i >= 0; i-- {
			if err = writeData(&data[i], w); err != nil {
				return err
			}
		}
	}

	return nil
}

// ParseCommand is a cli command that parses snapchat chat history
var ParseCommand = &cli.Command{
	Flags: flags,
	Action: func(ctx *cli.Context) error {
		parseFlags := &parseFlags{
			file:   ctx.String("file"),
			dir:    ctx.Path("dir"),
			user:   ctx.String("user"),
			write:  ctx.Bool("write"),
			create: ctx.Bool("create"),
		}

		return parseFunc(parseFlags)
	},
}
