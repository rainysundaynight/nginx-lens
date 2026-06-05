package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell|install]",
		Short: "Генерация и установка shell completion",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if args[0] == "install" {
				fmt.Println(installCompletion())
				return nil
			}
			switch args[0] {
			case "bash":
				return NewRoot().GenBashCompletion(os.Stdout)
			case "zsh":
				return NewRoot().GenZshCompletion(os.Stdout)
			case "fish":
				return NewRoot().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return NewRoot().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return cmd.Help()
			}
		},
	}
	return cmd
}
