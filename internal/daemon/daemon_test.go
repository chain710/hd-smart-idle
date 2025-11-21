package daemon

import (
	"context"
	"fmt"
	"maps"
	"testing"
	"testing/synctest"
	"time"

	"github.com/chain710/hd-smart-idle/internal/hw"
)

func TestDaemon_mainLoop_PollDrivenScenarios(t *testing.T) {
	cases := []struct {
		name  string
		devs  []string
		cfg   Config
		steps []time.Duration
		setup func(*hw.MockHDDControl)
	}{
		{
			name: "polling_invokes_scan",
			devs: []string{"/dev/sda", "/dev/sdb"},
			cfg: Config{
				PollInterval: 10 * time.Second,
				Cron:         &CronExpr{Hour: 22, Min: 0},
				StandbyValue: 120,
			},
			steps: []time.Duration{10 * time.Second, 10 * time.Second, 10 * time.Second},
			setup: func(m *hw.MockHDDControl) {
				m.EXPECT().GetState("/dev/sda").Return(hw.DriveStateActive, nil).Times(3)
				m.EXPECT().GetState("/dev/sdb").Return(hw.DriveStateActive, nil).Times(3)
			},
		},
		{
			name: "standby_to_active_disables_spindown",
			devs: []string{"/dev/sda"},
			cfg: Config{
				PollInterval: 5 * time.Second,
				Cron:         &CronExpr{Hour: 22, Min: 0},
				StandbyValue: 120,
			},
			steps: []time.Duration{5 * time.Second, 5 * time.Second},
			setup: func(m *hw.MockHDDControl) {
				m.EXPECT().GetState("/dev/sda").Return(hw.DriveStateStandby, nil).Once()
				m.EXPECT().GetState("/dev/sda").Return(hw.DriveStateActive, nil).Once()
				m.EXPECT().SetStandbyTimeout("/dev/sda", 0).Return(nil).Once()
			},
		},
		{
			name: "standby_to_active_disable_error_logged",
			devs: []string{"/dev/sda"},
			cfg: Config{
				PollInterval: 3 * time.Second,
				Cron:         &CronExpr{Hour: 22, Min: 0},
				StandbyValue: 120,
			},
			steps: []time.Duration{3 * time.Second, 3 * time.Second},
			setup: func(m *hw.MockHDDControl) {
				m.EXPECT().GetState("/dev/sda").Return(hw.DriveStateStandby, nil).Once()
				m.EXPECT().GetState("/dev/sda").Return(hw.DriveStateActive, nil).Once()
				m.EXPECT().SetStandbyTimeout("/dev/sda", 0).Return(fmt.Errorf("hdparm failed")).Once()
			},
		},
		{
			name: "get_state_error_is_tolerated",
			devs: []string{"/dev/sda"},
			cfg: Config{
				PollInterval: 5 * time.Second,
				Cron:         &CronExpr{Hour: 22, Min: 0},
				StandbyValue: 120,
			},
			steps: []time.Duration{5 * time.Second},
			setup: func(m *hw.MockHDDControl) {
				m.EXPECT().GetState("/dev/sda").Return("", fmt.Errorf("device error")).Once()
			},
		},
		{
			name: "multiple_devices_transitions",
			devs: []string{"/dev/sda", "/dev/sdb", "/dev/sdc"},
			cfg: Config{
				PollInterval: 7 * time.Second,
				Cron:         &CronExpr{Hour: 2, Min: 30},
				StandbyValue: 240,
			},
			steps: []time.Duration{7 * time.Second, 7 * time.Second},
			setup: func(m *hw.MockHDDControl) {
				m.EXPECT().GetState("/dev/sda").Return(hw.DriveStateActive, nil).Once()
				m.EXPECT().GetState("/dev/sdb").Return(hw.DriveStateStandby, nil).Once()
				m.EXPECT().GetState("/dev/sdc").Return(hw.DriveStateActive, nil).Once()

				m.EXPECT().GetState("/dev/sda").Return(hw.DriveStateActive, nil).Once()
				m.EXPECT().GetState("/dev/sdb").Return(hw.DriveStateActive, nil).Once()
				m.EXPECT().SetStandbyTimeout("/dev/sdb", 0).Return(nil).Once()
				m.EXPECT().GetState("/dev/sdc").Return(hw.DriveStateStandby, nil).Once()
			},
		},
	}

	for _, tt := range cases {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				mockCtrl := hw.NewMockHDDControl(t)
				tc.setup(mockCtrl)

				d := &Daemon{
					cfg:        tc.cfg,
					controller: mockCtrl,
					last:       make(map[string]string),
				}

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				done := make(chan struct{})
				go func() {
					d.mainLoop(ctx, tc.devs)
					close(done)
				}()

				synctest.Wait()
				for _, step := range tc.steps {
					time.Sleep(step)
					synctest.Wait()
				}

				cancel()
				<-done
			})
		})
	}
}

