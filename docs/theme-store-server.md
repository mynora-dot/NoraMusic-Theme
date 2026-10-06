# 独立主题商店服务

`server/` 是一个独立的 Go 服务，用于托管 `.noratheme` 包。它只依赖 SQLite 和本地文件目录，不需要运行 NoraMusic 主后端。

## 本地运行

```bash
cd server
GONOSUMDB='*' GOPROXY=off GOSUMDB=off go test ./...   # 已有模块缓存时可离线测试
go run ./cmd/themestore
```

首次打开 `/admin/` 时创建管理员。开发环境可使用 LAN 模式：

```bash
NORA_DEPLOY_MODE=lan \
ALLOW_LAN_HTTP=true \
LISTEN_ADDR=127.0.0.1:8090 \
DATA_DIR=./data \
go run ./cmd/themestore
```

## NAS 部署

```bash
cd server/deploy
docker compose -f docker-compose.nas.yml up -d --build
```

访问 `http://NAS_IP:8090/admin/` 完成初始化。默认仍然需要管理员密码；只有明确设置 `ALLOW_INSECURE_ADMIN=true` 才会允许无认证管理。

## 公网部署

编辑 `docker-compose.public.yml` 和 `Caddyfile` 中的域名后运行：

```bash
cd server/deploy
docker compose -f docker-compose.public.yml up -d --build
```

公网模式要求 `PUBLIC_BASE_URL` 为 HTTPS，管理接口拒绝明文 HTTP。Caddy 负责证书和反向代理，管理员网段默认仅容器本机；如需通过堡垒机访问，显式设置 `ADMIN_ALLOWED_CIDRS` 和 `TRUSTED_PROXY_CIDRS`。

## 主要配置

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `NORA_DEPLOY_MODE` | `public` | `public` 或 `lan` |
| `LISTEN_ADDR` | `127.0.0.1:8090` | 监听地址 |
| `PUBLIC_BASE_URL` | 必填（公网） | 无用户信息、查询串和 fragment 的 HTTPS URL |
| `DATA_DIR` | `./data` | SQLite 和 Blob 目录 |
| `DATABASE_PATH` | `$DATA_DIR/themes.db` | SQLite 路径 |
| `MAX_THEME_MIB` | `32` | 上传限制，范围 1–128 |
| `THEME_ACCESS` | `anonymous` | 主题公开读取：`anonymous` 或 `login` |
| `OFFLINE_POLICY` | `allow` | 客户端离线策略：`allow` 或 `strict` |
| `ADMIN_ALLOWED_CIDRS` | 回环地址 | 管理入口允许的来源网段 |
| `TRUSTED_PROXY_CIDRS` | 空 | 允许读取 `X-Real-IP`/`X-Forwarded-Proto` 的代理网段 |
| `ALLOW_LAN_HTTP` | `false` | 仅 LAN 模式允许 HTTP |
| `ALLOW_INSECURE_ADMIN` | `false` | 仅 LAN 模式可显式开启无认证管理 |

## API

公开 API 为 `/api/v1`，管理 API 为 `/admin/v1`。响应统一包含 `data`、`requestId` 和 `serverTime`；错误包含 `error.code`。主题上传使用 multipart 字段 `file`：

```bash
curl -b cookies.txt -c cookies.txt \
  -F file=@dist/my-first-theme-1.0.0.noratheme \
  http://127.0.0.1:8090/admin/v1/themes/uploads
```

上传后版本状态为 `draft`。发布、撤回、恢复和删除都写入审计记录。删除默认是软删除；永久清理需要 `POST /admin/v1/themes/{id}/purge` 并提交 `{"confirm":"package-id"}`。

管理员 Web 会话使用 HttpOnly Cookie 和 CSRF Token。脚本可通过 `POST /admin/v1/auth/tokens` 创建 Bearer Token（明文只返回一次），并通过 `DELETE /admin/v1/auth/tokens/{id}` 撤销。公网模式首次登录后必须在“安全设置”流程中绑定 TOTP 验证器，完成绑定前不会开放主题管理接口。

## 备份与恢复

停止服务后同时备份 `DATA_DIR/themes.db` 和 `DATA_DIR/theme-blobs/`。恢复时保持目录权限，并确保数据库与 Blob 目录来自同一份备份。服务启动时会自动执行数据库迁移。

## 从 NoraMusic 主后端迁移

独立服务不直接读取主后端数据库。迁移时从原后端的 `theme-blobs/` 取出仍需保留的 `.noratheme` 文件，通过管理后台或 `POST /admin/v1/themes/uploads` 重新上传；重新上传会重新校验包、计算哈希并建立新的草稿版本，确认内容后再发布。
