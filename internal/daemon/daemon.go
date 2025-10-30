package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/example/hd-smart-idle/internal/hw"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Devices      []string
	PollInterval time.Duration
	ScheduleTime string // HH:MM
	StandbyValue int
	DryRun       bool
}

type Daemon struct {
	cfg        *Config
	controller hw.HDDControl
	mu         sync.Mutex
	// device -> last known state
	last map[string]string
}

func New(cfg *Config) *Daemon {
	var controller = hw.NewHDDControl()

	// Honor DryRun by wrapping the controller with a dry-run wrapper.
	if cfg != nil && cfg.DryRun {
		controller = hw.NewDryRunHDDControl(controller)
	}

	return &Daemon{
		cfg:        cfg,
		controller: controller,
		last:       make(map[string]string),
	}
}

// Run starts the daemon loops and blocks until error or context cancel
func (d *Daemon) Run() error {
	if d.cfg == nil {
		return fmt.Errorf("nil config")
	}

	// canonicalize devices
	devs := append([]string{}, d.cfg.Devices...)
	sort.Strings(devs)
	logrus.Infof("monitoring devices: %v", devs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// handle signals in a goroutine and cancel context
	go func() {
		sig := <-sigChan
		logrus.Infof("received signal: %v, initiating graceful shutdown", sig)
		cancel()
	}()

	// run main loop
	d.mainLoop(ctx, devs)

	return nil
}

func (d *Daemon) mainLoop(ctx context.Context, devs []string) {
	pollTicker := time.NewTicker(d.cfg.PollInterval)
	defer pollTicker.Stop()

	// parse schedule time
	cron := &CronExpr{}
	schedulerEnabled := true
	if err := cron.Parse(d.cfg.ScheduleTime); err != nil {
		logrus.Warnf("invalid schedule time %q, scheduler disabled: %v", d.cfg.ScheduleTime, err)
		schedulerEnabled = false
	}

	var nextScheduledTime time.Time
	if schedulerEnabled {
		nextScheduledTime = cron.Next(time.Now())
		logrus.Infof("scheduler: next run at %s", nextScheduledTime.Format(time.RFC3339))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			d.checkAll(devs)
		case <-time.After(time.Until(nextScheduledTime)):
			if schedulerEnabled {
				logrus.Infof("scheduler triggered at %s — setting standby=%d for all devices", time.Now().Format(time.RFC3339), d.cfg.StandbyValue)
				for _, dev := range devs {
					if err := d.controller.SetStandbyTimeout(dev, d.cfg.StandbyValue); err != nil {
						logrus.Errorf("failed to set standby on %s: %v", dev, err)
					} else {
						logrus.Infof("set standby timer %d on %s", d.cfg.StandbyValue, dev)
					}
				}
				nextScheduledTime = cron.Next(time.Now())
				logrus.Infof("scheduler: next run at %s", nextScheduledTime.Format(time.RFC3339))
			}
		}
	}
}

func (d *Daemon) checkAll(devs []string) {
	for _, dev := range devs {
		state, err := d.controller.GetState(dev)
		if err != nil {
			logrus.Debugf("hdparm -C %s returned parse error: %v (raw: %s)", dev, err, state)
		}
		state = strings.ToLower(strings.TrimSpace(state))
		last := d.getLast(dev)
		if last == "" {
			// first time: if non-standby -> ensure spindown disabled
			logrus.Debugf("initial state for %s: %s", dev, state)
			if !strings.Contains(state, "standby") {
				d.disableSpindown(dev)
			}
		} else {
			if !strings.Contains(state, "standby") && strings.Contains(last, "standby") {
				logrus.Infof("device %s left standby (state=%s) — disabling spindown timer", dev, state)
				d.disableSpindown(dev)
			}
		}
		d.setLast(dev, state)
	}
}

func (d *Daemon) disableSpindown(dev string) {
	if err := d.controller.SetStandbyTimeout(dev, 0); err != nil {
		logrus.Errorf("failed to disable spindown on %s: %v", dev, err)
	} else {
		logrus.Infof("disabled spindown timer on %s", dev)
	}
}

func (d *Daemon) getLast(dev string) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.last[dev]
}

func (d *Daemon) setLast(dev, state string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.last[dev] = state
}
