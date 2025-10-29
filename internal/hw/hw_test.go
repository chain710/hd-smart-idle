package hw

import (
	"testing"
	"testing/fstest"
)

func TestListMechanicalDisksFS(t *testing.T) {
	m := fstest.MapFS{
		"sys/block/sda/queue/rotational":     &fstest.MapFile{Data: []byte("1")},
		"sys/block/nvme0n1/queue/rotational": &fstest.MapFile{Data: []byte("0")},
		"dev/sda":                            &fstest.MapFile{Data: []byte("")},
	}

	d := defaultHDDControl{fsys: m}
	disks, err := d.List()
	if err != nil {
		t.Fatalf("ListMechanicalDisksFS error: %v", err)
	}
	if len(disks) != 1 {
		t.Fatalf("expected 1 disk, got %d: %v", len(disks), disks)
	}
	if disks[0] != "/dev/sda" {
		t.Fatalf("expected /dev/sda, got %s", disks[0])
	}
}

func TestListMechanicalDisksFS_NoDevEntry(t *testing.T) {
	m := fstest.MapFS{
		"sys/block/sdb/queue/rotational": &fstest.MapFile{Data: []byte("1")},
		// note: no dev/sdb entry
	}
	d := defaultHDDControl{fsys: m}
	disks, err := d.List()
	if err != nil {
		t.Fatalf("ListMechanicalDisksFS error: %v", err)
	}
	if len(disks) != 0 {
		t.Fatalf("expected 0 disks when /dev entry missing, got %v", disks)
	}
}
