package tmc

import "github.com/spf13/cobra"

func newCompetitorsCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "competitors",
		Aliases: []string{"competitor"},
		Short:   "Work with competitor intelligence resources",
	}
	cmd.AddCommand(newPagedListCommand(opts, listCommandSpec{
		Use:   "list",
		Short: "List competitors",
		Path:  "/competitors",
		Filters: []queryFlagSpec{
			{Flag: "search", Param: "search", Description: "search keyword"},
			{Flag: "importance", Param: "importance", Description: "importance filter"},
			{Flag: "market", Param: "market", Description: "market filter"},
			{Flag: "status", Param: "status", Description: "status filter"},
			{Flag: "include-archived", Param: "include_archived", Description: "include archived competitors"},
		},
	}))

	activities := &cobra.Command{
		Use:   "activities",
		Short: "Work with competitor activities",
	}
	activities.AddCommand(newPagedListCommand(opts, listCommandSpec{
		Use:   "list",
		Short: "List competitor activities",
		Path:  "/competitors/activities",
		Filters: []queryFlagSpec{
			{Flag: "competitor-id", Param: "competitor_id", Description: "competitor ID filter"},
			{Flag: "market", Param: "market", Description: "market filter"},
			{Flag: "nice-class", Param: "nice_class", Description: "Nice class filter"},
			{Flag: "activity-type", Param: "activity_type", Description: "activity type filter"},
			{Flag: "importance", Param: "importance", Description: "importance filter"},
			{Flag: "include-dismissed", Param: "include_dismissed", Description: "include dismissed activities"},
			{Flag: "dismissed", Param: "dismissed", Description: "dismissed filter: include or only"},
		},
	}))
	cmd.AddCommand(activities)

	reports := &cobra.Command{
		Use:   "reports",
		Short: "Work with competitor reports",
	}
	reports.AddCommand(newPagedListCommand(opts, listCommandSpec{
		Use:   "list",
		Short: "List competitor reports",
		Path:  "/competitors/reports",
	}))
	cmd.AddCommand(reports)
	return cmd
}
