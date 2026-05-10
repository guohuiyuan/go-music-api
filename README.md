# go-music-api

> ⭐ 如果这个项目正在帮你省时间，欢迎顺手点一个 Star。Star 越多，作者越能确认这个工具确实有人在用，也会更有动力优先修复失效站点、适配新站点和更新版本。

`go-music-api` 是基于 `music-lib` 的统一 HTTP API 服务，用于把 `go-music-dl` 中已经验证的多平台音乐能力开放给 Web、桌面端、机器人插件或其它后端服务调用。

项目提供两套路由：

- **标准 REST API**：`/api/v1/*`，推荐新项目接入。
- **兼容 API**：`/music/*`，保留早期平面路由，方便旧前端或旧脚本平替迁移。

## 核心特性

- **多源音乐搜索**：支持歌曲、歌单、专辑三类搜索，默认按平台并发聚合。
- **链接解析**：支持主流平台歌曲、歌单、专辑分享链接解析。
- **音频流代理**：统一代理播放和下载，处理常见防盗链请求头；Soda/汽水音乐支持加密音频后端解密。
- **歌词与封面**：支持歌词 JSON、纯文本歌词、`.lrc` 文件下载和封面代理下载。
- **音频探测**：通过 Range 请求探测资源可用性、文件大小和估算码率。
- **智能换源**：基于歌名、歌手和时长匹配可播放的替代音源。
- **扫码登录**：支持网易云、QQ、QQ 音乐微信扫码、酷狗、Bilibili，成功后自动写入 `cookies.json`。
- **歌单能力**：支持推荐歌单、分类标签、分类歌单、个人歌单和歌单详情。
- **QQ 微信账号适配**：支持 `qq_wx` 扫码登录；QQ 个人目录歌单、我喜欢、收藏歌单可通过统一歌单接口读取。

## 支持平台

| 平台            | Source       | 歌曲 | 歌词 | 歌单 | 专辑 | 推荐歌单 | 分类歌单 | 个人歌单 | 扫码登录 | 备注                             |
| :-------------- | :----------- | :--: | :--: | :--: | :--: | :------: | :------: | :------: | :------: | :------------------------------- |
| 网易云音乐      | `netease`  |  ✅  |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ✅    |    ✅    |                                  |
| QQ 音乐         | `qq`       |  ✅  |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ✅    |    ✅    | 支持 QQ 扫码与微信扫码           |
| QQ 音乐微信扫码 | `qq_wx`    |  -  |  -  |  -  |  -  |    -    |    -    |    -    |    ✅    | 登录成功写入 `qq` Cookie       |
| 酷狗音乐        | `kugou`    |  ✅  |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    ✅    |    ✅    | App/Lite Cookie 可用于部分高音质 |
| 酷我音乐        | `kuwo`     |  ✅  |  ✅  |  ✅  |  ✅  |    ✅    |    ✅    |    -    |    -    |                                  |
| 咪咕音乐        | `migu`     |  ✅  |  ✅  |  ✅  |  ✅  |    -    |    ✅    |    -    |    -    |                                  |
| 千千音乐        | `qianqian` |  ✅  |  ✅  |  ✅  |  ✅  |    -    |    ✅    |    -    |    -    |                                  |
| 汽水音乐        | `soda`     |  ✅  |  ✅  |  ✅  |  ✅  |    -    |    -    |    -    |    -    | 支持加密音频解密                 |
| 5sing           | `fivesing` |  ✅  |  ✅  |  ✅  |  -  |    -    |    -    |    -    |    -    |                                  |
| Jamendo         | `jamendo`  |  ✅  |  ✅  |  ✅  |  ✅  |    -    |    -    |    -    |    -    | CC 音乐                          |
| JOOX            | `joox`     |  ✅  |  ✅  |  ✅  |  ✅  |    -    |    ✅    |    -    |    -    |                                  |
| Bilibili        | `bilibili` |  ✅  |  ✅  |  ✅  |  -  |    -    |    -    |    -    |    ✅    | 音频来自视频资源                 |

> 说明：表格表示 API 层已接入对应 `music-lib` 能力；实际资源是否可播放、是否有歌词或是否需要 Cookie，取决于平台策略和具体资源。

## 快速开始

### 本地运行

要求 Go 版本：**Go 1.25+**。

```bash
go mod tidy
go run main.go
```

服务默认监听：

```text
http://localhost:8080
```

### 构建

```bash
go build -o go-music-api .
```

### Docker

```bash
docker build -t guohuiyuan/go-music-api:latest .
docker run -p 8080:8080 -v $(pwd)/cookies.json:/home/appuser/cookies.json guohuiyuan/go-music-api:latest
```

也可以使用：

```bash
docker-compose up -d
```

## Swagger 文档

服务启动后访问：

```text
http://localhost:8080/swagger/index.html
```

