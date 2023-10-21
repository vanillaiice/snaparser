# Snaparser

Snaparser is a program that parses your snapchat chat history into a human friendly way.
The parsed data can either be written to file or printed to stdout.

# Installation

```
> go install github.com/vanillaiice/snaparser/cmd/snaparser@latest
```

# Usage

```
> snaparser -u johndoe -w chat_history.json
> snaparser -u chat_history.json | more
```

# Options

## ```-u```
Extract chats only from specified user.

## ```-w```
If chats should be written to file.

# Author

vanillaiice

# License

BSD-3-Clause
