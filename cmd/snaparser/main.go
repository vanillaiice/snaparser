/*
Snaparser parses snapchat chat history json files to human friendly format.
The output can be printed to stdout or written to file.
If no path is provided, stdin is processed.
If stdin is empty, the program stops execution.
By default, the parsed chat history is printed to stdout.

Usage:

	snaparser [flags] [path ...]

The options are:

	-u
			Only extract chats with this user.
	-w
			Write chats to file.
	-d
			Write to this directory.
	-f
			Create directory if it does not exist.
*/
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	parser "github.com/vanillaiice/snaparser/parser"
)

// checkIllegalString checks if a string contains '/' characters.
// If yes, they are replaced with '-'
func checkIllegalString(str *string) {
	if strings.Contains(*str, "/") == true {
		*str = strings.ReplaceAll(*str, "/", "-")
	}
}

func main() {
	user := flag.String("u", "", "only extract chats with this user")
	writeToFile := flag.Bool("w", false, "write chats to file")
	dir := flag.String("d", "", "write to this directory")
	forceDir := flag.Bool("f", false, "create directory if it does not exist")
	flag.Parse()

	var in io.Reader

	if filename := flag.Arg(0); filename != "" {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalln(err)
		}
		defer f.Close()
		in = f
	} else {
		fi, err := os.Stdin.Stat()
		if err != nil {
			log.Fatalln(err)
		}
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			in = os.Stdin
		} else {
			fmt.Fprintln(os.Stderr, "Usage: snaparser [flags] [path ...]")
			os.Exit(1)
		}
	}

	if *writeToFile == true && *dir != "" {
		if _, err := os.Stat(*dir); errors.Is(err, os.ErrNotExist) {
			if *forceDir == true {
				if err = os.Mkdir(*dir, os.ModePerm); err == nil {
					err = os.Chdir(*dir)
					if err != nil {
						log.Fatalln(err)
					}
				}
			} else {
				log.Fatalln(err)
			}
		} else {
			err = os.Chdir(*dir)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}

	var writer *bufio.Writer
	if *user != "" {
		data, err := parser.ParseUser(in, *user)
		if err != nil {
			log.Fatalln(err)
		}

		if len(data) == 0 {
			log.Printf("No chats with user - %s\n", &user)
			os.Exit(0)
		}

		if *writeToFile == true {
			checkIllegalString(user)
			userFile, err := os.Create(fmt.Sprintf("%s.txt", *user))
			if err != nil {
				log.Fatalln(err)
			}
			defer userFile.Close()

			writer = bufio.NewWriter(userFile)
		}

		for i := len(data) - 1; i >= 0; i-- {
			if data[i].MediaType != "TEXT" {
				continue
			}
			str := fmt.Sprintf("%s (%s): %s\n", data[i].From, data[i].Created, data[i].Content)
			if *writeToFile == false {
				fmt.Print(str)
			} else {
				writer.WriteString(str)
			}
		}
	} else {
		users, data, err := parser.ParseAll(in)
		if err != nil {
			log.Fatalln(err)
		}

		for i := 0; i < len(users); i++ {
			if len(data[users[i]]) == 0 {
				continue
			}

			checkIllegalString(&users[i])
			userFile, err := os.Create(fmt.Sprintf("%s.txt", users[i]))
			if err != nil {
				log.Fatalln(err)
			}
			defer userFile.Close()

			writer = bufio.NewWriter(userFile)
			for j := len(data[users[i]]) - 1; j >= 0; j-- {
				if data[users[i]][j].MediaType != "TEXT" {
					continue
				}
				str := fmt.Sprintf("%s (%s): %s\n", data[users[i]][j].From, data[users[i]][j].Created, data[users[i]][j].Content)
				if *writeToFile == false {
					fmt.Print(str)
				} else {
					writer.WriteString(str)
				}
			}
		}
	}

	if writer != nil && writer.Size() > 0 {
		writer.Flush()
	}
}
