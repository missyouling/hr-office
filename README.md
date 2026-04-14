# 人事行政管理系统 (hr-office)

> 一体化的人事花名册与社保账期处理平台，提供账期管理、补退数据处理、报表导出与操作审计等能力。后端采用 Go + PostgreSQL，前端基于 Next.js，支持容器化部署及企业级运维。

---

## 功能亮点
- **社保账期全流程**：账期创建、险种文件上传、补退数据开关（默认关闭，按期手动启用）、自动解析与汇总、扣款明细导出。
- **人事花名册管理**：在职/离职员工视图、字段自定义、滚动表格、批量导入与导出。
- **安全体系完善**：注册邮箱验证、找回密码（限流 + 48 小时有效链接）、JWT 鉴权、危险操作二次确认、审计日志、独立密码重置页面。
- **运维可观测**：健康检查、系统指标、数据库状态页、操作日志查询。
- **容器化交付**：提供完整 Docker Compose 编排，默认集成本地 PostgreSQL，也可切换至 Supabase。

---

## 技术架构与目录

| 层级 | 技术栈 | 说明 |
|------|--------|------|
| 前端 | Next.js 15 · React 19 · TypeScript · Tailwind CSS · shadcn/ui | 采用 App Router，支持服务端渲染与动态环境变量解析 |
| 后端 | Go 1.24 · Chi · GORM · JWT · Excelize | 多数据库适配、补退数据处理、审计/监控 |
| 数据库 | PostgreSQL（默认生产） / SQLite（单机） / Supabase（云端） | 可按环境切换 |
| 运维 | Docker & Compose · Nginx 反向代理 · Sonner 通知 | 支持本地/云端一键部署 |

```
├── backend/                     # Go 后端代码
│   ├── internal/
│   │   ├── api/                # HTTP 接口
│   │   ├── middleware/         # JWT、审计等中间件
│   │   ├── models/             # 数据模型
│   │   ├── service/            # 业务服务
│   │   └── supabase/           # Supabase 集成
│   └── main.go                 # 程序入口，自动迁移与系统初始化
├── frontend/                    # Next.js 前端
│   ├── app/                    # 页面（App Router）
│   ├── components/             # UI 组件
│   └── lib/                    # API 客户端、Auth 封装
├── docs/                       # 文档与迁移说明
├── docker-compose.yml          # 开发环境编排（前后端）
├── docker-compose.production.yml # 生产编排（含 Postgres、Nginx）
├── .env.production.example     # 生产环境变量样例
└── supabase.txt                # Supabase 连接信息备忘
```

---

## 环境准备

| 工具 | 版本建议 | 说明 |
|------|----------|------|
| Go  | 1.24.0+   | 后端开发与测试 |
| Node.js | 20 LTS | 前端开发与构建 |
| PostgreSQL | 15+ | 默认生产数据库，可用 Docker |
| Docker | 24+ | 推荐用于部署与本地一键环境 |
| npm | 10+ | 前端依赖管理 |

> ⚠️ 生产环境推荐使用自建或托管的 PostgreSQL 服务（如 Supabase、RDS）。具体配置见下文“Supabase 在线服务接入”与“本地 PostgreSQL 初始化”。

### 环境变量速览

所有容器默认读取根目录 `.env`（或 `.env.production`），前端还会在 `frontend/docker-entrypoint.sh` 中根据这些变量生成 `public/runtime-config.js`，在浏览器运行时动态解析 API 地址。常见变量如下：

