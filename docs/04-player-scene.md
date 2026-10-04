# 04 · 播放页与歌词页定制（player 节）

播放页是主题最有辨识度的部分。本篇逐个参数说明**它控制什么、在哪个页面哪个部件上生效**，所有结论均来自客户端源码（`mobile/lib/features/player/` 与 `mobile/lib/theme/tokens/`）。

## 配置写在哪里

播放页相关的**全部**配置写在 `theme.json` 顶层的 `"player"` 一个对象里。这个对象同时容纳两类字段：

- **场景字段**（枚举类）：`preset`、`background`、`cover`、`lyrics`、`lyricsBackground`、`layout`、`foreground` —— 决定播放页"长什么样"。
- **参数字段**（数值/布尔/ARGB 颜色）：`coverSize`、`haloBlur`、`lightBackgroundBase`、`vinylEnabled` 等 —— 决定各部件的具体尺寸与强度。

```json
{
  "schemaVersion": 2,
  "colors": { "...": "..." },
  "player": {
    "preset": "vinyl",
    "background": { "mode": "coverGradient", "flow": true },
    "cover": { "spinDuration": 20000 },
    "lightBackgroundBase": 1930693896
  }
}
```

> 注意：`components.player` 是另一个独立对象，目前只有 `progressThickness` / `thumbSize` 两个字段生效（作用于全局滑块样式），**播放页配置不要写在那里**。

## 配置作用于哪些页面

| 页面 | 路径 | 读取的配置 |
| --- | --- | --- |
| 播放页（封面页） | `/player` | 全部字段 |
| 全屏歌词页 | `/player/lyrics` | `lyricsBackground`（缺省沿用 `background`）、`lyrics*` 系列参数、`lyrics` 歌词样式、控制区参数 |
| 播放页内嵌歌词视图 | `/player` 内点封面外的歌词区 | `lyrics` 歌词样式、`lyricsFontSize*` 三档字号 |
| 迷你播放条 | 各页面底部 | **不读** `player` 节，样式由 `components.miniPlayer` 与全局颜色决定 |

播放页自上而下分三层理解，参数也按这三层组织：

1. **背景层** —— `background.mode` 决定的整屏画面（模糊封面 / 渐变 / 图片 / 纯色）。
2. **遮罩层** —— 压在背景上、保证文字可读的渐变色，由 `lightBackgroundBase` / `darkBackgroundBase` / `backgroundOverlayOpacity` 决定。
3. **内容层** —— 顶栏、封面舞台（含黑胶/光晕/呼吸）、歌曲信息、进度条、控制区，由其余字段决定。

## 解析顺序：预设 → 显式字段

先按 `preset` 展开一整套场景值，再用你显式写出的字段逐项覆盖（`background` / `cover` / `lyrics` 是逐字段合并，不是整块替换）。不写 `preset` 时以默认场景为底，与旧版播放页一致。

### 五种预设展开后的实际值

| 预设 | background | cover | lyrics | layout |
| --- | --- | --- | --- | --- |
| `standard`（默认） | `coverBlur` | 圆角 + `sweep` 光晕 | 渐变高亮 | standard |
| `immersive` | `coverBlur` | 圆角 + `glow` 光晕 | `solid` 高亮、`activeScale` 7、`blurInactive` 1.2、`inactiveOpacity` [0.55, 0.35, 0.2] | **immersive** |
| `vinyl` | `coverGradient` | **黑胶** + 无光晕 + `spin` | 渐变高亮 | standard |
| `aurora` | `coverGradient` + `flow` | 圆角 + `glow` 光晕 | `glow` 高亮 | standard |
| `minimal` | `solid` | **方形** + 无光晕 | `solid` 高亮 + 左对齐 + `activeScale` 3 | standard |

预设只展开场景字段；`coverSize`、`lightBackgroundBase` 这类参数字段不受预设影响，永远按你自己的值或默认值来。

## 背景层：background 的六种模式

```json
"background": {
  "mode": "coverBlur",
  "flow": false,
  "flowDuration": 18000,
  "colors": [],
  "backgroundImage": "assets/backgrounds/bg.webp",
  "imageBlur": 0
}
```

