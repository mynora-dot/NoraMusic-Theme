#!/usr/bin/env python3
"""生成示例主题包的背景图与商店预览图。

预览图以各主题包里现有的背景图铺底绘制。背景图缺失时用程序化绘制（固定随机种子）补齐；
已有背景图（包括手动替换的素材）默认保留，传 --backgrounds 才会强制重绘覆盖。
用法：python3 theme/tools/generate_assets.py [--backgrounds]
依赖：Pillow
"""
from __future__ import annotations

import json
import math
import random
import sys
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw, ImageFilter, ImageFont

ROOT = Path(__file__).resolve().parent.parent
PACKAGES = ROOT / "packages"

BG_SIZE = (1080, 1920)
PREVIEW_SIZE = (720, 1280)


def hex_rgb(value: str) -> tuple[int, int, int]:
    value = value.lstrip("#")[-6:]
    return tuple(int(value[i : i + 2], 16) for i in (0, 2, 4))  # type: ignore[return-value]


def vertical_gradient(size, stops):
    """stops: [(位置 0–1, '#RRGGBB'), ...]"""
    w, h = size
    column = Image.new("RGB", (1, h))
    px = column.load()
    for y in range(h):
        t = y / max(h - 1, 1)
        for i in range(len(stops) - 1):
            (p0, c0), (p1, c1) = stops[i], stops[i + 1]
            if p0 <= t <= p1:
                k = (t - p0) / max(p1 - p0, 1e-6)
                a, b = hex_rgb(c0), hex_rgb(c1)
                px[0, y] = tuple(round(a[j] + (b[j] - a[j]) * k) for j in range(3))
                break
    return column.resize(size)


def blob(canvas: Image.Image, center, radius, color, alpha, blur):
    """在 canvas 上叠加一个模糊圆形色块。"""
    mask = Image.new("L", canvas.size, 0)
    x, y = center
    ImageDraw.Draw(mask).ellipse((x - radius, y - radius, x + radius, y + radius), fill=round(255 * alpha))
    mask = mask.filter(ImageFilter.GaussianBlur(blur))
    canvas.paste(Image.new("RGB", canvas.size, hex_rgb(color)), (0, 0), mask)


