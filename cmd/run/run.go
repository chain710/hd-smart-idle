package run

import (
	"time"

	"github.com/chain710/hd-smart-idle/internal/daemon"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	cron := &daemon.CronExpr{}
	// Set default value: 22:00
	if err := cron.Parse("22 00"); err != nil {
		panic("cron.Parse should NOT fail!")
	}

	var (
		standbyValue int
		pollInterval time.Duration
		dryRun       bool
		devices      []string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.Infof("starting hd-smart-idle (schedule=%s standby=%d poll=%s dry-run=%v)", cron, standbyValue, pollInterval, dryRun)

			d, err := daemon.New(daemon.Config{
				Devices:      devices,
				PollInterval: pollInterval,
				Cron:         cron,
				StandbyValue: standbyValue,
				DryRun:       dryRun,
			})
			if err != nil {
				logrus.Fatalf("failed to create daemon: %v", err)
				return err
			}

			return d.Run()
		},
	}

	// command-local flags (previously on root) - bind directly to local vars
	cmd.Flags().VarP(cron, "time", "t", "daily time (hour min) to set standby timeout for all mechanical disks")
	cmd.Flags().IntVarP(&standbyValue, "standby", "s", 120, "hdparm -S value to set at scheduled time (e.g. 120)")
	cmd.Flags().DurationVarP(&pollInterval, "poll", "p", 10*time.Second, "poll interval for checking disk state")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "do not execute hdparm, only log actions")
	cmd.Flags().StringSliceVarP(&devices, "devices", "D", nil, "specific devices to monitor (e.g. /dev/sda,/dev/sdb); if not set, auto-detect all rotational disks")

	return cmd
}
