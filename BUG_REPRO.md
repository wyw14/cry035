# Bug 是什么

已经开工的保养计划超过停用窗口后仍保持进行中状态，也不会生成对应的严重超窗告警。

# 如何触发

在原始 Bug 环境执行：

```text
go test ./tests -run '^TestInProgressPlanBecomesOverdueAndAlerts004$' -count=20
```

# 错误信息

```text
the expired plan remained in_progress and no critical overrun alert was created
```
