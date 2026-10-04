# 02 · 主题包格式与 manifest.json

`.noratheme` 是一个 ZIP 压缩包。后台主题商店和客户端用**同一套规则**校验：后台在上传时检查，客户端在下载安装前再检查一遍（本地导入、链接导入同样过这套校验）。本篇列出全部规则和 `manifest.json` 的完整字段表。

## 包结构

```text
my-theme.noratheme  (ZIP)
├── manifest.json        # 必需，主题元信息
├── theme.json           # 必需，样式定义
├── preview.jpg          # 可选，商店/主题列表预览图
└── assets/              # 可选
    ├── backgrounds/bg.jpg    # 背景图
    └── fonts/MyFont.ttf      # 自定义字体
```

两个 JSON 必须在**压缩包根目录**，不能多套一层文件夹。多套一层会得到 `THEME_MANIFEST_REQUIRED` 错误。

## 文件限制

| 限制 | 值 | 超限错误码 |
| --- | --- | --- |
| 允许的扩展名（大小写不限） | `.json` `.png` `.jpg` `.jpeg` `.webp` `.ttf` `.otf` | `UNSUPPORTED_THEME_FILE` |
| 文件总数 | ≤ 512 | `THEME_TOO_MANY_FILES` |
| 单文件解压后大小 | ≤ 32 MiB | `THEME_EXPANSION_LIMIT` |
| 单文件压缩比 | ≤ 200 倍 | `THEME_EXPANSION_LIMIT` |
| 解压后总大小 | ≤ 128 MiB | `THEME_EXPANSION_LIMIT` |
| 包体大小（下载/导入） | ≤ 128 MiB | `THEME_SIZE_LIMIT` |
| 商店上传大小 | 由后台策略决定，默认 32 MiB，可在 1–128 MiB 之间调整 | `THEME_SIZE_LIMIT`（413） |
| `manifest.json` 大小 | ≤ 64 KiB | `INVALID_THEME_MANIFEST` |
| `theme.json` 大小 | ≤ 1 MiB | `INVALID_THEME_CONFIG` |

