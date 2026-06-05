package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	var full bool
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Показать активный конфиг nginx-lens",
		RunE: func(cmd *cobra.Command, args []string) error {
			loader := config.Get()
			if loader.ConfigPath == "" {
				fmt.Println("Конфиг не найден, используются значения по умолчанию")
			} else {
				fmt.Printf("Config path: %s\n", loader.ConfigPath)
			}
			if full {
				data, err := yaml.Marshal(loader.Config)
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&full, "full", "f", false, "Полный YAML конфиг")
	cmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Проверка схемы config.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			loader := config.Get()
			if err := config.RequireConfigFile(loader); err != nil {
				return err
			}
			errs := config.ValidateSchema(&loader.Config)
			if len(errs) == 0 {
				fmt.Println("✓ Конфигурация корректна")
				return nil
			}
			fmt.Println("✗ Ошибки конфигурации:")
			for _, e := range errs {
				fmt.Printf("  - %s\n", e)
			}
			return fmt.Errorf("найдено %d ошибок", len(errs))
		},
	})
	return cmd
}