| 字段 | 类型 | 默认值 | 取值范围 | 作用范围 |
| --- | --- | --- | --- | --- |
| `mode` | 枚举 | `coverBlur` | 下表六选一 | 播放页与歌词页（歌词页可被 `lyricsBackground` 覆盖）的整屏背景 |
| `flow` | 布尔 | `false` | — | 仅 `coverGradient` / `themeGradient`：光斑是否缓慢漂移；只在**播放中**且系统未开"移除动画"时运行 |
| `flowDuration` | 整数毫秒 | `18000` | 4000–120000（截断） | 漂移一个周期的时长 |
| `colors` | 颜色数组 | `[]` | `#RRGGBB` / `#AARRGGBB` / ARGB 整数 | `themeGradient` 的光斑颜色；`solid` 取第一个；其他模式忽略 |
| `backgroundImage` | 包内路径 | 无 | 必须存在于包内 | 仅 `image` 模式 |
| `imageBlur` | 数字 | `0` | 0–60（截断） | 仅 `image` 模式的高斯模糊强度 |

### 六种 mode 的渲染原理

**`coverBlur`（默认）—— 模糊封面**
把当前歌曲封面放大铺满全屏，施加高斯模糊（模糊半径 = 参数字段 `backgroundBlurIntensity`，默认 70）。封面随歌曲切换，取不到封面时回退为封面取色的柔和对角渐变。

**`coverFocus` —— 中心清晰、边缘渐隐（类 Apple Music）**
两层叠加：底层是把封面放大 1.25 倍后高斯模糊（半径同样取 `backgroundBlurIntensity`）；上层是一张完整清晰的方形封面（按宽度铺满、不裁切），套用竖向渐隐遮罩（顶部 0–18% 淡入、18–70% 全清晰、70–100% 淡出），让封面上下边缘柔和融进底层模糊中。封面取不到时两层都回退为取色渐变。

**`coverGradient` —— 封面取色流光渐变**
客户端从封面提取主色板，在"主色压暗 35%"的底色上画 3 团径向光斑（位置固定在左上、右上、下方），`flow: true` 时光斑以 `flowDuration` 为周期缓慢漂移。完全不依赖主题色，每首歌画面都不同。`aurora`、`vinyl` 预设用它。

**`themeGradient` —— 主题色流光渐变**
与 `coverGradient` 相同的渲染方式，但光斑颜色来自你自己的 `colors` 数组（建议 2–3 个颜色），不随歌曲变化。`colors` 为空时回退到封面取色，行为退化为 `coverGradient`。

**`image` —— 主题背景图**
整屏铺满包内图片（`BoxFit.cover` 居中裁切），可用 `imageBlur` 加模糊。图片路径安装时会解析为主题目录下的绝对路径。加载失败回退为封面取色渐变。**歌词页若也用图片背景，遮罩不再混入封面色**（见下文"歌词页遮罩"），主题画面不会被歌曲取色污染。

**`solid` —— 纯色**
`colors` 第一个值铺满；`colors` 为空时用封面取色主色向白色混合 82% 的浅色。

> 所有模式之上都会再叠一层遮罩（下一节），所以"背景很亮但文字依然可读"是遮罩的功劳，不是背景模式的。

## 遮罩层：base 色 + overlay 不透明度

遮罩是背景层之上的竖向线性渐变，作用是压住背景、保证文字可读。它由三个参数控制：

| 字段 | 类型 | 默认值 | 作用范围 |
| --- | --- | --- | --- |
| `lightBackgroundBase` | ARGB 整数 | `0xFFF7F7F5`（不透明米白） | **浅色主题下**播放页遮罩的底色 |
| `darkBackgroundBase` | ARGB 整数 | `0xFF06090E`（不透明墨黑） | **深色主题下**播放页遮罩的底色 |
| `backgroundOverlayOpacity` | 数字 0–1 | `0.4` | 遮罩向白（浅色）/黑（深色）混合的强度，越大背景越"灰"、越平淡 |

选 `light` 还是 `dark` 底色由**主题自身的亮度**决定（`brightness` / 深色派生），与 `foreground` 无关。

### alpha 双分支（最重要的规则）

客户端先看 base 色的**透明度**，走两条完全不同的路：

