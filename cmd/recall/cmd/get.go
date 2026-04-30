package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/PedroEvaldt/recall/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var getCmd = &cobra.Command{
	Use:   "get [query]",
	Short: "Get the documents matching a query",
	Args:  cobra.MinimumNArgs(1),
	Example: `	recall get go struct
	recall get go struct --server "http://localhost:8080"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := viper.GetString("server")
		query := strings.Join(args, " ")

		c, err := client.New(serverURL, 30*time.Second)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		docs, err := c.ListDocuments(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("list documents: %w", err)
		}
		if len(docs) == 0 {
			printNoResults(os.Stderr, query)
			return errSilent
		}
		doc := docs[0] // TODO put interactive menu

		body, err := c.GetContent(cmd.Context(), doc.ID)
		if err != nil {
			if client.IsNotFound(err) {
				printMissingContent(os.Stderr, doc.Title)
				return errSilent
			}
			return fmt.Errorf("get content: %w", err)
		}
		defer body.Close()

		if _, err := io.Copy(os.Stdout, body); err != nil {
			return fmt.Errorf("copy content: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
