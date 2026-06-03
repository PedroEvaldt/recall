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
		query, limitStr := buildListParams(args, listAll, limit)
		serverURL := viper.GetString("server")
		c, err := client.New(serverURL, 30*time.Second, viper.GetString("auth_token"))
		if err != nil {
			return fmt.Errorf("create client: %w", err)
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

func buildListParams(args []string, all bool, limit int) (query, limitStr string) {
	if !all {
		query = strings.Join(args, " ")
	}
	if limit != 0 {
		limitStr = strconv.Itoa(limit)
	}
	return query, limitStr
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
