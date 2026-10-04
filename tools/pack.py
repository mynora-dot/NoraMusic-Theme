#!/usr/bin/env python3
"""校验并打包示例主题为可上传商店的 .noratheme。

校验规则对齐后台 backend/internal/server/admin_themes.go 的 checkThemePackage
与客户端 mobile/lib/theme/theme_package.dart 的 validateStaticArchive。
打包输出可重复：条目排序、固定时间戳与权限。

用法：
  python3 theme/tools/pack.py              # 打包 theme/packages 下全部主题
  python3 theme/tools/pack.py ink-wash     # 只打包指定主题
输出：theme/dist/<id>-<version>.noratheme
"""
from __future__ import annotations

import hashlib
import io
import json
import posixpath
import re
import sys
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
PACKAGES = ROOT / "packages"
DIST = ROOT / "dist"

MIB = 1024 * 1024
ALLOWED_EXT = {".json", ".png", ".jpg", ".jpeg", ".webp", ".ttf", ".otf"}
MAX_FILES = 512
MAX_FILE = 32 * MIB
MAX_TOTAL = 128 * MIB
MAX_RATIO = 200
MAX_MANIFEST = 64 * 1024
MAX_THEME = 1 * MIB
MAX_UPLOAD_DEFAULT = 32 * MIB

ID_RE = re.compile(r"^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$")
VERSION_RE = re.compile(r"^\d{1,5}\.\d{1,5}\.\d{1,5}$")
COLOR_RE = re.compile(r"^#[a-fA-F0-9]{6}([a-fA-F0-9]{2})?$")
PATH_KEYS = {"path", "background", "backgroundImage", "preview"}
ARGB_KEYS = {
    "lightBackgroundBase", "darkBackgroundBase",
    "lyricsLightOverlayStart", "lyricsLightOverlayEnd",
    "lyricsDarkOverlayStart", "lyricsDarkOverlayEnd",
}
INT_KEYS = {"haloDuration", "breathDuration", "flowDuration", "spinDuration"}
BOOL_KEYS = {"haloEnabled", "breathEnabled", "flow", "spin", "showTranslation"}


class PackError(Exception):
    pass


def collect_files(src: Path) -> list[tuple[str, Path]]:
    files = []
    for path in sorted(src.rglob("*")):
        if path.is_symlink():
            raise PackError(f"不允许符号链接：{path.relative_to(src)}")
        if not path.is_file():
            continue
        rel = path.relative_to(src).as_posix()
        if any(part.startswith(".") for part in rel.split("/")):
            continue  # .DS_Store 等隐藏文件不进包
        files.append((rel, path))
    return files


def check_path(name: str):
    if not name or len(name) > 240:
        raise PackError(f"路径长度不合法：{name!r}")
    if "\\" in name or ":" in name or "\x00" in name or name.startswith("/"):
        raise PackError(f"路径包含非法字符或为绝对路径：{name}")
    if posixpath.normpath(name) != name or ".." in name.split("/"):
        raise PackError(f"路径未规范化：{name}")
    if name.startswith("__MACOSX/") or name.split("/")[-1] == ".DS_Store":
        raise PackError(f"不允许的系统文件：{name}")
    if posixpath.splitext(name)[1].lower() not in ALLOWED_EXT:
        raise PackError(f"不允许的扩展名：{name}")


def walk_json(value, names: set[str], where: str, key: str = ""):
    lower = key.lower()
    if lower.startswith("script") or lower.endswith("script") or lower.startswith("lua"):
        raise PackError(f"{where}: 不允许的字段 {key}")
    if isinstance(value, dict):
        for k, v in value.items():
            walk_json(v, names, where, k)
    elif isinstance(value, list):
        for v in value:
            walk_json(v, names, where, key)
    elif isinstance(value, str):
        if "://" in value or value.lower().startswith(("file:", "data:")):
            raise PackError(f"{where}: {key} 含 URL：{value}")
        if key in PATH_KEYS and value not in names:
            raise PackError(f"{where}: {key} 指向的文件不在包内：{value}")


def check_manifest(manifest: dict, names: set[str]):
    if manifest.get("schemaVersion") != 2:
        raise PackError("manifest.schemaVersion 必须是 2")
    if not isinstance(manifest.get("id"), str) or not ID_RE.match(manifest["id"]):
        raise PackError("manifest.id 不合法")
    if not isinstance(manifest.get("version"), str) or not VERSION_RE.match(manifest["version"]):
        raise PackError("manifest.version 不合法")
    name = manifest.get("name")
    if not isinstance(name, str) or not name.strip() or len(name.encode()) > 160:
        raise PackError("manifest.name 不合法")
    desc = manifest.get("description", "")
    if not isinstance(desc, str) or len(desc.encode()) > 2000:
        raise PackError("manifest.description 超长")
    author = manifest.get("author")
    if author is not None:
        if not isinstance(author, dict) or len(str(author.get("name", "")).encode()) > 160:
            raise PackError("manifest.author 必须是对象，name 不超过 160 字节")
    tags = manifest.get("tags", [])
    if not isinstance(tags, list) or len(tags) > 12 or any(
        not isinstance(t, str) or len(t.encode()) > 64 for t in tags
    ):
        raise PackError("manifest.tags 不合法")
    preview = manifest.get("preview")
    if preview is not None and preview not in names:
        raise PackError(f"manifest.preview 不在包内：{preview}")


