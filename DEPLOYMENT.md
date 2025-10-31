# 生产环境部署指南

## 🚀 快速部署

### 1. 环境准备

确保服务器安装了以下软件：
```bash
# Docker 和 Docker Compose
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 重新登录或运行
newgrp docker
```

### 2. 代码部署

```bash
# 克隆代码仓库
git clone https://github.com/missyouling/hr-office.git
cd hr-office

# 创建生产环境配置
cp .env.production.example .env.production
```

### 3. 安全配置

**⚠️ 重要：生产环境必须修改以下配置**

编辑 `.env.production`：

```bash
# 1. 生成新的JWT密钥
openssl rand -base64 64

# 2. 修改JWT配置
JWT_SECRET_KEY=你的强随机密钥
JWT_TOKEN_DURATION=12h

# 3. 配置CORS域名
ALLOWED_ORIGINS=https://your-domain.com,https://www.your-domain.com

# 4. 配置邮件服务 (可选)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

### 4. 启动服务

```bash
# 生产环境部署
docker compose -f docker-compose.production.yml up -d

# 检查服务状态
docker compose -f docker-compose.production.yml ps
```

## 🔧 高级配置

### SSL/HTTPS配置

## ✅ 本地构建与测试记录

| 日期(UTC+8) | 命令 | 输出摘要 | 状态 | 备注 |
|-------------|------|----------|------|------|
| 2025-10-31  | `GOROOT=/path/to/go1.24.5 PATH=$GOROOT/bin:$PATH CGO_ENABLED=1 go test ./...` | 所有包成功编译，无测试文件 (`? ... [no test files]`) | ✅ 成功 | 运行前需提供 Go ≥1.24.5 工具链 |
| 2025-10-31  | `npm ci && npm run build` | Turbopack 构建完成，伴随 ESLint 警告（未使用变量、缺失依赖） | ✅ 成功 | `next.config.js` 中 `allowedDevOrigins` 被标记为无效实验配置 |
| 2025-10-31  | `npm run lint` | 通过，存在 26 条 warning（@typescript-eslint/no-unused-vars、react-hooks/exhaustive-deps 等） | ⚠️ 警告 | 需逐步清理无用变量和缺失依赖 |

> 命令已在具备外网访问权限的环境中验证；复现时请准备相同的 Go 与 Node.js 版本，并关注输出中的 ESLint 提示。

## 🛠️ CI/CD 环境准备

- **Go 工具链**：在流水线启动阶段安装 Go 1.24.5，示例（Linux）：
  ```bash
  curl -fsSL https://go.dev/dl/go1.24.5.linux-amd64.tar.gz -o /tmp/go.tar.gz
  sudo tar -C /usr/local -xzf /tmp/go.tar.gz
  export GOROOT=/usr/local/go
  export PATH="$GOROOT/bin:$PATH"
  go version  # should output go1.24.5
  ```
- **CI 示例（GitHub Actions）**：
  ```yaml
  - name: Setup Go 1.24.5
    uses: actions/setup-go@v5
    with:
      go-version: '1.24.5'
  - name: Backend Tests
    run: CGO_ENABLED=1 go test ./...
  - name: Frontend Lint & Build
    run: |
      npm ci
      npm run lint
      npm run build
    working-directory: frontend
  ```
- **Docker 构建**：使用流水线命令 `docker build -f backend/Dockerfile -t hr-office-backend:latest backend` 与 `docker build -f frontend/Dockerfile -t hr-office-frontend:latest frontend`，确保环境变量与生产配置一致。

1. **获取SSL证书**：
```bash
# 使用 Let's Encrypt (推荐)
sudo apt install certbot
sudo certbot certonly --standalone -d your-domain.com
```

2. **配置Nginx**：
```bash
# 创建nginx配置目录
mkdir -p nginx/ssl

# 复制SSL证书
sudo cp /etc/letsencrypt/live/your-domain.com/fullchain.pem nginx/ssl/
sudo cp /etc/letsencrypt/live/your-domain.com/privkey.pem nginx/ssl/
```

3. **创建Nginx配置**：
```bash
# 编辑 nginx/nginx.conf
```

### 数据库备份

```bash
# 创建备份脚本
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
docker exec siapp-backend-prod sqlite3 /app/data/siapp.db ".backup /app/data/backup_$DATE.db"

# 设置定时备份
crontab -e
# 添加：0 2 * * * /path/to/backup.sh
```

### 监控配置

```bash
# 健康检查
curl http://localhost:8081/health

# 查看日志
docker compose -f docker-compose.production.yml logs -f backend

# 监控资源使用
docker stats siapp-backend-prod siapp-frontend-prod
```

## 🔒 安全检查清单

### 必须配置项

- [ ] **JWT密钥**: 已修改为强随机密钥
- [ ] **CORS域名**: 仅允许生产域名
- [ ] **HTTPS**: 配置SSL证书
- [ ] **防火墙**: 仅开放必要端口(80, 443)
- [ ] **用户权限**: 容器以非root用户运行

### 推荐配置项

- [ ] **反向代理**: 使用Nginx
- [ ] **速率限制**: 防止API滥用
- [ ] **日志轮转**: 避免磁盘满
- [ ] **自动备份**: 定期备份数据库
- [ ] **监控告警**: 系统状态监控

## 🚨 故障排除

### 常见问题

1. **容器启动失败**：
```bash
# 查看详细错误日志
docker compose -f docker-compose.production.yml logs backend

# 检查配置文件
docker compose -f docker-compose.production.yml config
```

2. **JWT认证失败**：
```bash
# 检查JWT密钥配置
docker exec siapp-backend-prod env | grep JWT
```

3. **CORS错误**：
```bash
# 检查允许的域名
docker exec siapp-backend-prod env | grep ALLOWED_ORIGINS
```

4. **数据库访问问题**：
```bash
# 检查数据库文件权限
docker exec siapp-backend-prod ls -la /app/data/
```

### 性能优化

```bash
# 调整容器资源限制
# 编辑 docker-compose.production.yml 中的 deploy.resources

# 启用gzip压缩
# 在nginx配置中添加压缩设置

# 数据库优化
# 考虑升级到PostgreSQL (参见数据库升级文档)
```

## 📝 维护操作

### 更新部署

```bash
# 1. 备份数据
./backup.sh

# 2. 拉取最新代码
git pull origin master

# 3. 重新构建和部署
docker compose -f docker-compose.production.yml build --no-cache
docker compose -f docker-compose.production.yml up -d

# 4. 验证服务
curl -f http://localhost:8081/health
```

### 日志管理

```bash
# 查看实时日志
docker compose -f docker-compose.production.yml logs -f --tail=100

# 清理旧日志
docker system prune -f
```

## 📞 支持

如果遇到问题，请：

1. 查看日志文件
2. 检查配置是否正确
3. 参考故障排除章节
4. 在GitHub仓库提交Issue

---

**⚠️ 重要提醒**：
- 生产环境部署前请仔细检查所有配置
- 定期备份重要数据
- 及时更新系统和依赖包
- 监控系统运行状态
