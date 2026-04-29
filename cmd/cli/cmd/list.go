package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PedroEvaldt/recall/internal/client"
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
			return err
		} // TODO verificar se é isso mesmo o jeito de fazer

		docs, err := c.ListDocuments(cmd.Context(), query)
		if err != nil {
			return err
		}
		if len(docs) == 0 {
			return fmt.Errorf("no documents found for: %s", query)
		}
		if len(docs) > limit {
			docs = docs[:limit]
		}

		for i, doc := range docs {
			if _, err := fmt.Fprintf(os.Stdout, "%d.\nId: %v\nTitle: %s\nCreated at: %v\n", (i + 1), doc.ID, doc.Title, doc.CreatedAt); err != nil {
				return fmt.Errorf("copy content: %w", err)
			}
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
