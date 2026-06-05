package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newSyntaxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "syntax",
		Short: "Проверка синтаксиса через nginx -t (настройки: syntax, defaults)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			path, err := nginxConfigPath(cfg)
			if err != nil {
				return err
			}
			valid, errors, synErr := checkNginxSyntax(cfg, cfg.Syntax.SkipWarns)
			if synErr != nil {
				return synErr
			}
			st := newStyler(cfg)
			if valid {
				fmt.Println(st.ok(fmt.Sprintf("Синтаксис корректен: %s", path)))
				return nil
			}
			fmt.Println(st.fail("Ошибки синтаксиса:"))
			for _, e := range errors {
				fmt.Println(" ", st.red(e))
			}
			os.Exit(1)
			return nil
		},
	}
}
