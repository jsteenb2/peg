package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func completionCmd(cliName string) *cobra.Command {
	return &cobra.Command{
		Use:                   "completion [bash|fish|oh-my-zsh|zsh]",
		Short:                 "Generates shell completions",
		Args:                  cobra.ExactValidArgs(1),
		ValidArgs:             []string{"bash", "fish", "oh-my-zsh", "zsh", "powershell"},
		DisableFlagsInUseLine: true,
		Long: `
	Outputs shell completion for the given shell (bash, fish, oh-my-zsh, or zsh)
	OS X:
		$ source $(brew --prefix)/etc/bash_completion	# for bash users
		$ source <(cbt completion bash)			# for bash users
		$ source <(cbt completion oh-my-zsh)		# for oh-my-zsh users
		$ source <(cbt completion zsh)			# for zsh users
	Ubuntu:
		$ source /etc/bash-completion	   	# for bash users
		$ source <(cbt completion bash) 	# for bash users
		$ source <(cbt completion oh-my-zsh) 	# for oh-my-zsh users
		$ source <(cbt completion zsh)  	# for zsh users
	Additionally, you may want to add this to your .bashrc/.zshrc
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd, writer := cmd.Root(), cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(writer)
			case "fish":
				return rootCmd.GenFishCompletion(writer, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletion(writer)
			case "oh-my-zsh":
				if err := rootCmd.GenZshCompletion(writer); err != nil {
					return err
				}
				compdef := fmt.Sprintf("\n compdef _%[1]s %[1]s\n", cliName)
				_, err := io.WriteString(writer, compdef)
				return err
			case "zsh":
				return rootCmd.GenZshCompletion(writer)
			}
			return nil
		},
	}
}
