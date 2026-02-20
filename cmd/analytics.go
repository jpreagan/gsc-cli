package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/api/searchconsole/v1"
)

func newAnalyticsCmd() *cobra.Command {
	var site string
	var startDate string
	var endDate string
	var tz string
	var dimensions string
	var filters []string
	var rowLimit int64
	var startRow int64
	var typ string

	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Query Search Analytics",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFrom(cmd)
			if err != nil {
				return err
			}
			if rowLimit <= 0 {
				return errors.New("--row-limit must be > 0")
			}
			if startRow < 0 {
				return errors.New("--start-row must be >= 0")
			}
			loc, err := time.LoadLocation(tz)
			if err != nil {
				return fmt.Errorf("invalid --tz %q: %w", tz, err)
			}

			now := time.Now()
			if startDate == "" || endDate == "" {
				defStart, defEnd := defaultAnalyticsDateRange(now, loc)
				if startDate == "" {
					startDate = defStart
				}
				if endDate == "" {
					endDate = defEnd
				}
			}

			if _, err := parseDate(startDate); err != nil {
				return fmt.Errorf("invalid --start-date: %w", err)
			}
			if _, err := parseDate(endDate); err != nil {
				return fmt.Errorf("invalid --end-date: %w", err)
			}
			if startDate > endDate {
				return errors.New("--start-date must be <= --end-date")
			}

			dims := parseCSV(dimensions)
			group, err := parseFilters(filters)
			if err != nil {
				return err
			}
			if typ != "" {
				if err := validateSearchType(typ); err != nil {
					return err
				}
			}

			svc, err := newSearchConsoleService(cmd.Context(), app.Store, app.Account)
			if err != nil {
				return err
			}

			req := &searchconsole.SearchAnalyticsQueryRequest{
				StartDate:  startDate,
				EndDate:    endDate,
				Type:       typ,
				RowLimit:   rowLimit,
				StartRow:   startRow,
				Dimensions: dims,
			}
			if group != nil {
				req.DimensionFilterGroups = []*searchconsole.ApiDimensionFilterGroup{group}
			}

			resp, err := svc.Searchanalytics.Query(site, req).Do()
			if err != nil {
				return err
			}

			type rowOut struct {
				Keys        []string `json:"keys"`
				Clicks      float64  `json:"clicks"`
				Impressions float64  `json:"impressions"`
				Ctr         float64  `json:"ctr"`
				Position    float64  `json:"position"`
			}
			out := struct {
				Site      string   `json:"site"`
				StartDate string   `json:"start_date"`
				EndDate   string   `json:"end_date"`
				Type      string   `json:"type"`
				Rows      []rowOut `json:"rows"`
			}{
				Site:      site,
				StartDate: startDate,
				EndDate:   endDate,
				Type:      typ,
				Rows:      make([]rowOut, 0),
			}

			headers := make([]string, 0, len(dims)+4)
			for _, d := range dims {
				headers = append(headers, strings.ToUpper(d))
			}
			headers = append(headers, "CLICKS", "IMPRESSIONS", "CTR", "POSITION")

			var rows [][]string
			if resp != nil {
				for _, r := range resp.Rows {
					if r == nil {
						continue
					}
					out.Rows = append(out.Rows, rowOut{
						Keys:        r.Keys,
						Clicks:      r.Clicks,
						Impressions: r.Impressions,
						Ctr:         r.Ctr,
						Position:    r.Position,
					})
					row := make([]string, 0, len(dims)+4)
					for _, k := range r.Keys {
						row = append(row, k)
					}
					row = append(row,
						fmt.Sprintf("%g", r.Clicks),
						fmt.Sprintf("%g", r.Impressions),
						fmt.Sprintf("%g", r.Ctr),
						fmt.Sprintf("%g", r.Position),
					)
					rows = append(rows, row)
				}
			}

			return app.Printer.Print(headers, rows, out)
		},
	}

	cmd.Flags().StringVar(&site, "site", "", "Site URL (required)")
	_ = cmd.MarkFlagRequired("site")
	cmd.Flags().StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&tz, "tz", "America/Los_Angeles", "Timezone for default date range (IANA name)")
	cmd.Flags().StringVar(&dimensions, "dimensions", "", "Comma-separated dimensions")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter (repeatable): dimension=operator=value")
	cmd.Flags().Int64Var(&rowLimit, "row-limit", 1000, "Maximum rows")
	cmd.Flags().Int64Var(&startRow, "start-row", 0, "Start row offset")
	cmd.Flags().StringVar(&typ, "type", "", "Search type: web/image/video/news/discover/googleNews")
	return cmd
}

func defaultAnalyticsDateRange(now time.Time, loc *time.Location) (string, string) {
	const dataLagDays = 2
	const windowDays = 28

	n := now.In(loc)
	end := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -dataLagDays)
	start := end.AddDate(0, 0, -(windowDays - 1))
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func parseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func validateSearchType(t string) error {
	switch t {
	case "web", "image", "video", "news", "discover", "googleNews":
		return nil
	default:
		return fmt.Errorf("invalid --type %q", t)
	}
}

func parseFilters(raw []string) (*searchconsole.ApiDimensionFilterGroup, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	filters := make([]*searchconsole.ApiDimensionFilter, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid filter %q (expected dimension=operator=value)", item)
		}
		dim := strings.TrimSpace(parts[0])
		op := strings.TrimSpace(parts[1])
		val := strings.TrimSpace(parts[2])
		if dim == "" || op == "" || val == "" {
			return nil, fmt.Errorf("invalid filter %q (empty dimension/operator/value)", item)
		}
		if err := validateFilterOperator(op); err != nil {
			return nil, err
		}
		filters = append(filters, &searchconsole.ApiDimensionFilter{
			Dimension:  dim,
			Operator:   op,
			Expression: val,
		})
	}
	if len(filters) == 0 {
		return nil, nil
	}
	return &searchconsole.ApiDimensionFilterGroup{
		GroupType: "and",
		Filters:   filters,
	}, nil
}

func validateFilterOperator(op string) error {
	switch op {
	case "equals", "notEquals", "contains", "notContains", "includingRegex", "excludingRegex":
		return nil
	default:
		return fmt.Errorf("unsupported filter operator %q", op)
	}
}