| 分类 | 变量 | 说明 | 示例 |
|------|------|------|------|
| 服务 | `SIAPP_ADDR` | 后端监听地址 | `:8080` |
| 安全 | `JWT_SECRET_KEY` / `JWT_TOKEN_DURATION` | JWT 签名密钥与有效期 | `change-me` / `12h` |
| CORS | `ALLOWED_ORIGINS` | 允许访问的前端源（逗号分隔） | `https://your-domain.com` |
| 数据库 | `SIAPP_DATABASE_TYPE` | `postgres` / `sqlite` | `postgres` |
| 数据库 | `SIAPP_DB_HOST` / `PORT` / `USER` / `PASSWORD` / `NAME` | PostgreSQL 连接信息 | `aws-1-xxx.pooler.supabase.com` |
| 数据库 | `SIAPP_DB_SSLMODE` | SSL 模式（Supabase 必须 `require`） | `require` |
| 数据库 | `DATABASE_URL` | 供外部工具使用的 DSN，建议追加 `?sslmode=require` | `postgresql://...` |
| 前端 API | `NEXT_PUBLIC_API_BASE_URL` | 默认域名 API 地址（含 `/api`） | `https://your-domain.com/api` |
| 前端 API | `NEXT_PUBLIC_API_BASE_URL_DOMAIN` | 域名兜底地址 | 同上 |
| 前端 API | `NEXT_PUBLIC_API_BASE_URL_IP` | 暴露 IP + 端口时的 API 地址 | `http://8.211.x.x:10101/api` |
| 前端 API | `NEXT_PUBLIC_API_IPV4_FALLBACK_PORT` | 裸 IP 访问时拼接端口 | `10101` |
| 前端 API | `INTERNAL_API_BASE_URL` | 容器内部访问后端使用 | `http://backend:8080/api` |
| Supabase | `NEXT_PUBLIC_SUPABASE_URL` / `NEXT_PUBLIC_SUPABASE_ANON_KEY` | 浏览器访问 Supabase 所需 | `https://xxx.supabase.co` |
| Supabase | `SUPABASE_URL` / `SUPABASE_SERVICE_ROLE_KEY` | 后端调用 Supabase 模块所需 | 同上 |
| Supabase | `SUPABASE_DB_PASSWORD` | Supabase 数据库密码（Dashboard 设置） | `secure-pass` |
| Supabase | `SUPABASE_JWT_SECRET` | Supabase Auth JWT Secret | `super-secret` |
| 邮件 | `SMTP_*` / `FROM_NAME` / `BASE_URL` | SMTP 发信配置 | 视服务商而定 |
| 上传 & 监控 | `MAX_UPLOAD_SIZE` / `ENABLE_METRICS` / `METRICS_PORT` | 上传与监控相关参数 | `10MB` / `true` / `9090` |

> **运行时 API 解析提示**：`frontend/lib/api.ts` 会优先读取 `window.__RUNTIME_CONFIG__` 中的值（由入口脚本注入），若未设置则回退至 `NEXT_PUBLIC_API_BASE_URL*`，确保“内网直连 / 内网穿透 / 域名反代”三种访问方式都能命中正确的后端地址。

---

## 本地开发（默认连接本地 PostgreSQL）

1. **克隆仓库**
   ```bash
   git clone https://github.com/missyouling/hr-office.git
   cd hr-office
   ```

2. **准备 PostgreSQL（二选一）**
   - 使用容器快速启动：
     ```bash
     docker run -d \
       --name siapp-postgres-dev \
       -e POSTGRES_USER=siapp \
       -e POSTGRES_PASSWORD=siapp_dev_pass \
       -e POSTGRES_DB=siapp \
       -p 5432:5432 \
       postgres:15-alpine
     ```
   - 或使用已有 PostgreSQL，提前创建数据库和用户：
     ```sql
     CREATE DATABASE siapp;
     CREATE USER siapp WITH ENCRYPTED PASSWORD 'siapp_dev_pass';
     GRANT ALL PRIVILEGES ON DATABASE siapp TO siapp;
     ```

