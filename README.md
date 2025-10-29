# hd-smart-idle

这是一个用 Go 实现的守护进程，用于智能监控机器上所有机械盘的 standby 状态：

- 当某个设备进入非-standby（被唤醒或活跃）状态时，会取消该设备的 standby 超时（等价于 `hdparm -S 0 /dev/sdX`），保持唤醒状态。
- 在每天指定时间（默认 22:00），为所有机械盘设置 standby 超时（等价于 `hdparm -S 120 /dev/sdX`）。

实现细节
- 语言：Go（pure Go 项目）
- CLI：cobra
- 日志：使用 `logrus`（默认），可选择使用 Go 标准库 `log` 风格输出
- 磁盘检测：通过检查 `/sys/block/*/queue/rotational` 为 `1` 来识别机械盘
- 与磁盘交互：使用 `hdparm -C` 查询状态，使用 `hdparm -S` 设置/取消 standby 超时（因此要求系统上安装 `hdparm`）。

日志框架比较（标准库 log vs logrus）

- 标准库 `log`：
  - 优点：内置、依赖少、使用简单；输出到 stdout 很方便。
  - 缺点：功能较少，不支持日志级别（需自己约定），不支持结构化日志，也没有内建的灵活格式化选项。

- `logrus`：
  - 优点：支持日志级别（Info, Warn, Error, Debug），支持结构化字段，格式配置灵活（json/text），社区广泛使用，便于未来扩展（例如发送到集中式日志）。
  - 缺点：增加少量依赖，二进制体积略增。

本项目默认使用 `logrus` 输出到 stdout，且保留 `--use-stdlog` 开关以切换为更简单的输出风格。

构建和运行

1. 安装依赖并构建：

```bash
go build -o hd-smart-idle ./...
```

2. 运行（示例）：

```bash
# 以守护进程模式运行（此程序当前不会自动 daemonize，建议用 systemd/runit/supervisor 管理）
./hd-smart-idle run --time 22:00 --standby 120 --poll 10s
```

选项
- `--time`: 每天执行设置 standby 的时间，格式 HH:MM，默认 `22:00`。
- `--standby`: hdparm -S 的值（例如 120），默认 `120`。
- `--poll`: 轮询间隔，默认 `10s`。
- `--dry-run`: 不执行 hdparm，仅记录将要运行的命令。
- `--use-stdlog`: 切换为更简单的标准风格输出。

注意事项
- 本程序依赖系统上安装 `hdparm`，并且需要对磁盘设备有足够的权限（通常需要 root）。
- 我选择调用系统 `hdparm` 二进制来做查询和设置，因为直接在 Go 中以 ioctl 完整复刻 hdparm 的功能需要实现大量低层 ATA/SCSI 交互，工程量较大且易出错。若你需要零外部依赖的实现，可以作为后续项，通过直接使用 ioctl/SG_IO 实现 ATA 命令传递。

下一步建议
- 添加 systemd unit 示例并提供安装脚本。
- 增加自启动/安装说明和简单的集成测试（模拟设备或在 CI 上用容器测试）。
