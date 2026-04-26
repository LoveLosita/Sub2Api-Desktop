# sub2api-desktop

AI API 桌面反向代理网关，支持 Claude / OpenAI / Gemini 多账号管理、负载均衡、自动故障转移与健康检查。基于 [Wails](https://wails.io/) (Go + Vue 3) 构建的本地桌面应用，开箱即用，无需部署服务器。

> 本项目灵感来源于 [sub2api](https://github.com/Wei-Shaw/sub2api)，将其核心的 API 网关能力移植为轻量级桌面应用，面向个人开发者和小团队的本地化使用场景。

## 功能特性

- **多账号管理** — 支持 API Key、OAuth、Cookie 等多种认证方式
- **智能负载均衡** — 基于优先级的轮询调度，自动选择最优账号
- **自动故障转移** — 请求失败时自动切换到下一个可用账号，客户端无感知
- **健康检查** — 手动或批量检测账号可用性，异常账号自动标记
- **认证失败自动标记** — 代理请求遇到 401/403 时自动将账号标记为异常
- **限速/过载冷却** — 账号触发限速或过载时自动进入冷却期，到期后自动恢复
- **分组管理** — 账号按组隔离，支持多租户场景
- **API Key 分发** — 生成独立 API Key，支持绑定分组
- **用量统计** — Token 级别的使用记录与费用追踪
- **模型定价** — 内置模型价格表，支持远程同步
- **代理支持** — 每个账号可独立配置 HTTP/SOCKS5 代理

## 支持平台

| 平台 | 认证方式 | 默认 Base URL |
|------|---------|--------------|
| Claude | API Key / OAuth / Cookie | `https://api.anthropic.com/v1` |
| OpenAI | API Key / OAuth | `https://api.openai.com/v1` |
| Gemini | API Key / OAuth | `https://generativelanguage.googleapis.com` |

## 快速开始

### 下载

从 [Releases](../../releases) 页面下载最新版本的可执行文件。

### 运行

双击运行即可，无需安装。应用会自动在本地启动代理服务器（默认 `127.0.0.1:8787`）。

### 配置 Claude Code

```bash
export ANTHROPIC_BASE_URL="http://127.0.0.1:8787"
export ANTHROPIC_AUTH_TOKEN="sk-your-api-key"
```

### 配置 OpenAI Codex

```bash
export OPENAI_BASE_URL="http://127.0.0.1:8787"
export OPENAI_API_KEY="sk-your-api-key"
```

## 从源码构建

### 前置要求

- Go 1.21+
- Node.js 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### 构建

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 克隆仓库
git clone https://github.com/LoveLosita/Sub2Api-Desktop.git
cd sub2api-desktop

# 开发模式
wails dev

# 生产构建
wails build
```

构建产物在 `build/bin/` 目录下。

## 项目结构

```
sub2api-desktop/
├── app.go                          # Wails 应用入口与前端绑定
├── internal/
│   ├── config/                     # 配置管理
│   ├── database/                   # SQLite 数据库初始化与迁移
│   ├── model/                      # 数据模型
│   ├── server/                     # 反向代理网关 (Claude/OpenAI/Gemini)
│   └── service/                    # 业务逻辑层
│       ├── account.go              # 账号 CRUD
│       ├── gateway.go              # 请求转发与重试
│       ├── healthcheck.go          # 健康检查
│       ├── pricing.go              # 模型定价
│       └── usage.go                # 用量记录
├── frontend/
│   └── src/
│       └── views/                  # 页面组件
│           ├── Dashboard.vue       # 仪表盘
│           ├── Accounts.vue        # 账号管理 + 健康检查
│           ├── Groups.vue          # 分组管理
│           ├── Proxies.vue         # 代理配置
│           ├── ApiKeys.vue         # API Key 管理
│           ├── Usage.vue           # 使用记录
│           ├── Pricing.vue         # 模型定价
│           └── Settings.vue        # 设置
└── config.yaml                     # 运行配置文件
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go, Gin |
| 前端 | Vue 3, TypeScript, TailwindCSS |
| 桌面框架 | Wails v2 |
| 数据库 | SQLite (内嵌) |

## 致谢

- [sub2api](https://github.com/Wei-Shaw/sub2api) — 本项目的灵感来源，提供了 API 网关的核心设计思路
- [Wails](https://wails.io/) — 优秀的 Go 桌面应用框架

## License

[MIT License](LICENSE)

本项目灵感来源于 [sub2api](https://github.com/Wei-Shaw/sub2api)（LGPL v3）。
