package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli/v3"
	"github.com/vanillaiice/snaparser/db"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	return path
}

func captureServe(t *testing.T, args ...string) map[string]any {
	t.Helper()

	app := newApp()

	var values map[string]any

	for _, command := range app.Commands {
		if command.Name != "serve" {
			continue
		}

		command.Action = func(_ context.Context, c *cli.Command) error {
			values = map[string]any{
				"db": c.String("db"), "data": c.String("data"), "listen": c.String("listen"),
				"read-header-timeout": c.Duration("read-header-timeout"), "read-timeout": c.Duration("read-timeout"),
				"write-timeout": c.Duration("write-timeout"), "idle-timeout": c.Duration("idle-timeout"),
				"shutdown-timeout": c.Duration("shutdown-timeout"),
			}

			return nil
		}
	}

	if err := app.Run(context.Background(), append([]string{"snaparser"}, args...)); err != nil {
		t.Fatal(err)
	}

	if values == nil {
		t.Fatal("serve action did not run")
	}

	return values
}

func TestConfigurationDefaultsAndExample(t *testing.T) {
	example, err := os.ReadFile("../../snaparser.example.toml")
	if err != nil {
		t.Fatal(err)
	}

	t.Chdir(t.TempDir())

	want := map[string]any{
		"db": "snaparser.db", "data": "", "listen": "127.0.0.1:8080",
		"read-header-timeout": 5 * time.Second, "read-timeout": 15 * time.Second,
		"write-timeout": 30 * time.Second, "idle-timeout": 60 * time.Second, "shutdown-timeout": 5 * time.Second,
	}

	if got := captureServe(t, "serve"); !reflect.DeepEqual(got, want) {
		t.Fatalf("built-in defaults: %+v, want %+v", got, want)
	}

	if err := os.WriteFile(defaultConfigFile, example, 0600); err != nil {
		t.Fatal(err)
	}

	if got := captureServe(t, "serve"); !reflect.DeepEqual(got, want) {
		t.Fatalf("example/default file disagrees with built-in defaults: %+v", got)
	}
}

func TestTOMLValuesAndFlagPrecedence(t *testing.T) {
	config := writeConfig(t, `db = "configured.db"
data = "export"
[server]
listen = "localhost:9000"
read_header_timeout = "2s"
read_timeout = "3s"
write_timeout = "0s"
idle_timeout = "2m"
shutdown_timeout = "9s"
`)

	for _, args := range [][]string{{"--config", config, "serve"}, {"serve", "--config", config}} {
		got := captureServe(t, args...)

		want := map[string]any{
			"db": "configured.db", "data": "export", "listen": "localhost:9000",
			"read-header-timeout": 2 * time.Second, "read-timeout": 3 * time.Second,
			"write-timeout": time.Duration(0), "idle-timeout": 2 * time.Minute, "shutdown-timeout": 9 * time.Second,
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("TOML sources with args %v: %+v, want %+v", args, got, want)
		}
	}

	got := captureServe(t, "serve", "--config", config, "--db", "override.db", "--data=", "--listen", "127.0.0.1:1234", "--write-timeout", "45s")

	if got["db"] != "override.db" || got["data"] != "" || got["listen"] != "127.0.0.1:1234" || got["write-timeout"] != 45*time.Second || got["idle-timeout"] != 2*time.Minute {
		t.Fatalf("CLI flags must override TOML and preserve other file settings: %+v", got)
	}
}

func TestImportUsesTOMLConfiguration(t *testing.T) {
	root := exportRoot(t)

	exportFile(t, root, "account.json", `{"Basic Information":{"Username":"configured-owner"}}`)

	path := filepath.Join(t.TempDir(), "configured.db")

	config := writeConfig(t, "data = "+strconv.Quote(root)+"\ndb = "+strconv.Quote(path)+"\n")

	app := newApp()

	app.Writer = &bytes.Buffer{}

	if err := app.Run(context.Background(), []string{"snaparser", "import", "--config", config}); err != nil {
		t.Fatal(err)
	}

	store, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	account, err := store.GetAccount(context.Background())
	if err != nil || account.BasicInformation.Username != "configured-owner" {
		t.Fatalf("configured archive was not imported: %+v, %v", account, err)
	}
}