3. **配置根目录 `.env`（供 docker-compose 与后端读取）**
   ```dotenv
# 服务端 & 安全
SIAPP_ADDR=:8080
APP_ENV=development
JWT_SECRET_KEY=dev-secret-change-it
JWT_TOKEN_DURATION=12h
ALLOWED_ORIGINS=http://localhost:3000

# 数据库（本地 Postgres 默认）
SIAPP_DATABASE_TYPE=postgres
SIAPP_DB_HOST=localhost
SIAPP_DB_PORT=5432
SIAPP_DB_USER=siapp
SIAPP_DB_PASSWORD=siapp_dev_pass
SIAPP_DB_NAME=siapp
SIAPP_DB_SSLMODE=disable
DATABASE_URL=postgresql://siapp:siapp_dev_pass@localhost:5432/siapp?sslmode=disable

# Supabase（暂未接入可保留占位）
NEXT_PUBLIC_SUPABASE_URL=http://localhost:54321
NEXT_PUBLIC_SUPABASE_ANON_KEY=public-anon-key-placeholder
SUPABASE_URL=http://localhost:54321
SUPABASE_SERVICE_ROLE_KEY=service-role-placeholder
SUPABASE_DB_PASSWORD=siapp_dev_pass
SUPABASE_JWT_SECRET=dev-jwt-placeholder

# 前端 API 地址（容器入口脚本会据此生成 runtime-config）
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api
NEXT_PUBLIC_API_BASE_URL_DOMAIN=http://localhost:8080/api
NEXT_PUBLIC_API_BASE_URL_IP=http://127.0.0.1:8080/api
NEXT_PUBLIC_API_IPV4_FALLBACK_PORT=8080
INTERNAL_API_BASE_URL=http://backend:8080/api

# 邮件（本地若无 SMTP 可暂留空）
SMTP_HOST=
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=
FROM_NAME=人事行政管理系统
BASE_URL=http://localhost:3000
   ```

4. **配置前端环境变量 `frontend/.env.local`**
   ```dotenv
   NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api
   NEXT_PUBLIC_API_BASE_URL_DOMAIN=http://localhost:8080/api
   NEXT_PUBLIC_API_BASE_URL_IP=http://127.0.0.1:8080/api
   NEXT_PUBLIC_API_IPV4_FALLBACK_PORT=8080
   NEXT_PUBLIC_SUPABASE_URL=http://localhost:54321
   NEXT_PUBLIC_SUPABASE_ANON_KEY=public-anon-key-placeholder
   INTERNAL_API_BASE_URL=http://localhost:8080/api
   ```

5. **启动后端**
   ```bash
   cd backend
   CGO_ENABLED=1 go run .
   # 默认监听 http://localhost:8080，首次启动会自动迁移数据库并初始化 admin/admin123
   ```

6. **启动前端**
   ```bash
   cd frontend
   npm install
   npm run dev
   # http://localhost:3000
   ```

7. **验证**
   - 浏览器访问 `http://localhost:3000/auth` 使用 `admin/admin123` 登录。
   - 检查登录页“找回密码”弹窗是否弹出并提示邮件发送。
   - 使用邮件中的重置链接访问 `http://localhost:3000/reset-password?token=...`，确认可完成密码校验与密码重置。

### Supabase 在线服务接入

当需要将数据库与认证托管在 Supabase 时，按以下步骤配置环境变量：