- **base 不透明（alpha ≥ 0.99）**：遮罩 = base 色**混入当前封面取色**的渐变（浅色：顶部混主色 55%、底部混辅色 28%，再按 `backgroundOverlayOpacity` 向白柔化；深色：顶部混 40%、底部混 20%，向黑压深）。效果：背景几乎被遮罩盖死，只剩淡淡的封面色调——内置主题就是这样。
- **base 半透明（alpha < 0.99）**：尊重你的透明度，遮罩退化为"base 色向白/黑轻微混合"的两段渐变（顶部按 `backgroundOverlayOpacity × 0.5`、底部 × 0.7 混合，底部 alpha 再放大 1.4 倍封顶不透明）。**背景层的模糊封面/渐变/图片能直接透出来**——这才是 `coverBlur`、`image` 等模式想要的搭配。

> **想让播放页背景透出来，必须把 base 色写成半透明。** 表现"设了 `background.mode` 却还是一块纯色"，99% 是 `lightBackgroundBase` 用了默认的不透明值。

### ARGB 整数的写法与换算

这 6 个播放页颜色字段（`lightBackgroundBase`、`darkBackgroundBase`、`lyricsLightOverlayStart/End`、`lyricsDarkOverlayStart/End`）是 **0–4294967295 的十进制整数**（格式 `0xAARRGGBB`），不是 `#` 字符串。换算：

```bash
printf '%d\n' 0x73140D08    # 45% 透明的暖黑
# 1930693896
```

```json
"player": {
  "background": { "mode": "coverBlur" },
  "lightBackgroundBase": 1930693896
}
```

参考手感：alpha `0x5C`(36%)～`0x8C`(55%) 是"透出背景"的常用区间；`0xBD`(74%)+ 适合歌词页。

## 前景明暗：foreground

| 值 | 行为 |
| --- | --- |
| `auto`（默认） | 跟随主题亮度：深色主题用浅色文字，浅色主题用深色文字 |
| `light` | 强制浅色前景（白字白图标）——配半透明**深色**遮罩使用 |
| `dark` | 强制深色前景 |

`foreground` 只改变内容层的配色，**不改变遮罩颜色的选择**（遮罩仍按主题亮度选 light/dark base）。所以它必须和半透明 base 色配套使用：浅色主题做深色沉浸播放页 = `lightBackgroundBase` 半透明深色 + `foreground: "light"`，缺一不可（缺前者背景压不暗，缺后者深底上是深字）。

具体影响的元素与色值：

| 元素 | 深色背景（light 前景） | 浅色背景 |
| --- | --- | --- |
| 歌名 / 主文字 | `#F7F6F3` | `#171717` |
| 歌手 / 副文字 / 时间 | `#D4D6DB` | `#666666` |
| 上一首/下一首/收藏/更多 | `#E2E4E8` | `#202328` |
| 循环/队列等次要按钮 | `#AEB3BB` | `#555A61` |
| 顶栏圆形按钮底色 | 黑 25% | 白 55% |

## 布局：layout

| 值 | 差异 |
| --- | --- |
| `standard`（默认） | 经典布局：顶栏（收起 + "正在播放"标题）→ 封面舞台 → 歌曲信息行（浮动当前歌词 + 歌名 + 收藏/更多）→ 进度条 → 控制区 |
| `immersive` | 沉浸布局：**没有独立封面部件**（cover/halo/breath/vinyl 全部不渲染），顶栏只剩收起按钮，当前歌词浮动在左下角歌曲信息上方，点击中部任意位置进全屏歌词页。`immersive` 预设自带此布局 |

`layout` 独立于 `background.mode`：immersive 布局配 `coverFocus` 背景就是"类 Apple Music"的完整形态。

## 封面舞台：cover 与相关参数

封面舞台 = 光晕 + 封面本体（+ 黑胶/呼吸/旋转动画），仅在 `standard` 布局渲染。

### shape：封面形态

| 值 | 渲染 | 相关参数 |
| --- | --- | --- |
| `rounded`（默认） | 圆角矩形 | 圆角取 `coverBorderRadius`（默认 34） |
| `square` | 直角方形 | — |
| `circle` | 圆形 | — |
| `vinyl` | **封面自身渲染成黑胶唱片**：盘面直径 = `coverSize` × 0.70，中间圆形封面标签 = 盘面 × 0.62，盘面带沟槽与反光，另有一支画出来的唱针随播放/暂停以 460ms 动画落下/抬起 | 旋转由 `cover.spin` 控制 |

