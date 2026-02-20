package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/api/searchconsole/v1"
)

func newInspectCmd() *cobra.Command {
	var site string
	var url string

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "URL Inspection",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}

			svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
			if err != nil {
				return err
			}

			req := &searchconsole.InspectUrlIndexRequest{
				InspectionUrl: url,
				SiteUrl:       site,
			}
			resp, err := svc.UrlInspection.Index.Inspect(req).Do()
			if err != nil {
				return err
			}

			var verdict, coverage, indexing, robots, lastCrawl string
			if resp != nil && resp.InspectionResult != nil && resp.InspectionResult.IndexStatusResult != nil {
				isr := resp.InspectionResult.IndexStatusResult
				verdict = isr.Verdict
				coverage = isr.CoverageState
				indexing = isr.IndexingState
				robots = isr.RobotsTxtState
				if isr.LastCrawlTime != "" {
					if t, err := time.Parse(time.RFC3339, isr.LastCrawlTime); err == nil {
						lastCrawl = t.UTC().Format(time.RFC3339)
					} else {
						lastCrawl = isr.LastCrawlTime
					}
				}
			}

			out := struct {
				Site        string `json:"site"`
				URL         string `json:"url"`
				Verdict     string `json:"verdict"`
				Coverage    string `json:"coverage_state"`
				Indexing    string `json:"indexing_state"`
				RobotsTxt   string `json:"robots_txt_state"`
				LastCrawlAt string `json:"last_crawl_time"`
			}{
				Site:        site,
				URL:         url,
				Verdict:     verdict,
				Coverage:    coverage,
				Indexing:    indexing,
				RobotsTxt:   robots,
				LastCrawlAt: lastCrawl,
			}

			headers := []string{"SITE", "URL", "VERDICT", "COVERAGE_STATE", "INDEXING_STATE", "ROBOTS_TXT_STATE", "LAST_CRAWL_TIME"}
			rows := [][]string{{site, url, verdict, coverage, indexing, robots, lastCrawl}}
			return app.Printer.Print(headers, rows, out)
		},
	}

	cmd.Flags().StringVar(&site, "site", "", "Site URL (required)")
	cmd.Flags().StringVar(&url, "url", "", "URL to inspect (required)")
	_ = cmd.MarkFlagRequired("site")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}
