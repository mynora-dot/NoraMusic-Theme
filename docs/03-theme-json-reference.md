# 03 · theme.json 全字段参考

`theme.json` 定义主题的全部样式。本篇按节逐一说明。播放页（`player` 节）内容较多，单独放在 [04-player-scene.md](04-player-scene.md)；背景图与字体（`backgroundImage` / `assets` / `fontFamily`）放在 [05-assets-and-fonts.md](05-assets-and-fonts.md)。

## 通用规则

- **所有节都可以省略**，省略后回退到内置浅色主题 **Nora Light** 的默认值。唯一的例外是 `colors`：必填且不能为空。
- **拼错的键名会被静默忽略**，不会报错——这是"改了没生效"的最常见原因，写完务必对照本篇表格检查拼写。
- 两个 JSON 里都要写 `"schemaVersion": 2`。`theme.json` 不写时按 1 处理，`motion`、`components`、`glass` 三节会被**整段忽略**。
- 数值字段写 JSON 数字；标明"整数"的字段不能写成 `200.0`，否则客户端安装失败（`player` 节例外，类型错误只回退默认值）。

## 顶层字段一览

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `schemaVersion` | 整数 | 写 `2` |
| `colors` | 对象 | **必填且非空**，见下文 |
| `brightness` | 字符串 | `"light"`（默认）或 `"dark"`，见下文 |
| `typo` | 对象 | 排版（8 种文字样式）。注意键名是 `typo` 不是 `typography` |
| `space` | 对象 | 间距阶梯 |
| `shape` | 对象 | 圆角阶梯 |
| `heights` | 对象 | 固定尺寸 |
| `motion` | 对象 | 动效时长与曲线（需 schema 2） |
| `glass` | 对象 | 毛玻璃参数（需 schema 2） |
| `components` | 对象 | 组件级参数（需 schema 2） |
| `player` | 对象 | 播放页参数与场景，见 [04-player-scene.md](04-player-scene.md) |
| `backgroundImage` | 字符串 | 全局背景图包内路径 |
| `backgroundImageOpacity` | 数字 | 背景图不透明度，默认 `0.3` |
| `backgroundImageBlur` | 数字 | 背景图模糊半径，默认 `0` |
| `fontFamily` | 字符串 | 全局字体族名，应用到所有文字样式 |
| `assets` | 对象 | 资源声明，目前 `background` 和 `fonts` 生效 |

`theme.json` 里的 `id`、`name` 写了也没用——安装时会被商店/manifest 信息覆盖。

## brightness：深色主题

```json
{
  "brightness": "dark",
  "colors": { "canvas": "#141416", "...": "..." }
}
```

- 写 `"dark"` 时主题按深色渲染；不写或写 `"light"` 按浅色渲染。值只能是这两个字符串，其他值会被拒绝。
- 深色主题应当提供一整套为深色设计的颜色（至少把 `canvas`、`surface`、`textPrimary` 等表面与文字色改深/改浅），只改 `brightness` 不改颜色会得到一个"深色模式的浅色主题"。
- manifest 里的 `brightnessSupport: "dual"` 只是商店展示声明；客户端的深色派生是实验性的。认真做深色就请单独做一个 `brightness: "dark"` 的主题包。

## colors

颜色值写成 `#RRGGBB` 或 `#AARRGGBB`。**8 位写法的前两位是透明度（alpha 在前）**：`#1A7FA37A` = 10% 不透明的绿色。不透明色两种写法等价：`#7FA37A` ≡ `#FF7FA37A`。

要点：

- `#RGB` 三位简写不被接受。
- 按 `#RRGGBBAA`（alpha 在后）写的颜色可能通过商店上传校验，但客户端固定把前两位当透明度，**显示颜色会错**，务必按 `#AARRGGBB` 写。
- `colors` 里每个值（包括你不认识的自定义键）都必须是合法颜色字符串，否则 `INVALID_THEME_COLOR`。
- `colors` 里**不能出现名为 `background` 的键**——内容扫描会把它当资源路径检查，报 `MISSING_THEME_RESOURCE`。
- 不认识的键名（如 `primary`、`onPrimary`、`error`、`outline` 这类 Material 风格命名）**不会生效**，静默忽略。请只用下表 26 个键。
- 没写的键回退到 Nora Light 默认值。

### 26 个颜色键

