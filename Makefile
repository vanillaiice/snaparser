# Determine the operating system
ifeq ($(OS), Windows_NT)
	OS_TARGET = Windows
else
	OS_TARGET := $(shell uname -s)
endif

# Compile depending on OS
os: $(OS_TARGET)

# Compile for windows
Windows:
	mkdir -p bin/windows
	GOOS=windows go build -ldflags="-s -w" -o bin/windows/snaparser.exe cmd/snaparser/main.go

# Compile for linux
Linux:
	mkdir -p bin/linux
	GOOS=linux go build -ldflags="-s -w" -o bin/linux/snaparser-linux cmd/snaparser/main.go

# Compile for darwin
Darwin:
	mkdir -p bin/darwin
	GOOS=darwin go build -ldflags="-s -w" -o bin/darwin/snaparser-darwin cmd/snaparser/main.go

# Compile for all OS
all: Windows Linux Darwin
