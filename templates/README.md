# templates · 模板说明

本仓库的入门模板直接放在 [`../packages/starter-theme/`](../packages/starter-theme/)——因为 `tools/pack.py` 默认扫描 `packages/` 目录，模板放在那里可以**开箱即打包**，无需任何移动：

```bash
python3 tools/pack.py starter-theme
# 输出 dist/my-first-theme-1.0.0.noratheme
```

`templates/` 目录本身预留给未来的额外卖点模板（例如深色模板、带字体模板）。目前请直接使用 `packages/starter-theme/`。

## 模板包含什么

- `manifest.json`：合法的最小 + 常用字段示例。`id` 是占位符 `my-first-theme`，`author` / `license` / `tags` / `brightnessSupport` / `category` 均为示例值。
- `theme.json`：完整可用的示例——全部 26 个颜色键（米白暖底 + 鼠尾草绿的和谐配色）、8 种 `typo` 文字样式、`motion`、`glass`、以及一个开启了半透明播放页遮罩（`lightBackgroundBase` 带 alpha）的 `player` 节。**已通过 `pack.py` 全部校验，可直接安装。**

模板特意不带背景图、字体和预览图——这些是按需添加的可选内容，添加方法见 [../docs/05-assets-and-fonts.md](../docs/05-assets-and-fonts.md)。

## 从模板新建你自己的主题

```bash
# 1. 复制（目录名建议与你的主题 id 一致）
cp -r packages/starter-theme packages/my-theme

# 2. 编辑 packages/my-theme/manifest.json
#    - id：改成你的主题 id（正则 ^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$，不能是 nora-light / nora-dark）
#    - name / author / description / license / tags 改成你的
#    - version 从 1.0.0 开始

# 3. 编辑 packages/my-theme/theme.json，改颜色与排版

# 4. 校验并打包
python3 tools/pack.py my-theme

# 5. 安装（三选一）
#    设置 → 主题 → 本地导入 dist/my-theme-1.0.0.noratheme
#    设置 → 主题 → 链接导入（公网 HTTPS URL）
#    提交管理员上架主题商店
```

完整流程见 [../docs/01-quick-start.md](../docs/01-quick-start.md)。

## 发布前必改

- [ ] `manifest.id`：不要用 `my-first-theme` 发布，那是占位符
- [ ] `manifest.name`、`author`、`description`：改成你的主题信息
- [ ] `manifest.license`：声明你的主题配置许可证（含 OFL 字体时用组合写法）
- [ ] 每次发布新版本记得提升 `version`（同 id+version 上传会被 409 拒绝）
