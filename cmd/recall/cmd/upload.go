package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PedroEvaldt/recall/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	uploadTitle string
	uploadTags  []string
)

var uploadCmd = &cobra.Command{
	Use:   "upload file [-t title] [--tags tags]",
	Short: "upload a document to the recall server",
	Example: `	recall upload notes.md --title "Go struct"	
	recall upload paper.pdf -t "Algebra linear" --tags math,linalg`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := viper.GetString("server")
		path := args[0]
		title := uploadTitle
		tags := uploadTags

		if title == "" {
			title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}

		file, err := os.Open(path) // #nosec G304 -- path provided by CLI user, runs with user's own privileges
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		defer file.Close()

		c, err := client.New(serverURL, 30*time.Second, viper.GetString("auth_token"))
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}
		doc, err := c.PostDocument(cmd.Context(), title, filepath.Base(path), file, tags)
		if err != nil {
			return fmt.Errorf("post document: %w", err)
		}

		fmt.Printf("Uploaded document\nName: %s\nId: %v \n", doc.Title, doc.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringVarP(
		&uploadTitle,
		"title",
		"t",
		"",
		"title of the document (defaults to filename without extension)",
	)
	uploadCmd.Flags().StringSliceVar(
		&uploadTags,
		"tags",
		nil,
		"comma-separated list of tags",
	)
}
