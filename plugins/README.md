# ClawPanel 插件仓库

本目录是 ClawPanel 的官方插件仓库，包含官方插件和社区贡献插件。

## 目录结构

```
plugins/
├── registry.json        # 插件注册表（ClawPanel 插件中心读取此文件）
├── official/            # 官方插件
│   └── hello-world/     # 示例插件
└── community/           # 社区贡献插件
```

## 插件开发

详细的插件开发指南请参阅 [docs/plugin-dev/README.md](../docs/plugin-dev/README.md)

## 提交插件

1. Fork 本仓库
2. 在 `community/` 目录下创建你的插件目录
3. 确保包含 `plugin.json` 和 `README.md`
4. 在 `registry.json` 中添加你的插件信息
5. 提交 Pull Request

## 安装方式

- **在线安装**：在 ClawPanel 插件中心浏览并一键安装
- **自定义安装**：输入 Git 仓库 URL 或下载链接
- **本地安装**：将插件目录放入 OpenClaw 的 `extensions/` 目录