封面阴影（所有形态通用）：`coverShadowBlur`（默认 80）、`coverShadowOpacity`（默认 0.5），阴影色深色主题为黑、浅色主题为 `colors.surfaceSunken`，向下偏移 = 模糊半径 × 0.3。

### halo：光晕

| 值 | 渲染 |
| --- | --- |
| `sweep`（默认） | 封面取色的锥形渐变，**播放时**以 `haloDuration` 为周期旋转，经 `haloBlur` 高斯模糊，形态跟随封面（圆/圆角） |
| `glow` | 静态径向柔光，无动画，尺寸可超出封面（`coverSize` × `haloSize`） |
| `none` | 无光晕 |

光晕实际透明度 = 基准值 ×（`haloOpacity` / 0.3）。基准值：浅色主题播放中 0.32、暂停 0.12；深色主题播放中 0.55、暂停 0.20。即 `haloOpacity` 默认 0.3 时等比，调大更亮。`haloEnabled: false` 时无论 `halo` 写什么一律不显示。光晕颜色取自当前封面取色，主题无法直接指定。

### spin：旋转

`spin: true` 时封面**播放中**匀速旋转（周期 `spinDuration`，默认 24000ms，范围 4000–120000），暂停停在当前角度。`shape: "vinyl"` 时是盘面旋转；其他形态是整个封面矩形旋转（视觉上只建议配 `circle`/`vinyl`）。

### 呼吸与其他舞台参数

| 字段 | 默认值 | 作用 |
| --- | --- | --- |
| `coverSize` | 262 | 封面舞台边长（逻辑像素）。黑胶形态时盘面 = 它 × 0.70 |
| `breathEnabled` | `true` | 呼吸动画开关：播放中封面在 1.0 ↔ `breathScale` 之间缓慢缩放 |
| `breathDuration` | 4500（整数毫秒） | 呼吸一个来回的周期 |
| `breathScale` | 1.02 | 呼吸放大到的比例 |

光晕/呼吸/旋转只在**播放中**运行，系统开"移除动画"时全部自动停止。

## 黑胶专题：两套黑胶机制

客户端有**两层**黑胶装饰，来源不同、可独立开关，这是最容易混淆的地方：

**① 外层黑胶框（参数字段控制，默认开启）**
`vinylEnabled`（默认 `true`）为真时，整个封面舞台被包在一个黑胶唱片装饰里（`VinylRecord`）：外圈是带沟槽纹理和反光的黑胶盘，右上角有一支唱臂。它是"封面外面套个唱片框"，**任何 shape 都会被它包住**。

| 字段 | 默认值 | 作用 |
| --- | --- | --- |
| `vinylEnabled` | `true` | 外层黑胶框总开关 |
| `vinylSize` | 290 | 外层黑胶的目标直径，实际取 `min(vinylSize, 可用高 × 0.92, 可用宽 × 0.78)` |
| `vinylRotationDuration` | 20000（整数毫秒） | 外层黑胶播放时旋转一圈的周期 |
| `tonearmEnabled` | `true` | 外层唱臂是否显示 |
| `tonearmRotationAngle` | 20（度） | 外层唱臂随播放状态抬落的角度 |

**② 内层黑胶封面（场景字段控制，`cover.shape: "vinyl"`）**
封面本体直接画成黑胶唱片（见上一节 shape 表），唱针为封面内绘制、随播放/暂停抬落。

**搭配建议**：

- 想要"唱片框里的方形/圆形封面"（默认应用的样子）：保持 `vinylEnabled: true`，shape 用 `rounded`/`circle`。
- 想要"封面本身就是黑胶"（`vinyl` 预设、官方 `opera-velvet-dual` 的做法）：`cover.shape: "vinyl"`，**同时设 `vinylEnabled: false`、`tonearmEnabled: false`**，避免外层唱片框和内层黑胶叠成双重装饰。

## 歌词样式：lyrics

同时作用于两处：**播放页内嵌歌词视图**（standard 布局下点封面切入的歌词）与**全屏歌词页**。沉浸布局左下角的浮动歌词是单行组件，只跟随 `foreground` 配色，不套用本节样式。

