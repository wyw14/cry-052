# cry052 离线数据脱敏流水线治理平台

cry052 是面向数据管理员的离线脱敏治理系统。它对本地 PostgreSQL 或内置样例库执行可审计的字段分类、策略审批、脱敏预演、批次写入、失败恢复、行数对账与本地报告导出。应用不会连接第三方接口、云存储、CDN、在线模型或真实消息服务。

## 模块职责

- `cmd/server`：进程装配、信号处理和优雅停机。
- `internal/domain`：数据源、目录、策略版本、预演、批次、审批和审计不变量。
- `internal/application`：角色权限、业务用例、事务边界和审计编排。
- `internal/repository/postgres`：PostgreSQL 持久化、唯一约束和乐观更新。
- `internal/repository/memory`：并发安全的测试仓储，不用于生产持久化。
- `internal/service`：掩码、替换、泛化、HMAC 哈希、保留和组合策略执行。
- `internal/transport/http`：`/api/v1`、分页排序、白名单筛选与稳定错误响应。
- `internal/middleware`：request_id、超时、CORS、安全头、本地会话认证和不泄露 panic 内容的恢复。
- `internal/platform`：本地样例数据库、秘密引用、通知、附件、回调、定时事件和日志脱敏适配器。
- `web`：Vue 3、TypeScript、Vite 和 Pinia 中文工作台。

## 本地启动

需要 Go 1.24+、Node.js 22+ 和 PostgreSQL 16+。复制 `.env.example` 中需要的环境变量到本地 shell；不要提交 `.env`。

```bash
createdb cry052
psql cry052 -f migrations/001_init.sql
psql cry052 -f migrations/002_indexes.sql
psql cry052 -f scripts/seed.sql
go run ./cmd/server
```

另开终端启动前端：

```bash
cd web
npm ci
npm run dev
```

前端开发服务器监听 `http://localhost:5173` 并将 API 转发到 `http://localhost:8080`。后端提供 `/healthz` 和需要数据库可用的 `/readyz`。

## 配置

| 变量 | 默认值 | 说明 |
|---|---|---|
| `HTTP_ADDR` | `:8080` | HTTP 监听地址 |
| `DATABASE_URL` | 本地 `cry052` DSN | PostgreSQL 连接，仅从环境读取 |
| `REQUEST_TIMEOUT` | `5s` | 单次请求超时 |
| `ATTACHMENT_DIR` | `./data/attachments` | 受控本地附件目录 |
| `MAX_UPLOAD_BYTES` | `5242880` | 附件大小上限 |
| `SERVICE_ACCOUNT_ROLE` | `masking_executor` | 最小权限执行角色，其他值拒绝启动 |
| `ADMIN_SESSION_TOKEN` | `local-admin-session` | 数据管理员本地会话令牌 |
| `REVIEWER_SESSION_TOKEN` | `local-reviewer-session` | 策略复核人本地会话令牌 |
| `AUDITOR_SESSION_TOKEN` | `local-auditor-session` | 审计员本地会话令牌 |
| `EXECUTOR_SESSION_TOKEN` | `local-executor-session` | 最小权限执行账户本地会话令牌 |

API 通过 `Authorization: Bearer <本地会话令牌>` 验证身份，再由服务端配置映射角色。客户端提供的 actor 或 role 头不会参与授权。演示角色包括 `data_admin`、`policy_reviewer`、`auditor` 与最小权限 `masking_executor`；正式环境必须替换默认令牌。

## 迁移与演示数据

迁移使用 `CREATE ... IF NOT EXISTS` 和稳定索引名称，可重复执行。`scripts/seed.sql` 使用 `ON CONFLICT DO NOTHING`，不会覆盖已有数据。连接信息只保留秘密引用，例如 `secret/demo`；日志清洗器遮罩口令、令牌、DSN 和长数字。

## 主要状态规则

- 数据源从 `draft` 或 `paused` 进入 `ready`，更新必须携带当前版本。
- 策略版本从 `draft` 提交为 `review`，申请人不能审批自己的版本；批准后进入 `approved`。
- 预演只有在策略已批准、策略作用域覆盖源表、映射使用的策略类型均获批准、源表和目标表不同且映射无冲突时才能确认；确认后映射随预演持久化，生产执行拒绝替换为另一套映射。
- 生产批次必须引用已确认预演；同一幂等键只可重放完全相同的预演与输入快照。
- 批次状态为 `pending → running → completed`；待执行批次取消后直接进入 `cancelled`，运行中取消会传播到执行上下文。失败保留行游标，只有输入指纹未变化时可进入 `recovering`。
- 每个目标写入块使用稳定 operation ID；目标写入成功但进度持久化失败时，恢复重放同一块不会重复追加目标行。
- 取消请求采用乐观版本并向运行上下文传播；源表永不被原地写入。
- 批次启动前保存目标表行数检查点，完成时读取目标真实行数并校验差异；完成、失败或取消批次可回滚到该检查点并再次计数确认。
- 策略提交与审批决定均在一个 PostgreSQL 事务中同时更新策略和审批，任一唯一约束或版本冲突会整体回滚。
- 数据源、目录、策略、预演和批次的业务写入与审计事件使用统一事务入口。本地附件和定时事件无法参与 PostgreSQL 事务，审计失败时会执行删除/撤销补偿，避免留下不可审计副作用。

## API 示例

```bash
curl -X POST http://localhost:8080/api/v1/data-sources \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer local-admin-session' \
  -d '{"name":"本地样例库","kind":"local_sample","connection_reference":"secret/demo"}'
```

列表统一接受 `page`、`size`、`sort` 和 `filter_<白名单字段>`。错误响应包含稳定 `code`、可读 `message`、可选 `fields` 和 `request_id`。

## 测试

```bash
go test ./...
POSTGRES_TEST_URL='postgres://...' go test ./internal/repository/postgres -run TestPostgresPolicySubmissionRollsBackWhenApprovalConflicts
go test -race ./...
go vet ./...
go build ./...
cd web && npm test
cd web && npm run build
```

2026-08-21 在 Windows/amd64、Go 1.26.2（`go.mod` 为 1.25.0）和 Node.js 25.9.0 上实际验证：`go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`npm test` 与 `npm run build` 均通过；前端运行时依赖审计为 0 项漏洞。独立 PostgreSQL 16 容器通过本地 `POSTGRES_TEST_URL` 串行执行核心持久化、事务回滚和启动种子测试并通过；未配置该变量时测试会明确跳过，不会误用开发数据库。

## 容器

`benzhi.Dockerfile` 使用官方 Go 与 Node 多阶段镜像，可由当前 `amd64` 或 `arm64` builder 构建。它不内置数据库或外部服务，运行时传入本地 PostgreSQL 的 `DATABASE_URL`。
