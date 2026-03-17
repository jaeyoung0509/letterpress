package letterpress

import (
	"github.com/jaeyoung0509/letterpress/internal/cli"
	"github.com/jaeyoung0509/letterpress/internal/tui"
	"github.com/spf13/cobra"
)

// Dependencies keeps root command wiring explicit so later issues can extend it
// without hiding shared entrypoints behind package globals.
type Dependencies struct {
	RunTUI func() error
}

// Execute runs the root command for the letterpress CLI.
func Execute() error {
	return NewRootCmd(defaultDependencies()).Execute()
}

// NewRootCmd constructs the root command with injectable behavior for tests.
func NewRootCmd(deps Dependencies) *cobra.Command {
	if deps.RunTUI == nil {
		deps.RunTUI = tui.Run
	}

	cmd := &cobra.Command{
		Use:           "letterpress",
		Short:         "A terminal-first letter and card composer",
		Long:          "letterpress is a terminal-first letter and card composer for beautiful printable messages.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.RunTUI()
		},
	}

	cmd.AddCommand(newTUICmd(deps))
	cmd.AddCommand(cli.NewTemplatesCmd())
	cmd.AddCommand(cli.NewValidateCmd())
	return cmd
}

func newTUICmd(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.RunTUI()
		},
	}
}

func defaultDependencies() Dependencies {
	return Dependencies{
		RunTUI: tui.Run,
	}
}
