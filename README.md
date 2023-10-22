# Snaparser

Snaparser is a program that parses your snapchat chat history into a human friendly way.
The parsed data can either be written to file or printed to stdout.

# Installation

```
> go install github.com/vanillaiice/snaparser/cmd/snaparser@latest
```

# Usage

To download your chat history data, follow the guide available on snapchat's [website](https://help.snapchat.com/hc/en-us/articles/7012305371156-How-do-I-download-my-data-from-Snapchat-). You can then do the following:

```
// Extract chats only with user 'johndoe' and write chats to file
> snaparser -u johndoe -w chat_history.json
// Extract all chats and pipe output to more
> snaparser chat_history.json | more
```

# Options

## ```-u``` (username)
Extract chats only from specified user.

## ```-w``` (write-to-file)
If chats should be written to file.

# Author

vanillaiice

# License

BSD-3-Clause
