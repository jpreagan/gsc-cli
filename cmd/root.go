package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jpreagan/gsc-cli/internal/auth"
	"github.com/jpreagan/gsc-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

var ErrVersionRequested = errors.New("version requested")

type appKeyT struct{}

type App struct {
	Account string
	NoInput bool

	Printer *cli.Printer
	Logger  *log.Logger
	Store   *auth.Store
}

func NewRootCmd() *cobra.Command {
	var (
		accountFlag string
		jsonFlag    bool
		plainFlag   bool
		noInputFlag bool
		verboseFlag bool
		colorFlag   bool
		versionFlag bool
	)

	rootCmd := &cobra.Command{
		Use:           "gsc",
		Short:         "Google Search Console CLI",
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if versionFlag {
				printVersion(cmd.OutOrStdout())
				root := cmd.Root()
				root.SilenceErrors = true
				root.SilenceUsage = true
				return ErrVersionRequested
			}
			if isHelpInvocation(cmd, args) {
				return nil
			}
			if jsonFlag && plainFlag {
				return errors.New("--json and --plain are mutually exclusive")
			}

			if !flagWasSet(cmd, "color") {
				colorFlag = autoDetectColor(cmd.OutOrStdout())
			}

			store, err := auth.NewStore()
			if err != nil {
				return err
			}

			var logOut io.Writer = io.Discard
			if verboseFlag {
				logOut = cmd.ErrOrStderr()
			}
			logger := log.New(logOut, "gsc: ", log.LstdFlags)

			p := cli.NewPrinter(cmd.OutOrStdout(), jsonFlag, plainFlag, colorFlag)

			app := &App{
				Account: accountFlag,
				NoInput: noInputFlag,
				Printer: p,
				Logger:  logger,
				Store:   store,
			}

			cmd.SetContext(context.WithValue(cmd.Context(), appKeyT{}, app))
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&accountFlag, "account", "default", "Select named account")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "JSON output")
	rootCmd.PersistentFlags().BoolVar(&plainFlag, "plain", false, "Plain tab-separated output (no headers)")
	rootCmd.PersistentFlags().BoolVar(&noInputFlag, "no-input", false, "Disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "Verbose logging to stderr")
	rootCmd.PersistentFlags().BoolVar(&colorFlag, "color", false, "Colored output")
	rootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "Print version")

	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newSitesCmd())
	rootCmd.AddCommand(newAnalyticsCmd())
	rootCmd.AddCommand(newInspectCmd())
	rootCmd.AddCommand(newSitemapsCmd())

	return rootCmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		if errors.Is(err, ErrVersionRequested) {
			return
		}
		os.Exit(1)
	}
}

func appFrom(cmd *cobra.Command) (*App, error) {
	v := cmd.Context().Value(appKeyT{})
	app, ok := v.(*App)
	if !ok || app == nil {
		return nil, errors.New("internal error: app context not initialized")
	}
	return app, nil
}

func printVersion(w io.Writer) {
	v := Version
	if Commit != "" {
		v = fmt.Sprintf("%s (%s)", v, Commit)
	}
	if Date != "" {
		v = fmt.Sprintf("%s %s", v, Date)
	}
	fmt.Fprintln(w, v)
}

func isHelpInvocation(cmd *cobra.Command, args []string) bool {
	if cmd == nil {
		return false
	}
	if cmd.Name() == "help" {
		return true
	}
	if cmd == cmd.Root() && len(args) == 0 {
		return true
	}
	if !cmd.Runnable() && len(args) == 0 {
		return true
	}
	for _, fs := range []*pflag.FlagSet{cmd.Flags(), cmd.InheritedFlags(), cmd.PersistentFlags()} {
		if fs == nil || fs.Lookup("help") == nil {
			continue
		}
		b, err := fs.GetBool("help")
		if err == nil && b {
			return true
		}
	}
	return false
}

func flagWasSet(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	root := cmd.Root()
	var rootPersistent *pflag.FlagSet
	if root != nil {
		rootPersistent = root.PersistentFlags()
	}
	for _, fs := range []*pflag.FlagSet{cmd.Flags(), cmd.InheritedFlags(), cmd.PersistentFlags(), rootPersistent} {
		if fs == nil || fs.Lookup(name) == nil {
			continue
		}
		if fs.Changed(name) {
			return true
		}
	}
	return false
}

func autoDetectColor(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
