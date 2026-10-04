# 01 · 快速开始：10 分钟做出第一个主题

本篇目标：复制模板 → 改一处颜色 → 打包 → 在 NoraMusic 里看到自己的主题。

## 环境要求

| 工具 | 是否必需 | 说明 |
| --- | --- | --- |
| Python 3.8+ | 必需 | `tools/pack.py` 只用标准库，无需 `pip install` 任何东西 |
| Pillow | 可选 | 仅 `tools/generate_assets.py`（程序化生成背景图/预览图）需要，`pip install Pillow` |
| 任意文本编辑器 | 必需 | VS Code 等带 JSON 高亮的编辑器体验更好 |
| 一台装了 NoraMusic 的手机/模拟器 | 必需 | 用来安装和预览主题 |

检查环境：

```bash
python3 --version
# Python 3.x.x
```

## 第一步：复制模板

模板主题是 `packages/starter-theme/`（id 为 `my-first-theme`）。复制一份并起一个属于你自己的 id：

```bash
cp -r packages/starter-theme packages/my-theme
```

打开 `packages/my-theme/manifest.json`，至少改这三个字段：

```json
{
  "schemaVersion": 2,
  "id": "my-theme",
  "name": "我的主题",
  "version": "1.0.0"
}
```

`id` 的规则：以字母或数字开头，只能含字母、数字、`_`、`-`，最长 64 个字符；不能叫 `nora-light` 或 `nora-dark`（这是两个内置主题的保留 id）。发布后 `id` 就是这个主题的唯一身份，同 `id` 的不同 `version` 被视为同一主题的新版本。

> 一个仓库目录下每个 `packages/xxx/` 的目录名不需要等于 manifest 里的 `id`，但保持一致最不容易混乱。`tools/pack.py` 的参数是**目录名**。

## 第二步：改颜色

打开 `packages/my-theme/theme.json`，先只改强调色感受一下：

```json
"colors": {
  "accent": "#C2574B",
  "accentPress": "#A64439",
  "accentSubtle": "#1AC2574B",
  "currentPlaying": "#C2574B"
}
```

颜色写法是 `#RRGGBB` 或 `#AARRGGBB`——注意 8 位写法里**透明度在最前面**：`#1AC2574B` 表示 10% 不透明（0x1A）的红色。写 `#C2574B1A`（alpha 在后）能通过部分校验，但显示出来的颜色是错的，见 [03-theme-json-reference.md](03-theme-json-reference.md#colors)。

`theme.json` 里只有 `colors` 是必填节，其余字段不写就回退到内置浅色主题 Nora Light 的默认值。模板里已经写全了 26 个颜色键和常用配置，你可以逐段删改。

## 第三步：校验并打包

```bash
python3 tools/pack.py my-theme
```

参数是 `packages/` 下的**目录名**；不带参数则打包 `packages/` 下全部主题：

```bash
python3 tools/pack.py
```

打包成功时的真实输出（以本仓库模板为例）：

```text
starter-theme
  NoraMusic-Theme/dist/my-first-theme-1.0.0.noratheme  2 个文件  0.00 MiB  sha256 a8dd153fa9f155fb4df96e1b82174f1591e472d1f595b9dec8c3f31669da9788
```

- 产物在 `dist/<id>-<version>.noratheme`。
- `pack.py` 会先完整校验（文件限制、路径安全、manifest 格式、颜色格式、内容扫描），**校验不通过就不出包**，失败原因会直接打印出来，例如：

```text
my-theme
  失败：theme.colors.accent 不是合法颜色：#C2574B1
```

- 打包结果可重复：条目按路径排序、时间戳固定为 1980-01-01，同样的输入永远得到字节级相同的文件，方便核对 SHA-256。
- `*.noratheme` 已在本仓库 `.gitignore` 中，产物不会进版本库。

校验规则的完整清单见 [02-package-format.md](02-package-format.md)。

## 第四步：安装到应用

三种方式任选其一，详细说明见 [06-publishing.md](06-publishing.md)。

**方式一：本地文件导入（开发调试首选）**

1. 把 `dist/my-theme-1.0.0.noratheme` 传到手机（AirDrop、adb push、聊天工具发给自己均可）。
2. 打开 NoraMusic → **设置 → 主题 → 本地导入**，选择这个文件。
3. 安装成功后立刻可以启用。

**方式二：HTTPS 链接导入**

把文件传到任意公网静态托管（对象存储、GitHub Release 等），在 **设置 → 主题 → 链接导入** 输入完整 URL。链接有硬性限制：

- 必须是 `https://`，端口必须是 **443**（不写端口即默认 443，可以）；
- 最多跟随 **3 次重定向**，重定向目标也要满足同样的限制；
- 文件不超过 **128 MiB**；
- 目标必须是公网地址，回环、内网、链路本地地址会被拒绝（防止导入入口被当成内网代理）。

**方式三：主题商店**

把 `.noratheme` 交给部署了 NoraMusic 后台的管理员上传发布，所有用户在主题商店里即可看到并一键安装。流程见 [06-publishing.md](06-publishing.md#主题商店)。

## 第五步：迭代

改完 JSON 后重新执行第三步打包即可。本地导入可以反复装同一个版本；走商店渠道时，同 `id` + 同 `version` 重复上传会被拒绝（409 `THEME_VERSION_EXISTS`），必须提升 `version`。

建议的迭代节奏：

1. 先只调 `colors`，确认配色；
2. 再调 `typo` / `space` / `shape` / `motion`，确认质感；
3. 然后做 `player` 播放页场景（见 [04-player-scene.md](04-player-scene.md)）；
4. 最后加背景图和字体（见 [05-assets-and-fonts.md](05-assets-and-fonts.md)），补 `preview` 预览图，发布。

## 打包前自查清单

- [ ] `manifest.json` 和 `theme.json` 在主题目录根部，两个文件都写了 `"schemaVersion": 2`
- [ ] `id` 只含字母、数字、`_`、`-`；`version` 是 `x.y.z` 三段纯数字
- [ ] `author` 是对象（`{"name": "..."}`），不是字符串
- [ ] 包内（含 JSON 的任何字段）没有任何 URL——`author.url` 写 `https://...` 也会被拒
- [ ] 写了 `preview` 字段时，包里确实存在这个文件；没有预览图就删掉该字段
- [ ] 颜色键名来自 26 个官方键，值是 `#RRGGBB` 或 `#AARRGGBB`
- [ ] `colors` 里没有名为 `background` 的键（它会被当作资源路径检查）
- [ ] 背景图、字体文件确实在包里，路径与 JSON 完全一致（区分大小写）
- [ ] 没有 `.DS_Store`、`__MACOSX/`、`.bak`、`.md` 等多余文件（`pack.py` 会自动排除隐藏文件，但手动 zip 时不会）
- [ ] 毫秒类字段（`haloDuration` 等）和 `player` 的 6 个颜色字段写的是**整数**
- [ ] 播放页用了封面模糊/渐变/图片背景时，`lightBackgroundBase` 带了透明度（见 [04-player-scene.md](04-player-scene.md#lightbackgroundbase-的-alpha-陷阱)）

用 `pack.py` 打包时以上大部分会被自动检查；如果你选择自己 `zip` 打包，务必逐条自查。
