package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/PedroEvaldt/recall/internal/tui/docmultinput"
	"github.com/PedroEvaldt/recall/internal/tui/docresult"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "Inicialize init recall configurations",
	Args:    cobra.ExactArgs(0),
	Example: `	recall init   `,
	RunE: func(cmd *cobra.Command, args []string) error {
		model := docmultinput.InitialModel()
		finalModel, err := tea.NewProgram(model).Run()
		if err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		m, ok := finalModel.(docmultinput.MultInputModel)
		if !ok {
			return fmt.Errorf("invalid model type %T", m)
		}
		serverURL := m.Inputs[0].Value()
		authToken := m.Inputs[1].Value()

		if authToken == "" {
			bytes := make([]byte, 32)
			_, err := rand.Read(bytes)
			if err != nil {
				return fmt.Errorf("generate token: %w", err)
			}
			authToken = hex.EncodeToString(bytes)
		}
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home dir: %w", err)
		}
		configPath := filepath.Join(homeDir, ".config", "recall")
		err = os.MkdirAll(configPath, 0o750)
		if err != nil {
			return fmt.Errorf("create recall config dir: %w", err)
		}
		configFilePath := filepath.Join(configPath, "config.yaml")
		if _, statErr := os.Stat(configFilePath); statErr == nil {
			finalModel, err := tea.NewProgram(docresult.ResultModel{}).Run()
			if err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			m, ok := finalModel.(docresult.ResultModel)
			if !ok {
				return fmt.Errorf("invalid model type %T", m)
			}
			fmt.Printf("\n---\nYou chose %s!\n", m.Choice)

			if m.Choice == docresult.Choices[0] {
				return nil
			}
		}
		content := fmt.Sprintf("server: %s\nauth_token: %s", serverURL, authToken)
		err = os.WriteFile(configFilePath, []byte(content), 0o600)
		if err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}

		// Imprimir mensagem
		sucessMsg := fmt.Sprintf("Config saved to %s\n\n%s\n\nAdd this token to your server's AUTH_TOKEN enviroment variable if your want to change it", configFilePath, content)
		fmt.Printf(sucessMsg)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
