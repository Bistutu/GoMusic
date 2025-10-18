# GoMusic 部署文档

## 部署步骤

### 1. 启动数据库

使用如下 `docker-compose.yaml` 启动 MySQL 和 Redis：

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: gomusic_mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: 12345678
      MYSQL_DATABASE: go_music
      MYSQL_USER: go_music
      MYSQL_PASSWORD: 12345678
      TZ: Asia/Shanghai
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    command:
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_unicode_ci
      - --default-authentication-plugin=mysql_native_password
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p12345678"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: gomusic_redis
    restart: always
    command: redis-server --requirepass SzW7fh2Fs5d2ypwT --port 16379
    ports:
      - "16379:16379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-p", "16379", "-a", "SzW7fh2Fs5d2ypwT", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  mysql_data:
    driver: local
  redis_data:
    driver: local
```

启动命令：

```bash
docker compose up -d
```

**注意**：如果需要修改数据库密码，需同步修改以下文件：
- MySQL 密码：`repo/db/mysql.go` 中的 DSN 连接字符串
- Redis 密码：`repo/cache/redis.go` 中的 Password 字段

### 2. 构建后端项目

在项目根目录执行：

```bash
go build
```

### 3. 启动前端项目

进入前端目录并安装依赖：

```bash
cd static
yarn install
```

**本地开发**：

```bash
yarn serve
```

**生产部署**：

```bash
yarn build
```

### 4. 配置后端请求地址

编辑 `static/src/App.vue`，根据环境修改后端 URL：

```diff
    // 本地开发：使用 http://127.0.0.1:8081
+   const resp = await axios.post('http://127.0.0.1:8081/songlist' + queryParams, params, {
    // 生产部署：替换为你的域名
-   const resp = await axios.post('https://your-domain.com/songlist' + queryParams, params, {
```

- **本地开发**：使用 `http://127.0.0.1:8081`
- **生产部署**：将 URL 改为你的域名，如 `https://your-domain.com`

### 5. 访问应用

**本地开发**：

- 后端：运行编译后的二进制文件 `./GoMusic`
- 前端：访问 `http://localhost:8080`

**生产部署**：

- 将构建后的前端文件（`static/dist`）部署到 Web 服务器
- 配置反向代理将后端 API 请求转发到 Go 服务
- 通过域名访问应用
