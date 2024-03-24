/*
Snaparser parses snapchat chat history json files to human friendly format.
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
		Version: "v0.1.4",
		Authors: []*cli.Author{{Name: "vanillaiice", Email: "vanillaiice1@proton.me"}},
		Usage:   "parse snapchat chat history to human friendly format",
		Flags:   snaparser.ParseCommand.Flags,
		Action:  snaparser.ParseCommand.Action,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