| 键 | 用途 | 默认值 |
| --- | --- | --- |
| `canvas` | 页面底色 | `#FAF9F7` |
| `canvasElevated` | 浮起的底色，如弹层 | `#FFFFFF` |
| `surface` | 卡片、列表块 | `#FFFFFF` |
| `surfaceVariant` | 次级卡片、分组背景 | `#F2F0EC` |
| `surfaceSunken` | 凹陷区域，如输入框、进度条底 | `#F2F0EC` |
| `hairline` | 细分割线 | `#12000000` |
| `hairlineStrong` | 较明显的描边 | `#1F000000` |
| `textPrimary` | 主文字 | `#17181A` |
| `textSecondary` | 次要文字 | `#6B6E76` |
| `textTertiary` | 弱化文字、占位符 | `#9A9CA3` |
| `textOnAccent` | 强调色上的文字 | `#FFFFFF` |
| `accent` | 强调色：按钮、选中态 | `#2F6BFF` |
| `accentPress` | 强调色按下态 | `#2455D6` |
| `accentSubtle` | 强调色浅底，用于选中背景 | `#1A2F6BFF` |
| `success` | 成功色 | `#1F9D5B` |
| `successSubtle` | 成功色浅底 | `#1A1F9D5B` |
| `warning` | 警告色 | `#C77A16` |
| `warningSubtle` | 警告色浅底 | `#1AC77A16` |
| `danger` | 错误、危险色 | `#D64545` |
| `dangerSubtle` | 危险色浅底 | `#1AD64545` |
| `scrim` | 弹窗遮罩 | `#73000000` |
| `shadow` | 阴影色，同时决定全局投影 | `#0F000000` |
| `currentPlaying` | 正在播放的歌曲高亮 | `#2F6BFF` |
| `rankFirst` | 排行榜第 1 名 | `#E0A32E` |
| `rankSecond` | 排行榜第 2 名 | `#9AA3AF` |
| `rankThird` | 排行榜第 3 名 | `#C08457` |
| `favorite` | 收藏/喜欢标记（红心） | `#FF4081` |

配色建议：

- `accentSubtle` / `successSubtle` / `warningSubtle` / `dangerSubtle` 习惯上用对应主色 + `1A`（10%）透明度。
- `hairline` / `hairlineStrong` / `scrim` / `shadow` 用主文字色（或黑色）加透明度的写法，能自动贴合各种底色。
- 商店列表的主题色块取自 `canvas`、`accent`、`textPrimary` 三个键，即使走商店分发也建议都写上。

## typo（排版）

8 种文字样式，每种是一个对象，对象内字段均可省略：

| 样式 | 默认字号 / 字重 / 行高 | 用途 |
| --- | --- | --- |
| `display` | 28 / 700 / 1.2，字间距 -0.5 | 大标题 |
| `title` | 22 / 700 / 1.25，字间距 -0.2 | 页面标题 |
| `heading` | 17 / 600 / 1.3 | 区块标题 |
| `body` | 15 / 500 / 1.4 | 正文 |
| `bodyStrong` | 15 / 600 / 1.4 | 加粗正文、歌名 |
| `caption` | 13 / 400 / 1.4 | 说明文字、歌手名 |
| `label` | 11 / 500 / 1.2，字间距 0.3 | 小标签 |
| `numeric` | 13 / 600 / 1.2 | 时长、计数等数字 |

样式对象内可用字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `fontFamily` | 字符串 | 字体族名（需配合 `assets.fonts` 注册，见 [05](05-assets-and-fonts.md)） |
| `fontSize` | 数字 | 字号，逻辑像素 |
| `fontWeight` | 整数 | 100–900 的**整百数**，其他值被忽略 |
| `letterSpacing` | 数字 | 字间距 |
| `height` | 数字 | 行高倍数 |

```json
"typo": {
  "title": { "fontSize": 24, "fontWeight": 800 },
  "body": { "fontSize": 15, "height": 1.5 }
}
```

## space / shape / heights

单位都是逻辑像素；唯一例外是 `heights.coverLarge`，它是**占屏幕宽度的比例**。

| 节 | 字段及默认值 |
| --- | --- |
| `space` 间距 | `xxs` 2, `xs` 4, `sm` 8, `md` 12, `lg` 16, `xl` 20, `xxl` 24, `huge` 32, `pageH` 20（页面左右边距） |
| `shape` 圆角 | `xs` 6, `sm` 10, `md` 14, `lg` 20, `pill` 999 |
| `heights` 尺寸 | `navBar` 56, `miniPlayer` 60, `rowCompact` 56, `rowComfortable` 68, `appBarCompact` 44, `touchTarget` 48, `coverLarge` 0.62, `iconSm` 16, `iconMd` 20, `iconLg` 24 |

```json
"space": { "pageH": 24, "md": 14 },
"shape": { "sm": 12, "md": 16, "lg": 24 }
```

阴影没有独立字段，由 `colors.shadow` 自动生成：向下偏移 4、模糊半径 16。想让界面更"平"，把 `shadow` 的透明度调低即可。

## motion（动效）

