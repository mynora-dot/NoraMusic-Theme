# NoraMusic Theme Development Kit

NoraMusic 主题开发套件（Theme Development Kit），面向第三方主题作者。NoraMusic 是一款 Flutter 音乐应用，它的界面外观完全由声明式的主题包驱动——一个 `.noratheme` 文件就能改变应用的颜色、字体、排版、圆角、动效、播放页样式和背景图。本仓库提供制作、校验、打包和发布主题所需的全部文档、工具、模板和示例说明。

## 什么是 .noratheme

`.noratheme` 是一个 ZIP 压缩包，根目录下必须有两个 JSON 文件：

- `manifest.json` —— 主题的"身份证"：id、名称、版本、作者、许可证、预览图路径。
- `theme.json` —— 主题的全部样式定义：颜色、排版、间距、圆角、动效、组件参数、播放页场景。

此外可以可选地带一张 `preview.jpg` 预览图和一个 `assets/` 目录（放背景图、自定义字体）。包里只有纯数据，没有任何可执行代码。

最小可用的 `manifest.json`：

```json
{
  "schemaVersion": 2,
  "id": "my-first-theme",
  "name": "我的第一个主题",
  "version": "1.0.0",
  "author": { "name": "Your Name" },
  "license": "MIT"
}
```

最小可用的 `theme.json`（`colors` 是唯一必填节，但不能为空；没写的字段全部回退到内置浅色主题 Nora Light 的默认值）：

```json
{
  "schemaVersion": 2,
  "colors": {
    "canvas": "#F6F3EC",
    "accent": "#7FA37A",
    "accentPress": "#6B8C66",
    "accentSubtle": "#1A7FA37A",
    "currentPlaying": "#7FA37A",
    "textPrimary": "#37322A"
  }
}
```

## 能做什么，不能做什么

| 能力 | 状态 |
| --- | --- |
| 颜色（26 个语义化颜色键）、排版、间距、圆角、尺寸 | ✅ 支持 |
| 动效参数（时长、20 种命名曲线、按压缩放） | ✅ 支持 |
| 毛玻璃效果、组件级参数（按钮、列表、弹层、导航栏等） | ✅ 支持 |
| 背景图（含透明度、模糊）、自定义字体（OFL 许可） | ✅ 支持 |
| 播放页场景：5 种预设、背景模式、黑胶封面、歌词样式 | ✅ 支持 |
| 深色主题（`brightness: "dark"`） | ✅ 支持（声明 `"dual"` 的自动派生为实验性） |
| 本地导入 `.noratheme` 文件 | ✅ 支持（设置 → 主题） |
| 通过 HTTPS 链接导入 | ✅ 支持（仅 https、443 端口、≤3 次重定向、≤128 MiB） |
| Lua 脚本、自定义 Shader、粒子、频谱可视化 | ❌ 不支持，出现即被拒绝 |
| 替换页面结构或组件 | ❌ 不支持，主题只能调整参数 |
| 引用任何包外资源（http 图片、网络字体等） | ❌ 不支持，所有资源必须打进包里 |

`assets.backgroundLayers`、`assets.textures`、`assets.particles`、`FontAsset.weights` 这些字段可以被解析但**暂不生效**，属于保留字段，不要在主题里依赖它们。

## 快速开始

只需要 Python 3（标准库，无需安装依赖）。详细步骤见 [docs/01-quick-start.md](docs/01-quick-start.md)。

```bash
# 1. 复制模板，改成你自己的主题
cp -r packages/starter-theme packages/my-theme
#    修改 packages/my-theme/manifest.json 里的 id、name、version
#    修改 packages/my-theme/theme.json 里的颜色

# 2. 校验并打包（输出到 dist/<id>-<version>.noratheme）
python3 tools/pack.py my-theme

# 3. 安装：任选一种
#    ① 设置 → 主题 → 本地导入，选择生成的 .noratheme 文件
#    ② 把文件放到公网 HTTPS 服务器，设置 → 主题 → 链接导入
#    ③ 提交给 NoraMusic 主题商店管理员上传发布
```

## 目录导览

```text
NoraMusic-Theme/
├── README.md                 # 本文件：套件总览
├── LICENSE                   # MIT
├── docs/                     # 完整开发文档（建议按顺序阅读）
│   ├── 01-quick-start.md         # 10 分钟上手
│   ├── 02-package-format.md      # 包结构、manifest.json、校验规则、错误码
│   ├── 03-theme-json-reference.md# theme.json 全字段参考
│   ├── 04-player-scene.md        # 播放页定制
│   ├── 05-assets-and-fonts.md    # 背景图与自定义字体
│   ├── 06-publishing.md          # 三种分发方式与版本管理
│   ├── 07-faq.md                 # 常见问题
│   └── theme-store-server.md     # 独立主题商店部署与 API
├── tools/
│   ├── pack.py               # 校验 + 可重复打包器（Python 3 标准库）
│   ├── generate_assets.py    # 背景图/预览图生成器（需要 Pillow）
│   └── README.md             # 工具用法
├── packages/
│   └── starter-theme/        # 入门模板主题（可直接打包安装）
├── server/                   # 独立主题商店服务（Go + SQLite）
│   ├── cmd/themestore/       # 服务入口
│   ├── internal/             # 校验、存储、认证与 API
│   └── deploy/               # 公网/NAS Docker Compose 示例
├── templates/
│   └── README.md             # 模板说明与新建主题指引
└── examples/
    └── README.md             # 主仓库 6 套官方示例主题导览
```

## 推荐阅读顺序

1. [快速开始](docs/01-quick-start.md) —— 先把模板装上手机看到效果。
2. [包格式](docs/02-package-format.md) —— 搞清 manifest 和打包限制。
3. [theme.json 参考](docs/03-theme-json-reference.md) —— 逐字段查阅，改色改字都在这里。
4. [播放页定制](docs/04-player-scene.md) —— 做出有辨识度的播放页。
5. [背景图与字体](docs/05-assets-and-fonts.md) —— 加素材。
6. [发布分发](docs/06-publishing.md) —— 把主题分享给别人。
7. [FAQ](docs/07-faq.md) —— 出问题时先查这里。

需要自托管主题商店时，参阅 [独立主题商店服务](docs/theme-store-server.md)。服务支持公网强安全模式和 NAS 便捷模式，上传校验与客户端规则保持一致。

## 许可证

本套件（文档、工具脚本、模板）以 [MIT License](LICENSE) 发布，版权方 NoraMusic。

你在模板基础上制作的**主题作品**属于你，可以自选许可证（通过 `manifest.json` 的 `license` 字段声明）。注意：主题中使用的图片、字体、人像等素材的版权与肖像权由你自行负责，详见 [发布分发 · 安全与合规](docs/06-publishing.md)。