func TestDaemon_mainLoop_ScheduledStandbySet(t *testing.T) {
	cases := []struct {
		name      string
		devs      []string
		poll      time.Duration
		standby   int
		lastState map[string]string
		expect    []string
		setupMock func(*hw.MockHDDControl)
	}{
		{
			name:    "all_devices_active",
			devs:    []string{"/dev/sda", "/dev/sdb"},
			poll:    10 * time.Second,
			standby: 120,
			lastState: map[string]string{
				"/dev/sda": hw.DriveStateActive,
				"/dev/sdb": hw.DriveStateActive,
			},
			expect: []string{"/dev/sda", "/dev/sdb"},
			setupMock: func(m *hw.MockHDDControl) {
				m.On("GetState", "/dev/sda").Return(hw.DriveStateActive, nil).Maybe()
				m.On("GetState", "/dev/sdb").Return(hw.DriveStateActive, nil).Maybe()
			},
		},
		{
			name:    "inactive_devices_filtered",
			devs:    []string{"/dev/sda", "/dev/sdb", "/dev/sdc"},
			poll:    15 * time.Second,
			standby: 300,
			lastState: map[string]string{
				"/dev/sda": hw.DriveStateActive,
				"/dev/sdb": hw.DriveStateStandby,
				"/dev/sdc": hw.DriveStateActive,
			},
			expect: []string{"/dev/sda", "/dev/sdc"},
			setupMock: func(m *hw.MockHDDControl) {
				m.On("GetState", "/dev/sda").Return(hw.DriveStateActive, nil).Maybe()
				m.On("GetState", "/dev/sdb").Return(hw.DriveStateStandby, nil).Maybe()
				m.On("GetState", "/dev/sdc").Return(hw.DriveStateActive, nil).Maybe()
			},
		},
	}

	for _, tt := range cases {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				mockCtrl := hw.NewMockHDDControl(t)
				tc.setupMock(mockCtrl)

				now := time.Now()
				cron := &CronExpr{Hour: now.Hour(), Min: now.Minute()}

				d := &Daemon{
					cfg: Config{
						PollInterval: tc.poll,
						Cron:         cron,
						StandbyValue: tc.standby,
					},
					controller: mockCtrl,
					last:       maps.Clone(tc.lastState),
				}

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				done := make(chan struct{})
				go func() {
					d.mainLoop(ctx, tc.devs)
					close(done)
				}()

				// Wait for timers to be set up
				synctest.Wait()

				for _, dev := range tc.expect {
					mockCtrl.EXPECT().SetStandbyTimeout(dev, tc.standby).Return(nil).Once()
				}

				// Advance time to trigger the scheduled event (24 hours + 1 second)
				time.Sleep(24*time.Hour + time.Second)
				synctest.Wait()

				cancel()
				<-done
			})
		})
	}
}

func TestDaemon_mainLoop_ContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mockCtrl := hw.NewMockHDDControl(t)
		devs := []string{"/dev/sda"}

		// May or may not be called depending on timing
		mockCtrl.On("GetState", "/dev/sda").Return(hw.DriveStateActive, nil).Maybe()

		cron := &CronExpr{Hour: 1, Min: 0}
		d := &Daemon{
			cfg: Config{
				PollInterval: 5 * time.Second,
				Cron:         cron,
				StandbyValue: 120,
			},
			controller: mockCtrl,
			last:       make(map[string]string),
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		go func() {
			d.mainLoop(ctx, devs)
			close(done)
		}()

		// Wait for goroutine to initialize
		synctest.Wait()

		cancel()

		// Verify mainLoop exits after context cancellation
		<-done
	})
}
