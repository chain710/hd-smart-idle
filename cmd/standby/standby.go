package standby

import (
	"fmt"

	"github.com/chain710/hd-smart-idle/internal/hw"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewStandbyCmd() *cobra.Command {
	var (
		standbyValue int
		dryRun       bool
		devices      []string
	)

	cmd := &cobra.Command{
		Use:   "standby",
		Short: "Set standby timeout for mechanical disks",
		RunE: func(cmd *cobra.Command, args []string) error {
			controller := hw.NewHDDControl()
			if dryRun {
				controller = hw.NewDryRunHDDControl(controller)
			}

			logrus.Infof("setting standby timeout %d for devices: %v", standbyValue, devices)

			// Set standby timeout for each device
			hasError := false
			for _, dev := range devices {
				if err := controller.SetStandbyTimeout(dev, standbyValue); err != nil {
					logrus.Errorf("failed to set standby on %s: %v", dev, err)
					hasError = true
				} else {
					logrus.Infof("set standby timeout %d on %s", standbyValue, dev)
				}
			}

			if hasError {
				return fmt.Errorf("failed to set standby timeout on one or more devices")
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&standbyValue, "value", "s", 120, "standby timeout value in 5 seconds units (e.g. 120 = 10 minutes)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "do not issue standby, only log actions")
	cmd.Flags().StringSliceVarP(&devices, "devices", "D", nil, "specific devices to configure (e.g. /dev/sda,/dev/sdb) [required]")
	// nolint:errcheck
	cmd.MarkFlagRequired("devices")
	return cmd
}
