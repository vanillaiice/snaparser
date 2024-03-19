# Snaparser

Snaparser parses snapchat chat history json files to human friendly format.
The output can be printed to stdout or written to file.
If no path is provided, stdin is processed.
If stdin is empty, the program stops execution.
By default, the parsed chat history is printed to stdout.

# Installation

```
$ go install github.com/vanillaiice/snaparser/cmd/snaparser@latest
```

# Usage

To download your chat history data, follow the guide available on snapchat's 
[website](https://help.snapchat.com/hc/en-us/articles/7012305371156-How-do-I-download-my-data-from-Snapchat-). 
You can then do the following:

```sh
# Generic Usage
$ snaparser parse [command options] [arguments...]

# Extract chats only with user 'johndoe' and write chats to file
$ snaparser parse -u johndoe -f chat_history.json -w

# Extract chats only with user 'janedoe', read chat history file from stdin,
# and print to Stdout
$ cat chat_history.json | snaparser -u janedoe

# Extract all chats and pipe output to more
$ snaparser parse -f chat_history.json | more
```

# Flags

```sh
NAME:
   snaparser parse - parse chats

USAGE:
   snaparser parse [command options] [arguments...]

OPTIONS:
   --file FILE, -f FILE                                  read chats from FILE
   --user value, -u value                                only extract chat with user
   --write, -w                                           write parsed chats to disk (default: false)
   --directory DIRECTORY, -d DIRECTORY, --dir DIRECTORY  write parsed chats to DIRECTORY
   --create, -c                                          create directory if it does not exist (default: false)
   --help, -h                                            show help
```

# Author

vanillaiice

# License

BSD-3-Clause
