# 楼宇垂直运输设备保养计划平台

cry035 是一个离线运行的物业设备保养闭环示例。它管理电梯、货梯和升降设备的档案、法定周期、停用窗口、执行证据、复核、缺陷整改、复检、恢复运行、供应商费用和审计时间线，不包含客服工单或采购下单。

## 模块职责

- `internal/domain/equipment`：楼宇、型号、设备和运行状态。
- `internal/domain/maintenance`：保养项目、半开时间窗、计划状态机和周期计算。
- `internal/domain/inspection`：清单快照、实测项、照片证据和评价。
- `internal/domain/defect`：缺陷等级、整改、复检和恢复资格。
- `internal/application`：catalog、planning、execution、review、remediation、reporting 用例；每个用例只依赖自己的仓储接口。
- `internal/repository`：内存演示仓储；`internal/repository/postgres` 是 pgx 事务实现。
- `internal/service/scheduler`：本地周期生成、逾期扫描和安全告警。
- `internal/platform`：时钟、ID、受控附件目录和本地通知适配器。
- `internal/transport/http`：Gin 路由、请求超时、角色边界、统一错误和分页。
- `web`：Vue 3 + TypeScript + Vite + Pinia 的设备总览、计划日历、执行、整改、复检、时间线和配置页。

## 本地启动

需要 Go 1.24+、Node.js 20+ 和 npm。没有数据库时，服务使用可重复的内存演示数据：

```powershell
go run ./cmd/server
cd web
npm ci
npm run dev
```

浏览器访问 `http://localhost:5173`。Vite 会把 `/api` 代理到 `http://localhost:8080`。

使用 PostgreSQL 时，复制 `.env.example` 为 `.env`，设置 `DATABASE_URL`，然后执行：

```powershell
$env:DATABASE_URL = "postgres://vertical:vertical@localhost:5432/vertical_maintenance?sslmode=disable"
go run ./cmd/migrate
go run ./cmd/server
```

迁移脚本按文件名顺序执行，种子使用 `ON CONFLICT DO NOTHING`，不会覆盖既有数据。附件默认落在 `./var/attachments`，只允许 PNG/JPEG/WebP/PDF，单个文件默认不超过 10 MiB。

## Docker Compose

```powershell
docker compose up --build
```

访问 `http://localhost:8088`。Compose 启动 PostgreSQL、一次性迁移容器、Go API 和 Nginx 前端。官方基础镜像可构建 amd64/arm64；本项目只在当前机器平台做验证。停止服务：

```powershell
docker compose down
```

## 演示角色

本地演示使用请求头模拟已登录身份，不包含真实账号或密钥：

- `X-Role: safety_manager`：配置、复核、恢复运行。
- `X-Role: planner`：设备档案和计划排程。
- `X-Role: technician`：执行记录、证据和整改。
- `X-Role: reviewer`：执行复核和复检复核。
- `X-Role: auditor`：只读查询和导出。

可选 `X-Actor` 写入审计事件；未传角色时只获得 `auditor` 只读权限。生产环境应在网关接入正式身份系统并移除演示默认身份。

## 业务状态规则

计划通常从 `planned` 开始，可进入 `in_progress`、`suspended` 或逾期；提交完整清单后进入 `pending_review`。复核通过进入 `qualified`，失败进入 `restricted` 并把设备置为 `limited`。限用计划只能在本次限制周期的缺陷全部整改、关联复检合格并完成复检复核后进入 `qualified`，随后显式恢复为 `restored`/运行。直接恢复和跨周期复检都会被拒绝。

同一设备的时间窗使用 `[start,end)` 语义：包含、交叉和相同窗口都拒绝，首尾相接允许。PostgreSQL 使用排斥约束，内存仓储使用同一锁保护检查和写入。计划创建支持 `Idempotency-Key`；自动生成键由设备、项目版本和法定到期日组成。

## API 示例

```powershell
curl http://localhost:8080/healthz
curl -H "X-Role: auditor" http://localhost:8080/api/v1/equipment?page=1&page_size=20

curl -X POST http://localhost:8080/api/v1/plans `
  -H "Content-Type: application/json" -H "X-Role: planner" -H "Idempotency-Key: demo-cycle-1" `
  -d '{"equipment_id":"equipment-1","program_id":"program-monthly","start":"2026-08-20T01:00:00Z","end":"2026-08-20T03:00:00Z","assignee":"tech-a","spares":[]}'
```

完整 OpenAPI 3.0 描述在 `api/openapi/openapi.yaml`。错误包含稳定 `code`、可读 `message`、`field_errors` 和 `request_id`；列表统一返回 `items/page/page_size/total`。

## 验证命令

```powershell
gofmt -w cmd internal tests
go test ./...
go test -race ./...
go vet ./...
go build ./...
cd web
npm ci
npm test
npm run build
```

本次 baseline 生成阶段已实际执行并通过 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`npm test` 和 `npm run build`。PostgreSQL 集成测试在设置 `TEST_DATABASE_URL` 且先执行迁移时运行，否则会明确跳过，不伪造数据库结果。

## 安全与交付边界

服务使用结构化 Zap 日志、panic 恢复、安全响应头、CORS、优雅停机、请求超时和审计事件；日志不写入密码、令牌或附件内容。仓库不应提交 `.env`、`node_modules`、`var`、`dist`、依赖缓存或运行期数据库文件。
