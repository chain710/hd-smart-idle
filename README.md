# hd-smart-idle

`hd-smart-idle` is a Go daemon that intelligently manages the standby state of mechanical hard drives (HDDs). It keeps drives awake during active periods and reapplies standby timers based on a scheduled cron expression, preventing frequent transitions between standby and active modes to extend drive lifespan.

## Features

- **Auto-detect rotational disks**: Automatically discovers mechanical hard drives in the system.
- **Intelligent standby management**: Sets standby timers according to cron expressions and keeps drives awake when active.
- **Configurable polling interval**: Regularly checks drive status.
- **Dry-run mode**: Logs actions without executing hdparm commands, for testing.
- **Specify devices**: Allows manual specification of devices to monitor.
- **Systemd integration**: Provides a systemd service file for running as a system service.

## Installation

```bash
go install github.com/chain710/hd-smart-idle@latest
```

## Usage

### Basic Usage

Run the daemon:

```bash
./bin/hd-smart-idle run
```

Manually set standby timeout:

```bash
./bin/hd-smart-idle standby --devices /dev/sda --value 120
```

### Global Options

- `--log-level <level>`: Set log level (debug|info|warn|error). Default is info.

### run Command Options

The `run` command supports the following flags:

- `-t, --time <hour min>`: Set the daily time (hour min) to apply standby timeout for all mechanical disks. Default is `22 00` (10 PM). E.g., `-t "23 30"` sets to 11:30 PM.
- `-s, --standby <value>`: Standby timeout value in 5-second units (e.g., 120 = 10 minutes). Default is 120.
- `-p, --poll <duration>`: Polling interval for checking disk state. Default is 10 seconds.
- `-d, --dry-run`: Enable dry-run mode, only log actions without executing hdparm commands.
- `-D, --devices <device1,device2,...>`: Specific devices to monitor (e.g., /dev/sda,/dev/sdb); if not set, auto-detect all rotational disks.

### standby Command Options

The `standby` command is used to manually set standby timeout for specified devices:

- `-s, --value <value>`: Standby timeout value in 5-second units (e.g., 120 = 10 minutes). Default is 120.
- `-d, --dry-run`: Enable dry-run mode, only log actions without executing hdparm commands.
- `-D, --devices <device1,device2,...>`: Specific devices to configure (required, e.g., /dev/sda,/dev/sdb).

### Examples

1. **Run daemon with default config**:
   ```bash
   ./bin/hd-smart-idle run
   ```

2. **Custom standby time and timeout**:
   ```bash
   ./bin/hd-smart-idle run --time "01 00" --standby 240
   ```
   This sets a 20-minute standby timeout at 1 AM.

3. **Dry-run mode**:
   ```bash
   ./bin/hd-smart-idle run --dry-run
   ```

4. **Specify specific devices**:
   ```bash
   ./bin/hd-smart-idle run --devices /dev/sda,/dev/sdb
   ```

5. **Manually set standby timeout**:
   ```bash
   ./bin/hd-smart-idle standby --devices /dev/sda --value 120 --dry-run
   ```
   This sets a 10-minute standby timeout for /dev/sda in dry-run mode.

6. **Enable debug logging**:
   ```bash
   ./bin/hd-smart-idle --log-level debug run
   ```

### Environment Variables

- `HDPARM_PATH`: Specify the path to the hdparm executable. Defaults to `/sbin/hdparm`. Used to configure an alternate path for testing.
