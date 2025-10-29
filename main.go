package main

import (
	"fmt"
	"os"

	runcmd "github.com/example/hd-smart-idle/cmd/run"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	flagLogLevel = "log-level"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "hd-smart-idle",
		Short: "Daemon to smartly manage HDD standby timers",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			lvl, err := cmd.Flags().GetString(flagLogLevel)
			if err != nil {
				return err
			}
			level, err := logrus.ParseLevel(lvl)
			if err != nil {
				return fmt.Errorf("wrong log level `%v`: %w", lvl, err)
			}
			logrus.SetOutput(os.Stdout)
			logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
			logrus.SetLevel(level)
			return nil
		},
	}

	rootCmd.PersistentFlags().String(flagLogLevel, "info", "log level: debug|info|warn|error")
	rootCmd.AddCommand(runcmd.NewRunCmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
