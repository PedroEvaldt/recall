package cmd

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/PedroEvaldt/recall/internal/client"
	"github.com/PedroEvaldt/recall/internal/tui/doclist"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var limit int

var listCmd = &cobra.Command{
	Use:   "list query [-l limit]",
	Short: "List documents matching a query",
	Example: `	recall list go struct
	recall list go struct -l 5`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := viper.GetString("server")
		query := strings.Join(args, " ")

		c, err := client.New(serverURL, 30*time.Second)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		model := doclist.New(c, query, cmd.Context())
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
		10,
		"max results",
	)
}