1. **创建项目**：登录 [Supabase](https://supabase.com/)，创建新 Project，记下 `Project ref`（后续用于数据库用户名）。
2. **获取数据库连接信息**：Dashboard → *Database* → *Connection pooling*，复制 `pooler` 地址与数据库密码，并填入：
   ```dotenv
   SIAPP_DATABASE_TYPE=postgres
   SIAPP_DB_HOST=aws-1-xxxxx.pooler.supabase.com
   SIAPP_DB_PORT=5432
   SIAPP_DB_USER=postgres.your-project-ref
   SIAPP_DB_PASSWORD=your-db-password
   SIAPP_DB_NAME=postgres
   SIAPP_DB_SSLMODE=require
   DATABASE_URL=postgresql://postgres.your-project-ref:your-db-password@aws-1-xxxxx.pooler.supabase.com:5432/postgres?sslmode=require
   SUPABASE_DB_PASSWORD=your-db-password
   ```
3. **获取 API Key**：Dashboard → *Project Settings* → *API*
   ```dotenv
   NEXT_PUBLIC_SUPABASE_URL=https://your-project-ref.supabase.co
   NEXT_PUBLIC_SUPABASE_ANON_KEY=anon-public-key
   SUPABASE_URL=https://your-project-ref.supabase.co
   SUPABASE_SERVICE_ROLE_KEY=service-role-key
   ```
4. **同步 JWT Secret**：Dashboard → *Authentication* → *Settings* → *JWT Secret*，确保与 `.env` 中的 `SUPABASE_JWT_SECRET` 一致。
5. **调整 API 地址**：根据部署方式设置 `NEXT_PUBLIC_API_BASE_URL*`（域名/IP 均要包含 `/api`），并在有内网穿透时同步更新 `NEXT_PUBLIC_API_IPV4_FALLBACK_PORT`。

保存后重启后端，日志若出现 `Connecting to PostgreSQL database: host=...supabase.com` 表示成功接入 Supabase。

### 本地 PostgreSQL 初始化（裸机）

若在物理机部署 PostgreSQL，可执行：

```bash
sudo apt install postgresql postgresql-contrib
sudo -u postgres psql
```

在 psql 中运行：

```sql
CREATE DATABASE siapp;
CREATE USER siapp WITH ENCRYPTED PASSWORD 'strong-siapp-pass';
GRANT ALL PRIVILEGES ON DATABASE siapp TO siapp;
ALTER ROLE siapp SET client_encoding TO 'UTF8';
ALTER ROLE siapp SET default_transaction_isolation TO 'read committed';
ALTER ROLE siapp SET timezone TO 'Asia/Shanghai';
```

完成后在 `pg_hba.conf` 中启用 `scram-sha-256` 并开放需要的内网段，即可与后端服务连通。

---

## Docker Compose 编排

仓库提供三套编排文件，可依据部署场景选择：

| 文件 | 场景 | 特性 |
|------|------|------|
| `docker-compose.yml` | 开发 / 测试 | 前端映射 `10100`，后端映射 `10011`，读取根目录 `.env`，兼容裸 IP 与内网穿透访问 |
| `docker-compose.production.yml` | 生产一体化 | 包含 Nginx + Frontend + Backend + Postgres，适合单机或小规模部署 |
| `docker-compose.postgres.yml` | 生产（外部数据库） | 仅前后端容器，连接 Supabase 或自建 PostgreSQL 集群 |

使用示例：

```bash
# 准备配置模板
cp .env.production.example .env.production
# 按照前文环境变量指引填充域名、数据库、Supabase 等信息

# 启动（生产示例）
docker compose -f docker-compose.production.yml up -d

# 查看服务状态
docker compose -f docker-compose.production.yml ps
```

> 提示：当同时开放域名与 IP/穿透入口时，请务必设置 `NEXT_PUBLIC_API_BASE_URL_IP` 与 `NEXT_PUBLIC_API_IPV4_FALLBACK_PORT`，前端容器会在运行时拼接正确的后端地址。

---


## 生产环境部署（默认使用本地 PostgreSQL）

1. **服务器准备**
   - Ubuntu 22.04+/CentOS 8+，安装 Docker & Docker Compose Plugin。
   - 预留目录：
     ```bash
     sudo mkdir -p /opt/hr-office/{nginx,logs} /var/lib/siapp/{postgres,data}
     sudo chown -R 1001:1001 /var/lib/siapp/data
     sudo chown -R 999:999 /var/lib/siapp/postgres
     ```

2. **获取代码与配置**
   ```bash
   git clone https://github.com/missyouling/hr-office.git /opt/hr-office
   cd /opt/hr-office
   cp .env.production.example .env.production
   ```
   根据实际域名/IP 调整 `.env.production`：
   - `NEXT_PUBLIC_API_BASE_URL=https://your-domain.com/api`
   - `ALLOWED_ORIGINS=https://your-domain.com`
   - `SIAPP_DB_PASSWORD` 替换为强口令。

3. **启动**
   ```bash
   docker compose -f docker-compose.production.yml up -d --build
   ```
   编排会同时拉起 `postgres`、`backend`、`frontend` 与 `nginx`，默认端口：
   - 前端：80/443（由 Nginx 暴露）
   - 后端：8080（内部网络访问）
   - PostgreSQL：容器内部 5432

4. **配置 HTTPS（可选）**
   - 将证书与私钥放入 `nginx/ssl/`，更新 `nginx/nginx.conf`。
   - 重新加载：`docker compose -f docker-compose.production.yml restart nginx`

### Nginx 反向代理示例

若在宿主机上单独部署 Nginx，可参考如下示例（假设域名为 `hr-office.example.com`）：

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

upstream hr_office_frontend {
    server 127.0.0.1:10100;
}

upstream hr_office_backend {
    server 127.0.0.1:10101;
}

server {
    listen 80;
    listen [::]:80;
    server_name hr-office.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name hr-office.example.com;

    ssl_certificate /etc/nginx/certs/hr-office.example.com_cert.pem;
    ssl_certificate_key /etc/nginx/certs/hr-office.example.com_key.pem;
    client_max_body_size 1000m;

    location /api/ {
        proxy_pass http://hr_office_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 120s;
    }

    location / {
        proxy_pass http://hr_office_frontend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 120s;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|ttf|woff2?)$ {
        proxy_pass http://hr_office_frontend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_cache my_proxy_cache;
        aio threads;
        access_log off;
    }
}
```

根据需要可额外添加 `listen 443 quic;` 与 `Alt-Svc` 头以开启 HTTP/3，证书路径和域名请改为实际值。

5. **部署验证**
   ```bash
   docker compose -f docker-compose.production.yml ps
   curl http://localhost:8080/health
   curl -I https://your-domain.com/auth
   ```

6. **升级流程**
   ```bash
   git pull
   docker compose -f docker-compose.production.yml build --no-cache
   docker compose -f docker-compose.production.yml up -d
   ```

---

## 环境变量速查

### 后端核心变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SIAPP_ADDR` | HTTP 监听地址 | `:8080` |
| `SIAPP_DATABASE_TYPE` | `postgres` / `sqlite` | `sqlite`（生产建议 `postgres`） |
| `SIAPP_DB_HOST` / `PORT` / `USER` / `PASSWORD` | PostgreSQL 主机、端口、用户、密码 | `aws-1-xxx.pooler.supabase.com / 5432 / postgres.xxx / strong-pass` |
| `SIAPP_DB_NAME` | 数据库库名 | `siapp`（本地） / `postgres`（Supabase） |
| `SIAPP_DB_SSLMODE` | SSL 模式 | `require`（Supabase）/`disable`（本地） |
| `DATABASE_URL` | 统一 DSN，建议附 `?sslmode=` | `postgresql://...` |
| `SUPABASE_URL` | Supabase 服务端 API | `https://xxx.supabase.co` |
| `SUPABASE_SERVICE_ROLE_KEY` | Supabase Service Role Key | Supabase Dashboard 获取 |
| `SUPABASE_DB_PASSWORD` | Supabase 数据库密码 | Dashboard → Database 设置 |
| `SUPABASE_JWT_SECRET` | Supabase JWT Secret | Dashboard → Authentication 设置 |
| `JWT_SECRET_KEY` | JWT 签名密钥 | 必填 |
| `ALLOWED_ORIGINS` | 允许的前端域名 | `https://your-domain.com` |
| `SMTP_*` / `BASE_URL` | 邮件服务器 & 邮件中的访问地址 | 生产需配置完整域名 |

### 前端核心变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `NEXT_PUBLIC_API_BASE_URL` | API 基础地址 | `https://your-domain.com/api` |
| `NEXT_PUBLIC_API_BASE_URL_DOMAIN` | 域名形式 API（容器构建用） | 同上 |
| `NEXT_PUBLIC_API_BASE_URL_IP` | IP 形式 API（可选） | `http://<ip>:10101/api` |
| `NEXT_PUBLIC_API_IPV4_FALLBACK_PORT` | 裸 IP 访问时拼接端口 | `10101` |
| `NEXT_PUBLIC_SUPABASE_URL` | Supabase URL | `https://xxxx.supabase.co` |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | Supabase 公钥 | Supabase Dashboard 获取 |
| `INTERNAL_API_BASE_URL` | SSR/容器内部访问 API 的地址 | `http://backend:8080/api` |


## 密码重置机制

- **链接有效期**：密码重置链接有效期延长至 48 小时，邮件模板已同步提示。
- **频率限制**：同一账号连续申请密码重置时需间隔 5 分钟，系统会提示“请求过于频繁”。
- **每日上限**：单账号每天最多可申请 3 次密码重置，超出后需联系管理员协助。
- **独立页面**：前端新增 `/reset-password` 页面，提供链接校验、密码复杂度校验与重置成功引导。

---

## 运维与日常操作

| 操作 | 命令 |
|------|------|
| 查看容器日志 | `docker compose -f docker-compose.production.yml logs -f backend` |
| 检查健康状态 | `curl http://localhost:8080/health` |
| 备份 PostgreSQL | `docker exec siapp-postgres-prod pg_dump -U siapp siapp > backup.sql` |
| 清理旧镜像 | `docker image prune -f` |
| 本地测试后端 | `cd backend && go test ./...` |
| 前端静态检查 | `cd frontend && npm run lint` |

默认管理员：`admin / admin123`，请安装后立即登录后台修改密码。

---

## 常见问题排查

| 症状 | 排查要点 |
|------|----------|
| 登录提示 401 | 确认后台 `ALLOWED_ORIGINS` 包含访问域名，检查管理员密码是否已修改，查看 `backend` 日志。 |
| 找回密码无邮件 | 确认 SMTP 环境变量正确、服务器对外 587 端口可用，后台日志是否出现 “SendPasswordResetEmail” 错误。 |
| 找回密码提示“请求过于频繁” | 同一账号需间隔 5 分钟再申请，系统将提示稍后再试；若当日累计 3 次仍需重置，请联系管理员处理。 |
| 重置链接提示无效/已过期 | 链接有效期为 48 小时且仅可使用一次，请以最新邮件中的链接为准，必要时重新申请。 |
| 前端 API 请求失败 | 核对 `.env` 中的 `NEXT_PUBLIC_API_BASE_URL*` 是否与实际访问协议/端口一致，必要时清理浏览器缓存。 |
| Docker 启动失败 | 查看 `docker compose logs`, 确认 `.env.production` 填写完整且未出现格式错误。 |

---

## 开发协作

1. 新建分支：`git checkout -b feature/xxx`
2. 后端运行 `go test ./...`，前端运行 `npm run lint`
3. 完成后提交：`git commit -m "feat: 描述"`
4. 推送并发起 PR，说明改动和测试情况。

---

## 许可证

本项目遵循 MIT License，详见 [LICENSE](LICENSE)。

---

如需更多运维细节，可参考：
- 《DEPLOYMENT_SUMMARY.md》：历史部署记录与操作要点
- 《SUPABASE_BACKEND_INTEGRATION.md》：Supabase 集成指南
- `.env.production.example`：完整生产配置模板

欢迎提出 Issue 或 PR，共同完善系统。
