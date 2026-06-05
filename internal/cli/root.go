package cli

import (
	"fmt"
	"os"

	"github.com/rainysundaynight/nginx-lens/internal/version"
	"github.com/spf13/cobra"
)

// ---------- Корневая CLI-команда ----------
// Все параметры берутся из config.yaml.

// Execute запускает CLI.
func Execute() error {
	return NewRoot().Execute()
}

// NewRoot создаёт корневую cobra-команду.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "nginx-lens",
		Short: "CLI для анализа, визуализации и диагностики конфигураций Nginx",
		Long:  "nginx-lens — все настройки в config.yaml. Выполните nginx-lens init для первичной настройки.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.AddCommand(
		newHealthCmd(),
		newResolveCmd(),
		newAnalyzeCmd(),
		newValidateCmd(),
		newSyntaxCmd(),
		newRouteCmd(),
		newExplainCmd(),
		newBlastRadiusCmd(),
		newCertsCmd(),
		newScoreCmd(),
		newIngressAuditCmd(),
		newTreeCmd(),
		newGraphCmd(),
		newIncludeTreeCmd(),
		newDiffCmd(),
		newLogsCmd(),
		newMetricsCmd(),
		newUpstreamsCmd(),
		newInitCmd(),
		newConfigCmd(),
		newVersionCmd(),
		newCompletionCmd(),
	)
	root.CompletionOptions.DisableDefaultCmd = false
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Version)
		},
	}
}

func exitCode(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
		os.Exit(1)
	}
}
