# GoMusic 部署

## 一键运行：前端 + API + MySQL + Redis

安装 Docker 和 Compose 后，在仓库根目录执行：

```sh
cp .env.example .env
# 共享部署前修改 .env 中的示例密码。
docker compose --profile app up -d --build --wait
```

访问 <http://localhost:8081>。应用会等待 MySQL、Redis 健康后启动，首次创建歌曲缓存及使用统计表。数据卷分别属于这个 Compose 项目。端口默认仅绑定本机；对外部署时通过反向代理提供 HTTPS。

```sh
docker compose ps
docker compose logs --tail=100 app
docker compose stop
```

`.env` 中的 `GOMUSIC_MYSQL_PORT`（13306）、`GOMUSIC_REDIS_PORT`（16379）、`GOMUSIC_PORT`（8081）可修改。Compose 专属变量使用 `GOMUSIC_` 前缀，避免与机器上其他项目的 `REDIS_PORT` 等变量冲突。`docker compose stop` 保留容器和数据。

## 轻量运行：不依赖数据库

需要 Go 1.27（仓库锁定工具链 1.27.4）、Node.js 22.12+。

```sh
cd static
npm ci
npm run build
cd ..
go run .
```

访问 <http://localhost:8081>。未配置存储地址时，直接从音乐平台获取信息，正常支持四个平台，无导入时连接数据库的副作用。

前端开发可另开终端：

```sh
cd static
npm run dev
```

访问 <http://localhost:5173>，`/api` 默认代理至 `127.0.0.1:8081`。可用 `GOMUSIC_API_PROXY` 修改开发代理目标。生产构建使用同源 API；分离部署时在 `static/.env.local` 设置 `VITE_API_BASE_URL=https://your-api.example` 后重新构建，并在后端显式配置允许的前端来源。来源须为完整的协议、域名和可选端口，不带路径或通配符。前端请求和 CORS 均支持匿名访客 Cookie；如需访客去重，前端与 API 应使用同站点地址（例如 `music.example.com` 与 `api.example.com`），并保持 HTTPS。跨站点部署仍受浏览器的 SameSite 和第三方 Cookie 策略限制。

## 在宿主机连接 Docker 数据服务

```sh
docker compose up -d --wait mysql redis
export MYSQL_DSN='go_music:gomusic_local@tcp(127.0.0.1:13306)/go_music?charset=utf8mb4&parseTime=true&loc=UTC&timeout=3s&readTimeout=3s&writeTimeout=3s'
export REDIS_ADDR=127.0.0.1:16379
export REDIS_PASSWORD=gomusic_local
go run .
```

上述密码是 `.env.example` 的本地示例，修改后同步修改导出的值。Go 进程读取环境变量，不自动读取 Compose 的 `.env`。

| 后端变量 | 默认 / 说明 |
| --- | --- |
| `GOMUSIC_ADDR` | `:8081` |
| `GOMUSIC_STATIC_DIR` | `static/dist` |
| `MYSQL_DSN` | 空，禁用 MySQL；指定但连接失败时启动报错 |
| `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB` | 地址为空禁用 Redis；数据库默认 0 |
| `GOMUSIC_REQUEST_TIMEOUT` | `90s` |
| `GOMUSIC_UPSTREAM_TIMEOUT` | `12s` |
| `GOMUSIC_MAX_SONGS` | 10000，可调低 |
| `GOMUSIC_CONCURRENCY` | 4，范围 1–16 |
| `GOMUSIC_CACHE_TTL` | `720h`，Redis 到期和 MySQL 查询新鲜度 |
| `GOMUSIC_ALLOWED_ORIGINS` | 空，同源部署；跨域时填写逗号分隔的明确来源 |
| `GOMUSIC_STATS_TOKEN` | 空时关闭统计查询接口；设置后使用 Bearer 令牌访问 `/api/v1/stats` |
| `GOMUSIC_STATS_COOKIE_SECURE` | 默认 `false`；网站使用 HTTPS 时设为 `true` |
| `GOMUSIC_DISABLE_DB` / `GOMUSIC_DISABLE_REDIS` | 兼容旧部署，设为 `1` 强制禁用对应存储 |

新缓存保存原始标题和歌手，不读取旧的格式化缓存；Redis 使用 `gomusic:v2:` 前缀。旧的 `net_easy_songs` 表及旧 Redis 键保持原样，不执行删表、重写或全库清空。新版本可以正常读取后再由维护者自行决定历史缓存的保留方式。

配置 MySQL 后，两条歌单获取接口会将按北京时间统计的请求数、成功/失败数、实际返回歌曲数和匿名去重访客写入 `gomusic_usage_daily`、`gomusic_usage_visitors`。统计跨重启保留，不受歌曲缓存过期时间影响。详情、历史日/周/月查询和计数口径见 [API 文档](docs/API.md#persistent-usage-statistics)。

## 验证

```sh
go test -race ./...
go vet ./...
cd static
npm run check
npm test
npm run build
npm run test:e2e
```

本地浏览器测试使用已安装 Chrome；CI 安装 Playwright Chromium。默认测试不请求真实音乐平台。存储集成测试通过 `GOMUSIC_INTEGRATION_MYSQL_DSN`、`GOMUSIC_INTEGRATION_REDIS_ADDR`、`GOMUSIC_INTEGRATION_REDIS_PASSWORD` 显式启用，请指向隔离测试服务。测试使用独立记录，不执行全库清空。

真实平台冒烟测试：`GOMUSIC_LIVE_TEST=1 go test -v -run '^TestLiveProviders$' ./logic`。平台歌单的公开权限可能随时改变。

运行中的完整页面检查：在 `static/` 执行 `GOMUSIC_WEB_URL=http://localhost:8081 node scripts/inspect-ui.mjs`，会实际请求公开歌单、核对剪贴板，并保存桌面/手机、中文/英文截图。可设置 `GOMUSIC_UI_PLAYLIST_URL` 和 `GOMUSIC_UI_EVIDENCE_DIR` 更换歌单和截图目录。

接口契约、错误码和平台限制见 [API 文档](docs/API.md)。
