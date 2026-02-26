# OpenClaw 插件开发指南

> 本文档详细介绍如何为 OpenClaw 开发插件，涵盖插件结构、API 接口、配置规范、生命周期管理、调试方法和提交流程。

---

## 目录

1. [概述](#概述)
2. [快速开始](#快速开始)
3. [插件目录结构](#插件目录结构)
4. [plugin.json 规范](#pluginjson-规范)
5. [配置系统 (JSON Schema)](#配置系统-json-schema)
6. [插件入口文件](#插件入口文件)
7. [OpenClaw 插件 API](#openclaw-插件-api)
8. [ClawPanel 管理 API](#clawpanel-管理-api)
9. [插件分类](#插件分类)
10. [生命周期](#生命周期)
11. [日志系统](#日志系统)
12. [权限系统](#权限系统)
13. [冲突检测](#冲突检测)
14. [调试与测试](#调试与测试)
15. [发布插件](#发布插件)
16. [最佳实践](#最佳实践)
17. [示例插件](#示例插件)

---

## 概述

OpenClaw 插件系统允许开发者在不修改 OpenClaw 核心代码的前提下，扩展 AI 助手的功能。插件通过 ClawPanel 的**插件中心**进行安装、配置、启用/禁用和更新。

### 核心设计原则

- **零侵入**：插件不修改 OpenClaw 底层代码
- **热插拔**：启用/禁用插件无需重启 OpenClaw
- **声明式配置**：通过 JSON Schema 自动生成配置 UI
- **标准化接口**：统一的 API 规范，降低开发门槛
- **安全隔离**：插件运行在独立上下文中，声明式权限管理

---

## 快速开始

### 1. 创建插件目录

```bash
mkdir my-awesome-plugin
cd my-awesome-plugin
```

### 2. 编写 plugin.json

```json
{
  "id": "my-awesome-plugin",
  "name": "我的插件",
  "version": "1.0.0",
  "author": "你的名字",
  "description": "一句话描述插件功能",
  "category": "tool",
  "entryPoint": "index.js",
  "minOpenClaw": "2025.1.0",
  "minPanel": "v5.0.0"
}
```

### 3. 编写入口文件 index.js

```javascript
module.exports = {
  name: 'my-awesome-plugin',
  
  // 插件激活时调用
  async activate(context) {
    console.log('[MyPlugin] 插件已激活');
    
    // 注册消息处理器
    context.onMessage(async (msg) => {
      if (msg.content === '/hello') {
        await context.reply(msg, '你好！这是我的插件回复 👋');
      }
    });
  },
  
  // 插件停用时调用
  async deactivate() {
    console.log('[MyPlugin] 插件已停用');
  }
};
```

### 4. 本地安装测试

在 ClawPanel 插件中心，使用「自定义安装」功能，输入本地路径或 Git 仓库 URL。

---

## 插件目录结构

```
my-plugin/
├── plugin.json          # 必需 - 插件元数据
├── index.js             # 必需 - 插件入口文件
├── config.json          # 可选 - 默认配置
├── plugin.schema.json   # 可选 - 配置的 JSON Schema（用于自动生成表单）
├── README.md            # 推荐 - 插件说明文档
├── LICENSE              # 推荐 - 开源许可证
├── package.json         # 可选 - npm 依赖管理
├── assets/              # 可选 - 静态资源
│   ├── icon.png         # 插件图标 (128x128 PNG)
│   └── screenshot.png   # 插件截图
├── lib/                 # 可选 - 工具库
├── test/                # 可选 - 测试文件
└── plugin.log           # 运行时自动生成 - 插件日志
```

---

## plugin.json 规范

`plugin.json` 是插件的核心元数据文件，**必须**存在于插件根目录。

### 完整字段说明

```json
{
  "id": "unique-plugin-id",
  "name": "插件显示名称",
  "version": "1.0.0",
  "author": "作者名",
  "description": "插件功能的一句话描述",
  "homepage": "https://github.com/user/plugin",
  "repository": "https://github.com/user/plugin.git",
  "license": "MIT",
  "category": "ai",
  "tags": ["AI", "消息处理", "自动回复"],
  "icon": "brain",
  "minOpenClaw": "2025.1.0",
  "minPanel": "v5.0.0",
  "entryPoint": "index.js",
  "configSchema": {
    "type": "object",
    "properties": {
      "apiKey": {
        "type": "string",
        "title": "API Key",
        "description": "第三方服务的 API Key"
      },
      "enabled": {
        "type": "boolean",
        "title": "启用功能",
        "default": true
      }
    },
    "required": ["apiKey"]
  },
  "dependencies": {
    "axios": "^1.6.0",
    "dayjs": "^1.11.0"
  },
  "permissions": [
    "message.send",
    "message.receive",
    "config.read",
    "config.write",
    "http.request",
    "file.read"
  ]
}
```

### 字段详解

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `id` | string | ✅ | 插件唯一标识符，全小写，使用 `-` 分隔，如 `smart-scheduler` |
| `name` | string | ✅ | 显示名称，支持中文 |
| `version` | string | ✅ | 语义化版本号，如 `1.0.0` |
| `author` | string | ✅ | 作者名 |
| `description` | string | ✅ | 功能描述，不超过 100 字 |
| `homepage` | string | ❌ | 插件主页 URL |
| `repository` | string | ❌ | Git 仓库地址 |
| `license` | string | ❌ | 开源许可证，推荐 MIT |
| `category` | string | ❌ | 分类：`basic`, `ai`, `message`, `fun`, `tool` |
| `tags` | string[] | ❌ | 标签数组，用于搜索 |
| `icon` | string | ❌ | Lucide 图标名称 |
| `minOpenClaw` | string | ❌ | 最低 OpenClaw 版本要求 |
| `minPanel` | string | ❌ | 最低 ClawPanel 版本要求 |
| `entryPoint` | string | ❌ | 入口文件路径，默认 `index.js` |
| `configSchema` | object | ❌ | JSON Schema 配置定义 |
| `dependencies` | object | ❌ | npm 依赖 |
| `permissions` | string[] | ❌ | 所需权限列表 |

---

## 配置系统 (JSON Schema)

插件可以通过 `configSchema` 字段或独立的 `plugin.schema.json` 文件定义配置结构。ClawPanel 会根据 Schema **自动生成配置表单**，用户无需手动编辑 JSON。

### 支持的 Schema 类型

```json
{
  "type": "object",
  "properties": {
    "apiKey": {
      "type": "string",
      "title": "API Key",
      "description": "用于认证的密钥"
    },
    "maxRetries": {
      "type": "integer",
      "title": "最大重试次数",
      "description": "请求失败时的重试次数",
      "default": 3,
      "minimum": 0,
      "maximum": 10
    },
    "timeout": {
      "type": "number",
      "title": "超时时间(秒)",
      "default": 30.0
    },
    "enableDebug": {
      "type": "boolean",
      "title": "调试模式",
      "default": false
    },
    "logLevel": {
      "type": "string",
      "title": "日志级别",
      "enum": ["debug", "info", "warn", "error"],
      "default": "info"
    },
    "allowedGroups": {
      "type": "array",
      "title": "允许的群组",
      "items": { "type": "string" },
      "description": "限定插件仅在这些群组中生效"
    }
  },
  "required": ["apiKey"]
}
```

### 前端自动渲染规则

| Schema 类型 | 渲染为 |
|------------|--------|
| `string` | 文本输入框 |
| `string` + `enum` | 下拉选择框 |
| `number` / `integer` | 数字输入框 |
| `boolean` | 开关/复选框 |
| `array` | 多值输入 |

### 配置读写

插件配置保存在 `{插件目录}/config.json`。ClawPanel 提供 API 读写配置：

- **读取**：`GET /api/plugins/{id}/config`
- **写入**：`PUT /api/plugins/{id}/config`

插件代码中读取配置：

```javascript
async activate(context) {
  const config = context.getConfig();
  console.log('API Key:', config.apiKey);
  console.log('Max Retries:', config.maxRetries ?? 3);
}
```

---

## 插件入口文件

### 标准导出格式

```javascript
// index.js
module.exports = {
  name: 'my-plugin',       // 插件名（必需）
  version: '1.0.0',        // 版本号（可选，优先读取 plugin.json）
  
  /**
   * 插件激活 - 在启用插件时调用
   * @param {PluginContext} context - 插件上下文
   */
  async activate(context) {
    // 初始化逻辑
  },
  
  /**
   * 插件停用 - 在禁用插件时调用
   * 用于清理资源：取消定时器、关闭连接等
   */
  async deactivate() {
    // 清理逻辑
  },
  
  /**
   * 配置变更回调 - 用户修改配置后触发
   * @param {object} newConfig - 新配置
   * @param {object} oldConfig - 旧配置
   */
  async onConfigChange(newConfig, oldConfig) {
    // 响应配置变更
  }
};
```

### TypeScript 支持

```typescript
// index.ts
import { PluginContext, Message } from '@openclaw/plugin-sdk';

export default {
  name: 'my-plugin',
  
  async activate(context: PluginContext) {
    context.onMessage(async (msg: Message) => {
      // 处理消息
    });
  },
  
  async deactivate() {}
};
```

---

## OpenClaw 插件 API

### PluginContext 接口

```typescript
interface PluginContext {
  // === 配置 ===
  getConfig(): Record<string, any>;
  setConfig(key: string, value: any): void;
  
  // === 消息 ===
  onMessage(handler: (msg: Message) => Promise<void>): void;
  reply(msg: Message, content: string): Promise<void>;
  send(channelId: string, targetId: string, content: string): Promise<void>;
  
  // === 事件 ===
  on(event: string, handler: (...args: any[]) => void): void;
  emit(event: string, ...args: any[]): void;
  
  // === 任务（面向智能任务调度等高级插件） ===
  createTask(name: string, steps: TaskStep[]): Task;
  updateTaskProgress(taskId: string, step: number, status: string): void;
  notifyUser(taskId: string, message: string): void;
  
  // === 存储 ===
  storage: {
    get(key: string): Promise<any>;
    set(key: string, value: any): Promise<void>;
    delete(key: string): Promise<void>;
    list(): Promise<string[]>;
  };
  
  // === 日志 ===
  log: {
    debug(msg: string): void;
    info(msg: string): void;
    warn(msg: string): void;
    error(msg: string): void;
  };
  
  // === HTTP ===
  http: {
    get(url: string, options?: RequestOptions): Promise<Response>;
    post(url: string, body: any, options?: RequestOptions): Promise<Response>;
    put(url: string, body: any, options?: RequestOptions): Promise<Response>;
    delete(url: string, options?: RequestOptions): Promise<Response>;
  };
  
  // === 定时器 ===
  setInterval(callback: () => void, ms: number): string;
  setTimeout(callback: () => void, ms: number): string;
  clearTimer(id: string): void;
  
  // === 插件信息 ===
  pluginId: string;
  pluginDir: string;
  openclawVersion: string;
  panelVersion: string;
}

interface Message {
  id: string;
  channel: string;       // 'qq' | 'wechat' | 'telegram' | ...
  type: string;           // 'private' | 'group'
  content: string;
  senderId: string;
  senderName: string;
  groupId?: string;
  groupName?: string;
  timestamp: number;
  raw: any;               // 原始消息对象
}

interface TaskStep {
  name: string;
  description: string;
  handler: () => Promise<any>;
}

interface Task {
  id: string;
  name: string;
  steps: TaskStep[];
  status: 'pending' | 'running' | 'completed' | 'failed';
  currentStep: number;
  createdAt: number;
  run(): Promise<void>;
  cancel(): void;
}
```

---

## ClawPanel 管理 API

以下 API 供 ClawPanel 前端和外部工具调用，用于管理插件：

### 列表相关

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/plugins/list` | 获取已安装 + 仓库插件列表 |
| GET | `/api/plugins/installed` | 仅获取已安装插件 |
| GET | `/api/plugins/:id` | 获取单个插件详情 |
| POST | `/api/plugins/registry/refresh` | 刷新插件仓库 |

### 生命周期

| 方法 | 路径 | Body | 说明 |
|------|------|------|------|
| POST | `/api/plugins/install` | `{"pluginId":"xxx","source":""}` | 安装插件 |
| DELETE | `/api/plugins/:id` | - | 卸载插件 |
| PUT | `/api/plugins/:id/toggle` | `{"enabled":true}` | 启用/禁用 |
| POST | `/api/plugins/:id/update` | - | 更新到最新版本 |

### 配置

| 方法 | 路径 | Body | 说明 |
|------|------|------|------|
| GET | `/api/plugins/:id/config` | - | 获取配置和 Schema |
| PUT | `/api/plugins/:id/config` | `{...config}` | 更新配置 |

### 日志

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/plugins/:id/logs` | 获取插件运行日志 |

### 响应格式

所有 API 返回 JSON 格式：

```json
{
  "ok": true,
  "message": "操作成功",
  "data": { ... }
}
```

错误响应：

```json
{
  "ok": false,
  "error": "错误描述"
}
```

---

## 插件分类

| 分类 | category 值 | 说明 | 示例 |
|------|------------|------|------|
| 基础功能 | `basic` | 核心功能增强 | 自动回复、关键词触发 |
| AI 增强 | `ai` | AI 能力扩展 | GPT 增强、多模态、RAG |
| 消息处理 | `message` | 消息过滤/转发 | 消息转发、内容过滤 |
| 娱乐互动 | `fun` | 游戏/娱乐 | 骰子、抽签、小游戏 |
| 工具 | `tool` | 实用工具 | 天气查询、翻译、提醒 |

---

## 生命周期

```
安装 → 初始化 → 激活 → 运行中 ⟷ 配置变更
                              ↓
                           停用 → 卸载
```

1. **安装** (`install`)：从仓库/本地安装，解压到 extensions 目录
2. **初始化**：读取 plugin.json，解析依赖，安装 npm 包
3. **激活** (`activate`)：调用入口文件的 `activate(context)`
4. **运行中**：插件正常运行，处理消息和事件
5. **配置变更** (`onConfigChange`)：用户修改配置时触发
6. **停用** (`deactivate`)：调用 `deactivate()` 清理资源
7. **卸载** (`uninstall`)：删除插件目录和配置

---

## 日志系统

插件可以通过 `context.log` 输出日志，日志自动保存到 `{插件目录}/plugin.log`。

```javascript
context.log.info('消息已处理');
context.log.warn('配置项缺失，使用默认值');
context.log.error('API 调用失败: ' + error.message);
context.log.debug('收到消息: ' + JSON.stringify(msg));
```

日志可在 ClawPanel 插件中心的「日志」标签页实时查看。

---

## 权限系统

插件必须在 `plugin.json` 中声明所需权限。安装时 ClawPanel 会提示用户确认。

| 权限 | 说明 |
|------|------|
| `message.send` | 发送消息 |
| `message.receive` | 接收消息 |
| `config.read` | 读取 OpenClaw 配置 |
| `config.write` | 修改 OpenClaw 配置 |
| `http.request` | 发起 HTTP 请求 |
| `file.read` | 读取工作目录文件 |
| `file.write` | 写入工作目录文件 |
| `process.exec` | 执行系统命令（危险） |
| `task.create` | 创建后台任务 |
| `task.notify` | 发送任务通知 |

---

## 冲突检测

安装插件前，ClawPanel 会自动检测以下冲突：

1. **ID 冲突**：不能安装 ID 相同的插件
2. **端口冲突**：检查插件是否占用已使用的端口
3. **依赖冲突**：检查 npm 依赖版本兼容性
4. **权限冲突**：高危权限需要用户确认

---

## 调试与测试

### 本地调试

1. 将插件目录放入 OpenClaw 的 `extensions/` 目录
2. 在 ClawPanel 中刷新插件列表
3. 启用插件并查看日志输出

### 日志调试

```javascript
// 在入口文件中添加详细日志
async activate(context) {
  context.log.debug('[DEBUG] 插件激活参数: ' + JSON.stringify(context.getConfig()));
  
  context.onMessage(async (msg) => {
    context.log.debug('[DEBUG] 收到消息: ' + msg.content);
    // ...
  });
}
```

### 单元测试

```javascript
// test/index.test.js
const plugin = require('../index');

describe('MyPlugin', () => {
  it('should activate without error', async () => {
    const mockContext = {
      getConfig: () => ({ apiKey: 'test' }),
      onMessage: jest.fn(),
      log: { info: jest.fn(), error: jest.fn(), debug: jest.fn(), warn: jest.fn() },
    };
    await plugin.activate(mockContext);
    expect(mockContext.onMessage).toHaveBeenCalled();
  });
});
```

---

## 发布插件

### 提交到官方仓库

1. Fork `ClawPanel-Plugins` 仓库
2. 在 `community/` 目录下创建你的插件目录
3. 确保包含完整的 `plugin.json` 和 `README.md`
4. 更新 `registry.json`，添加你的插件信息
5. 提交 Pull Request

### 提交审核清单

- [ ] `plugin.json` 字段完整
- [ ] `README.md` 包含使用说明
- [ ] 代码无恶意行为
- [ ] 声明了所有必要权限
- [ ] 无硬编码密钥/Token
- [ ] 版本号遵循语义化版本

### 独立发布

你也可以将插件发布到自己的 Git 仓库，用户通过「自定义安装」输入 URL 即可安装。

---

## 最佳实践

1. **错误处理**：所有异步操作必须 try-catch，避免未捕获异常导致 OpenClaw 崩溃
2. **资源清理**：在 `deactivate()` 中清理所有定时器、WebSocket 连接等
3. **配置校验**：读取配置后校验必需字段，缺失时使用合理默认值
4. **日志规范**：使用 `context.log` 而非 `console.log`，便于 ClawPanel 收集
5. **异步操作**：长耗时操作使用 `setTimeout` 或 Promise，避免阻塞消息处理
6. **版本兼容**：在 `plugin.json` 中明确 `minOpenClaw` 和 `minPanel` 版本
7. **国际化**：如果面向国际用户，同时提供中英文说明

---

## 示例插件

### 智能任务调度插件（设计参考）

此示例展示如何开发一个高级插件，实现消息秒回 + 后台任务调度 + 进度通知。

```javascript
// smart-scheduler/index.js
module.exports = {
  name: 'smart-scheduler',
  
  async activate(context) {
    context.log.info('[SmartScheduler] 智能任务调度插件已启动');
    
    context.onMessage(async (msg) => {
      // 1. 收到消息后立即回复
      await context.reply(msg, '收到！正在为你处理... 📋');
      
      // 2. 创建任务列表
      const task = context.createTask('处理用户请求', [
        {
          name: '分析意图',
          description: '使用 AI 分析用户消息意图',
          handler: async () => {
            // AI 意图分析逻辑
            return { intent: 'query', confidence: 0.95 };
          }
        },
        {
          name: '执行操作',
          description: '根据意图执行对应操作',
          handler: async () => {
            // 执行业务逻辑
            return { result: '查询完成' };
          }
        },
        {
          name: '生成回复',
          description: '生成最终回复内容',
          handler: async () => {
            return { reply: '这是处理结果...' };
          }
        }
      ]);
      
      // 3. 通知用户任务已创建
      await context.reply(msg, `📋 任务已创建：\n1️⃣ 分析意图\n2️⃣ 执行操作\n3️⃣ 生成回复\n\n正在处理中...`);
      
      // 4. 执行任务并实时通知进度
      task.on('stepComplete', async (step, result) => {
        await context.notifyUser(task.id, `✅ ${step.name} 完成`);
      });
      
      task.on('error', async (step, error) => {
        await context.notifyUser(task.id, `❌ ${step.name} 失败: ${error.message}`);
      });
      
      await task.run();
      
      // 5. 任务完成后发送最终结果
      await context.reply(msg, '✅ 所有任务已完成！');
    });
  },
  
  async deactivate() {
    // 清理资源
  }
};
```

---

*本文档持续更新，如有问题请提交 Issue 或 PR。*

*© 2025 ClawPanel Team. MIT License.*
