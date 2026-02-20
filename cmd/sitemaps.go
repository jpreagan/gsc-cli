package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newSitemapsCmd() *cobra.Command {
	var site string

	cmd := &cobra.Command{
		Use:   "sitemaps",
		Short: "Manage sitemaps",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSitemapsList(cmd, site)
		},
	}

	cmd.PersistentFlags().StringVar(&site, "site", "", "Site URL (required)")
	_ = cmd.MarkPersistentFlagRequired("site")

	cmd.AddCommand(newSitemapsGetCmd(&site))
	cmd.AddCommand(newSitemapsSubmitCmd(&site))
	cmd.AddCommand(newSitemapsDeleteCmd(&site))
	return cmd
}

func runSitemapsList(cmd *cobra.Command, site string) error {
	app, err := appFrom(cmd)
	if err != nil {
		return err
	}
	svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
	if err != nil {
		return err
	}
	resp, err := svc.Sitemaps.List(site).Do()
	if err != nil {
		return err
	}

	type sitemapOut struct {
		Path           string `json:"path"`
		Type           string `json:"type"`
		LastSubmitted  string `json:"last_submitted"`
		LastDownloaded string `json:"last_downloaded"`
		Warnings       int64  `json:"warnings"`
		Errors         int64  `json:"errors"`
		IsPending      bool   `json:"is_pending"`
	}
	out := struct {
		Site     string       `json:"site"`
		Sitemaps []sitemapOut `json:"sitemaps"`
	}{Site: site, Sitemaps: make([]sitemapOut, 0)}

	var rows [][]string
	if resp != nil {
		for _, sm := range resp.Sitemap {
			if sm == nil {
				continue
			}
			out.Sitemaps = append(out.Sitemaps, sitemapOut{
				Path:           sm.Path,
				Type:           sm.Type,
				LastSubmitted:  sm.LastSubmitted,
				LastDownloaded: sm.LastDownloaded,
				Warnings:       sm.Warnings,
				Errors:         sm.Errors,
				IsPending:      sm.IsPending,
			})
			rows = append(rows, []string{
				sm.Path,
				sm.Type,
				sm.LastSubmitted,
				sm.LastDownloaded,
				fmt.Sprintf("%d", sm.Warnings),
				fmt.Sprintf("%d", sm.Errors),
				strconv.FormatBool(sm.IsPending),
			})
		}
	}

	headers := []string{"PATH", "TYPE", "LAST_SUBMITTED", "LAST_DOWNLOADED", "WARNINGS", "ERRORS", "IS_PENDING"}
	return app.Printer.Print(headers, rows, out)
}

func newSitemapsGetCmd(site *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <feedpath>",
		Short: "Get a specific sitemap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedpath := args[0]
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
			if err != nil {
				return err
			}
			sm, err := svc.Sitemaps.Get(*site, feedpath).Do()
			if err != nil {
				return err
			}
			if sm == nil {
				return errors.New("empty response")
			}

			out := struct {
				Site           string `json:"site"`
				Path           string `json:"path"`
				Type           string `json:"type"`
				LastSubmitted  string `json:"last_submitted"`
				LastDownloaded string `json:"last_downloaded"`
				Warnings       int64  `json:"warnings"`
				Errors         int64  `json:"errors"`
				IsPending      bool   `json:"is_pending"`
			}{
				Site:           *site,
				Path:           sm.Path,
				Type:           sm.Type,
				LastSubmitted:  sm.LastSubmitted,
				LastDownloaded: sm.LastDownloaded,
				Warnings:       sm.Warnings,
				Errors:         sm.Errors,
				IsPending:      sm.IsPending,
			}

			headers := []string{"SITE", "PATH", "TYPE", "LAST_SUBMITTED", "LAST_DOWNLOADED", "WARNINGS", "ERRORS", "IS_PENDING"}
			rows := [][]string{{
				*site,
				sm.Path,
				sm.Type,
				sm.LastSubmitted,
				sm.LastDownloaded,
				fmt.Sprintf("%d", sm.Warnings),
				fmt.Sprintf("%d", sm.Errors),
				strconv.FormatBool(sm.IsPending),
			}}
			return app.Printer.Print(headers, rows, out)
		},
	}
	return cmd
}

func newSitemapsSubmitCmd(site *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit <feedpath>",
		Short: "Submit a sitemap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedpath := args[0]
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
			if err != nil {
				return err
			}
			if err := svc.Sitemaps.Submit(*site, feedpath).Do(); err != nil {
				return err
			}
			out := struct {
				Site   string `json:"site"`
				Feed   string `json:"feedpath"`
				Status string `json:"status"`
			}{Site: *site, Feed: feedpath, Status: "ok"}
			headers := []string{"SITE", "FEEDPATH", "STATUS"}
			rows := [][]string{{*site, feedpath, "ok"}}
			return app.Printer.Print(headers, rows, out)
		},
	}
	return cmd
}

func newSitemapsDeleteCmd(site *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <feedpath>",
		Short: "Delete a sitemap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedpath := args[0]
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
			if err != nil {
				return err
			}
			if err := svc.Sitemaps.Delete(*site, feedpath).Do(); err != nil {
				return err
			}
			out := struct {
				Site   string `json:"site"`
				Feed   string `json:"feedpath"`
				Status string `json:"status"`
			}{Site: *site, Feed: feedpath, Status: "ok"}
			headers := []string{"SITE", "FEEDPATH", "STATUS"}
			rows := [][]string{{*site, feedpath, "ok"}}
			return app.Printer.Print(headers, rows, out)
		},
	}
	return cmd
}
