# Bug 是什么

供应商费用可以为设备 A 关联设备 B 的保养计划并成功入账，造成费用台账、计划和审计事件的设备范围不一致。

# 如何触发

在原始 Bug 环境执行：

```text
go test ./tests -run '^TestSupplierLedgerRejectsCrossEquipmentPlan005$' -count=20
```

# 错误信息

```text
a service for equipment A that referenced equipment B's plan was accepted
```
