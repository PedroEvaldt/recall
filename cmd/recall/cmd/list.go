package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/PedroEvaldt/recall/internal/client"
	"github.com/PedroEvaldt/recall/internal/tui/doclist"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	limit   int
	listAll bool
)

var listCmd = &cobra.Command{
	Use:   "list query [-l limit]",
	Short: "List documents matching a query",
	Example: `	recall list go struct
	recall list go struct -l 5`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if listAll && len(args) > 0 {
			return fmt.Errorf("--all cannot be combined with a query")
		}
		if !listAll && len(args) == 0 {
			return fmt.Errorf("either provide a query or use --all")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			query    string
			limitStr string
		)
		serverURL := viper.GetString("server")
		if !listAll {
			query = strings.Join(args, " ")
		}
		c, err := client.New(serverURL, 30*time.Second)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		if limit != 0 {
			limitStr = strconv.Itoa(limit)
		}

		model := doclist.New(c, query, limitStr, cmd.Context())
		finalModel, err := tea.NewProgram(model).Run()
		if err != nil {
			return fmt.Errorf("tui: %w", err)
		}

		m, ok := finalModel.(doclist.Model)
		if !ok {
			return fmt.Errorf("unexpected tui model type: %T", finalModel)
		}
		if m.Aborted || m.Selected == nil {
			return nil
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().IntVarP(
		&limit,
		"limit",
		"l",
		0,
		"max results",
	)
	listCmd.Flags().BoolVarP(
		&listAll,
		"all",
		"a",
		false,
		"list all documents",
	)
}
