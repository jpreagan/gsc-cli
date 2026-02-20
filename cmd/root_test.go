package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmd_ColorAutoDetect_DefaultsToNoColorForNonFileWriter(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := NewRootCmd()
	var out bytes.Buffer
	var errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)

	appCh := make(chan *App, 1)
	root.AddCommand(&cobra.Command{
		Use: "cap",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			appCh <- app
			return nil
		},
	})
	root.SetArgs([]string{"cap"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, errOut.String())
	}

	app := <-appCh
	if app.Printer.ColorEnabled() {
		t.Fatalf("ColorEnabled=%v want false for non-file output writer", app.Printer.ColorEnabled())
	}
}

func TestRootCmd_ColorFlagOverridesAutoDetect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := NewRootCmd()
	var out bytes.Buffer
	var errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)

	appCh := make(chan *App, 1)
	root.AddCommand(&cobra.Command{
		Use: "cap",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			appCh <- app
			return nil
		},
	})
	root.SetArgs([]string{"--color", "cap"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, errOut.String())
	}

	app := <-appCh
	if !app.Printer.ColorEnabled() {
		t.Fatalf("ColorEnabled=%v want true when --color is explicitly set", app.Printer.ColorEnabled())
	}
}

func TestRootCmd_HelpDoesNotCreateStoreDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := NewRootCmd()
	var out bytes.Buffer
	var errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, errOut.String())
	}

	if _, err := os.Stat(filepath.Join(home, ".gsc")); err == nil {
		t.Fatalf("expected no ~/.gsc directory to be created for --help")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat ~/.gsc: %v", err)
	}
}

func TestRootCmd_VersionFlagPrintsAndReturnsSentinel(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	defer func() {
		Version, Commit, Date = oldVersion, oldCommit, oldDate
	}()
	Version, Commit, Date = "1.2.3", "abc123", "2000-01-02"

	root := NewRootCmd()
	var out bytes.Buffer
	var errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"--version"})

	err := root.Execute()
	if !errors.Is(err, ErrVersionRequested) {
		t.Fatalf("Execute() err=%v want ErrVersionRequested", err)
	}
	if got := strings.TrimSpace(out.String()); got != "1.2.3 (abc123) 2000-01-02" {
		t.Fatalf("stdout=%q want %q", got, "1.2.3 (abc123) 2000-01-02")
	}
	if got := strings.TrimSpace(errOut.String()); got != "" {
		t.Fatalf("stderr=%q want empty", got)
	}
}
