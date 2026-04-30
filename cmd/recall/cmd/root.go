package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const configPath = "/.config/recall"

var serverURL string

var rootCmd = &cobra.Command{
	Use:   "recall",
	Short: "recall is a resume maker",
	Long: `A easy to use cli tool to get resumes and fast tips
		   for concepts that you already know but forgot, easy
		   access while programming`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, errSilent) {
			printError(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(
		&serverURL,
		"server",
		"s",
		"",
		"URL of the recall server (env: RECALL_SERVER for default)",
	)
	if err := viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server")); err != nil {
		panic(fmt.Sprintf("bind server flag: %v", err))
	}
}

func initConfig() {
	// Load .env from current directory (best effort)
	_ = godotenv.Load()

	// Read config file
	home, _ := os.UserHomeDir()
	viper.AddConfigPath(home + configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Read env file
	viper.SetEnvPrefix("RECALL")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try reading arquive
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Error reading config:", err)
		}
	}
}
