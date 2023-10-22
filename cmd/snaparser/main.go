package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	parser "github.com/vanillaiice/snaparser/parser"
)

const TEXT = "TEXT"

func checkIllegalString(str *string) {
	if strings.Contains(*str, "/") == true {
		*str = strings.ReplaceAll(*str, "/", "-")
	}
}

func main() {
	user := flag.String("u", "", "extract chats only with specified user")
	writeToFile := flag.Bool("w", false, "if chats should be written to file")
	dir := flag.String("d", "", "write files to specified directory")
	forceDir := flag.Bool("f", false, "if directory does not exist, create it")
	flag.Parse()

	if flag.Arg(0) == "" {
		fmt.Println("Usage: sc-data-parser [option] [argument] chat_history.json")
		os.Exit(1)
	}

	file, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	if *dir != "" {
		if _, err := os.Stat(*dir); errors.Is(err, os.ErrNotExist) {
			if *forceDir == true {
				if err = os.Mkdir(*dir, os.ModePerm); err == nil {
					err = os.Chdir(*dir)
					if err != nil {
						panic(err)
					}
				}
			} else {
				panic(err)
			}
		} else {
			err = os.Chdir(*dir)
			if err != nil {
				panic(err)
			}
		}
	}

	var writer *bufio.Writer
	if *user != "" {
		data, err := parser.ParseUser(file, *user)
		if err != nil {
			panic(err)
		}
		if len(data) == 0 {
			os.Exit(0)
		}
		checkIllegalString(user)
		userFile, err := os.Create(fmt.Sprintf("%s.txt", *user))
		if err != nil {
			panic(err)
		}
		defer userFile.Close()
		writer = bufio.NewWriter(userFile)
		for i := len(data) - 1; i >= 0; i-- {
			if data[i].MediaType != TEXT {
				continue
			}
			str := fmt.Sprintf("%s (%s): %s\n", data[i].From, data[i].Created, data[i].Content)
			if *writeToFile == false {
				fmt.Print(str)
			} else {
				writer.WriteString(str)
			}
		}
		writer.Flush()
	} else {
		users, data, err := parser.ParseAll(file)
		if err != nil {
			panic(err)
		}

		for i := 0; i < len(users); i++ {
			if len(data[users[i]]) == 0 {
				continue
			}
			checkIllegalString(&users[i])
			userFile, err := os.Create(fmt.Sprintf("%s.txt", users[i]))
			if err != nil {
				panic(err)
			}
			defer userFile.Close()
			writer = bufio.NewWriter(userFile)
			for j := len(data[users[i]]) - 1; j >= 0; j-- {
				if data[users[i]][j].MediaType != TEXT {
					continue
				}
				str := fmt.Sprintf("%s (%s): %s\n", data[users[i]][j].From, data[users[i]][j].Created, data[users[i]][j].Content)
				if *writeToFile == false {
					fmt.Print(str)
				} else {
					writer.WriteString(str)
				}
			}
			writer.Flush()
		}
	}
}
