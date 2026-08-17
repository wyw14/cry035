# Bug 是什么

同一设备在同一天有两个保养项目到期时，批量自动排程会给两项分配相同停用窗口；第一项入库后，第二项因窗口重叠失败。

# 如何触发

在原始 Bug 环境执行：

```text
go test ./tests -run '^TestAutomaticGenerationAllocatesDistinctShutdownWindows001$' -count=20
```

# 错误信息

```text
the second due program conflicts with generation-001-1 instead of receiving the next adjacent shutdown window
```
