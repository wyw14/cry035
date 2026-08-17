# Bug 是什么

两个强制拍照检查项可以复用同一张照片并成功提交，导致无法证明每个检查项都有独立现场证据。

# 如何触发

在原始 Bug 环境执行：

```text
go test ./tests -run '^TestExecutionRejectsReusedChecklistEvidence002$' -count=20
```

# 错误信息

```text
two required checklist measurements referencing the same photo were accepted without an error
```
