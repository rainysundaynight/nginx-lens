package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		force bool
		user  bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Создать конфигурационный файл по умолчанию",
		RunE: func(cmd *cobra.Command, args []string) error {
			var target, dir string
			if user {
				dir = filepath.Dir(config.UserConfigPath())
				target = config.UserConfigPath()
			} else {
				dir = config.SystemConfigDir
				target = config.SystemConfigPath
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				if !user {
					return fmt.Errorf("не удалось создать %s: %w (попробуйте: sudo nginx-lens init)", dir, err)
				}
				return err
			}
			if _, err := os.Stat(target); err == nil && !force {
				return fmt.Errorf("файл %s уже существует, используйте --force", target)
			}
			if err := os.WriteFile(target, []byte(config.ConfigTemplate), 0644); err != nil {
				if !user {
					return fmt.Errorf("не удалось записать %s: %w (попробуйте: sudo nginx-lens init)", target, err)
				}
				return err
			}
			fmt.Printf("Конфиг создан: %s\n", target)
			if !user {
				fmt.Println("Подсказка: export NGINX_LENS_CONFIG=/opt/nginx-lens/config.yaml")
			}
			if msg := installCompletion(); msg != "" {
				fmt.Println(msg)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Перезаписать существующий конфиг")
	cmd.Flags().BoolVar(&user, "user", false, "Пользовательский конфиг ~/.nginx-lens/config.yaml")
	return cmd
}