出现任何白名单之外的文件，**整个包都会被拒**——包括 `.lua`、`.frag`、`.gif`、`.md`、`.txt`、`.bak`、`.DS_Store`。这也带来一个实际后果：**字体的许可证文本（OFL.txt）放不进包里**，署名请写在 `manifest.json` 的 `description` 和 `license` 字段里，见 [05-assets-and-fonts.md](05-assets-and-fonts.md#字体许可与署名)。

## 路径限制

- 只能使用相对路径，以 `/` 分隔（POSIX 风格），总长度 ≤ 240 字符。
- 不允许：`..`、`\`、`:`、`./`、`//`、绝对路径、空字符。
- 不允许符号链接。
- 不允许 `__MACOSX/` 目录和 `.DS_Store` 文件（所以**不要用 macOS 访达的"压缩"功能**打包，它会自动塞入 `__MACOSX/`）。
- 文件名在忽略大小写后不能重名（例如 `BG.jpg` 和 `bg.jpg` 不能同时存在）。

违反以上任一条 → `UNSAFE_THEME_PATH`。

## manifest.json 字段表

| 字段 | 类型 | 必填 | 规则 / 说明 |
| --- | --- | --- | --- |
| `schemaVersion` | 整数 | **是** | 必须是 `2`，其他值直接拒绝 |
| `id` | 字符串 | **是** | `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`；不能是保留 id `nora-light` / `nora-dark` |
| `name` | 字符串 | **是** | 非空，≤ 160 字节（一个汉字 3 字节，约 53 个汉字） |
| `version` | 字符串 | **是** | `^\d{1,5}\.\d{1,5}\.\d{1,5}$`，即 `主.次.修订` 三段纯数字，每段 1–5 位；**不能**带 `-beta` 等后缀 |
| `author` | 对象 | 否 | `{"name": "...", "email": "...", "url": "..."}`，`name` ≤ 160 字节。**必须是对象**，写成字符串会被拒 |
| `description` | 字符串 | 否 | ≤ 2000 字节。主题介绍、素材署名都写这里 |
| `tags` | 字符串数组 | 否 | ≤ 12 个，每个 ≤ 64 字节 |
| `license` | 字符串 | 否 | 无格式限制，建议用 SPDX 写法，如 `"MIT"`、`"MIT AND OFL-1.1"` |
| `preview` | 字符串 | 否 | 预览图在包内的相对路径，**文件必须真实存在于包内**，否则 `MISSING_THEME_RESOURCE`。没有预览图就不要写这个字段 |
| `brightnessSupport` | 字符串 | 否 | 商店展示字段：`"light"` / `"dark"` / `"dual"`，见下文 |
| `category` | 字符串 | 否 | 商店分类字段，≤ 64 字符，如 `"nature"`、`"chinese"`、`"simple"` |

### 关于 brightnessSupport

这是**面向商店列表的展示字段**，客户端加载主题时不读它，实际渲染亮度由 `theme.json` 的 `brightness` 决定：

- `"light"`：主题只提供浅色外观。
- `"dark"`：主题只提供深色外观。
- `"dual"`：主题在浅色/深色模式下都可用；当 `theme.json` 只提供一套颜色时，深色外观由客户端**自动派生**。自动派生是**实验性**能力，效果（尤其是背景图、遮罩类字段）未必理想，追求深色质量时应单独提供深色主题（`theme.json` 写 `"brightness": "dark"` + 一套深色颜色）。

### 注意事项

- 长度限制按**字节**计算，中文要按 UTF-8 三倍宽度估算。
- **包里任何地方都不能出现 URL**：内容扫描规则规定任何字符串值含 `://` 或以 `file:`、`data:` 开头都会被拒，所以 `author.url` 写 `https://example.com` 会让整个包上传失败。`author.email` 写纯邮箱地址（不含 `://`）是可以的。
- `id` 用于商店识别同一主题的不同版本；用户设备上的安装目录名由客户端另行生成，与 `id` 无关。
- `id` + `version` 一旦上传就**不可覆盖**：改了任何内容必须提升 `version` 再上传，否则 409 `THEME_VERSION_EXISTS`。

## 内容扫描规则

后台和客户端会递归扫描 `manifest.json` 和 `theme.json` 里的每个键和每个值，任何一层命中即整包拒绝：

| 规则 | 错误码 |
| --- | --- |
| 键名以 `script` 开头或结尾，或以 `lua` 开头（不区分大小写） | `THEME_SCRIPTS_DISABLED` |
| 字符串值含 `://`，或以 `file:`、`data:` 开头 | `EXTERNAL_THEME_RESOURCE` |
| 键名是 `path`、`background`、`backgroundImage`、`preview` 之一，但值不是包内真实存在的文件 | `MISSING_THEME_RESOURCE` |

第三条**只看键名、不看层级**。所以：

- `colors` 里不能出现名为 `background` 的键——它会被当成资源路径检查，而一个颜色值显然不是包内文件；
- `assets.fonts.*.path` 必须指向包内真实字体文件；
- 自定义的任何子对象里如果起了 `path` 这样的键名，同样会触发检查。

设计上记住一句话：**主题包是自包含的，不允许引用包外的任何资源**。网络图片、CDN 字体一律不行，必须打进包里。

## 错误码速查表

| 错误码 | 含义 | 常见原因 |
| --- | --- | --- |
| `THEME_SIZE_LIMIT`（413） | 包体超限 | 超过商店上传上限或 128 MiB 硬上限 |
| `INVALID_THEME_ARCHIVE` | 不是合法 ZIP | 文件损坏、扩展名改了但内容不是 ZIP |
| `UNSUPPORTED_THEME_FILE` | 文件类型不允许 | 混入 `.DS_Store`、`__MACOSX/`、`.txt`、`.md`、`.bak`、`.lua` 等 |
| `UNSAFE_THEME_PATH` | 路径不安全 | `..`、`./`、反斜杠、绝对路径、忽略大小写重名 |
| `THEME_TOO_MANY_FILES` | 文件数超限 | > 512 个文件 |
| `THEME_EXPANSION_LIMIT` | 解压超限 | 单文件 > 32 MiB、压缩比 > 200、总解压 > 128 MiB（zip 炸弹防护） |
| `THEME_MANIFEST_REQUIRED` | 缺 `manifest.json` | 最常见：打包时多套了一层文件夹 |
| `INVALID_THEME_MANIFEST` | manifest 不合法 | `schemaVersion` 不是 2、`id`/`version` 格式错、`author` 写成字符串、长度超限 |
| `THEME_CONFIG_REQUIRED` | 缺 `theme.json` | 文件名拼错 |
| `INVALID_THEME_CONFIG` | theme.json 不是合法 JSON | 语法错误、尾逗号 |
| `THEME_COLORS_REQUIRED` | `colors` 缺失或为空 | `colors` 是唯一必填节 |
| `INVALID_THEME_COLOR` | 颜色格式错误 | 不是 `#RRGGBB` 或 8 位十六进制；`#RGB` 简写不被接受 |
| `THEME_VERSION_EXISTS`（409） | 版本已存在 | 同 `id` 同 `version` 重复上传，需提升版本号 |
| `THEME_SCRIPTS_DISABLED` | 含脚本类字段 | 键名以 `script`/`lua` 开头或以 `script` 结尾 |
| `EXTERNAL_THEME_RESOURCE` | 含外部资源 | 字符串值含 `://` 或以 `file:`/`data:` 开头 |
| `MISSING_THEME_RESOURCE` | 引用的文件不在包内 | `preview`/`backgroundImage`/`path`/`background` 指向的文件不存在 |

能通过商店校验、但仍可能在**客户端安装时**失败的情况：

- 字段类型写错，例如 `motion.normal` 写成 `200.0`（标明整数的字段必须是 JSON 整数）。`player` 节内的字段例外——类型错误只会静默回退默认值，不报错。
- `components` 的子块写成了非对象（数组、字符串等）。

这类错误在客户端显示为"安装失败：…"。

## 打包方式

**推荐：用 `tools/pack.py`**，它与后台/客户端校验规则对齐，打包前完整校验，且产物可重复（条目排序、时间戳固定）：

```bash
python3 tools/pack.py my-theme          # 打包 packages/my-theme
python3 tools/pack.py                    # 打包 packages/ 下全部主题
```

**手动 zip（不推荐，容易踩坑）**：在主题目录内执行

```bash
cd packages/my-theme
zip -r -X ../../dist/my-theme.noratheme manifest.json theme.json assets -x '*.DS_Store' '*.bak'
```

没有 `assets` 目录就去掉命令里的 `assets`。打包后用 `unzip -l my-theme.noratheme` 检查文件清单，确认根目录就是 `manifest.json`、没有 `__MACOSX/` 和 `.DS_Store`。