| 字段 | 类型 | 默认值 | 取值范围 | 作用 |
| --- | --- | --- | --- | --- |
| `align` | 枚举 | `center` | `center` / `left` | 歌词对齐 |
| `highlight` | 枚举 | `gradient` | 见下表 | 当前行高亮方式 |
| `activeScale` | 数字 | 5 | 0–16 | 当前行比其他行**大多少字号**（加法，不是倍数），当前行恒为 w800 |
| `inactiveOpacity` | 数组 | `[0.6, 0.4, 0.25]` | 每项 0–1 | 距当前行 1、2、≥3 行的不透明度（颜色为 `textSecondary`） |
| `blurInactive` | 数字 | 0 | 0–6 | 非当前行模糊强度，每远一行 ×距离，上限 3 |
| `lineHeight` | 数字 | 1.5 | 1–2.4 | 行高倍数 |
| `fadeEdge` | 数字 | 0.16 | 0–0.4 | 歌词列表上下边缘渐隐比例（ShaderMask） |
| `showTranslation` | 布尔 | `true` | — | 有翻译歌词时显示在原文下方（按时间戳 ±500ms 匹配） |
| `translationScale` | 数字 | 0.75 | 0.5–1 | 翻译字号 = 正文字号 × 该值 |

三种 `highlight` 的具体渲染（当前行）：

| 值 | 颜色来源 | 效果 |
| --- | --- | --- |
| `gradient` | `colors.accent` → `colors.accentPress` | 文字填充双色渐变 + 一层 accent 50% 柔和投影 |
| `solid` | `colors.accent` | 纯色，无投影 |
| `glow` | `colors.accent` | 纯色 + 双层发光阴影（80% / 模糊 18 与 40% / 模糊 36） |

歌词字体的族跟随全局 `fontFamily`；歌词实际字号由 `lyricsFontSizeSmall/Medium/Large` 三档决定（用户在歌词页可循环切换），`activeScale` 在此之上叠加。

## 歌词页独立背景：lyricsBackground

全屏歌词页默认沿用 `background`，写 `lyricsBackground` 可单独定制。字段与 `background` 完全相同，**以主背景为底做增量合并**——只写差异字段即可：

```json
"player": {
  "background": { "mode": "coverBlur" },
  "lyricsBackground": {
    "mode": "image",
    "backgroundImage": "assets/backgrounds/lyrics.webp",
    "imageBlur": 0
  }
}
```

### 歌词页遮罩规则（与播放页不同）

歌词页遮罩颜色由 4 个独立字段控制（ARGB 整数），按主题亮度选 light/dark 一组：

| 字段 | 默认值 | 含义 |
| --- | --- | --- |
| `lyricsLightOverlayStart` / `lyricsLightOverlayEnd` | `0x8CF7F6F3` / `0xDBF7F6F3` | 浅色：遮罩顶部/底部色（默认即半透明米白） |
| `lyricsDarkOverlayStart` / `lyricsDarkOverlayEnd` | `0xBD06090E` / `0xEB06090E` | 深色：遮罩顶部/底部色 |

叠加方式分两种：

- 背景是 `image` 模式且图片存在，或封面取色失败：简单的 start→end **线性**渐变，不混任何封面色（保护主题画面）。
- 其他情况：以 start/end 为底**混入封面主色**的径向渐变（圆心偏上，混入比例 50% / 28% / 15%），歌词页色调随歌曲变化。

歌词页背景的模糊半径取 `lyricsBackgroundBlur`（默认 70），与播放页的 `backgroundBlurIntensity` 各自独立。

## 歌词页专属参数

| 字段 | 默认值 | 作用范围 |
| --- | --- | --- |
| `lyricsBackgroundBlur` | 70 | 歌词页背景模糊半径（coverBlur/coverFocus 模式） |
| `lyricsTopBarHeight` | 40 | 歌词页顶栏高度 |
| `lyricsTopButtonSize` | 38 | 歌词页顶栏圆形按钮尺寸 |
| `lyricsTopButtonMargin` | 12 | 顶栏按钮水平边距 |
| `lyricsTopButtonIconSize` | 20 | 顶栏按钮图标尺寸 |
| `lyricsTitleFontSize` | 14 | 歌词页顶栏歌曲名字号 |
| `lyricsFontSizeSmall` / `Medium` / `Large` | 12 / 15 / 19 | 歌词三档字号，用户点顶栏 Aa 按钮循环切换 |

歌词页顶栏按钮底色用 `colors.surface`、描边 `colors.hairline`、图标 `colors.textPrimary`（走全局颜色，不受 `foreground` 影响）。

## 内容层其余参数