def check_player(player: dict):
    """客户端类型错误会静默回退，这里提前拦下，避免效果不对。"""
    for key, value in player.items():
        if key in ARGB_KEYS and (type(value) is not int or not 0 <= value <= 0xFFFFFFFF):
            raise PackError(f"player.{key} 必须是 ARGB 十进制整数")
        if key in INT_KEYS and type(value) is not int:
            raise PackError(f"player.{key} 必须是整数")
        if key in BOOL_KEYS and type(value) is not bool:
            raise PackError(f"player.{key} 必须是布尔值")
    for group in ("background", "cover", "lyrics", "lyricsBackground"):
        sub = player.get(group)
        if sub is None:
            continue
        if not isinstance(sub, dict):
            raise PackError(f"player.{group} 必须是对象")
        check_player(sub)
    colors = (player.get("background") or {}).get("colors", [])
    for c in colors:
        if not (isinstance(c, int) or (isinstance(c, str) and COLOR_RE.match(c))):
            raise PackError(f"player.background.colors 颜色不合法：{c}")


def check_theme(theme: dict):
    if theme.get("schemaVersion") != 2:
        raise PackError("theme.schemaVersion 必须写 2")
    colors = theme.get("colors")
    if not isinstance(colors, dict) or not colors:
        raise PackError("theme.colors 必填且不能为空")
    if "background" in colors:
        raise PackError("theme.colors 不能包含 background 字段")
    for k, v in colors.items():
        if not isinstance(v, str) or not COLOR_RE.match(v):
            raise PackError(f"theme.colors.{k} 不是合法颜色：{v}")
    if isinstance(theme.get("player"), dict):
        check_player(theme["player"])


def validate_archive(data: bytes) -> dict:
    with zipfile.ZipFile(io.BytesIO(data)) as zf:
        infos = zf.infolist()
        if len(infos) > MAX_FILES:
            raise PackError("文件数超过 512")
        seen, names, total = set(), set(), 0
        for info in infos:
            if info.is_dir():
                continue
            check_path(info.filename)
            if info.filename.lower() in seen:
                raise PackError(f"忽略大小写后重名：{info.filename}")
            seen.add(info.filename.lower())
            names.add(info.filename)
            if info.file_size > MAX_FILE:
                raise PackError(f"单文件超过 32 MiB：{info.filename}")
            if info.compress_size and info.file_size / info.compress_size > MAX_RATIO:
                raise PackError(f"压缩比超过 200：{info.filename}")
            total += info.file_size
        if total > MAX_TOTAL:
            raise PackError("解压后总大小超过 128 MiB")
        for required, limit in (("manifest.json", MAX_MANIFEST), ("theme.json", MAX_THEME)):
            if required not in names:
                raise PackError(f"根目录缺少 {required}")
            if zf.getinfo(required).file_size > limit:
                raise PackError(f"{required} 超过大小限制")
        manifest = json.loads(zf.read("manifest.json"))
        theme = json.loads(zf.read("theme.json"))
    check_manifest(manifest, names)
    check_theme(theme)
    walk_json(manifest, names, "manifest.json")
    walk_json(theme, names, "theme.json")
    return manifest


def build(src: Path) -> Path:
    files = collect_files(src)
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as zf:
        for rel, path in files:
            check_path(rel)
            info = zipfile.ZipInfo(rel, date_time=(1980, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o100644 << 16
            zf.writestr(info, path.read_bytes())
    data = buf.getvalue()
    manifest = validate_archive(data)
    if len(data) > MAX_UPLOAD_DEFAULT:
        print(f"  注意：包大小超过后台默认上传上限 32 MiB")
    DIST.mkdir(exist_ok=True)
    out = DIST / f"{manifest['id']}-{manifest['version']}.noratheme"
    out.write_bytes(data)
    digest = hashlib.sha256(data).hexdigest()
    print(f"  {out.relative_to(ROOT.parent)}  {len(files)} 个文件  {len(data) / MIB:.2f} MiB  sha256 {digest}")
    return out


def main(argv: list[str]) -> int:
    targets = argv or sorted(p.name for p in PACKAGES.iterdir() if p.is_dir())
    failed = False
    for name in targets:
        print(name)
        try:
            build(PACKAGES / name)
        except (PackError, json.JSONDecodeError, OSError) as e:
            print(f"  失败：{e}")
            failed = True
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