如果修改了 `handler` 注释，需要重新生成 Swagger：

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init --parseDependency --parseInternal
```

## REST API 概览

### System

| 方法     | 路径                                        | 说明                                    |
| :------- | :------------------------------------------ | :-------------------------------------- |
| `GET`  | `/api/v1/system/cookies`                  | 获取当前 Cookie 配置                    |
| `POST` | `/api/v1/system/cookies`                  | 热更新并保存 Cookie                     |
| `GET`  | `/api/v1/system/qr_login/sources`         | 获取支持扫码登录的平台                  |
| `POST` | `/api/v1/system/qr_login/:source`         | 创建扫码登录会话                        |
| `GET`  | `/api/v1/system/qr_login/:source?key=...` | 轮询扫码登录状态，成功后自动保存 Cookie |

扫码登录 `source` 支持：

```text
netease, qq, qq_wx, kugou, bilibili
```

### Music

| 方法    | 路径                                         | 说明                       |
| :------ | :------------------------------------------- | :------------------------- |
| `GET` | `/api/v1/music/search?q=...&type=song`     | 搜索歌曲                   |
| `GET` | `/api/v1/music/search?q=...&type=playlist` | 搜索歌单                   |
| `GET` | `/api/v1/music/search?q=...&type=album`    | 搜索专辑                   |
| `GET` | `/api/v1/music/url`                        | 获取音频裸直链             |
| `GET` | `/api/v1/music/stream`                     | 代理音频流/下载音频        |
| `GET` | `/api/v1/music/inspect`                    | 探测音频可用性、大小、码率 |
| `GET` | `/api/v1/music/switch`                     | 智能切换可用音源           |
| `GET` | `/api/v1/music/lyric`                      | 获取 JSON 格式歌词         |
| `GET` | `/api/v1/music/lyric/file`                 | 下载 `.lrc` 歌词文件     |
| `GET` | `/api/v1/music/cover`                      | 代理下载封面图             |

`/api/v1/music/search` 的 `q` 可以是关键词，也可以是平台分享链接。链接解析会自动识别歌曲、歌单或专辑。

### Playlist

| 方法    | 路径                                                         | 说明                 |
| :------ | :----------------------------------------------------------- | :------------------- |
| `GET` | `/api/v1/playlist/detail?source=qq&id=...`                 | 获取歌单歌曲         |
| `GET` | `/api/v1/playlist/recommend`                               | 获取推荐歌单         |
| `GET` | `/api/v1/playlist/categories`                              | 获取歌单分类         |
| `GET` | `/api/v1/playlist/category?source=netease&category_id=...` | 获取分类下歌单       |
| `GET` | `/api/v1/playlist/user?source=qq&page=1&limit=30`          | 获取登录账号个人歌单 |

QQ 个人歌单支持特殊 ID：

- `profile:favorites`：我喜欢。
- `profile:dir:<dirid>`：QQ/微信账号个人目录歌单。

这些 ID 可直接传给 `/api/v1/playlist/detail`。

### Album

| 方法    | 路径                                           | 说明         |
| :------ | :--------------------------------------------- | :----------- |
| `GET` | `/api/v1/album/detail?source=netease&id=...` | 获取专辑歌曲 |

## 兼容 API

兼容路由位于 `/music/*`，主要用于旧版前端或脚本：

| 路径                           | 对应能力          |
| :----------------------------- | :---------------- |
| `/music/search`              | 综合搜索/链接解析 |
| `/music/download`            | 音频代理下载      |
| `/music/download_lrc`        | 歌词文件下载      |
| `/music/download_cover`      | 封面下载          |
| `/music/lyric`               | 纯文本歌词        |
| `/music/inspect`             | 音频探测          |
| `/music/switch_source`       | 智能换源          |
| `/music/playlist`            | 歌单详情          |
| `/music/album`               | 专辑详情          |
| `/music/recommend`           | 推荐歌单          |
| `/music/playlist_categories` | 歌单分类          |
| `/music/category_playlists`  | 分类歌单          |
| `/music/user_playlists`      | 个人歌单          |
| `/music/qr_login/:source`    | 扫码登录创建/轮询 |

## Cookie 配置

部分平台资源、VIP 音质、个人歌单或扫码登录能力需要 Cookie。服务启动时会读取项目根目录的 `cookies.json`。

示例：

```json
{
  "netease": "MUSIC_U=xxx; __csrf=yyy;",
  "qq": "qm_keyst=xxx; uin=yyy;",
  "kugou": "token=xxx; userid=yyy;",
  "bilibili": "SESSDATA=xxx;",
  "soda": "sessionid=xxx;"
}
```

也可以通过接口热更新：

```bash
curl -X POST http://localhost:8080/api/v1/system/cookies \
  -H "Content-Type: application/json" \
  -d '{"qq":"qm_keyst=xxx; uin=yyy;"}'
```

扫码登录成功时，服务会自动把返回的 Cookie 写入 `cookies.json`；`qq_wx` 会写入 `qq`。

## 常用示例

### 搜索歌曲

```bash
curl "http://localhost:8080/api/v1/music/search?q=稻香&type=song"
```

### 搜索专辑

```bash
curl "http://localhost:8080/api/v1/music/search?q=我很忙&type=album&sources=netease&sources=qq"
```

### 获取 QQ 微信扫码登录二维码

```bash
curl -X POST "http://localhost:8080/api/v1/system/qr_login/qq_wx"
```

### 轮询扫码登录状态

```bash
curl "http://localhost:8080/api/v1/system/qr_login/qq_wx?key=返回的key"
```

### 获取 QQ 个人歌单

```bash
curl "http://localhost:8080/api/v1/playlist/user?source=qq&page=1&limit=30"
```

### 获取歌单详情

```bash
curl "http://localhost:8080/api/v1/playlist/detail?source=qq&id=profile:favorites"
```

## 开发说明

- 本仓库通过 `go.work` 可直接使用本地 `music-lib`，便于和 `go-music-dl` 同步开发验证。
- 修改 handler 注释后，请重新运行 `swag init --parseDependency --parseInternal` 更新 Swagger 生成文件。
- 新增平台能力时，优先在 `service/factory.go` 同步 source 列表、工厂函数和 README 支持矩阵。

## 许可证

本项目遵循开源协议，详情请参见仓库根目录的 LICENSE 文件。

## Star History

[![Star History Chart](https://api.star-history.com/image?repos=guohuiyuan/go-music-api&type=date&legend=top-left)](https://www.star-history.com/?repos=guohuiyuan%2Fgo-music-api&type=date&legend=top-left)
