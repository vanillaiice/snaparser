package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli-altsrc/v3"
	altsrctoml "github.com/urfave/cli-altsrc/v3/toml"
	"github.com/urfave/cli/v3"
)

const defaultConfigFile = "snaparser.toml"

// This schema validates the file; urfave's TOML sources apply values to flags.
// Validation prevents malformed files and misspelled keys silently using defaults.
type fileConfig struct {
	Data   string `toml:"data"`
	DB     string `toml:"db"`
	Server struct {
		Listen            string `toml:"listen"`
		ReadHeaderTimeout string `toml:"read_header_timeout"`
		ReadTimeout       string `toml:"read_timeout"`
		WriteTimeout      string `toml:"write_timeout"`
		IdleTimeout       string `toml:"idle_timeout"`
		ShutdownTimeout   string `toml:"shutdown_timeout"`
	} `toml:"server"`
}

func configSources(path *string) func(string) cli.ValueSourceChain {
	source := altsrc.NewStringPtrSourcer(path)

	return func(key string) cli.ValueSourceChain {
		return cli.NewValueSourceChain(altsrctoml.TOML(key, source))
	}
}

func validateConfig(path string, explicit bool) error {
	var config fileConfig

	metadata, err := toml.DecodeFile(path, &config)
	if err != nil {
		if !explicit && errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("load configuration %q: %w", path, err)
	}

	var unknown []string

	for _, key := range metadata.Undecoded() {
		unknown = append(unknown, key.String())
	}

	if len(unknown) != 0 {
		return fmt.Errorf("configuration %q has unknown keys: %s", path, strings.Join(unknown, ", "))
	}

	return nil
}

func databaseFlag(source cli.ValueSourceChain) *cli.StringFlag {
	return &cli.StringFlag{
		Name: "db", Value: "snaparser.db", Sources: source,
		Usage: "archive database path",
		Validator: func(path string) error {
			if strings.TrimSpace(path) == "" {
				return errors.New("database path must not be empty")
			}

			return nil
		},
	}
}

func timeoutFlag(name string, value time.Duration, source func(string) cli.ValueSourceChain) *cli.DurationFlag {
	return &cli.DurationFlag{
		Name: name, Value: value,
		Sources: source("server." + strings.ReplaceAll(name, "-", "_")),
		Usage:   "HTTP " + strings.ReplaceAll(name, "-", " ") + " (duration, e.g. 30s)",
		Validator: func(duration time.Duration) error {
			if duration < 0 {
				return fmt.Errorf("%s must be nonnegative", name)
			}

			if name == "shutdown-timeout" && duration == 0 {
				return errors.New("shutdown-timeout must be positive")
			}

			return nil
		},
	}
}
