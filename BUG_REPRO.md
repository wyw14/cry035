# Bug 是什么

同一限用批次有两项重大缺陷时，只关闭并复检其中一项就能恢复设备，另一项未解决的阻断缺陷被忽略。

# 如何触发

在原始 Bug 环境执行：

```text
go test ./tests -run '^TestRestoreRequiresEveryBlockingDefectClosed003$' -count=20
```

# 错误信息

```text
equipment was restored after only one of two critical defects was rectified and reviewed
```
