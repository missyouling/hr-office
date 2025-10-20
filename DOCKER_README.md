# 社保整合系统 Docker 部署指南

## 快速开始

### 1. 一键部署
```bash
./deploy.sh
```

### 2. 手动部署
```bash
# 构建并启动服务
docker compose up --build -d

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f
```

## 访问地址

- **前端界面**: http://localhost:8080
- **后端API**: http://localhost:8081/api

## 服务管理

### 常用命令
```bash
# 查看运行状态
docker compose ps

# 查看实时日志
docker compose logs -f

# 重启服务
docker compose restart

# 停止服务
docker compose down

# 停止并删除数据
docker compose down -v
```

### 单独操作
```bash
# 重启前端
docker compose restart frontend

# 重启后端
docker compose restart backend

# 查看后端日志
docker compose logs backend

# 查看前端日志
docker compose logs frontend
```

## 端口配置

- 前端：主机端口 8080 → 容器端口 8080
- 后端：主机端口 8081 → 容器端口 8080

## 数据持久化

- 后端数据存储在Docker卷 `backend_data` 中
- SQLite数据库文件位置：容器内 `/root/data/siapp.db`

## 环境变量

### 后端
- `SIAPP_ADDR`: HTTP服务地址 (默认: :8080)
- `SIAPP_DATABASE_PATH`: SQLite数据库路径 (默认: ./data/siapp.db)

### 前端
- `NODE_ENV`: 运行环境 (production)
- `PORT`: HTTP服务端口 (8080)
- `NEXT_PUBLIC_API_BASE_URL`: 后端API地址

## 故障排除

### 查看详细日志
```bash
docker compose logs --tail=50 backend
docker compose logs --tail=50 frontend
```

### 检查健康状态
```bash
curl http://localhost:8081/api/periods
curl -I http://localhost:8080
```

### 重新构建镜像
```bash
docker compose down
docker compose up --build -d
```

### 清理系统
```bash
# 停止所有服务
docker compose down

# 清理未使用的镜像和缓存
docker system prune -f

# 删除所有相关容器和镜像
docker compose down --rmi all -v
```

## 开发模式

如需切换回开发模式：
```bash
# 停止Docker服务
docker compose down

# 启动前端开发服务器
cd frontend && npm run dev

# 启动后端开发服务器
cd backend && go run .
```

## 架构说明

- **前端**: Next.js 15 + React 19 + TypeScript + Tailwind CSS
- **后端**: Go + Chi Router + GORM + SQLite
- **容器**: Alpine Linux 基础镜像
- **网络**: Docker bridge 网络，服务间通信
- **存储**: Docker 卷持久化数据库数据