### 顶栏（仅 standard 布局）

| 字段 | 默认值 | 作用 |
| --- | --- | --- |
| `topBarHeight` | 40 | 顶栏高度 |
| `topBarPaddingH` | 22 | 顶栏水平边距 |
| `topBarPaddingTop` | 10 | 顶栏上边距 |
| `topBarPaddingBottom` | 20 | 控制区底部留白 |
| `titleUnderlineEnabled` | `true` | "正在播放"标题下方的装饰下划线 |
| `titleUnderlineWidth` / `titleUnderlineHeight` | 32 / 3 | 下划线宽/高（实际渲染分别 ×0.82、×0.9） |
| `titleUnderlineColor` | 无（ARGB 整数） | 下划线颜色，不写用 `colors.accent` |

### 控制区（播放页与歌词页底部共用）

| 字段 | 默认值 | 作用 |
| --- | --- | --- |
| `playButtonSize` | 70 | 播放/暂停主按钮直径 |
| `playIconSize` | 28 | 主按钮图标尺寸 |
| `sideButtonSize` | 46 | 上一首/下一首/循环/队列按钮直径 |
| `sideIconSize` | 22 | 次要按钮图标尺寸 |
| `playButtonShadowBlur` / `OffsetY` / `Opacity` | 26 / 10 / 0.45 | 主按钮投影 |

进度条（播放页为自绘组件，时间气泡用 `colors.accent`；歌词页用 Slider，轨道 `accent` + `hairlineStrong`）：**粗细与滑块尺寸请在 `components.player` 里设 `progressThickness` / `thumbSize`**，顶层 `player` 里的同名字段当前没有消费方。

## 当前可解析但不生效的字段

以下字段写在顶层 `player` 里不会报错，但客户端当前没有读取它们，属于历史遗留/保留字段，**不要依赖**：

`coverRadius`、`coverMaxInset`、`titleGap`、`controlGap`、`backdropOpacity`、`backdropBlurSigma`、`progressThickness`、`thumbSize`、`actionBarEnabled`、`actionBarSpacing`、`actionBarIconSize`、`actionBarLabelSize`

（圆角请用 `coverBorderRadius`；进度条样式用 `components.player`。）

## 类型容错规则

`player` 节类型写错**不会安装失败**，只会静默回退默认值——效果不对时先对照下表检查类型：

- 布尔：`flow`、`spin`、`haloEnabled`、`breathEnabled`、`vinylEnabled`、`tonearmEnabled`、`titleUnderlineEnabled`、`showTranslation` 等。
- **整数**毫秒：`flowDuration`、`spinDuration`、`haloDuration`、`breathDuration`、`vinylRotationDuration`（写 `20000.0` 会被忽略）。
- **ARGB 十进制整数**：`lightBackgroundBase`、`darkBackgroundBase`、`lyricsLight/DarkOverlayStart/End`、`titleUnderlineColor`。
- 枚举值不认识 → 回退默认；数值超出范围 → 截断到边界。

## 深色模式下的播放页

- 主题自身 `brightness: "dark"`：遮罩选 `darkBackgroundBase` / `lyricsDarkOverlay*`，`foreground: auto` 时前景为浅色。
- 浅色主题 + 用户开深色模式（即 manifest `brightnessSupport: "dual"` 宣称的能力）：客户端**自动派生深色变体**——26 色换成固定深色板（`accent`/`accentPress`/`accentSubtle`/`currentPlaying`/`rank*`/`favorite` 继承你的浅色值），`player` 参数整体保留，但 `backgroundOverlayOpacity` 被强制改为 `0.55`。派生是实验性的；认真做深色请单独出 `brightness: "dark"` 的主题包。

## 三个完整示例

**① 黑胶唱片页**（封面即黑胶 + 封面取色渐变背景）：

```json
"player": {
  "preset": "vinyl",
  "vinylEnabled": false,
  "tonearmEnabled": false,
  "cover": { "shape": "vinyl", "spin": true, "spinDuration": 20000 },
  "lyrics": { "highlight": "solid" },
  "lightBackgroundBase": 1930693896
}
```

要点：`preset: "vinyl"` 已含 `coverGradient` 背景与黑胶封面；手动关外层黑胶框避免双重装饰；半透明 base 让渐变透出。

**② 主题背景图**（播放页/歌词页用包内插画）：

