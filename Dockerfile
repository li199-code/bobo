# =============================
# Stage 1: Build Frontend
# =============================
FROM node:20 as frontend-builder

WORKDIR /app

COPY frontend/package*.json ./
RUN npm install

COPY frontend/ ./
RUN npm run build

# =============================
# Stage 2: Build Backend
# =============================
FROM golang:1.23 as backend-builder

WORKDIR /app

COPY backend/go.mod .
COPY backend/go.sum .
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /blog-backend

# =============================
# Stage 3: Final Debuggable Image
# =============================
FROM alpine:3.19

# 安装一些调试工具（可选）
RUN apk add --no-cache bash curl

WORKDIR /app

# 复制前端构建产物
COPY --from=frontend-builder /app/dist /app/dist

# 复制后端可执行文件
COPY --from=backend-builder /blog-backend /app/blog-backend

# 设置可执行权限（保险起见）
RUN chmod +x /app/blog-backend

# 暴露端口
EXPOSE 8080

# 启动服务（或换成 sh 以调试）
CMD ["/app/blog-backend"]
