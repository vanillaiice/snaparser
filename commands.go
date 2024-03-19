package snaparser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/vanillaiice/snaparser/parser"
)

var Parse = &cli.Command{
	Name:    "parse",
	Aliases: []string{"p"},
	Usage:   "parse chats",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "file",
			Aliases: []string{"f"},
			Usage:   "read chats from `FILE`",
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
	},
	Action: func(ctx *cli.Context) error {
		var in io.Reader
		var w *bufio.Writer

		file := ctx.String("file")
		dir := ctx.Path("dir")
		write := ctx.Bool("write")
		create := ctx.Bool("create")
		user := ctx.String("user")

		if file == "" {
			fi, err := os.Stdin.Stat()
			if err != nil {
				return err
			}
			if (fi.Mode() & os.ModeCharDevice) == 0 {
				in = os.Stdin
			} else {
				return fmt.Errorf("no input file provided")
			}
		} else {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()
			in = f
		}

		if write && dir != "" {
			if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
				if create {
					if err = os.Mkdir(dir, os.ModePerm); err != nil {
						return err
					}
				}
			}
			if err := os.Chdir(dir); err != nil {
				return err
			}
		}

		if user == "" {
			users, data, err := parser.ParseAll(in)
			if err != nil {
				return err
			}

			for _, u := range users {
				if len(data[u]) == 0 {
					continue
				}

				if write {
					CheckIllegalString(&u)
					userFile, err := os.Create(u + ".txt")
					if err != nil {
						return err
					}
					defer userFile.Close()
					w = bufio.NewWriter(userFile)
				}

				for j := len(data[u]) - 1; j >= 0; j-- {
					if data[u][j].MediaType != "TEXT" {
						continue
					}
					s := fmt.Sprintf("%s (%s): %s\n", data[u][j].From, data[u][j].Created, data[u][j].Content)

					if write {
						if _, err := w.WriteString(s); err != nil {
							return err
						}
					} else {
						fmt.Fprint(os.Stdout, s)
					}
				}
			}
		} else {
			data, err := parser.ParseUser(in, user)
			if err != nil {
				return err
			}

			if write {
				CheckIllegalString(&user)
				userFile, err := os.Create(user + ".txt")
				if err != nil {
					return err
				}
				defer userFile.Close()
				w = bufio.NewWriter(userFile)
			}

			for i := len(data) - 1; i >= 0; i-- {
				if data[i].MediaType != "TEXT" {
					continue
				}
				s := fmt.Sprintf("%s (%s): %s\n", data[i].From, data[i].Created, data[i].Content)

				if write {
					if _, err := w.WriteString(s); err != nil {
						return err
					}
				} else {
					fmt.Fprint(os.Stdout, s)
				}
			}
		}

		if w != nil && w.Size() > 0 {
			if err := w.Flush(); err != nil {
				return err
			}
		}

		return nil
	},
}