def noise(size, rng: random.Random, scale=4, blur=1.2):
    w, h = size
    small = Image.new("L", (w // scale, h // scale))
    small.putdata([rng.randint(0, 255) for _ in range((w // scale) * (h // scale))])
    return small.resize(size, Image.BICUBIC).filter(ImageFilter.GaussianBlur(blur))


def save_jpg(image: Image.Image, path: Path, quality=86):
    path.parent.mkdir(parents=True, exist_ok=True)
    image.convert("RGB").save(path, "JPEG", quality=quality, optimize=True, progressive=True)
    print(f"  {path.relative_to(ROOT)}  {path.stat().st_size // 1024} KiB")


def rounded_bar(draw, box, color):
    x0, y0, x1, y1 = box
    draw.rounded_rectangle(box, radius=(y1 - y0) / 2, fill=color)


# ---------------------------------------------------------------- 朱红

def crimson_backdrop(size, seed=163):
    """白底上的朱红光晕、黑胶纹路与声波线，整体保持很淡，保证列表文字可读。"""
    rng = random.Random(seed)
    w, h = size
    canvas = vertical_gradient(size, [(0, "#FFF4F2"), (0.35, "#FAF8F8"), (1, "#F6F6F6")])
    blob(canvas, (w * 0.95, h * 0.02), w * 0.55, "#F7B4AC", 0.55, w * 0.18)
    blob(canvas, (w * 0.05, h * 0.98), w * 0.45, "#F9D3CE", 0.45, w * 0.16)
    layer = Image.new("RGBA", size, (217, 54, 54, 0))
    d = ImageDraw.Draw(layer)
    # 右上角露出一部分黑胶纹路
    cx, cy = w * 1.02, h * 0.08
    for k in range(18):
        r = w * 0.22 + k * w * 0.028
        d.ellipse((cx - r, cy - r, cx + r, cy + r), outline=(217, 54, 54, 22 if k % 3 else 34), width=2)
    # 左下方几条声波线
    for i in range(5):
        base = h * (0.78 + i * 0.022)
        amp = h * (0.012 + 0.006 * rng.random())
        freq = rng.uniform(1.6, 2.6)
        phase = rng.uniform(0, math.tau)
        pts = [(x, base + amp * math.sin(x / w * math.tau * freq + phase)) for x in range(0, w + 1, 6)]
        d.line(pts, fill=(217, 54, 54, 26 - i * 3), width=3)
    out = canvas.convert("RGBA")
    out.alpha_composite(layer.filter(ImageFilter.GaussianBlur(0.6)))
    return out.convert("RGB")


def centered_text(draw, y, text, font, fill):
    tw = draw.textlength(text, font=font)
    draw.text(((PREVIEW_SIZE[0] - tw) / 2, y), text, font=font, fill=fill)


# ---------------------------------------------------------------- 水墨

def ridge(width, base_y, amplitude, rng, roughness=0.55):
    """中点位移生成山脊线，返回每个 x 的 y。"""
    n = 1
    while n < width:
        n *= 2
    pts = [0.0] * (n + 1)
    pts[0], pts[n] = rng.uniform(-1, 1), rng.uniform(-1, 1)
    step, amp = n, 1.0
    while step > 1:
        half = step // 2
        for i in range(half, n, step):
            pts[i] = (pts[i - half] + pts[i + half]) / 2 + rng.uniform(-amp, amp)
        step, amp = half, amp * roughness
    return [base_y - amplitude * pts[round(x * n / (width - 1))] for x in range(width)]


def ink_mountain(size, base_y, amplitude, ink, darkness, blur, rng):
    """一层山：山顶墨色浓，向下渐淡，带干笔纹理。"""
    w, h = size
    line = ridge(w, base_y, amplitude, rng)
    shape = Image.new("L", size, 0)
    ImageDraw.Draw(shape).polygon([(0, h)] + [(x, line[x]) for x in range(w)] + [(w, h)], fill=255)
    top = min(line)
    fade = Image.new("L", (1, h))
    fade.putdata([
        255 if y < top else max(0, round(255 * (1 - (y - top) / (h - top + 1) * 1.15)))
        for y in range(h)
    ])
    alpha = ImageChops.multiply(shape, fade.resize(size))
    texture = noise(size, rng, scale=3, blur=0.8).point(lambda v: 150 + v * 105 // 255)
    alpha = ImageChops.multiply(alpha, texture).filter(ImageFilter.GaussianBlur(blur))
    alpha = alpha.point(lambda v: round(v * darkness))
    return Image.new("RGB", size, hex_rgb(ink)), alpha


def mist(canvas, y, height, rng, color="#F4EFE4", alpha=0.75):
    w, _ = canvas.size
    mask = Image.new("L", canvas.size, 0)
    d = ImageDraw.Draw(mask)
    for _ in range(5):
        cx = rng.uniform(-0.1, 1.1) * w
        rw = rng.uniform(0.35, 0.7) * w
        d.ellipse((cx - rw, y - height / 2, cx + rw, y + height / 2), fill=round(255 * alpha))
    mask = mask.filter(ImageFilter.GaussianBlur(height / 2.2))
    canvas.paste(Image.new("RGB", canvas.size, hex_rgb(color)), (0, 0), mask)


def seal(canvas, box, color="#B5372B"):
    """朱红方印：只画抽象的阴刻笔画，不含文字。"""
    x0, y0, x1, y1 = box
    layer = Image.new("L", canvas.size, 0)
    d = ImageDraw.Draw(layer)
    d.rounded_rectangle(box, radius=6, fill=215)
    s = (x1 - x0) / 10
    for rect in [
        (2, 2, 4.4, 2.8), (2, 3.8, 4.4, 4.6), (2, 5.6, 4.4, 6.4), (2, 7.2, 4.4, 8),
        (3, 2, 3.6, 8), (5.6, 2, 8, 2.8), (6.5, 2, 7.1, 8), (5.6, 5, 8, 5.8), (5.6, 7.2, 8, 8),
    ]:
        a, b, c, e = rect
        d.rectangle((x0 + a * s, y0 + b * s, x0 + c * s, y0 + e * s), fill=0)
    texture = noise(canvas.size, random.Random(7), scale=2, blur=0.5).point(lambda v: 170 + v * 85 // 255)
    layer = ImageChops.multiply(layer, texture)
    canvas.paste(Image.new("RGB", canvas.size, hex_rgb(color)), (0, 0), layer)


def ink_landscape(size, seed=20261001, paper="#F3EEE3", with_seal=True):
    rng = random.Random(seed)
    w, h = size
    canvas = vertical_gradient(size, [(0, "#F6F2E9"), (0.6, paper), (1, "#EDE6D8")])
    grain = noise(size, rng, scale=2, blur=0.6)
    canvas = Image.composite(Image.new("RGB", size, hex_rgb("#E4DCCB")), canvas, grain.point(lambda v: v * 40 // 255))
    blob(canvas, (w * 0.72, h * 0.38), w * 0.085, "#C9564A", 0.55, w * 0.012)
    layers = [
        (0.66, 0.10, "#7D7A74", 0.35, 6),
        (0.74, 0.12, "#5C5A55", 0.55, 4),
        (0.84, 0.09, "#3A3936", 0.75, 2.5),
        (0.95, 0.07, "#1F1F1E", 0.92, 1.5),
    ]
    for i, (base, amp, ink, dark, blur) in enumerate(layers):
        color, alpha = ink_mountain(size, h * base, h * amp, ink, dark, blur, rng)
        canvas.paste(color, (0, 0), alpha)
        if i < len(layers) - 1:
            mist(canvas, h * (base + 0.035), h * 0.05, rng)
    if with_seal:
        seal(canvas, (round(w * 0.82), round(h * 0.08), round(w * 0.82) + 74, round(h * 0.08) + 74))
    return canvas


# ---------------------------------------------------------------- 樱色

def sparkle(layer: Image.Image, center, r, color, alpha):
    x, y = center
    d = ImageDraw.Draw(layer)
    rgb = hex_rgb(color)
    pts = []
    for i in range(8):
        ang = math.pi / 4 * i - math.pi / 2
        rad = r if i % 2 == 0 else r * 0.22
        pts.append((x + rad * math.cos(ang), y + rad * math.sin(ang)))
    d.polygon(pts, fill=rgb + (round(255 * alpha),))


def petal(layer: Image.Image, center, size, angle, color, alpha):
    w = round(size)
    tile = Image.new("RGBA", (w * 2, w * 2), (0, 0, 0, 0))
    d = ImageDraw.Draw(tile)
    rgb = hex_rgb(color)
    d.ellipse((w * 0.55, w * 0.2, w * 1.45, w * 1.8), fill=rgb + (round(255 * alpha),))
    d.polygon([(w, w * 0.15), (w * 0.85, w * 0.42), (w * 1.15, w * 0.42)], fill=(0, 0, 0, 0))
    tile = tile.rotate(angle, resample=Image.BICUBIC)
    layer.alpha_composite(tile, (round(center[0] - w), round(center[1] - w)))


def sakura_sky(size, seed=520):
    rng = random.Random(seed)
    w, h = size
    canvas = vertical_gradient(size, [(0, "#FFE3EE"), (0.45, "#F3E4FF"), (1, "#DDF0FF")])
    for color, cx, cy, r in [
        ("#FFC2DA", 0.15, 0.12, 0.42), ("#D9C8FF", 0.9, 0.35, 0.38),
        ("#BFE3FF", 0.2, 0.8, 0.45), ("#FFD6E8", 0.85, 0.92, 0.3),
    ]:
        blob(canvas, (w * cx, h * cy), w * r, color, 0.7, w * 0.12)
    layer = Image.new("RGBA", size, (255, 255, 255, 0))
    d = ImageDraw.Draw(layer)
    for _ in range(16):
        x, y, r = rng.uniform(0, w), rng.uniform(0, h), rng.uniform(18, 70)
        d.ellipse((x - r, y - r, x + r, y + r), fill=(255, 255, 255, rng.randint(40, 90)))
        d.ellipse((x - r, y - r, x + r, y + r), outline=(255, 255, 255, 120), width=2)
    layer = layer.filter(ImageFilter.GaussianBlur(2))
    for _ in range(22):
        sparkle(layer, (rng.uniform(0, w), rng.uniform(0, h)), rng.uniform(10, 30),
                rng.choice(["#FFFFFF", "#FFE08A", "#FF9CC6"]), rng.uniform(0.55, 0.9))
    for _ in range(26):
        petal(layer, (rng.uniform(0, w), rng.uniform(0, h)), rng.uniform(14, 28), rng.uniform(0, 360),
              rng.choice(["#FFB3CF", "#FF9CC2", "#FFC8DD"]), rng.uniform(0.6, 0.9))
    for _ in range(90):
        x, y, r = rng.uniform(0, w), rng.uniform(0, h), rng.uniform(1.5, 3.5)
        d.ellipse((x - r, y - r, x + r, y + r), fill=(255, 255, 255, rng.randint(120, 220)))
    out = canvas.convert("RGBA")
    out.alpha_composite(layer)
    return out.convert("RGB")


# ---------------------------------------------------------------- 玫瑰佳人（占位背景，建议替换为人像插画）

def rose_bloom(size, seed=1314):
    rng = random.Random(seed)
    w, h = size
    canvas = vertical_gradient(size, [(0, "#FBE3E4"), (0.5, "#FFF1EC"), (1, "#F6DCD6")])
    for color, cx, cy, r in [
        ("#F2B5C2", 0.85, 0.1, 0.45), ("#F6D3B8", 0.1, 0.4, 0.38),
        ("#E9A3B5", 0.15, 0.9, 0.42), ("#F3CBA8", 0.9, 0.78, 0.32),
    ]:
        blob(canvas, (w * cx, h * cy), w * r, color, 0.65, w * 0.13)
    # 柔焦光斑
    layer = Image.new("RGBA", size, (255, 236, 224, 0))
    d = ImageDraw.Draw(layer)
    for _ in range(22):
        x, y, r = rng.uniform(0, w), rng.uniform(0, h), rng.uniform(24, 90)
        d.ellipse((x - r, y - r, x + r, y + r), fill=(255, 236, 224, rng.randint(36, 80)))
    layer = layer.filter(ImageFilter.GaussianBlur(10))
    # 散落花瓣
    petals = Image.new("RGBA", size, (226, 140, 160, 0))
    for _ in range(30):
        petal(petals, (rng.uniform(0, w), rng.uniform(0, h)), rng.uniform(16, 34), rng.uniform(0, 360),
              rng.choice(["#E28CA0", "#F0AEBB", "#D9788F", "#F2C4A8"]), rng.uniform(0.45, 0.8))
    petals = petals.filter(ImageFilter.GaussianBlur(0.8))
    # 香槟金细点
    dust = Image.new("RGBA", size, (232, 196, 140, 0))
    dd = ImageDraw.Draw(dust)
    for _ in range(110):
        x, y, r = rng.uniform(0, w), rng.uniform(0, h), rng.uniform(1.2, 3.2)
        dd.ellipse((x - r, y - r, x + r, y + r), fill=(232, 196, 140, rng.randint(110, 220)))
    out = canvas.convert("RGBA")
    for part in (layer, petals, dust):
        out.alpha_composite(part)
    return out.convert("RGB")


def banknote(layer: Image.Image, center, width, angle, alpha):
    # 风格化红色钞票：只有边框和花纹，不含真实票面元素
    w, h = round(width), round(width * 0.48)
    note = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    d = ImageDraw.Draw(note)
    a = round(255 * alpha)
    d.rounded_rectangle((0, 0, w - 1, h - 1), radius=round(h * 0.08), fill=(196, 38, 48, a))
    d.rounded_rectangle((h * 0.08, h * 0.08, w - h * 0.08, h - h * 0.08), radius=round(h * 0.06),
                        outline=(232, 186, 96, a), width=max(2, round(h * 0.035)))
    r = h * 0.26
    d.ellipse((w * 0.72 - r, h / 2 - r, w * 0.72 + r, h / 2 + r), outline=(240, 200, 120, a),
              width=max(2, round(h * 0.03)))
    for i in range(5):
        y = h * (0.3 + i * 0.1)
        d.line((w * 0.14, y, w * 0.52, y), fill=(150, 20, 34, a), width=max(1, round(h * 0.02)))
    note = note.rotate(angle, expand=True, resample=Image.BICUBIC)
    layer.alpha_composite(note, (round(center[0] - note.width / 2), round(center[1] - note.height / 2)))


def gold_coin(layer: Image.Image, center, r, alpha):
    # 方孔金币
    d = ImageDraw.Draw(layer)
    x, y = center
    a = round(255 * alpha)
    d.ellipse((x - r, y - r, x + r, y + r), fill=(226, 176, 64, a), outline=(176, 124, 32, a),
              width=max(2, round(r * 0.08)))
    d.ellipse((x - r * 0.78, y - r * 0.78, x + r * 0.78, y + r * 0.78), outline=(250, 216, 130, a),
              width=max(1, round(r * 0.05)))
    s = r * 0.24
    d.rectangle((x - s, y - s, x + s, y + s), fill=(150, 24, 30, a))


def fortune_rain(size, seed=8888):
    rng = random.Random(seed)
    w, h = size
    canvas = vertical_gradient(size, [(0, "#B3141F"), (0.45, "#E2584A"), (0.75, "#FBE3C2"), (1, "#FFF6E6")])
    blob(canvas, (w * 0.5, h * 0.18), w * 0.6, "#F6C35A", 0.45, w * 0.18)
    layer = Image.new("RGBA", size, (0, 0, 0, 0))
    # 上半部分密集飘落，下半部分逐渐稀疏，给歌词留出空间
    for _ in range(46):
        y = h * (rng.random() ** 1.7) * 0.9
        banknote(layer, (rng.uniform(-40, w + 40), y), rng.uniform(220, 360), rng.uniform(-40, 40),
                 rng.uniform(0.75, 0.95))
    for _ in range(40):
        y = h * (rng.random() ** 1.4)
        gold_coin(layer, (rng.uniform(0, w), y), rng.uniform(18, 46), rng.uniform(0.8, 1))
    layer = layer.filter(ImageFilter.GaussianBlur(0.6))
    dust = Image.new("RGBA", size, (255, 214, 120, 0))
    dd = ImageDraw.Draw(dust)
    for _ in range(160):
        x, y, r = rng.uniform(0, w), rng.uniform(0, h), rng.uniform(1.2, 3.4)
        dd.ellipse((x - r, y - r, x + r, y + r), fill=(255, 214, 120, rng.randint(120, 230)))
    out = canvas.convert("RGBA")
    out.alpha_composite(layer)
    out.alpha_composite(dust)
    return out.convert("RGB")


# ---------------------------------------------------------------- 预览图（播放页示意）

def overlay(image: Image.Image, color: str, alpha: float):
    return Image.blend(image.convert("RGB"), Image.new("RGB", image.size, hex_rgb(color)), alpha)


def cover_fit(image: Image.Image, size, focus_y=0.5):
    """按 BoxFit.cover 缩放裁切，focus_y 控制纵向取景位置。"""
    tw, th = size
    scale = max(tw / image.width, th / image.height)
    resized = image.convert("RGB").resize(
        (math.ceil(image.width * scale), math.ceil(image.height * scale)), Image.LANCZOS
    )
    left = (resized.width - tw) // 2
    top = round((resized.height - th) * focus_y)
    return resized.crop((left, top, left + tw, top + th))


def page_backdrop(theme: str, gradient_alpha=None):
    """按 theme.json 的 canvas 与 backgroundImageOpacity 还原页面背景的实际观感。

    gradient_alpha: None 使用统一不透明度；tuple (top, bottom) 上实下虚渐变。
    """
    config = json.loads((PACKAGES / theme / "theme.json").read_text(encoding="utf-8"))
    canvas = Image.new("RGB", PREVIEW_SIZE, hex_rgb(config["colors"]["canvas"]))
    image = cover_fit(Image.open(PACKAGES / theme / config["backgroundImage"]), PREVIEW_SIZE)
    blur = config.get("backgroundImageBlur", 0)
    if blur:
        image = image.filter(ImageFilter.GaussianBlur(blur))

    if gradient_alpha is None:
        # 统一不透明度
        return Image.blend(canvas, image, config.get("backgroundImageOpacity", 1.0))
    else:
        # 垂直渐变不透明度
        top_alpha, bottom_alpha = gradient_alpha
        w, h = PREVIEW_SIZE
        mask = Image.new("L", PREVIEW_SIZE)
        px = mask.load()
        for y in range(h):
            t = y / (h - 1)
            alpha_val = round(255 * (top_alpha + (bottom_alpha - top_alpha) * t))
            for x in range(w):
                px[x, y] = alpha_val
        result = canvas.copy()
        result.paste(image, (0, 0), mask)
        return result


def controls(draw, y, accent, muted, play_shape="circle"):
    w = PREVIEW_SIZE[0]
    cx = w / 2
    # 进度条
    draw.rounded_rectangle((70, y - 90, w - 70, y - 84), radius=3, fill=muted)
    draw.rounded_rectangle((70, y - 90, 70 + (w - 140) * 0.38, y - 84), radius=3, fill=accent)
    draw.ellipse((70 + (w - 140) * 0.38 - 9, y - 96, 70 + (w - 140) * 0.38 + 9, y - 78), fill=accent)
    # 上一首 / 下一首
    for sign in (-1, 1):
        x = cx + sign * 150
        tri = [(x - sign * 16, y - 18), (x - sign * 16, y + 18), (x + sign * 14, y)]
        draw.polygon(tri, fill=muted)
        draw.rectangle((x + sign * 14 - 3, y - 18, x + sign * 14 + 3, y + 18), fill=muted)
    # 播放键
    r = 46
    if play_shape == "square":
        draw.rounded_rectangle((cx - r, y - r, cx + r, y + r), radius=10, fill=accent)
    else:
        draw.ellipse((cx - r, y - r, cx + r, y + r), fill=accent)
    draw.rectangle((cx - 14, y - 18, cx - 5, y + 18), fill="#FFFFFF")
    draw.rectangle((cx + 5, y - 18, cx + 14, y + 18), fill="#FFFFFF")


def lyric_bars(draw, top, accent, muted, align="center", widths=(0.52, 0.66, 0.44, 0.58)):
    w = PREVIEW_SIZE[0]
    y = top
    for i, k in enumerate(widths):
        bw = w * k
        h = 22 if i == 1 else 16
        x0 = (w - bw) / 2 if align == "center" else 70
        rounded_bar(draw, (x0, y, x0 + bw, y + h), accent if i == 1 else muted)
        y += 54


def preview_crimson(out: Path, font_path: Path):
    w, h = PREVIEW_SIZE
    # 以主题实际的全局背景铺底
    img = page_backdrop("crimson-cloud")
    d = ImageDraw.Draw(img)
    centered_text(d, 74, "朱红云音", ImageFont.truetype(str(font_path), 52), "#222222")
    rounded_bar(d, (w / 2 - 80, 146, w / 2 + 80, 162), "#8A8A8A")
    # 黑胶唱片
    cx, cy, r = w / 2, 470, 245
    d.ellipse((cx - r, cy - r, cx + r, cy + r), fill="#141414")
    for k in range(8):
        rr = r - 18 - k * 14
        d.ellipse((cx - rr, cy - rr, cx + rr, cy + rr), outline="#262626", width=2)
    cover = vertical_gradient((300, 300), [(0, "#F06B5B"), (1, "#8E3B70")])
    mask = Image.new("L", (300, 300), 0)
    ImageDraw.Draw(mask).ellipse((0, 0, 300, 300), fill=255)
    img.paste(cover, (round(cx - 150), round(cy - 150)), mask)
    d.ellipse((cx - 12, cy - 12, cx + 12, cy + 12), fill="#F7F7F7")
    # 唱臂
    d.line((w - 150, 170, w - 230, 300), fill="#D0D0D0", width=10)
    d.ellipse((w - 168, 152, w - 132, 188), fill="#E8E8E8")
    lyric_bars(d, 790, "#D93636", "#9A9A9A")
    controls(d, 1150, "#D93636", "#555555")
    save_jpg(img, out)


def preview_ink(out: Path, font_path: Path):
    w, h = PREVIEW_SIZE
    img = page_backdrop("ink-wash")
    d = ImageDraw.Draw(img)
    # 方形封面：取背景图下部山水作为示意封面
    box = (130, 170, w - 130, 170 + w - 260)
    d.rectangle((box[0] - 6, box[1] - 6, box[2] + 6, box[3] + 6), fill="#2B2B2B")
    source = Image.open(PACKAGES / "ink-wash" / "assets" / "backgrounds" / "ink.jpg")
    cover = cover_fit(source, (box[2] - box[0], box[3] - box[1]), focus_y=0.7)
    img.paste(cover, box[:2])
    centered_text(d, 690, "山水清音", ImageFont.truetype(str(font_path), 64), "#2B2B2B")
    centered_text(d, 778, "水墨丹青", ImageFont.truetype(str(font_path), 30), "#5E5A52")
    lyric_bars(d, 870, "#B23A2E", "#8C877C", widths=(0.48, 0.6, 0.42))
    controls(d, 1150, "#2B2B2B", "#5E5A52", play_shape="square")
    save_jpg(img, out)


def preview_sakura(out: Path, font_path: Path):
    w, h = PREVIEW_SIZE
    img = page_backdrop("sakura-dream").convert("RGBA")
    cx, cy, r = w / 2, 450, 220
    # 透明像素取发光色本身，避免模糊时混入黑色产生灰边
    glow = Image.new("RGBA", PREVIEW_SIZE, (255, 150, 200, 0))
    ImageDraw.Draw(glow).ellipse((cx - r - 40, cy - r - 40, cx + r + 40, cy + r + 40), fill=(255, 150, 200, 150))
    img.alpha_composite(glow.filter(ImageFilter.GaussianBlur(36)))
    cover = vertical_gradient((2 * r, 2 * r), [(0, "#FF9CC6"), (0.5, "#C7A6FF"), (1, "#8ED0FF")]).convert("RGBA")
    layer = Image.new("RGBA", cover.size, (0, 0, 0, 0))
    rng = random.Random(3)
    for _ in range(9):
        sparkle(layer, (rng.uniform(40, 2 * r - 40), rng.uniform(40, 2 * r - 40)), rng.uniform(12, 26), "#FFFFFF", 0.85)
    cover.alpha_composite(layer)
    mask = Image.new("L", cover.size, 0)
    ImageDraw.Draw(mask).ellipse((0, 0, 2 * r, 2 * r), fill=255)
    img.paste(cover, (round(cx - r), round(cy - r)), mask)
    img = img.convert("RGB")
    d = ImageDraw.Draw(img)
    d.ellipse((cx - r, cy - r, cx + r, cy + r), outline="#FFFFFF", width=6)
    centered_text(d, 708, "樱色星梦", ImageFont.truetype(str(font_path), 54), "#4A3A5C")
    rounded_bar(d, (w / 2 - 90, 784, w / 2 + 90, 800), "#9C88B0")
    # 发光高亮行
    halo = Image.new("RGBA", PREVIEW_SIZE, (255, 120, 180, 0))
    ImageDraw.Draw(halo).rounded_rectangle((w * 0.15, 900, w * 0.85, 936), radius=18, fill=(255, 120, 180, 160))
    img = img.convert("RGBA")
    img.alpha_composite(halo.filter(ImageFilter.GaussianBlur(14)))
    img = img.convert("RGB")
    d = ImageDraw.Draw(img)
    lyric_bars(d, 852, "#D63A7E", "#B9A6C9", widths=(0.5, 0.64, 0.46))
    controls(d, 1150, "#D63A7E", "#7D6A8F")
    save_jpg(img, out)


def preview_rose(out: Path, font_path: Path):
    w, h = PREVIEW_SIZE
    # 沉浸式：播放页背景图本身铺底，下半部分加浅色遮罩保证文字可读
    img = cover_fit(Image.open(PACKAGES / "rose-belle" / "assets" / "backgrounds" / "rose.jpg"), PREVIEW_SIZE)
    shade = Image.new("RGB", PREVIEW_SIZE, hex_rgb("#FFF4F1"))
    mask = Image.linear_gradient("L").resize(PREVIEW_SIZE).point(lambda v: round(70 + v * 0.62))
    img = Image.composite(shade, img, mask).convert("RGBA")
    # 发光圆角封面
    box = (w / 2 - 190, 210, w / 2 + 190, 590)
    glow = Image.new("RGBA", PREVIEW_SIZE, (242, 168, 186, 0))
    ImageDraw.Draw(glow).rounded_rectangle((box[0] - 30, box[1] - 30, box[2] + 30, box[3] + 30), radius=60,
                                           fill=(242, 168, 186, 150))
    img.alpha_composite(glow.filter(ImageFilter.GaussianBlur(34)))
    size = (round(box[2] - box[0]), round(box[3] - box[1]))
    cover = cover_fit(Image.open(PACKAGES / "rose-belle" / "assets" / "backgrounds" / "rose.jpg"), size, focus_y=0.25)
    cmask = Image.new("L", size, 0)
    ImageDraw.Draw(cmask).rounded_rectangle((0, 0, size[0], size[1]), radius=28, fill=255)
    img.paste(cover, (round(box[0]), round(box[1])), cmask)
    img = img.convert("RGB")
    d = ImageDraw.Draw(img)
    centered_text(d, 650, "玫瑰佳人", ImageFont.truetype(str(font_path), 56), "#3E2A30")
    centered_text(d, 730, "一曲倾城", ImageFont.truetype(str(font_path), 28), "#7A5C63")
    lyric_bars(d, 830, "#C2506E", "#B79AA0", widths=(0.48, 0.64, 0.42, 0.54))
    controls(d, 1150, "#C2506E", "#7A5C63")
    save_jpg(img, out)


def preview_yiyi(out: Path, font_path: Path):
    w, h = PREVIEW_SIZE
    source = Image.open(PACKAGES / "jiang-yiyi" / "assets" / "backgrounds" / "yiyi.jpg")
    # 沉浸式：写真铺底，自上而下渐入雾蓝遮罩保证文字可读
    img = cover_fit(source, PREVIEW_SIZE, focus_y=0.3)
    shade = Image.new("RGB", PREVIEW_SIZE, hex_rgb("#EEF3FA"))
    mask = Image.linear_gradient("L").resize(PREVIEW_SIZE).point(lambda v: round(125 + v * 0.45))
    img = Image.composite(shade, img, mask).convert("RGBA")
    # 发光圆形封面，裁人物面部
    cx, cy, r = w / 2, 400, 190
    glow = Image.new("RGBA", PREVIEW_SIZE, (150, 180, 225, 0))
    ImageDraw.Draw(glow).ellipse((cx - r - 30, cy - r - 30, cx + r + 30, cy + r + 30), fill=(150, 180, 225, 160))
    img.alpha_composite(glow.filter(ImageFilter.GaussianBlur(34)))
    side = 2 * r
    face = source.crop((90, 120, 630, 660)).resize((side, side), Image.LANCZOS)
    cmask = Image.new("L", (side, side), 0)
    ImageDraw.Draw(cmask).ellipse((0, 0, side, side), fill=255)
    img.paste(face, (round(cx - r), round(cy - r)), cmask)
    img = img.convert("RGB")
    d = ImageDraw.Draw(img)
    centered_text(d, 650, "依依蓝羽", ImageFont.truetype(str(font_path), 56), "#26324A")
    centered_text(d, 730, "羽落晴空", ImageFont.truetype(str(font_path), 28), "#5A6680")
    lyric_bars(d, 830, "#5B7FB8", "#9DAAC2", widths=(0.48, 0.64, 0.42, 0.54))
    controls(d, 1150, "#5B7FB8", "#5A6680")
    save_jpg(img, out)


def preview_fortune(out: Path, font_path: Path):
    w, h = PREVIEW_SIZE
    # 上实下虚：页面背景用渐变透明度遮罩
    img = page_backdrop("fortune-gold", gradient_alpha=(0.65, 0.15)).convert("RGBA")
    source = Image.open(PACKAGES / "fortune-gold" / "assets" / "backgrounds" / "fortune.jpg")
    # 金色光晕 + 金边圆形封面
    cx, cy, r = w / 2, 400, 190
    glow = Image.new("RGBA", PREVIEW_SIZE, (246, 195, 90, 0))
    ImageDraw.Draw(glow).ellipse((cx - r - 34, cy - r - 34, cx + r + 34, cy + r + 34), fill=(246, 195, 90, 190))
    img.alpha_composite(glow.filter(ImageFilter.GaussianBlur(34)))
    side = 2 * r
    cover = cover_fit(source, (side, side), focus_y=0.2)
    cmask = Image.new("L", (side, side), 0)
    ImageDraw.Draw(cmask).ellipse((0, 0, side, side), fill=255)
    img.paste(cover, (round(cx - r), round(cy - r)), cmask)
    ImageDraw.Draw(img).ellipse((cx - r, cy - r, cx + r, cy + r), outline=(214, 160, 52, 255), width=8)
    img = img.convert("RGB")
    d = ImageDraw.Draw(img)
    centered_text(d, 650, "家财万贯", ImageFont.truetype(str(font_path), 60), "#8E1A1A")
    centered_text(d, 730, "招财进宝", ImageFont.truetype(str(font_path), 30), "#A8742A")
    lyric_bars(d, 830, "#C8242C", "#D2AE86", widths=(0.48, 0.64, 0.42, 0.54))
    controls(d, 1150, "#C8242C", "#8A5A3A")
    save_jpg(img, out)


def missing_photo(theme: str, name: str):
    # 写真类主题没有程序化背景，素材缺失时直接报错
    raise SystemExit(f"缺少写真背景：theme/packages/{theme}/assets/backgrounds/{name}")


# 各主题内置字体（均来自 Google Fonts，SIL OFL 1.1，许可证见 theme/licenses/）
FONTS = {
    "crimson-cloud": "ZCOOLQingKeHuangYou-Regular.ttf",
    "ink-wash": "MaShanZheng-Regular.ttf",
    "sakura-dream": "ZCOOLKuaiLe-Regular.ttf",
    "rose-belle": "ZCOOLXiaoWei-Regular.ttf",
    "jiang-yiyi": "ZCOOLXiaoWei-Regular.ttf",
    "fortune-gold": "MaShanZheng-Regular.ttf",
}


def main():
    fonts = {}
    for theme, name in FONTS.items():
        path = PACKAGES / theme / "assets" / "fonts" / name
        if not path.exists():
            raise SystemExit(f"缺少字体文件：{path.relative_to(ROOT)}（Google Fonts，SIL OFL 1.1）")
        fonts[theme] = path

    # 默认保留已有背景图（可能是手动替换的素材），只有缺失或显式传 --backgrounds 时才程序化重绘
    redraw = "--backgrounds" in sys.argv[1:]
    backgrounds = {
        "crimson-cloud": ("crimson.jpg", lambda: crimson_backdrop(BG_SIZE)),
        "ink-wash": ("ink.jpg", lambda: ink_landscape(BG_SIZE)),
        "sakura-dream": ("sakura.jpg", lambda: sakura_sky(BG_SIZE)),
        "rose-belle": ("rose.jpg", lambda: rose_bloom(BG_SIZE)),
        "jiang-yiyi": ("yiyi.jpg", lambda: missing_photo("jiang-yiyi", "yiyi.jpg")),
        "fortune-gold": ("fortune.jpg", lambda: fortune_rain(BG_SIZE)),
    }
    previews = {
        "crimson-cloud": preview_crimson,
        "ink-wash": preview_ink,
        "sakura-dream": preview_sakura,
        "rose-belle": preview_rose,
        "jiang-yiyi": preview_yiyi,
        "fortune-gold": preview_fortune,
    }
    for theme, (name, draw) in backgrounds.items():
        print(theme)
        path = PACKAGES / theme / "assets" / "backgrounds" / name
        if redraw or not path.exists():
            save_jpg(draw(), path, quality=84)
            print(f"  重绘背景 {path.relative_to(ROOT)}")
        else:
            print(f"  保留背景 {path.relative_to(ROOT)}")
        previews[theme](PACKAGES / theme / "preview.jpg", fonts[theme])


if __name__ == "__main__":
    main()
