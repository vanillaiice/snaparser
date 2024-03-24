# Snaparser

Snaparser parses snapchat chat history json files to human friendly format.
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
$ snaparser [global options] command [command options] 

# Extract chats only with user 'johndoe' and write chats to file
$ snaparser -u johndoe -f chat_history.json -w

# Extract chats only with user 'janedoe', read chat history file from stdin,
# and print to Stdout
$ cat chat_history.json | snaparser -u janedoe

# Extract all chats and pipe output to more
$ snaparser -f chat_history.json | more
```

# Flags

```sh
NAME:
   snaparser - parse snapchat chat history to human friendly format

USAGE:
   snaparser [global options] command [command options] 

VERSION:
   v0.1.4

AUTHOR:
   vanillaiice <vanillaiice1@proton.me>

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --file FILE, -f FILE                                  read chats from FILE (use '-' for stdin)
   --directory DIRECTORY, -d DIRECTORY, --dir DIRECTORY  write parsed chats to DIRECTORY
   --user value, -u value                                only extract chat with user
   --write, -w                                           write parsed chats to disk (default: false)
   --create, -c                                          create directory if it does not exist (default: false)
   --color, -l                                           write colored output (default: false)
   --help, -h                                            show help
   --version, -v                                         print the version
```

# Author

vanillaiice

# License

BSD-3-Clause
