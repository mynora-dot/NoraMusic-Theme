# 05 · 背景图与自定义字体

主题包是自包含的：所有图片、字体都必须打进 `.noratheme`，**不允许引用任何包外资源**（URL、本地文件路径都会在内容扫描时被拒）。

## 全局背景图

```json
{
  "backgroundImage": "assets/backgrounds/bg.jpg",
  "backgroundImageOpacity": 0.2,
  "backgroundImageBlur": 2
}
```

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `backgroundImage` | 字符串 | 无 | 图片在包内的相对路径，文件必须真实存在，否则 `MISSING_THEME_RESOURCE` |
| `backgroundImageOpacity` | 数字 | `0.3` | 不透明度，0–1。全局背景叠在 `colors.canvas` 之上，值越大图越明显 |
| `backgroundImageBlur` | 数字 | `0` | 高斯模糊半径，0 表示不模糊 |

也可以写成 `"assets": { "background": "..." }`；两种写法同时存在时以 `assets.background` 为准。

播放页、歌词页可以另设背景，与全局背景互不影响，见 [04-player-scene.md](04-player-scene.md)。

### 图片规格建议

| 项 | 建议 |
| --- | --- |
| 尺寸 | **1080 × 1920**（竖屏）起步，长边控制在 2000 像素以内 |
| 格式 | 照片/插画用 `.webp` 或压缩后的 `.jpg`；带透明通道的图形用 `.png` |
| 体积 | 单张尽量 < 500 KiB；注意单文件 32 MiB、整包 128 MiB 的硬上限 |
| 构图 | 背景会被列表、卡片大面积遮挡，低对比、少主体的图更好用；重要内容别放在会被导航栏/迷你播放器盖住的位置 |
| 可读性 | 背景只负责氛围，文字可读性靠 `backgroundImageOpacity` 和 `backgroundImageBlur` 调。列表页看不清字就降透明度或加模糊 |

压缩建议：

```bash
# ImageMagick：缩到 1080 宽、质量 80 的 jpg
magick input.png -resize 1080x1920 -quality 80 bg.jpg
# 或 cwebp
cwebp -q 80 -resize 1080 1920 input.png -o bg.webp
```

没有现成素材时，可以用 `tools/generate_assets.py` 里的程序化绘制思路（渐变 + 噪点 + 色块，固定随机种子可复现）自己生成，脚本说明见 [tools/README.md](../tools/README.md)。

## 预览图 preview.jpg

- 在 `manifest.json` 写 `"preview": "preview.jpg"`，文件放在包根目录。
- 建议尺寸 **720 × 1280**，jpg 质量 80 左右。
- 商店列表/主题列表里展示的就是它，直接决定用户的第一印象。惯例是画一张播放页示意图（背景铺底 + 封面 + 歌词条 + 控制区）。
- 写了 `preview` 字段但文件不在包里 → `MISSING_THEME_RESOURCE`；没有预览图就删掉该字段。

## 自定义字体

### 声明与引用（两步缺一不可）

```json
{
  "fontFamily": "MyFont",
  "assets": {
    "fonts": {
      "body": {
        "path": "assets/fonts/MyFont.ttf",
        "family": "MyFont"
      }
    }
  }
}
```

1. **注册**：`assets.fonts` 把字体文件注册进应用。键名（如 `body`）只是占位，注册的字体族名取 `family` 字段；不写 `family` 时取键名。客户端用 `FontLoader` 在启用主题时加载。
2. **引用**：在顶层 `fontFamily`（全局生效，所有文字样式统一替换）或某个 `typo.*.fontFamily`（按样式单独指定）里使用这个族名。

规则与坑：

- 一个字体文件只注册**一种字重**；`weights` 字段是保留字段，暂不生效。需要多个字重时，每个字重用单独的文件和不同的 `family` 名，再分别指定到 `typo.title.fontFamily`、`typo.body.fontFamily` 等。
- 字体注册失败**不会报错**，静默回退系统字体。所以"字体没生效"时先检查：`path` 是否正确（区分大小写）、`family` 与引用处拼写是否一致、文件是不是合法的 `.ttf`/`.otf`。
- 大多数开源中文字体只有常规字重，界面上需要加粗的地方由系统合成粗体，笔画会略糊。
- 注意字体的字符覆盖：不少中文字体不含日文假名或特殊符号（如间隔号 `·`），缺字形时会按系统字体回退，歌词里可能出现两种字体混排。

### 字体许可与署名

**只使用允许再分发的字体**，推荐 SIL Open Font License（OFL）字体，例如 Google Fonts 上的中文开源字体：

| 字体 | 风格 | 许可证 |
| --- | --- | --- |
| Noto Sans SC / Noto Serif SC（思源黑体/宋体） | 现代无衬线 / 衬线 | OFL 1.1 |
| ZCOOL KuaiLe（站酷快乐体） | 活泼手写 | OFL 1.1 |
| ZCOOL QingKe HuangYou（站酷庆科黄油体） | 海报美术体 | OFL 1.1 |
| ZCOOL XiaoWei（站酷小薇体） | 柔美手写 | OFL 1.1 |
| Ma Shan Zheng（马善政毛笔楷书） | 毛笔书法 | OFL 1.1 |
| LXGW WenKai（霞鹜文楷） | 楷体 | OFL 1.1 |

**重要：许可证 .txt 文件放不进主题包**（扩展名白名单不含 `.txt`）。OFL 要求保留版权声明，惯例做法是把署名写进 `manifest.json`：

```json
{
  "license": "MIT AND OFL-1.1",
  "description": "……主题配置以 MIT 发布，内置字体 ZCOOL XiaoWei 遵循 SIL Open Font License 1.1。"
}
```

### 字体体积

中文字体动辄 1–10 MiB，直接决定包大小。优化方向：

- 优先选官方就较小的字体（如 ZCOOL KuaiLe 约 1.4 MiB）；
- 用 `pyftsubset`（fonttools）做子集化，只保留常用汉字和符号——注意 OFL 对修改版字体的 Reserved Font Name 条款，子集化后需改名；
- 多字重主题考虑可变字体（单个 `.ttf` 覆盖多个字重，但按当前规则仍只注册一种默认字重）。

## 版权与肖像权

背景图、插画、人像写真是主题侵权的高发区：

- 使用自己拍摄/绘制、或明确允许再分发（CC0、CC-BY 等）的素材；
- **人像写真涉及肖像权和著作权双重权利**，即便"仅个人使用"可以，也不要未经权利人授权就上传商店公开发布；
- 不要把有版权的专辑封面、动漫截图、品牌 logo 打进主题包。

发布前的合规清单见 [06-publishing.md](06-publishing.md#安全与合规)。
