# Go WebSocket Chatroom Demo

一个基于 Go 和原生 WebSocket 的实时聊天室学习项目，支持注册登录、在线人数、历史消息、emoji、本地表情包，以及可选本地 Ollama AI 助手。

这个项目适合用来学习 WebSocket、简单会话登录、前后端实时通信和 Go Web 服务的基本组织方式。它不是生产级聊天室系统，重点是清晰、可运行、容易改。

## 功能特性

- 注册、登录、退出登录
- `/` 和 `/ws` 登录校验
- 多用户实时聊天
- 在线人数显示
- 用户加入、离开系统提示
- 最近 20 条消息历史
- 普通文字、emoji、本地 SVG 表情包
- 自己消息、他人消息、系统消息、历史消息、AI 消息样式区分
- 输入长度限制和字数提示
- WebSocket 连接状态指示（已连接 / 已断开 / 重连中）
- 断线自动重连（指数退避）
- 用户名和密码前后端输入校验
- 可选本地 Ollama AI 助手，支持 `@AI`、`/ai` 和 `/summary`

## 技术栈

- Go
- `net/http`
- `github.com/gorilla/websocket`
- `golang.org/x/crypto/bcrypt`
- 原生 HTML/CSS/JavaScript
- 可选 Ollama 本地模型 API

## 快速启动

确保已经安装 Go 1.21+，然后执行：

```powershell
go mod tidy
go run main.go
```

浏览器访问：

```text
http://localhost:8088
```

首次启动时，程序会自动创建 `users.json`。该文件只用于本地演示，已经被 `.gitignore` 忽略，不会提交到仓库。

## Docker 启动方式

使用 Docker Compose：

```powershell
docker compose up --build
```

访问：

```text
http://localhost:8088
```

Compose 配置会用 volume 保存用户数据，并设置 Ollama 示例环境变量。即使本机没有启动 Ollama，聊天室主体功能也可以正常使用。

停止容器：

```powershell
docker compose down
```

修改代码后重新构建：

```powershell
docker compose up --build
```

## Ollama AI 助手

AI 助手是可选功能。先启动本地 Ollama，再指定 API 地址和模型：

```powershell
$env:OLLAMA_URL = "http://localhost:11434"
$env:OLLAMA_MODEL = "qwen2.5:3b"
go run main.go
```

聊天室中可以输入：

```text
@AI 用一句话解释 WebSocket
/ai 帮我总结 Gorilla WebSocket 的作用
/summary
```

如果 Ollama 不可用，程序不会崩溃，会在聊天室里返回友好的不可用提示。

## 项目亮点

- **零前端依赖**：纯 HTML/CSS/JavaScript，无需构建工具
- **bcrypt 密码哈希**：密码从不明文存储
- **HttpOnly + SameSite Cookie**：Session token 不可被 JavaScript 访问
- **消息长度限制**：每条消息 800 字符，实时字数统计
- **表情包 URL 白名单**：只接受预批准的表情包 URL
- **AI 优雅降级**：即使 Ollama 未运行，聊天室功能完全正常
- **Docker 就绪**：一条命令部署，用户数据持久化
- **WebSocket 自动重连**：网络中断时自动恢复连接

## 项目目录结构

```text
.
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── main.go
├── users.example.json
├── public/
│   ├── app.js
│   ├── index.html
│   ├── login.html
│   ├── register.html
│   ├── style.css
│   └── stickers/
│       ├── sticker-1.svg
│       ├── sticker-2.svg
│       ├── sticker-3.svg
│       ├── sticker-4.svg
│       ├── sticker-5.svg
│       └── sticker-6.svg
├── README.md
└── README_CN.md
```

## 安全说明

本项目用于学习和本地演示，不建议直接用于生产环境。

- `users.json` 是本地 JSON 文件，只适合 demo 场景。
- 密码使用 bcrypt 哈希保存，但账号系统整体仍然很简化。
- session 存在内存里，服务重启后会失效。
- 当前没有验证码、限流、邮箱验证、密码找回、CSRF 防护、审计日志等生产功能。
- 如果要部署到公网，需要重新设计认证、存储、安全策略和运维监控。

## 测试

运行：

```powershell
gofmt -w main.go
go mod tidy
go test ./...
```

手动检查：

- 访问 `/` 未登录时确认重定向到 `/login`
- 用两个账号在不同浏览器窗口注册并登录
- 发送文字、emoji 和表情包消息
- 刷新页面确认历史消息出现
- 启动 Ollama 后测试 `@AI` 和 `/summary`
- 停止 Ollama 后确认 AI 命令显示友好错误提示
- 测试用户名校验：尝试过短、过长或特殊字符
- 测试 WebSocket 断开：停止服务器观察重连行为

## 截图说明

目前还没有放真实截图。后续可以新增 `screenshots/` 目录，并补充这些图片：

- 登录页
- 两个用户在线的聊天室
- emoji 和本地表情包面板
- AI 助手回复效果

## 后续计划

- [x] WebSocket 断线自动重连（指数退避）
- [x] 用户名和密码输入校验
- [x] 连接状态指示器
- [ ] 消息日期分组
- [ ] 自动化浏览器测试
- [ ] 可选数据库存储
- [ ] 消息搜索/过滤
- [ ] 用户资料自定义
