package hw

import (
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
