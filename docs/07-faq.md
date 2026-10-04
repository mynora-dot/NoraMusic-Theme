# 07 · 常见问题（FAQ）

按出现频率排序。排查通用口诀：**先看拼写，再看类型，最后看路径**。

## 改了 theme.json 为什么不生效？

按可能性从高到低：

1. **键名拼错**。未知键名被**静默忽略**，不报错。常见错拼：`typography`（正确是 `typo`）、`fontSize` 写成 `size`、`accentPressed`（正确是 `accentPress`）。对照 [03-theme-json-reference.md](03-theme-json-reference.md) 的表格逐键检查。
2. **忘了 `schemaVersion: 2`**。`theme.json` 不写时按 schema 1 处理，`motion`、`components`、`glass` 三节会被**整段忽略**。
3. **改完没重新打包/安装**。本地导入不会自动覆盖，重新打包后再导入一次。
4. **该字段本就是保留字段**：`assets.backgroundLayers`、`assets.textures`、`assets.particles`、`FontAsset.weights` 能解析但不产生任何效果。
5. **`player` 节字段类型错误**：`player` 内类型错了不报错、静默回退默认值，见下面"毫秒与颜色字段"。

## 颜色设置了但没生效？

- **用了 Material 风格键名**：`primary`、`onPrimary`、`error`、`outline`、`background` 等都不生效。NoraMusic 只认 26 个语义键（`accent`、`textPrimary`、`canvas`……），完整表见 [03](03-theme-json-reference.md#colors)。
- **`colors.background` 是违禁键**：它会被内容扫描当成资源路径，导致整包被拒（`MISSING_THEME_RESOURCE`），不是"没生效"而是"装不上"。
- **alpha 写错位置**：8 位颜色是 `#AARRGGBB`（透明度在前）。`#7FA37A1A` 想表达"10% 透明的绿"，实际会被解析成"以 0x7F 为透明度的另一种颜色"。半透明应写 `#1A7FA37A`。
- **`#RGB` 三位简写不支持**，必须 6 位或 8 位。

## 播放页背景设了 coverBlur / coverGradient / image，但还是一块纯色？

这是 `lightBackgroundBase` 的 alpha 陷阱：播放页遮罩颜色由它推导，默认值完全不透明，会把背景整个盖住。给它一个带透明度的 ARGB 十进制整数即可：

```bash
printf '%d\n' 0x73140D08   # 1930693896
```

```json
"player": { "lightBackgroundBase": 1930693896 }
```

浅色主题配深色遮罩时别忘了 `"foreground": "light"`，否则深色遮罩上是深色文字。详见 [04-player-scene.md](04-player-scene.md#遮罩层base-色--overlay-不透明度)。

## 黑胶效果怎么配？为什么我的封面外面又套了一层唱片？

客户端有**两套**黑胶：`vinylEnabled`（默认 `true`）是套在封面**外面**的唱片框装饰；`cover.shape: "vinyl"` 是把封面**本身**画成黑胶唱片。两个都开就会"唱片套唱片"。想要封面即黑胶（如 `vinyl` 预设），请同时写 `"vinylEnabled": false, "tonearmEnabled": false`；想要默认那种唱片框效果则不用动任何配置。机制详解见 [04-player-scene.md](04-player-scene.md#黑胶专题两套黑胶机制)。

## 光晕颜色、渐变背景颜色能自己指定吗？

分情况：`coverGradient` 背景、`sweep`/`glow` 光晕的颜色都来自**当前歌曲封面取色**，主题无法指定（每首歌不同）；想要固定颜色用 `background.mode: "themeGradient"` + `background.colors` 数组。歌词高亮色用 `colors.accent` / `accentPress`，是可以指定的。

## 毫秒、颜色这些字段到底写什么类型？

- 整数毫秒（`haloDuration`、`breathDuration`、`flowDuration`、`spinDuration` 及 `motion.*` 时长）：写 JSON **整数** `16000`，不要写 `16000.0`。
- `player` 下 6 个颜色字段（`lightBackgroundBase`、`darkBackgroundBase`、`lyrics*Overlay*` 四个）：写 **ARGB 十进制整数**，用 `printf '%d\n' 0xAARRGGBB` 换算。
- 其余颜色一律写 `#RRGGBB` / `#AARRGGBB` 字符串。
- `player` 节内类型错了只回退默认值；`player` 节外（如 `motion.normal: 200.0`）类型错误会导致**安装失败**。

## 深色主题怎么做？

- `theme.json` 写 `"brightness": "dark"`，并提供一套为深色设计的颜色（只改 `brightness` 不改颜色会得到"深色壳的浅色主题"）。
- manifest 的 `brightnessSupport` 只是商店展示字段，客户端加载不读它。
- `brightnessSupport: "dual"` 表示深浅都能用；只有一套颜色时深色由客户端**自动派生**，属实验性，效果（尤其背景图和遮罩）未必理想。认真做深色就单独出一个 `brightness: "dark"` 的主题包。

## 本地能不能导入主题？一定要走商店吗？

可以。**设置 → 主题 → 本地导入**直接选 `.noratheme` 文件即可；也支持 **HTTPS 链接导入**（仅 https、443 端口、≤3 次重定向、≤128 MiB、公网地址）。商店只是面向全量用户的正式分发渠道，见 [06-publishing.md](06-publishing.md)。

## 自定义字体没生效？

字体加载失败**静默回退系统字体**，按顺序检查：

1. `assets.fonts.*.path` 指向的文件确实在包里，路径大小写完全一致；
2. `family` 声明的名字与引用处（顶层 `fontFamily` 或 `typo.*.fontFamily`）拼写一致；
3. 文件是合法 `.ttf` / `.otf`（扩展名白名单只允许这两种）；
4. 记住**注册 ≠ 生效**：`assets.fonts` 只负责注册，必须再被 `fontFamily` 引用才显示；
5. 多字重需求：一个文件只注册一种字重，`weights` 字段暂不生效；每个字重需要独立文件 + 独立 `family`。
6. 部分字形缺失（日文假名、间隔号 `·` 等）会按系统字体回退，表现为混排——这是字体本身的字符覆盖问题，不是配置错误。

## 字体许可证文件怎么放进包里？

放不进。扩展名白名单只有 `.json .png .jpg .jpeg .webp .ttf .otf`，`.txt` 会被拒。OFL 字体的署名写在 `manifest.json` 的 `license`（如 `"MIT AND OFL-1.1"`）和 `description` 里，见 [05-assets-and-fonts.md](05-assets-and-fonts.md#字体许可与署名)。

## 打包/上传报错了，错误码什么意思？

完整错误码表见 [02-package-format.md](02-package-format.md#错误码速查表)。最高频的几个：

| 错误码 | 一句话原因 | 怎么修 |
| --- | --- | --- |
| `THEME_MANIFEST_REQUIRED` | 根目录没有 `manifest.json` | 打包时多套了一层文件夹，进目录再 zip |
| `UNSUPPORTED_THEME_FILE` | 混入了 `.DS_Store` / `__MACOSX/` / `.txt` 等 | 别用访达压缩；用 `pack.py`；`unzip -l` 检查清单 |
| `INVALID_THEME_MANIFEST` | manifest 字段不合法 | 查 `schemaVersion: 2`、`id` 正则、`author` 必须是对象 |
| `THEME_COLORS_REQUIRED` | `colors` 缺失或空 | 它是 theme.json 唯一必填节 |
| `INVALID_THEME_COLOR` | 颜色格式错 | 只接受 `#RRGGBB` 或 `#AARRGGBB` |
| `THEME_VERSION_EXISTS`（409） | 同 id 同 version 重复上传 | 提升 `version` |
| `EXTERNAL_THEME_RESOURCE` | 包里出现了 URL | 删掉所有含 `://` 的字符串，包括 `author.url` |
| `MISSING_THEME_RESOURCE` | `preview`/`path`/`background`/`backgroundImage` 指向的文件不在包里 | 补文件或删字段；注意 `colors.background` 也会触发 |

## 包太大超限怎么办？

- 硬上限：整包 ≤128 MiB，解压后 ≤128 MiB，单文件 ≤32 MiB；商店上传默认 32 MiB（管理员可调至 1–128 MiB）。
- 大头永远是**字体**（1–10 MiB/个）和**背景图**。优化：图片转 webp/降质量 jpg、长边 ≤2000px；字体做子集化（`pyftsubset`，注意 OFL 的 Reserved Font Name 条款要求改名）或换更小的字体。
- `pack.py` 打包产物超过 32 MiB 时会打印提醒（不阻断），超过后台实际上传上限才会被拒。

## 为什么商店里我的主题色块不对？

商店列表色块取自 `colors.canvas`、`colors.accent`、`colors.textPrimary`。只写一部分时，缺的那个会显示 Nora Light 默认色。三个都写上。

## 更新主题后用户端会自动升级吗？

不会。客户端没有自动更新：新版本作为商店里的新条目出现，用户需手动下载；旧版本在被管理员撤回前一直可用。撤回后已安装用户会在下一次授权校验失败时回退到内置主题。
