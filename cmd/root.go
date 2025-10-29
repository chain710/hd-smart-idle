package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/example/hd-smart-idle/internal/daemon"
	"github.com/example/hd-smart-idle/internal/hw"
)

var (
	scheduleTime string
	standbyValue int
	pollInterval time.Duration
	dryRun       bool
)

var rootCmd = &cobra.Command{
	Use:   "hd-smart-idle",
	Short: "Daemon to smartly manage HDD standby timers",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the daemon (default)",
	Run: func(cmd *cobra.Command, args []string) {
		logrus.SetOutput(os.Stdout)
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

		logrus.Infof("starting hd-smart-idle (schedule=%s standby=%d poll=%s dry-run=%v)", scheduleTime, standbyValue, pollInterval, dryRun)

		// discover mechanical disks
		disks, err := hw.ListMechanicalDisks()
		if err != nil {
			logrus.Fatalf("failed to list disks: %v", err)
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

		if err := d.Run(); err != nil {
			logrus.Fatalf("daemon exited: %v", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&scheduleTime, "time", "22:00", "daily time (HH:MM) to set standby timeout for all mechanical disks")
	runCmd.Flags().IntVar(&standbyValue, "standby", 120, "hdparm -S value to set at scheduled time (e.g. 120)")
	runCmd.Flags().DurationVar(&pollInterval, "poll", 10*time.Second, "poll interval for checking disk state")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "do not execute hdparm, only log actions")
}
