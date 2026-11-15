<div align="center">

# GoMusic

**把歌单导出为文本，带去下一个音乐平台。**

简洁、开源的网易云音乐、QQ 音乐、汽水音乐和酷狗音乐歌单导出工具。

[在线使用](https://music.unmeta.cn) · [使用方法](#使用方法) · [自行部署](#自行部署) · [API](../docs/API.md) · [English](../README.md) · **简体中文** · [한국어](../README_KO.md) · [MIT](../LICENSE)

</div>

输入公开歌单链接或酷狗码，即可获取可编辑的歌名、歌手列表。复制结果或下载 TXT 后，可以通过 TuneMyMusic 等迁移工具导入其他音乐平台。

![GoMusic 中文界面：输入酷狗码获取 12 首歌曲，结果可编辑，左侧独立显示序号](images/gomusic-zh-desktop.png)

<details>
<summary>查看手机界面</summary>

<img src="images/gomusic-zh-mobile.png" alt="GoMusic 手机中文界面：歌单输入、导出选项和带序号的歌曲结果" width="320" />

</details>

## 功能

- **支持四个平台：**网易云音乐、QQ 音乐、汽水音乐和酷狗音乐的公开歌单。支持链接、整段分享文字，也可以只输入酷狗码，如 `40725013`。
- **可编辑导出结果：**选择「歌名 - 歌手」「歌手 - 歌名」或「仅歌名」，支持正序、倒序和保留原始歌名。
- **复制和 TXT 不带序号：**编辑框左侧的序号独立显示，随编辑、换行和滚动更新；大歌单可按每批 500 首导出。
- **明确提示完整性：**同时显示实际获取数量和平台总数，部分歌曲未获取时给出提示。
- **简洁的中英文界面：**适配电脑和手机，右上角可切换 English / 中文，GoMusic 无需登录音乐账号。
- **支持自行部署：**后端 Go + Gin，前端 Vue 3 + TypeScript + Vite。MySQL 和 Redis 可选；配置 MySQL 后，用量统计可跨重启保存。

## 使用方法

1. 打开 [GoMusic](https://music.unmeta.cn)，粘贴公开歌单链接、分享文字，或直接输入酷狗码。
2. 范例链接保留为 [`http://163cn.tv/zoIxm3`](http://163cn.tv/zoIxm3)；酷狗码可以只输入 `40725013`，不用添加「酷狗码」等文字。
3. 点击「获取歌单」，核对歌曲数量，再按需要调整格式、顺序或编辑文本。
4. 点击「复制结果」或「下载 TXT」。大歌单可选择「每批 500 首」，逐批导出。
5. 在 [TuneMyMusic](https://www.tunemymusic.com/transfer) 中选择文本或文件作为来源，导入结果，再选择 Apple Music、YouTube Music 或 Spotify 为目的地。导入 Spotify 也可以使用 [Spotlistr](https://www.spotlistr.com/)。确认匹配结果后再执行迁移。

GoMusic 导出歌名和歌手文本。歌曲匹配、写入目标账号由所选迁移工具完成，其支持平台、数量限制和收费规则可能变化。GoMusic 不提供音频下载。

网易云与 QQ 音乐之间的互转，可参考 [issue #17 的社区讨论](https://github.com/Bistutu/GoMusic/issues/17)。

## 支持的输入与限制

| 平台       | 输入方式                                       |
| ---------- | ---------------------------------------------- |
| 网易云音乐 | 官方歌单链接，包括 `163cn.tv` 短链及分享文字   |
| QQ 音乐    | 官方歌单链接及分享文字                         |
| 汽水音乐   | 官方歌单链接及分享文字                         |
| 酷狗音乐   | 官方歌单或分享链接、公开集合链接、纯数字酷狗码 |

默认每次最多获取 10,000 首歌曲。私密歌单、下架歌曲或平台访问限制可能导致结果不完整。部分酷狗分享页只提供预览；GoMusic 会尝试解析受支持的公开集合或酷狗码，分页获取歌曲，并明确提示仍未获取的部分。酷狗码可能过期，也可能指向公开歌单以外的内容。

接口契约、错误码和平台限制见 [API 文档](../docs/API.md)，开源参考与实现选择见 [REFERENCES.md](../docs/REFERENCES.md)。

## 自行部署

安装 Docker 和 Compose 后，在仓库根目录运行：

```sh
cp .env.example .env
# 共享部署前修改 .env 中的示例密码。
docker compose --profile app up -d --build --wait
```

打开 <http://localhost:8081>。前端、API、MySQL 和 Redis 会一同启动，数据库使用持久化数据卷。默认端口仅绑定本机，对外部署时通过反向代理提供 HTTPS。

也可以不配置 MySQL 和 Redis，直接运行：

```sh
cd static
npm ci
npm run build
cd ..
go run .
```

源码构建需要 Go 1.27（工具链 1.27.4）和 Node.js 22.12+。环境变量、连接已有数据库、前端开发和验证命令详见 [部署文档](../DEPLOYMENT.md)。

## 接口与开发

- `POST /api/v1/songlists`：JSON 歌单接口，`url` 字段支持链接、分享文字或纯数字酷狗码。
- `POST /songlist`：兼容旧版表单接口。
- `GET /api/v1/stats`：按日、周、月和累计查询用量，需要 MySQL 和配置好的 Bearer 令牌。匿名去重客户端通过 Cookie 识别，不等于精确的自然人数。

回归用例覆盖大歌单分页、重复或重叠页、短链解析、酷狗码与预览结果、缓存降级、统计持久化，以及浏览器复制和 TXT 内容。[部署文档的验证部分](../DEPLOYMENT.md#验证)说明了单元测试、存储集成、竞态检查和浏览器测试的运行方法；真实音乐平台检查需单独启用。

## 赞赏支持

网站免费、开源、保持简单。如果你想支持作者，可以使用微信扫描赞赏码。点击图片可查看原图。

[<img src="images/approve.png" alt="微信赞赏码，点击查看原图" width="320" />](images/approve.png)

感谢以下赞赏者。名单与金额由人工维护。

| 序号 | 赞赏者                 | 金额 / 元 |
| :--: | ---------------------- | --------: |
|  1   | 冰楠 ₿.Ξ 🧘            |       188 |
|  2   | 高胖胖                 |       101 |
|  3   | 不疯就行               |       100 |
|  4   | 魏鹏程                 |        88 |
|  5   | 什么长发及腰不如短发凉 |        87 |
|  6   | Youyo🍊                |        66 |
|  7   | 安分wa                 |        50 |
|  8   | 高小伦                 |        50 |
|  9   | 平                     |        30 |
|  10  | 匿名用户               |        30 |

## Star 历史

[![Star History Chart](https://api.star-history.com/svg?repos=Bistutu/GoMusic&type=Date)](https://star-history.com/#Bistutu/GoMusic&Date)

## 许可证与致谢

项目使用 [MIT 许可证](../LICENSE)。平台接口研究、第三方授权说明和重构设计选择见 [REFERENCES.md](../docs/REFERENCES.md) 与 [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md)。
