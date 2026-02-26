# go-music-api

基于 `music-lib` 底层库构建的跨平台音乐搜索与解析统一 HTTP API 服务。本项目提供了标准化的 RESTful Web 接口，并内置精美的 Swagger 交互式接口文档，方便任何前端项目快速接入。

## ✨ 核心特性

- 🎵 **多源聚合搜索**：支持全网主流音乐平台并发搜索，自动聚合单曲与歌单。
- 🔗 **智能链接解析**：直接输入各大平台的音乐/歌单分享链接，自动解析出真实数据。
- 🔓 **全能音频流代理**：内置跨域代理与防盗链破解，**独家支持 Soda(汽水音乐) 加密音频流实时后端解密**。
- 🔀 **智能音源切换**：基于 Levenshtein 距离算法与时长精准匹配，当某平台歌曲无版权(灰掉)时，自动寻源无缝切换平替音源。
- 📖 **全量元数据获取**：提供 LRC 滚动歌词拉取、封面大图无视防盗链代理下载、音频文件大小与码率(kbps)毫秒级探测。
- 🔄 **无缝向下兼容**：保留了原 `server.go` 的旧版路由规范，前端无需修改任何代码即可零成本迁移至本服务。

## 🎧 支持的音乐平台

| 平台       | 包名         | 搜索 | 下载 | 歌词 | 歌曲解析 | 歌单搜索 | 歌单推荐 | 歌单歌曲 | 歌单链接解析 | 备注     |
| :--------- | :----------- | :--: | :--: | :--: | :------: | :------: | :------: | :------: | :----------: | :------- |
| 网易云音乐 | `netease`  |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ✅    |    ✅    |      ✅      |          |
| QQ 音乐    | `qq`       |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ✅    |    ✅    |      ✅      |          |
| 酷狗音乐   | `kugou`    |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ✅    |    ✅    |      ✅      |          |
| 酷我音乐   | `kuwo`     |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ✅    |    ✅    |      ✅      |          |
| 咪咕音乐   | `migu`     |  ✅  |  ✅  |  ✅  |    ❌    |    ✅    |    ❌    |    ❌    |      ❌      |          |
| 千千音乐   | `qianqian` |  ✅  |  ✅  |  ✅  |    ❌    |    ❌    |    ❌    |    ✅    |      ❌      |          |
| 汽水音乐   | `soda`     |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ❌    |    ✅    |      ✅      | 音频解密 |
| 5sing      | `fivesing` |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ❌    |    ✅    |      ✅      |          |
| Jamendo    | `jamendo`  |  ✅  |  ✅  |  ❌  |    ✅    |    ❌    |    ❌    |    ❌    |      ❌      |          |
| JOOX       | `joox`     |  ✅  |  ✅  |  ✅  |    ❌    |    ✅    |    ❌    |    ❌    |      ❌      |          |
| Bilibili   | `bilibili` |  ✅  |  ✅  |  ❌  |    ✅    |    ✅    |    ❌    |    ✅    |      ✅      |          |


## 🚀 快速开始

### 1. 生成 Swagger API 文档
项目使用了 `swaggo` 自动生成接口文档，运行前请先生成（如果修改了 `handler` 中的注释也需要重新执行）：

```bash
# 安装 swag 命令行工具
go install github.com/swaggo/swag/cmd/swag@latest

# 生成 docs 目录及 Swagger JSON
swag init --parseDependency --parseInternal

```

### 2. 本地运行与构建

确保环境为 **Go 1.25+**。

```bash
# 下载依赖
go mod tidy

# 直接运行 (默认端口 8080)
go run main.go

# 编译为二进制文件
go build -o go-music-api .

```

### 3. Docker 容器化运行

```bash
# 构建镜像
docker build -t guohuiyuan/go-music-api:latest .

# 运行容器 (建议挂载 cookies 文件以维持平台登录状态)
docker run -p 8080:8080 -v $(pwd)/cookies.json:/home/appuser/cookies.json guohuiyuan/go-music-api:latest

# 或者使用 docker-compose 一键启动
docker-compose up -d

```

## 📚 接口文档概览

服务成功启动后，浏览器访问以下地址即可在线测试 API：
👉 **http://localhost:8080/swagger/index.html**

### 路由架构设计

本项目提供两套路由体系，共存运行：

1. **标准化 RESTful API (`/api/v1/*`)**：推荐外部新项目对接使用，结构清晰。
* `/api/v1/system/...`：配置管理（如动态设置 Cookies）。
* `/api/v1/music/...`：单曲核心操作（搜索、流代理、探测、歌词、智能切换等）。
* `/api/v1/playlist/...`：歌单操作（获取详情、热门推荐）。


2. **兼容性 API (`/music/*`)**：完美复刻了早期 `server.go` 的平面路由结构（如 `/music/switch_source`, `/music/download` 等），只要将旧版后端平替为本服务，旧版前端网页即可直接点亮。

## ⚙️ 配置说明 (`cookies.json`)

部分平台（如网易云 VIP 歌曲、汽水音乐等）需要登录态才能获取完整的音频流或高音质数据。
你可以在项目根目录创建 `cookies.json` 文件（Docker 环境需映射进容器），程序会在启动时自动加载。

**格式示例：**

```json
{
  "netease": "MUSIC_U=xxx; __csrf=yyy;",
  "qq": "qm_keyst=xxx; uin=yyy;",
  "soda": "sessionid=xxx;"
}

```

> 💡 提示：也可以直接在服务运行期间，通过 Swagger 调用 `/api/v1/system/cookies` 接口进行热更新。

## 🛠 开发与部署建议

* **本地调试**：使用 `go run main.go` 快速拉起服务，配合 Swagger UI 即可进行全功能调试。
* **CI / 发布**：仓库内置了 GitHub Actions 工作流。当你推送 Tag 时，会自动构建 Docker 镜像 (`guohuiyuan/go-music-api`) 并使用 GoReleaser 生成多平台二进制 Release 产物（前缀为 `go-music-api_`）。

## 📄 许可证

本项目遵循开源协议，详情请参见仓库根目录的 LICENSE 文件。