需要 `schemaVersion: 2`。

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `fast` | 整数（毫秒） | 150 | 快速反馈动画 |
| `normal` | 整数（毫秒） | 200 | 常规动画 |
| `slow` | 整数（毫秒） | 280 | 慢速动画 |
| `pageTransition` | 整数（毫秒） | 260 | 页面转场 |
| `stagger` | 整数（毫秒） | 30 | 列表逐项出现的间隔 |
| `enter` | 曲线名 | `easeOut` | 入场曲线 |
| `exit` | 曲线名 | `easeIn` | 出场曲线 |
| `emphasis` | 曲线名 | `easeOutCubic` | 强调曲线 |
| `pressScale` | 数字 | 0.98 | 按下时的缩放 |
| `enterOffset` | 数字 | 4 | 入场位移 |

可用的曲线名共 20 个：

- `linear`
- `easeIn`、`easeOut`、`easeInOut`
- `easeInCubic`、`easeOutCubic`、`easeInOutCubic`
- `easeInQuad`、`easeOutQuad`、`easeInOutQuad`
- `easeInQuart`、`easeOutQuart`、`easeInOutQuart`
- `fastOutSlowIn`、`slowMiddle`
- `bounceIn`、`bounceOut`、`bounceInOut`
- `elasticIn`、`elasticOut`、`elasticInOut`

不认识的曲线名回退默认值。系统开启"移除动画"（无障碍设置）时，客户端会自动停用大部分动效，主题无需处理。

## glass（毛玻璃）

需要 `schemaVersion: 2`。导航栏、迷你播放器等浮层的毛玻璃质感参数：

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `lightBlur` / `mediumBlur` / `heavyBlur` | 10 / 20 / 30 | 三档模糊半径 |
| `lightOpacityLight` / `mediumOpacityLight` / `heavyOpacityLight` | 0.7 / 0.85 / 0.9 | 浅色主题下三档底色不透明度 |
| `borderOpacityLight` | 0.5 | 浅色主题下描边不透明度 |
| `borderWidth` | 0.5 | 描边宽度 |
| `shadowBlur` / `shadowOffsetX` / `shadowOffsetY` | 24 / 0 / 8 | 投影模糊与偏移 |
| `shadowOpacityLight` | 0.08 | 浅色主题下投影不透明度 |
| `pressScale` | 0.96 | 按压缩放 |

glass 中还有一组以 `Dark` 结尾的对应字段（如 `lightOpacityDark`），用于深色主题；浅色主题只需设置上表字段。

## components（组件参数）

需要 `schemaVersion: 2`。两种缺省情况的回退行为**不一样**，理解它能少踩坑：

- **整个子块不写**：数值从 `space` 和 `shape` 推导。例如改了 `shape.sm`，列表封面圆角会跟着变。
- **写了子块但缺字段**：缺的字段用下表固定默认值，**不再**跟随 `space` / `shape`。

| 子块 | 字段及默认值 |
| --- | --- |
| `button` | `height` 48, `tonalHeight` 44, `ghostHeight` 44, `paddingH` 20, `iconGap` 8, `iconSize` 20, `borderWidth` 0 |
| `chip` | `height` 34, `paddingH` 14, `gap` 6, `iconSize` 16, `borderWidth` 0, `selectedBorderWidth` 1, `radiusFactor` 1.0 |
| `input` | `height` 48, `paddingH` 16, `borderWidth` 0, `focusedBorderWidth` 1, `iconSize` 20, `gap` 10 |
| `list` | `showDividers` true, `dividerIndent` 0, `dividerThickness` 1, `coverSize` 48, `coverRadius` 10, `gap` 12, `leadingBarWidth` 3, `leadingBarRadius` 2 |
| `nav` | `iconSize` 24, `selectedIconSize` 24, `labelGap` 3, `indicatorHeight` 32, `indicatorWidth` 64, `blurSigma` 12, `surfaceOpacity` 0.92, `hairlineWidth` 1 |
| `miniPlayer` | `coverSize` 40, `coverRadius` 8, `gap` 12, `buttonSize` 40, `iconSize` 24, `progressThickness` 2 |
| `sheet` | `radius` 20, `handleWidth` 36, `handleHeight` 4, `handleGap` 12, `maxHeightFactor` 0.9, `dragDownClose` true |
| `dialog` | `radius` 20, `padding` 24, `actionGap` 12 |
| `cover` | `radius` 14, `placeholderStyle`（`initial` / `icon` / `plain`，默认 `initial`）, `showRingWhenPlaying` true |
| `player` | 字段同播放页参数；目前只有 `progressThickness` 和 `thumbSize` 生效（全局进度条滑块） |

```json
"components": {
  "list": { "showDividers": false, "coverRadius": 12 },
  "sheet": { "radius": 28 }
}
```

注意：解析器只读取上表中的字段名。主程序内置主题文件里存在一些解析器并不读取的历史字段（如 `button.iconSpacing`、`list.dividerHeight`、`cover.radiusSmall`），**不要照抄**，以上表为准。

## assets 中的保留字段

`assets.backgroundLayers`、`assets.textures`、`assets.particles` 以及字体声明里的 `weights` 字段**可以被解析但暂不生效**，属于为未来能力保留的字段。不要在主题里依赖它们的效果。
