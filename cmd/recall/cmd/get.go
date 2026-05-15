package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PedroEvaldt/recall/internal/client"
	"github.com/PedroEvaldt/recall/internal/tui/docglamour"
	"github.com/PedroEvaldt/recall/internal/tui/doclist"
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

		c, err := client.New(serverURL, 30*time.Second, viper.GetString("auth_token"))
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		hasDarkBg := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

		model := doclist.New(c, query, "", cmd.Context())
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

		body, err := c.GetContent(cmd.Context(), m.Selected.ID)
		if err != nil {
			if client.IsNotFound(err) {
				printMissingContent(os.Stderr, m.Selected.Title)
				return errSilent
			}
			return fmt.Errorf("get content: %w", err)
		}
		defer body.Close()

		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return fmt.Errorf("read content: %w", err)
		}

		if !isMarkdownDocument(*m.Selected) {
			if _, err := io.Copy(os.Stdout, bytes.NewReader(bodyBytes)); err != nil {
				return fmt.Errorf("write content: %w", err)
			}
			return nil
		}

		glamourModel, err := docglamour.NewExample(string(bodyBytes), hasDarkBg)
		if err != nil {
			return fmt.Errorf("create glamour model: %w", err)
		}

		if _, err := tea.NewProgram(glamourModel).Run(); err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