func TestInvalidConfigurationFailsBeforeCreatingArchive(t *testing.T) {
	for _, content := range []string{
		`db = [`,
		`db = 123`,
		`database = "typo.db"`,
		`db = ""`,
		"[server]\nlisten = false",
		"[server]\nlisten = \"\"",
		"[server]\nidle_timout = \"1s\"",
		"[server]\nread_timeout = \"-1s\"",
		"[server]\nwrite_timeout = \"forever\"",
		"[server]\nshutdown_timeout = \"0s\"",
	} {
		t.Run(content, func(t *testing.T) {
			t.Chdir(t.TempDir())

			config := writeConfig(t, content)

			app := newApp()

			app.Writer, app.ErrWriter = &bytes.Buffer{}, &bytes.Buffer{}

			if err := app.Run(context.Background(), []string{"snaparser", "serve", "--config", config}); err == nil {
				t.Fatal("invalid configuration accepted")
			}

			if _, err := os.Stat("snaparser.db"); !os.IsNotExist(err) {
				t.Fatal("invalid configuration created a database")
			}
		})
	}

	for _, path := range []string{filepath.Join(t.TempDir(), "missing.toml"), t.TempDir()} {
		app := newApp()

		if err := app.Run(context.Background(), []string{"snaparser", "serve", "--config", path}); err == nil || !strings.Contains(err.Error(), "load configuration") {
			t.Fatalf("explicit missing/unreadable config should fail: %v", err)
		}
	}
}

func TestImportRequiresDataAndVersionBypassesConfig(t *testing.T) {
	t.Chdir(t.TempDir())

	config := writeConfig(t, `data = ""`)

	if err := newApp().Run(context.Background(), []string{"snaparser", "import", "--config", config}); err == nil || !strings.Contains(err.Error(), "import requires") {
		t.Fatalf("empty import source should fail: %v", err)
	}

	app := newApp()

	var output bytes.Buffer

	app.Writer = &output

	if err := app.Run(context.Background(), []string{"snaparser", "--config", "missing.toml", "--version"}); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "1.0.0") {
		t.Fatalf("wrong release version: %s", output.String())
	}
}

func TestServeCanDisableConfiguredImport(t *testing.T) {
	root := exportRoot(t)

	exportFile(t, root, "account.json", `{"Basic Information":{"Username":"original"}}`)

	path := filepath.Join(t.TempDir(), "archive.db")

	if _, err := importArchive(context.Background(), root, path); err != nil {
		t.Fatal(err)
	}

	exportFile(t, root, "account.json", `{"Basic Information":{"Username":"replacement"}}`)

	config := writeConfig(t, "data = "+strconv.Quote(root)+"\ndb = "+strconv.Quote(path)+"\n[server]\nlisten = \"invalid-address\"\n")

	for _, tc := range []struct {
		flags    []string
		username string
	}{
		{[]string{"--data="}, "original"},
		{nil, "replacement"},
	} {
		app := newApp()

		app.Writer = &bytes.Buffer{}

		args := append([]string{"snaparser", "serve", "--config", config}, tc.flags...)

		// Invalid addressing stops at server startup without binding a port.
		if err := app.Run(context.Background(), args); err == nil || !strings.Contains(err.Error(), "missing port") {
			t.Fatalf("expected configured server startup: %v", err)
		}

		store, err := db.Open(path)
		if err != nil {
			t.Fatal(err)
		}

		account, err := store.GetAccount(context.Background())

		store.Close()

		if err != nil || account.BasicInformation.Username != tc.username {
			t.Fatalf("serve flags %v imported unexpectedly: %+v, %v", tc.flags, account, err)
		}
	}
}
