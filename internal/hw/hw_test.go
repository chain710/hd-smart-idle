package hw

import (
	"errors"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestHDDController_List(t *testing.T) {
	tests := []struct {
		name     string
		fsys     fstest.MapFS
		wantLen  int
		wantDisk string
	}{
		{
			name: "single mechanical disk",
			fsys: fstest.MapFS{
				"sys/block/sda/queue/rotational":     &fstest.MapFile{Data: []byte("1")},
				"sys/block/nvme0n1/queue/rotational": &fstest.MapFile{Data: []byte("0")},
				"dev/sda":                            &fstest.MapFile{Data: []byte("")},
			},
			wantLen:  1,
			wantDisk: "/dev/sda",
		},
		{
			name: "no dev entry",
			fsys: fstest.MapFS{
				"sys/block/sdb/queue/rotational": &fstest.MapFile{Data: []byte("1")},
			},
			wantLen:  0,
			wantDisk: "",
		},
		{
			name: "multiple mechanical disks",
			fsys: fstest.MapFS{
				"sys/block/sda/queue/rotational": &fstest.MapFile{Data: []byte("1")},
				"sys/block/sdb/queue/rotational": &fstest.MapFile{Data: []byte("1")},
				"dev/sda":                        &fstest.MapFile{Data: []byte("")},
				"dev/sdb":                        &fstest.MapFile{Data: []byte("")},
			},
			wantLen: 2,
		},
		{
			name: "only nvme disk (non-rotational)",
			fsys: fstest.MapFS{
				"sys/block/nvme0n1/queue/rotational": &fstest.MapFile{Data: []byte("0")},
				"dev/nvme0n1":                        &fstest.MapFile{Data: []byte("")},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := defaultHDDControl{fsys: tt.fsys}
			disks, err := d.List()

			require.NoError(t, err)
			require.Len(t, disks, tt.wantLen)
			if tt.wantLen > 0 && tt.wantDisk != "" {
				require.Equal(t, tt.wantDisk, disks[0])
			}
		})
	}
}

func TestParseHDParmState(t *testing.T) {
	tests := []struct {
		name           string
		hdparmOutput   string
		cmdErr         error
		expectState    string
		expectError    bool
		expectErrorMsg string
	}{
		{
			name: "active/idle state",
			hdparmOutput: `/dev/sda:
 drive state is:  active/idle
`,
			cmdErr:      nil,
			expectState: DriveStateActive,
			expectError: false,
		},
		{
			name: "standby state",
			hdparmOutput: `/dev/sdd:
 drive state is:  standby
`,
			cmdErr:      nil,
			expectState: DriveStateStandby,
			expectError: false,
		},
		{
			name: "unknown state",
			hdparmOutput: `/dev/sde:
 drive state is:  unknown
`,
			cmdErr:      nil,
			expectState: DriveStateActive,
			expectError: false,
		},
		{
			name: "sleeping state",
			hdparmOutput: `/dev/sdf:
 drive state is:  sleeping
`,
			cmdErr:      nil,
			expectState: DriveStateStandby,
			expectError: false,
		},
		{
			name:           "device not found error",
			hdparmOutput:   `/dev/sdX: No such file or directory\n`,
			cmdErr:         errors.New("exit status 1"),
			expectState:    "",
			expectError:    true,
			expectErrorMsg: os.ErrNotExist.Error(),
		},
		{
			name:           "empty output",
			hdparmOutput:   ``,
			cmdErr:         nil,
			expectState:    "",
			expectError:    true,
			expectErrorMsg: "malformed hdparm output",
		},
		{
			name: "malformed output without colon",
			hdparmOutput: `/dev/sda:
 drive state is active/idle
`,
			cmdErr:         nil,
			expectState:    "",
			expectError:    true,
			expectErrorMsg: "malformed hdparm output",
		},
		{
			name: "case insensitive parsing",
			hdparmOutput: `/dev/sda:
 Drive State Is:  active/idle
`,
			cmdErr:      nil,
			expectState: DriveStateActive,
			expectError: false,
		},
		{
			name: "unknown state value",
			hdparmOutput: `/dev/sda:
 drive state is:  unknown_state
`,
			cmdErr:         nil,
			expectState:    "",
			expectError:    true,
			expectErrorMsg: "malformed hdparm output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := defaultHDDControl{}
			state, err := d.parseHDParmState(tt.hdparmOutput, tt.cmdErr)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectErrorMsg)
				require.Equal(t, "", state)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectState, state)
			}
		})
	}
}
