# tools · 工具说明

两个 Python 3 脚本，与 NoraMusic 后台/客户端的校验规则对齐。

## pack.py —— 校验 + 可重复打包

只依赖 Python 标准库，无需安装任何包。

```bash
python3 tools/pack.py                # 校验并打包 packages/ 下全部主题
python3 tools/pack.py starter-theme  # 只打包指定主题（参数是 packages/ 下的目录名）
```

输出到 `dist/<id>-<version>.noratheme`，并打印文件数、大小和 SHA-256。真实输出示例：

```text
starter-theme
  NoraMusic-Theme/dist/my-first-theme-1.0.0.noratheme  2 个文件  0.00 MiB  sha256 a8dd153fa9f155fb4df96e1b82174f1591e472d1f595b9dec8c3f31669da9788
```

行为要点：

- **先校验、后出包**：任何一项校验不过都不生成产物，失败原因直接打印（退出码 1）。
- **可重复构建**：ZIP 条目按路径排序，时间戳固定为 1980-01-01，权限固定 `0644`——同样输入永远得到字节级相同的文件，SHA-256 可用于核对发布产物。
- 自动跳过隐藏文件（`.DS_Store` 等以 `.` 开头的路径段不会进包），但其他非法文件仍会触发校验失败。
- 产物超过商店默认上传上限 32 MiB 时打印提醒（不阻断）。
- 脚本内嵌了 `player` 节的类型检查（ARGB 颜色必须是十进制整数、时长必须是整数、开关必须是布尔），把客户端"静默回退默认值"的坑提前拦下。

校验规则摘要（与 [../docs/02-package-format.md](../docs/02-package-format.md) 一致）：

- 扩展名白名单 `.json .png .jpg .jpeg .webp .ttf .otf`；≤512 个文件；单文件 ≤32 MiB、压缩比 ≤200；解压总量 ≤128 MiB；`manifest.json` ≤64 KiB、`theme.json` ≤1 MiB。
- 路径必须相对 POSIX、≤240 字符，禁止 `..` `\` `:` 绝对路径、符号链接、`__MACOSX/`、`.DS_Store`、忽略大小写重名。
- manifest：`schemaVersion: 2`、`id` / `version` 正则、`author` 为对象、`preview` 指向包内文件。
- theme：`colors` 必填非空、无 `background` 键、每个值都是 `#RRGGBB` / `#AARRGGBB`。
- 内容扫描：禁止 script/lua 类键名、含 `://` 或 `file:`/`data:` 的字符串；`path` / `background` / `backgroundImage` / `preview` 键的值必须是包内文件。

## generate_assets.py —— 程序化生成背景图与商店预览图

依赖 [Pillow](https://python-pillow.org/)：

```bash
pip install Pillow
python3 tools/generate_assets.py                # 缺失的背景图才补画，已有背景保留
python3 tools/generate_assets.py --backgrounds  # 强制重绘全部背景（覆盖手动替换的素材）
```

用途与约定：

- 为背景图生成**1080 × 1920**、为预览图生成**720 × 1280** 的 JPG。
- 绘制采用固定随机种子（渐变 + 模糊色块 + 噪点纹理），输出可复现，适合没有美术素材时快速得到"能看"的背景。
- 预览图按主题 `theme.json` 的真实 `canvas`、`backgroundImageOpacity`、`backgroundImageBlur` 还原页面观感，再叠封面、歌词条、控制区示意。
- 脚本是**面向你自己主题改造的参考实现**：它内置的主题绘制函数与主仓库官方示例绑定，用在自己的主题上时，复制其中的 `vertical_gradient` / `blob` / `noise` / `cover_fit` 等工具函数自行组合即可。
- 背景图也可以随时用手动制作的素材替换——脚本默认保留已有背景，只有文件缺失或显式传 `--backgrounds` 才重绘。

## 目录约定

两个脚本都以"脚本上一级目录"为根：

- 输入：`packages/<目录名>/`（含 `manifest.json`、`theme.json`、可选 `assets/`）
- 输出：`dist/<id>-<version>.noratheme`（仅 `pack.py`）

因此新建主题就是新建 `packages/` 下的一个目录，详见 [../templates/README.md](../templates/README.md)。
