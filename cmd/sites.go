package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func newSitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sites",
		Short: "Manage Search Console sites",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSitesList(cmd)
		},
	}

	cmd.AddCommand(newSitesGetCmd())
	cmd.AddCommand(newSitesAddCmd())
	cmd.AddCommand(newSitesDeleteCmd())
	return cmd
}

func runSitesList(cmd *cobra.Command) error {
	app, err := appFrom(cmd)
	if err != nil {
		return err
	}
	svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
	if err != nil {
		return err
	}

	resp, err := svc.Sites.List().Do()
	if err != nil {
		return err
	}

	type siteOut struct {
		SiteURL         string `json:"site_url"`
		PermissionLevel string `json:"permission_level"`
	}
	out := struct {
		Sites []siteOut `json:"sites"`
	}{Sites: make([]siteOut, 0)}

	var rows [][]string
	if resp != nil {
		for _, se := range resp.SiteEntry {
			if se == nil {
				continue
			}
			out.Sites = append(out.Sites, siteOut{
				SiteURL:         se.SiteUrl,
				PermissionLevel: se.PermissionLevel,
			})
			rows = append(rows, []string{se.SiteUrl, se.PermissionLevel})
		}
	}
	headers := []string{"SITE_URL", "PERMISSION_LEVEL"}
	return app.Printer.Print(headers, rows, out)
}

func newSitesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <siteUrl>",
		Short: "Get a site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			siteURL := args[0]
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
			if err != nil {
				return err
			}
			se, err := svc.Sites.Get(siteURL).Do()
			if err != nil {
				return err
			}
			if se == nil {
				return errors.New("empty response")
			}
			out := struct {
				SiteURL         string `json:"site_url"`
				PermissionLevel string `json:"permission_level"`
			}{
				SiteURL:         se.SiteUrl,
				PermissionLevel: se.PermissionLevel,
			}
			rows := [][]string{{se.SiteUrl, se.PermissionLevel}}
			headers := []string{"SITE_URL", "PERMISSION_LEVEL"}
			return app.Printer.Print(headers, rows, out)
		},
	}
	return cmd
}

func newSitesAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <siteUrl>",
		Short: "Add a site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			siteURL := args[0]
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
			if err != nil {
				return err
			}
			if err := svc.Sites.Add(siteURL).Do(); err != nil {
				return err
			}
			out := struct {
				SiteURL string `json:"site_url"`
				Status  string `json:"status"`
			}{
				SiteURL: siteURL,
				Status:  "ok",
			}
			rows := [][]string{{siteURL, "ok"}}
			headers := []string{"SITE_URL", "STATUS"}
			return app.Printer.Print(headers, rows, out)
		},
	}
	return cmd
}

func newSitesDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <siteUrl>",
		Short: "Delete a site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			siteURL := args[0]
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
			if err != nil {
				return err
			}
			if err := svc.Sites.Delete(siteURL).Do(); err != nil {
				return err
			}
			out := struct {
				SiteURL string `json:"site_url"`
				Status  string `json:"status"`
			}{
				SiteURL: siteURL,
				Status:  "ok",
			}
			rows := [][]string{{siteURL, "ok"}}
			headers := []string{"SITE_URL", "STATUS"}
			return app.Printer.Print(headers, rows, out)
		},
	}
	return cmd
}
