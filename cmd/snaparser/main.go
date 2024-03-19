/*
Snaparser parses snapchat chat history json files to human friendly format.
The output can be printed to stdout or written to file.
If no path is provided, stdin is processed.
If stdin is empty, the program stops execution.
By default, the parsed chat history is printed to stdout.
*/
package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/vanillaiice/snaparser"
)

func main() {
	app := &cli.App{
		Name:    "snaparser",
		Suggest: true,
		Version: "v0.0.7",
		Authors: []*cli.Author{{Name: "vanillaiice", Email: "vanillaiice1@proton.me"}},
		Usage:   "parse snapchat chat history to human friendly format",
		Commands: []*cli.Command{
			snaparser.Parse,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
