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

```
// Generic Usage
$ snaparser [flags] [path ...]

// Extract chats only with user 'johndoe' and write chats to file
$ snaparser -u johndoe -w chat_history.json

// Extract chats only with user 'janedoe' and read chat history file from stdin
$ cat chat_history.json | snaparser -u janedoe

// Extract all chats and pipe output to more
$ snaparser chat_history.json | more
```

# Flags

## ```-u``` (username)
Only extract chats with this user.

## ```-w``` (write-to-file)
write chats to file.

## ```-d``` (directory)
Write to this directory.

## ```-f``` (force-directory-creation)
Create directory if it does not exist.

# Author

vanillaiice

# License

BSD-3-Clause
