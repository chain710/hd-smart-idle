package daemon

import (
    "context"
    "fmt"
    "sort"
    "strings"
    "sync"
    "time"

    "github.com/sirupsen/logrus"

    "github.com/example/hd-smart-idle/internal/hw"
)

type Config struct {
    Devices      []string
    PollInterval time.Duration
    ScheduleTime string // HH:MM
    StandbyValue int
    DryRun       bool
    Logger       *logrus.Logger
}

type Daemon struct {
    cfg    *Config
    logger *logrus.Entry
    mu     sync.Mutex
    // device -> last known state
    last map[string]string
}

func New(cfg *Config) *Daemon {
    l := logrus.New()
    if cfg != nil && cfg.Logger != nil {
        l = cfg.Logger
    }
    return &Daemon{
        cfg:    cfg,
        logger: l.WithField("component", "daemon"),
        last:   make(map[string]string),
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
    d.logger.Infof("monitoring devices: %v", devs)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // start poller
    go d.poller(ctx, devs)

    // start scheduler
    go d.scheduler(ctx, devs)

    // block forever (or until killed)
    select {}
}

func (d *Daemon) poller(ctx context.Context, devs []string) {
    t := time.NewTicker(d.cfg.PollInterval)
    defer t.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-t.C:
            d.checkAll(devs)
        }
    }
}

func (d *Daemon) checkAll(devs []string) {
    for _, dev := range devs {
        state, err := hw.GetDriveState(dev)
        if err != nil {
            d.logger.Debugf("hdparm -C %s returned parse error: %v (raw: %s)", dev, err, state)
        }
        state = strings.ToLower(strings.TrimSpace(state))
        last := d.getLast(dev)
        if last == "" {
            // first time: if non-standby -> ensure spindown disabled
            d.logger.Debugf("initial state for %s: %s", dev, state)
            if !strings.Contains(state, "standby") {
                d.disableSpindown(dev)
            }
        } else {
            if !strings.Contains(state, "standby") && strings.Contains(last, "standby") {
                d.logger.Infof("device %s left standby (state=%s) — disabling spindown timer", dev, state)
                d.disableSpindown(dev)
            }
        }
        d.setLast(dev, state)
    }
}

func (d *Daemon) scheduler(ctx context.Context, devs []string) {
    // parse schedule time HH:MM
    parts := strings.Split(d.cfg.ScheduleTime, ":")
    if len(parts) != 2 {
        d.logger.Warnf("invalid schedule time %q, scheduler disabled", d.cfg.ScheduleTime)
        return
    }
    hour := 0
    min := 0
    fmt.Sscanf(d.cfg.ScheduleTime, "%d:%d", &hour, &min)

    for {
        now := time.Now()
        // next occurrence
        loc := now.Location()
        next := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, loc)
        if !next.After(now) {
            next = next.Add(24 * time.Hour)
        }
        wait := time.Until(next)
        d.logger.Infof("scheduler: next run at %s (in %s)", next.Format(time.RFC3339), wait)

        select {
        case <-ctx.Done():
            return
        case <-time.After(wait):
            d.logger.Infof("scheduler triggered at %s — setting standby=%d for all devices", time.Now().Format(time.RFC3339), d.cfg.StandbyValue)
            for _, dev := range devs {
                if d.cfg.DryRun {
                    d.logger.Infof("dry-run: would run hdparm -S %d %s", d.cfg.StandbyValue, dev)
                    continue
                }
                if err := hw.SetStandbyTimeout(dev, d.cfg.StandbyValue); err != nil {
                    d.logger.Errorf("failed to set standby on %s: %v", dev, err)
                } else {
                    d.logger.Infof("set standby timer %d on %s", d.cfg.StandbyValue, dev)
                }
            }
        }
    }
}

func (d *Daemon) disableSpindown(dev string) {
    if d.cfg.DryRun {
        d.logger.Infof("dry-run: would run hdparm -S 0 %s", dev)
        return
    }
    if err := hw.SetStandbyTimeout(dev, 0); err != nil {
        d.logger.Errorf("failed to disable spindown on %s: %v", dev, err)
    } else {
        d.logger.Infof("disabled spindown timer on %s", dev)
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