```json
"player": {
  "foreground": "light",
  "background": {
    "mode": "image",
    "backgroundImage": "assets/backgrounds/night.webp",
    "imageBlur": 0
  },
  "lyricsBackground": {
    "mode": "image",
    "backgroundImage": "assets/backgrounds/night_lyrics.webp"
  },
  "backgroundOverlayOpacity": 0.2,
  "lightBackgroundBase": 1544558630,
  "haloEnabled": false
}
```

要点：图片模式务必配半透明 base（这里是 36% 透明墨蓝 `0x5C101826`，十进制 1544558630）+ `foreground: "light"`；歌词页用图时遮罩自动不混封面色。

**③ 封面渐变流光**（每首歌背景随封面变化）：

```json
"player": {
  "preset": "aurora",
  "background": { "mode": "coverGradient", "flow": true, "flowDuration": 24000 },
  "cover": { "halo": "glow" },
  "lyrics": { "highlight": "glow" },
  "backgroundOverlayOpacity": 0.3,
  "lightBackgroundBase": 1930693896
}
```

要点：`aurora` 预设已含 `flow`，这里演示显式覆盖 `flowDuration`；`coverGradient` 不需要任何颜色字段，颜色全部来自封面取色。

更多实战配置见主仓库官方示例主题（[examples/README.md](../examples/README.md)）：`opera-velvet-dual` 与 `natural-warmth` 均为 `immersive` 预设 + `coverFocus` + 半透明遮罩的组合。

## 字段速查总表

按"字段 → 默认值 → 作用范围"汇总 `player` 节全部生效字段：

| 字段 | 默认值 | 作用范围 |
| --- | --- | --- |
| `preset` | 无（standard 场景） | 场景底值，播放页整体 |
| `layout` | `standard` | 播放页布局结构 |
| `foreground` | `auto` | 播放页内容层文字/图标配色 |
| `background.mode` | `coverBlur` | 播放页+歌词页背景层 |
| `background.flow` / `flowDuration` | `false` / 18000 | 渐变类背景漂移动画 |
| `background.colors` | `[]` | `themeGradient`/`solid` 用色 |
| `background.backgroundImage` / `imageBlur` | 无 / 0 | `image` 模式背景图 |
| `cover.shape` | `rounded` | 封面形态（standard 布局） |
| `cover.halo` | `sweep` | 光晕样式 |
| `cover.spin` / `spinDuration` | `false` / 24000 | 封面旋转 |
| `lyrics.*`（9 字段） | 见歌词样式表 | 三处歌词渲染 |
| `lyricsBackground.*` | 无（沿用 background） | 歌词页背景层 |
| `lightBackgroundBase` / `darkBackgroundBase` | 不透明米白/墨黑 | 播放页遮罩底色（alpha 决定透出与否） |
| `backgroundOverlayOpacity` | 0.4 | 播放页遮罩强度 |
| `backgroundBlurIntensity` | 70 | 播放页 coverBlur/coverFocus 模糊半径 |
| `coverSize` / `coverBorderRadius` | 262 / 34 | 封面尺寸与圆角 |
| `coverShadowBlur` / `coverShadowOpacity` | 80 / 0.5 | 封面投影 |
| `haloEnabled` / `haloDuration` / `haloBlur` / `haloOpacity` / `haloSize` | true / 16000 / 120 / 0.3 / 1.8 | 光晕开关与强度 |
| `breathEnabled` / `breathDuration` / `breathScale` | true / 4500 / 1.02 | 呼吸动画 |
| `vinylEnabled` / `vinylSize` / `vinylRotationDuration` | true / 290 / 20000 | **外层**黑胶框 |
| `tonearmEnabled` / `tonearmRotationAngle` | true / 20 | 外层唱臂 |
| `lyricsBackgroundBlur` | 70 | 歌词页背景模糊 |
| `lyricsLight/DarkOverlayStart/End` | 见上表 | 歌词页遮罩色 |
| `lyricsTopBar*` / `lyricsTitleFontSize` / `lyricsFontSize*` | 见上表 | 歌词页顶栏与字号 |
| `topBar*` / `titleUnderline*` | 见上表 | 播放页顶栏与标题装饰 |
| `playButtonSize` / `playIconSize` / `sideButtonSize` / `sideIconSize` / `playButtonShadow*` | 见上表 | 播放页+歌词页控制区 |
