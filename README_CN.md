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
- 可选本地 Ollama AI 助手，支持 `@AI`、`/ai` 和 `/summary`

## 技术栈

- Go
- `net/http`
- `github.com/gorilla/websocket`
- `golang.org/x/crypto/bcrypt`
- 原生 HTML/CSS/JavaScript
- 可选 Ollama 本地模型 API

## 快速启动

确保已经安装 Go，然后执行：

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

## 截图说明

目前还没有放真实截图。后续可以新增 `screenshots/` 目录，并补充这些图片：

- 登录页
- 两个用户在线的聊天室
- emoji 和本地表情包面板
- AI 助手回复效果

## 后续计划

- 增加 WebSocket 自动重连提示
- 增加消息日期分组
- 增加自动化浏览器测试
- 增加可选数据库存储
- 优化移动端输入区体验
