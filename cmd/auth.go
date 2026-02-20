package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jpreagan/gsc-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and account management",
	}
	authCmd.AddCommand(newAuthCredentialsCmd())
	authCmd.AddCommand(newAuthAddCmd())
	authCmd.AddCommand(newAuthListCmd())
	return authCmd
}

func newAuthCredentialsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials <path|->",
		Short: "Store OAuth client credentials (downloaded client ID JSON)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}

			var b []byte
			if args[0] == "-" {
				b, err = io.ReadAll(cmd.InOrStdin())
			} else {
				b, err = os.ReadFile(args[0])
			}
			if err != nil {
				return err
			}

			clientID, clientSecret, err := auth.ExtractClientIDSecretFromCredentialsJSON(b)
			if err != nil {
				return err
			}
			if err := app.Store.WriteClientCredentials(app.Account, clientID, clientSecret); err != nil {
				return err
			}

			if app.Printer.JSONEnabled() {
				out := struct {
					Account string `json:"account"`
					Status  string `json:"status"`
				}{
					Account: app.Account,
					Status:  "ok",
				}
				return app.Printer.PrintJSON(out)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Wrote client credentials for account %q.\n", app.Account)
			return err
		},
	}
	return cmd
}

func newAuthAddCmd() *cobra.Command {
	var readonly bool
	var manual bool
	var authURL string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Run OAuth installed-app flow and save tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}

			clientJSON, err := app.Store.ReadClientJSON(app.Account)
			if err != nil {
				return err
			}

			cfg, err := auth.OAuthConfigFromClientJSON(clientJSON, "", readonly)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			var tok *oauth2.Token
			switch {
			case manual:
				req, err := auth.NewAuthCodeRequest(cfg)
				if err != nil {
					return err
				}

				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Open this URL in a browser to continue:\n%s\n", req.AuthURL); err != nil {
					return err
				}

				if authURL == "" {
					if app.NoInput {
						return errors.New("--auth-url is required with --manual --no-input")
					}
					in := cmd.InOrStdin()
					r := bufio.NewReader(in)
					authURL, err = promptLine(cmd.ErrOrStderr(), r, "Paste redirect URL: ")
					if err != nil {
						return err
					}
				}

				tok, err = auth.ExchangeAuthCodeFromRedirectURL(ctx, cfg, req.RedirectURL, req.State, authURL)
				if err != nil {
					return err
				}

			default:
				tok, err = auth.RunInstalledAppFlow(ctx, cfg, app.NoInput, true, func(format string, args ...any) {
					if _, werr := fmt.Fprintf(cmd.ErrOrStderr(), format, args...); werr != nil {
						app.Logger.Printf("stderr write error: %v", werr)
					}
				})
				if err != nil {
					return err
				}
			}

			if tok != nil && tok.RefreshToken == "" {
				if oldTok, rerr := app.Store.ReadToken(app.Account); rerr == nil {
					carryRefreshToken(tok, oldTok)
				} else if !errors.Is(rerr, os.ErrNotExist) {
					app.Logger.Printf("warning: failed to read existing token for account %q: %v", app.Account, rerr)
				}
			}

			if err := app.Store.WriteToken(app.Account, tok); err != nil {
				return err
			}

			if app.Printer.JSONEnabled() {
				exp := ""
				if !tok.Expiry.IsZero() {
					exp = tok.Expiry.UTC().Format(time.RFC3339)
				}
				out := struct {
					Account     string `json:"account"`
					TokenExpiry string `json:"token_expiry"`
					Status      string `json:"status"`
				}{
					Account:     app.Account,
					TokenExpiry: exp,
					Status:      "ok",
				}
				return app.Printer.PrintJSON(out)
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Saved token for account %q.\n", app.Account); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&readonly, "readonly", false, "Request read-only scope")
	cmd.Flags().BoolVar(&manual, "manual", false, "Manual flow: print auth URL and exchange code from pasted redirect URL (useful for headless/SSH)")
	cmd.Flags().StringVar(&authURL, "auth-url", "", "Full redirect URL from browser address bar (used with --manual)")
	return cmd
}

func newAuthListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			infos, err := app.Store.ListAccounts()
			if err != nil {
				return err
			}

			type accountOut struct {
				Name        string `json:"name"`
				HasClient   bool   `json:"has_client"`
				HasToken    bool   `json:"has_token"`
				TokenExpiry string `json:"token_expiry"`
			}
			out := struct {
				Accounts []accountOut `json:"accounts"`
			}{Accounts: make([]accountOut, 0, len(infos))}

			rows := make([][]string, 0, len(infos))
			for _, info := range infos {
				exp := ""
				if !info.TokenExpiry.IsZero() {
					exp = info.TokenExpiry.UTC().Format(time.RFC3339)
				}
				out.Accounts = append(out.Accounts, accountOut{
					Name:        info.Name,
					HasClient:   info.HasClient,
					HasToken:    info.HasToken,
					TokenExpiry: exp,
				})
				rows = append(rows, []string{info.Name, strconv.FormatBool(info.HasClient), strconv.FormatBool(info.HasToken), exp})
			}

			headers := []string{"NAME", "HAS_CLIENT", "HAS_TOKEN", "TOKEN_EXPIRY"}
			return app.Printer.Print(headers, rows, out)
		},
	}
	return cmd
}

func promptLine(w io.Writer, r *bufio.Reader, prompt string) (string, error) {
	if _, err := fmt.Fprint(w, prompt); err != nil {
		return "", err
	}
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", errors.New("empty input")
	}
	return line, nil
}

func carryRefreshToken(newTok, oldTok *oauth2.Token) {
	if newTok == nil || oldTok == nil {
		return
	}
	if newTok.RefreshToken != "" {
		return
	}
	if oldTok.RefreshToken == "" {
		return
	}
	newTok.RefreshToken = oldTok.RefreshToken
}
