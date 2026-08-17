# 评测用镜像：保留完整 Go 工具链，依赖在只读模块模式下按需下载。
FROM golang:1.25
WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN go build -mod=readonly ./...
CMD ["bash"]

# 多架构交叉构建示例（如需交付双架构镜像）：
# docker buildx build --platform linux/arm64,linux/amd64 -f benzhi.Dockerfile -t <image> .
