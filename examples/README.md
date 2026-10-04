# examples · 官方示例主题

NoraMusic 主仓库（`NoraMusic/theme/packages/`）维护了 6 套官方示例主题，全部是**只用当前公开能力**实现、可以直接上架的完整主题。它们体积较大（含背景图和字体文件），因此**不复制进本仓库**——学习时请到主仓库对应目录阅读 `manifest.json` 和 `theme.json`，这两个 JSON 加起来不到 200 行，是最好的实战教材。

| 目录 | 名称 | 风格与学习要点 |
| --- | --- | --- |
| `blue-feather` | 蓝羽 | 清透浅蓝 + 洁白羽毛的治愈系。`immersive` 预设 + `coverFocus` 背景（封面中心清晰、边缘渐隐模糊）+ 半透明遮罩（`lightBackgroundBase` 带 alpha）+ 全局霞鹜文楷字体的标准组合 |
| `cloud-red` | 云村红 | 经典网易云风红白灰。默认黑胶播放页（唱针转盘俱全），`brightnessSupport: "dual"` 双模式声明的参考样本，全局思源黑体 |
| `natural-warmth` | 自然暖光 | 米色 + 柔和绿意的治愈配色。`lyricsBackground` 用 `image` 模式给歌词页单独指定背景图、`lyricsLightOverlayStart/End` 歌词页遮罩写法都值得抄 |
| `opera-red` | 梨园红 | 戏曲朱砂中国红 + 毛笔楷体。传统配色的 26 键完整映射样例，`coverFocus` 沉浸播放页 |
| `opera-velvet-dual` | 绛幕流光 | 戏曲绛幕 + 鎏金的双色主题。`brightnessSupport: "dual"` + webp 动图背景 + `Noto Serif SC` 可变字体 + 完整 8 样式 `typo` 表，是字段最全的参考 |
| `zhengyehuilicai` | 真野绘里菜 | 蓝调人像写真定制主题（与蓝羽同风格谱系）。注意：人像素材涉及肖像权/著作权，仅适合个人本地使用，**不要照此上架**，合规要求见 [../docs/06-publishing.md](../docs/06-publishing.md#安全与合规) |

## 怎么学

1. **先读 JSON，再看图**。每套主题的核心都在 `theme.json`：先看 `colors` 怎么配出整体调性，再看 `player` 节怎么做播放页。
2. **对照字段文档读**。遇到不认识的键，回 [../docs/03-theme-json-reference.md](../docs/03-theme-json-reference.md) 和 [../docs/04-player-scene.md](../docs/04-player-scene.md) 查表。
3. **拆着抄，别整抄**。把某个你喜欢的局部（比如 `natural-warmth` 的歌词页遮罩、`opera-velvet-dual` 的 typo 表）拷进自己的主题，配合自己的配色调。
4. **注意素材版权**。示例主题的背景图、字体各有授权条件，主题配置可以参考，素材文件不要直接拿去发布。示例内置字体均来自 Google Fonts（SIL OFL 1.1），许可证文本在主仓库 `theme/licenses/`。
5. 想要一个能直接改的起点，用本仓库的 [`packages/starter-theme/`](../packages/starter-theme/)（见 [../templates/README.md](../templates/README.md)）。
