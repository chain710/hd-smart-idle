package run

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/example/hd-smart-idle/internal/daemon"
	"github.com/example/hd-smart-idle/internal/hw"
)

func NewRunCmd() *cobra.Command {
	var (
		scheduleTime string
		standbyValue int
		pollInterval time.Duration
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.Infof("starting hd-smart-idle (schedule=%s standby=%d poll=%s dry-run=%v)", scheduleTime, standbyValue, pollInterval, dryRun)

			disks, err := hw.ListMechanicalDisks()
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
				ScheduleTime: scheduleTime,
				StandbyValue: standbyValue,
				DryRun:       dryRun,
			})

			return d.Run()
		},
	}

	// command-local flags (previously on root) - bind directly to local vars
	cmd.Flags().StringVarP(&scheduleTime, "time", "t", "22:00", "daily time (HH:MM) to set standby timeout for all mechanical disks")
	cmd.Flags().IntVarP(&standbyValue, "standby", "s", 120, "hdparm -S value to set at scheduled time (e.g. 120)")
	cmd.Flags().DurationVarP(&pollInterval, "poll", "p", 10*time.Second, "poll interval for checking disk state")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "do not execute hdparm, only log actions")

	return cmd
}
