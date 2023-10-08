## 迁移网易云歌单至-AppleMusic、YoutubeMusic、Spotify

链接：https://music.unmeta.cn/

项目使用 Golang + Gin 开发，Redis 作为缓存中间件，使用时需到 /repo/cache/redis.go 文件下配置 redis 的连接地址。

```go
rdb = redis.NewClient(&redis.Options{
	Addr:     "", // redis 服务端地址
	Password: "", // redis 密码
	DB:       0,
})
```

<img src="https://oss.thinkstu.com/typora/202310081905115.png?x-oss-process=style/optimize" alt="image-20231008190554003" style="width:80%;" />

## 使用指南

1. 输入网易云歌单链接，例如：https://music.163.com/#/playlist?id=8725919816
2. 转移到 Youtube Music or Spotify or Apple Music
   - Go to [TuneMyMusic](https://www.tunemymusic.com/zh-CN/transfer)
   - STEP 1：选择来源从「任意文本」
   - STEP 2：粘贴刚刚复制的内容到文本框中
   - STEP 3：选择 Youtube or Spotify or Apple Music作为目的地
   - STEP 4：开始移动

<img src="https://oss.thinkstu.com/typora/202310081907395.png?x-oss-process=style/optimize" alt="image-20231008190713343" style="width:80%;" />

<img src="https://oss.thinkstu.com/typora/202310081907435.png?x-oss-process=style/optimize" alt="image-20231008190730397" style="width:80%;" />
