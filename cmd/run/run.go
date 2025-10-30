package run

import (
	"time"

	"github.com/example/hd-smart-idle/internal/daemon"
	"github.com/example/hd-smart-idle/internal/hw"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	cron := &daemon.CronExpr{}
	// Set default value: 22:00
	_ = cron.Parse("22 00")

	var (
		standbyValue int
		pollInterval time.Duration
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.Infof("starting hd-smart-idle (schedule=%s standby=%d poll=%s dry-run=%v)", cron, standbyValue, pollInterval, dryRun)

			ctrl := hw.NewHDDControl()
			disks, err := ctrl.List()
			if err != nil {
				logrus.Fatalf("failed to list disks: %v", err)
				return err
			}
			if len(disks) == 0 {
				logrus.Warn("no mechanical disks found, nothing to do")
			} else {
				logrus.Infof("found mechanical disks: %v", disks)
			}

			d := daemon.New(&daemon.Config{
				Devices:      disks,
				PollInterval: pollInterval,
				Cron:         cron,
				StandbyValue: standbyValue,
				DryRun:       dryRun,
			})

			return d.Run()
		},
	}

	// command-local flags (previously on root) - bind directly to local vars
	cmd.Flags().VarP(cron, "time", "t", "daily time (hour min) to set standby timeout for all mechanical disks")
	cmd.Flags().IntVarP(&standbyValue, "standby", "s", 120, "hdparm -S value to set at scheduled time (e.g. 120)")
	cmd.Flags().DurationVarP(&pollInterval, "poll", "p", 10*time.Second, "poll interval for checking disk state")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "do not execute hdparm, only log actions")

	return cmd
}
