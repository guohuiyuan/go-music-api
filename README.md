# go-music-api

跨平台音乐搜索与解析统一 HTTP API 服务，基于 `music-lib` 构建，提供简易的 Web 接口与 Swagger 文档。

**Quick Start**
- **生成 Swagger 文档（若你修改了注释）：** 安装 `swag` 并运行 `swag init` 以生成 `docs` 目录和 Swagger JSON：
-   - 安装: `go install github.com/swaggo/swag/cmd/swag@latest`
-   - 运行: `swag init --parseDependency --parseInternal`
-
- **本地运行:** `go run main.go`
- **构建二进制:** `go build -o go-music-api .`
- **在容器中运行:**
  - 构建镜像: `docker build -t guohuiyuan/go-music-api:latest .`
  - 运行: `docker run -p 8080:8080 -v $(pwd)/cookies.json:/home/appuser/cookies.json guohuiyuan/go-music-api:latest`
- **使用 docker-compose:** `docker-compose up -d`

**默认端口**: 8080

文件与入口
- 主入口: [main.go](main.go#L1)
- 路由: [router/router.go](router/router.go)
- 服务 (加载 cookies 等): [service/factory.go](service/factory.go)

依赖
- Go 1.25+
- 依赖由 `go.mod` 管理，使用 `go mod tidy` 下载与整理依赖。

配置
- `cookies.json`：可选，放在项目根或容器挂载到容器内 `/home/appuser/cookies.json`，程序会在启动时加载（见 `service` 包）。

API 文档
- Swagger UI 暴露在: `http://localhost:8080/swagger/index.html`（启动后访问）

CI / 发布
- 仓库包含 GitHub Actions 工作流（构建与发布镜像、release）。镜像名默认：`guohuiyuan/go-music-api`。
- GoReleaser 配置位于 `.goreleaser.yml`，产物前缀 `go-music-api_`。

开发建议
- 使用 `go run main.go` 快速调试。
- 本地构建并在容器中测试以保证运行环境一致。

许可证
- 请参见仓库根目录的 LICENSE 文件。
