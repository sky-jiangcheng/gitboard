#!/usr/bin/env python3
"""Generate mock screenshots reflecting the v1.5.0 UI for README.

Renders a light, Claude-inspired theme with: 3-item nav + Cmd+K palette, a
dashboard hero goal ring, a knowledge hub page, a project detail overview, and
settings. Output is written to the repo's screenshots/ directory.

CJK-capable fonts are preferred so Chinese labels render correctly.
"""

from PIL import Image, ImageDraw, ImageFont
import math
import os

# Repo-relative output (script lives in scripts/).
OUTPUT_DIR = os.path.realpath(os.path.join(os.path.dirname(__file__), "..", "screenshots"))
WIDTH = 1200

# --- Palette (matches web/src/styles/global.css light theme) ---
BG = (249, 249, 247)
CARD = (255, 255, 255)
TERTIARY = (242, 242, 239)
BORDER = (235, 233, 228)
TEXT = (28, 28, 26)
TEXT_SEC = (95, 95, 90)
TEXT_TER = (154, 154, 146)
ACCENT = (74, 125, 74)
ACCENT_LIGHT = (107, 155, 107)
ACCENT_SOFT = (232, 240, 232)
DANGER = (201, 87, 87)
WARNING = (201, 152, 87)
WARNING_SOFT = (248, 240, 232)
INFO = (90, 127, 160)
INFO_SOFT = (232, 239, 245)

_FONT_CANDIDATES = [
    "/System/Library/Fonts/Supplemental/Arial Unicode.ttf",  # macOS (CJK + Latin)
    "/System/Library/Fonts/Hiragino Sans GB.ttc",
    "/System/Library/Fonts/STHeiti Medium.ttc",
    "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",  # Linux
    "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
]


def _load_font(size, bold=False):
    for path in _FONT_CANDIDATES:
        try:
            return ImageFont.truetype(path, size)
        except Exception:
            continue
    return ImageFont.load_default()


_FONT_CACHE = {}


def get_font(size, bold=False):
    # Arial Unicode has no separate bold face; emulate bold with a slightly
    # larger size for headings.
    key = (size, bold)
    if key not in _FONT_CACHE:
        _FONT_CACHE[key] = _load_font(size + (1 if bold else 0), bold)
    return _FONT_CACHE[key]


def text_w(draw, s, font):
    try:
        return draw.textlength(s, font=font)
    except Exception:
        return len(s) * size // 2


def rrect(draw, xy, radius, fill=None, outline=None, width=1):
    draw.rounded_rectangle(xy, radius=radius, fill=fill, outline=outline, width=width)


def text(draw, xy, s, fill, font, anchor="la"):
    draw.text(xy, s, fill=fill, font=font, anchor=anchor)


def chip(draw, x, y, label, fg, bg, font):
    w = text_w(draw, label, font) + 18
    rrect(draw, [(x, y), (x + w, y + 22)], 11, fill=bg)
    text(draw, (x + 9, y + 2), label, fg, font)
    return w


def draw_navbar(draw, active="仪表盘"):
    rrect(draw, [(0, 0), (WIDTH, 56)], 0, fill=CARD)
    draw.line([(0, 56), (WIDTH, 56)], fill=BORDER, width=1)
    brand = get_font(16, bold=True)
    text(draw, (32, 28), "▦ GitBoard", TEXT, brand)
    links = ["仪表盘", "知识库", "设置"]
    x = 180
    f = get_font(13)
    for l in links:
        w = text_w(draw, l, f) + 24
        if l == active:
            rrect(draw, [(x, 14), (x + w, 42)], 8, fill=TERTIARY)
            text(draw, (x + 12, 28), l, TEXT, get_font(13, bold=True))
        else:
            text(draw, (x + 12, 28), l, TEXT_SEC, f)
        x += w + 4
    # Cmd+K search pill (right)
    pill_w = 150
    px = WIDTH - 32 - pill_w
    rrect(draw, [(px, 16), (px + pill_w, 40)], 8, fill=TERTIARY, outline=BORDER)
    text(draw, (px + 12, 28), "搜索", TEXT_TER, get_font(12))
    kw = "⌘K"
    kfw = text_w(draw, kw, get_font(10))
    rrect(draw, [(px + pill_w - kfw - 22, 22), (px + pill_w - 10, 34)], 4, fill=CARD, outline=BORDER)
    text(draw, (px + pill_w - kfw - 16, 28), kw, TEXT_TER, get_font(10))


def draw_goal_ring(draw, cx, cy, r, ratio, value_label, sub_label):
    # Track
    draw.ellipse([(cx - r, cy - r), (cx + r, cy + r)], outline=TERTIARY, width=9)
    # Progress arc (clockwise from top)
    ratio = max(0.0, min(1.0, ratio))
    color = ACCENT if ratio >= 1 else (ACCENT_LIGHT if ratio >= 0.5 else WARNING)
    if ratio > 0:
        bbox = [(cx - r, cy - r), (cx + r, cy + r)]
        draw.arc(bbox, -90, -90 + 360 * ratio, fill=color, width=9)
    text(draw, (cx, cy - 2), value_label, TEXT, get_font(22, bold=True), anchor="mm")
    text(draw, (cx, cy + 16), sub_label, TEXT_TER, get_font(10), anchor="mm")


def draw_summary_bar(draw, x, y, w):
    h = 70
    rrect(draw, [(x, y), (x + w, y + h)], 10, fill=CARD, outline=BORDER)
    stats = [("仓库", "12"), ("团队新增", "+1,847"), ("个人新增", "+1,203"), ("文件", "38"), ("日期", "工作日")]
    n = len(stats)
    col_w = w / n
    for i, (label, val) in enumerate(stats):
        cx = x + 20 + i * col_w
        text(draw, (cx, y + 14), label, TEXT_TER, get_font(11))
        color = ACCENT if "+" in val and "团队" in label else (ACCENT if val.startswith("+") and i == 2 else TEXT)
        if i == 1:
            color = ACCENT
        text(draw, (cx, y + 32), val, color, get_font(18, bold=True))
    return h


def draw_heatmap(draw, x, y, w):
    rrect(draw, [(x, y), (x + w, y + 90)], 10, fill=CARD, outline=BORDER)
    text(draw, (x + 16, y + 14), "提交热力图", TEXT, get_font(13, bold=True))
    # 30 weeks x 7 days mini grid
    cell = 9
    gap = 2
    gx = x + 16
    gy = y + 40
    import random  # deterministic-ish mock
    rng = random.Random(7)
    for wi in range(30):
        for di in range(7):
            lvl = rng.choice([0, 0, 0, 1, 1, 2, 3, 4]) if rng.random() > 0.4 else 0
            colors = [(235, 233, 228), (155, 233, 168), (64, 196, 99), (48, 161, 78), (33, 110, 57)]
            cx = gx + wi * (cell + gap)
            cy = gy + di * (cell + gap)
            rrect(draw, [(cx, cy), (cx + cell, cy + cell)], 2, fill=colors[lvl])
    return 90


def draw_project_card(draw, x, y, w, name, added, deleted, files, goal_pct, reached, below):
    h = 168
    fill = CARD
    outline = ACCENT if reached else BORDER
    rrect(draw, [(x, y), (x + w, y + h)], 10, fill=fill, outline=outline, width=(2 if reached else 1))
    # name + badges
    text(draw, (x + 14, y + 14), name, TEXT, get_font(14, bold=True))
    bx = x + w - 14
    if reached:
        bw = chip(draw, bx - 50, y + 12, "达标", ACCENT, ACCENT_SOFT, get_font(10))
        bx -= 50 + 4
    if below:
        chip(draw, bx - 50, y + 12, "未达标", WARNING, WARNING_SOFT, get_font(10))
    # hero number
    text(draw, (x + 14, y + 46), "今日新增", TEXT_TER, get_font(10))
    text(draw, (x + 14, y + 60), f"+{added}", ACCENT if added > 0 else TEXT, get_font(20, bold=True))
    # goal bar
    by = y + 92
    rrect(draw, [(x + 14, by), (x + w - 14, by + 5)], 3, fill=TERTIARY)
    rrect(draw, [(x + 14, by), (x + 14 + int((w - 28) * goal_pct / 100), by + 5)], 3, fill=ACCENT)
    text(draw, (x + w - 14, by + 10), f"{goal_pct}% 目标", TEXT_TER, get_font(10), anchor="ra")
    # mini stats
    sy = y + 120
    cols = [("仓库", "3"), ("文件", str(files)), ("新增", f"+{added}"), ("删除", f"-{deleted}")]
    cw = (w - 28) / 4
    for i, (l, v) in enumerate(cols):
        cx = x + 14 + i * cw
        text(draw, (cx, sy), l, TEXT_TER, get_font(10))
        col = ACCENT if l == "新增" else (DANGER if l == "删除" else TEXT)
        text(draw, (cx, sy + 14), v, col, get_font(13, bold=True))
    return h


def draw_dashboard():
    H = 840
    img = Image.new("RGB", (WIDTH, H), BG)
    d = ImageDraw.Draw(img)
    draw_navbar(d, "仪表盘")

    # Hero card: goal ring + text
    hx, hy, hw = 24, 76, 560
    hh = 120
    rrect(d, [(hx, hy), (hx + hw, hy + hh)], 10, fill=CARD, outline=BORDER)
    draw_goal_ring(d, hx + 64, hy + 60, 44, 0.72, "72%", "今日目标")
    text(d, (hx + 130, hy + 22), "2026-07-20 · 工作日", TEXT_TER, get_font(11))
    text(d, (hx + 130, hy + 40), "今日目标已达成 🎉", TEXT, get_font(16, bold=True))
    text(d, (hx + 130, hy + 66), "个人新增 +1,203 · 文件 38 · 涉及 12 个仓库", TEXT_SEC, get_font(12))

    # Summary bar to the right of hero
    draw_summary_bar(d, 600, 76, 576)

    # Heatmap
    draw_heatmap(d, 24, 212, WIDTH - 48)

    # Cards grid (4 cols x 2 rows)
    projects = [
        ("business-toolkit", 420, 85, 12, 84, True, False),
        ("GitBoard", 180, 30, 5, 36, True, False),
        ("user-service", 0, 0, 0, 0, False, True),
        ("api-gateway", 95, 12, 3, 19, False, False),
        ("data-platform", 0, 0, 0, 0, False, True),
        ("frontend-app", 310, 45, 8, 62, True, False),
        ("infra-tools", 0, 0, 0, 0, False, True),
        ("monorepo-root", 156, 28, 4, 31, False, False),
    ]
    gap = 14
    cw = (WIDTH - 48 - 3 * gap) / 4
    sx, sy = 24, 322
    for i, (name, a, dl, f, gp, reached, below) in enumerate(projects):
        col = i % 4
        row = i // 4
        x = sx + col * (cw + gap)
        y = sy + row * (168 + gap)
        draw_project_card(d, int(x), int(y), int(cw), name, a, dl, f, gp, reached, below)

    img.save(os.path.join(OUTPUT_DIR, "dashboard.png"), "PNG")
    print("Generated dashboard.png")


def draw_knowledge():
    H = 820
    img = Image.new("RGB", (WIDTH, H), BG)
    d = ImageDraw.Draw(img)
    draw_navbar(d, "知识库")

    text(d, (32, 84), "知识库", TEXT, get_font(26, bold=True))
    text(d, (32, 120), "跨项目汇总 18 条笔记，支持全文搜索、标签筛选与置顶。", TEXT_TER, get_font(12))

    # Search box
    rrect(d, [(32, 150), (620, 184)], 8, fill=CARD, outline=BORDER)
    text(d, (48, 167), "搜索笔记与待办…", TEXT_TER, get_font(13))

    # Import button
    rrect(d, [(640, 150), (820, 184)], 8, fill=TERTIARY, outline=BORDER)
    text(d, (660, 167), "导入 Claude 记忆", TEXT_SEC, get_font(12))

    # Filter chips
    fx = 32
    fy = 204
    for label, active in [("全部", True), ("知识", False), ("其他", False)]:
        fg = ACCENT if active else TEXT_TER
        bg = ACCENT_SOFT if active else CARD
        w = text_w(d, label, get_font(12)) + 22
        rrect(d, [(fx, fy), (fx + w, fy + 28)], 8, fill=bg, outline=(ACCENT if active else BORDER))
        text(d, (fx + 11, fy + 7), label, fg, get_font(12))
        fx += w + 6
    # pinned chip
    rrect(d, [(fx, fy), (fx + 90, fy + 28)], 8, fill=WARNING_SOFT, outline=WARNING)
    text(d, (fx + 8, fy + 7), "★ 置顶 3", WARNING, get_font(12))
    fx += 96
    # tag chips
    for t in ["架构", "待办", "部署"]:
        w = text_w(d, "#" + t, get_font(11)) + 18
        rrect(d, [(fx, fy + 3), (fx + w, fy + 25)], 10, fill=CARD, outline=BORDER)
        text(d, (fx + 9, fy + 6), "#" + t, TEXT_SEC, get_font(11))
        fx += w + 6

    # Note cards grid (3 cols x 2 rows)
    notes = [
        ("项目知识", "business-toolkit", "knowledge", True, "## 分层架构\n前端 React + 后端 Go，通过 Wails 绑定直连…"),
        ("部署清单", "infra-tools", "knowledge", False, "## 环境变量\nPORT、GITBOARD_PORT 控制监听端口…"),
        ("接口约定", "api-gateway", "knowledge", False, "### 统一错误格式\n所有错误返回 {error: msg}…"),
        ("重构计划", "frontend-app", "idea", False, "考虑拆分 Dashboard 组件，抽出 Hero…"),
        ("修复记录", "user-service", "log", False, "登录态丢失：cookie domain 配置错误…"),
        ("性能笔记", "data-platform", "knowledge", True, "## 查询优化\n索引 daily_stats(stat_date)…"),
    ]
    gap = 14
    cw = (WIDTH - 48 - 2 * gap) / 3
    ch = 150
    sx, sy = 32, 256
    kind_meta = {"knowledge": ("知识", ACCENT, ACCENT_SOFT), "idea": ("想法", INFO, INFO_SOFT), "log": ("日志", WARNING, WARNING_SOFT)}
    for i, (title, proj, kind, pinned, body) in enumerate(notes):
        col = i % 3
        row = i // 3
        x = int(sx + col * (cw + gap))
        y = int(sy + row * (ch + gap))
        outline = WARNING if pinned else BORDER
        rrect(d, [(x, y), (x + int(cw), y + ch)], 10, fill=CARD, outline=outline, width=(2 if pinned else 1))
        # kind badge + pin
        kl, kfg, kbg = kind_meta[kind]
        chip(d, x + 12, y + 12, kl, kfg, kbg, get_font(10))
        if pinned:
            text(d, (x + int(cw) - 22, y + 12), "★", WARNING, get_font(13))
        text(d, (x + 12, y + 42), title, TEXT, get_font(13, bold=True))
        # body (truncated, 2 lines)
        text(d, (x + 12, y + 64), body.split("\n")[0][:34], TEXT_SEC, get_font(11))
        text(d, (x + 12, y + 80), body.split("\n")[1][:34] if "\n" in body else "", TEXT_TER, get_font(11))
        text(d, (x + 12, y + ch - 24), proj, ACCENT, get_font(11))
        text(d, (x + int(cw) - 12, y + ch - 24), "07-20", TEXT_TER, get_font(10), anchor="ra")

    img.save(os.path.join(OUTPUT_DIR, "knowledge.png"), "PNG")
    print("Generated knowledge.png")


def draw_project_detail():
    H = 980
    img = Image.new("RGB", (WIDTH, H), BG)
    d = ImageDraw.Draw(img)
    draw_navbar(d, "仪表盘")
    text(d, (32, 76), "← 返回仪表盘", TEXT_SEC, get_font(13))

    # Header card
    rrect(d, [(24, 104), (WIDTH - 24, 200)], 10, fill=CARD, outline=BORDER)
    text(d, (44, 120), "business-toolkit", TEXT, get_font(20, bold=True))
    text(d, (44, 152), "/home/user/projects/business-toolkit", TEXT_TER, get_font(11))
    # level buttons
    rrect(d, [(WIDTH - 280, 124), (WIDTH - 200, 150)], 6, fill=TERTIARY, outline=BORDER)
    text(d, (WIDTH - 270, 137), "向下拆分", TEXT_SEC, get_font(12))
    rrect(d, [(WIDTH - 190, 124), (WIDTH - 110, 150)], 6, fill=TERTIARY, outline=BORDER)
    text(d, (WIDTH - 180, 137), "向上合并", TEXT_SEC, get_font(12))
    # stat row
    stats = [("子仓库", "3"), ("活跃天数", "42"), ("文件变更", "1,847"), ("新增", "+12,300"), ("删除", "-2,100")]
    sx = 44
    for l, v in stats:
        text(d, (sx, 168), l, TEXT_TER, get_font(10))
        col = ACCENT if l == "新增" else (DANGER if l == "删除" else TEXT)
        text(d, (sx, 180), v, col, get_font(13, bold=True))
        sx += 120

    # Overview card
    oy = 216
    rrect(d, [(24, oy), (WIDTH - 24, oy + 250)], 10, fill=CARD, outline=BORDER)
    text(d, (44, oy + 16), "项目概览", TEXT, get_font(15, bold=True))
    text(d, (WIDTH - 44, oy + 18), "实时挖掘", TEXT_TER, get_font(10), anchor="ra")
    # tech chips
    tx = 44
    ty = oy + 48
    for t in ["Go", "JavaScript / TypeScript", "React", "Docker"]:
        w = text_w(d, t, get_font(11)) + 18
        rrect(d, [(tx, ty), (tx + w, ty + 24)], 12, fill=ACCENT_SOFT)
        text(d, (tx + 9, ty + 4), t, ACCENT, get_font(11))
        tx += w + 6
    # language bars
    ly = oy + 88
    text(d, (44, ly), "语言占比", TEXT_SEC, get_font(11))
    langs = [("Go", 1.0), ("TypeScript", 0.62), ("JavaScript", 0.31), ("CSS", 0.18)]
    ry = ly + 20
    for name, ratio in langs:
        text(d, (44, ry), name, TEXT_SEC, get_font(11))
        rrect(d, [(150, ry + 2), (150 + 360, ry + 10)], 3, fill=TERTIARY)
        rrect(d, [(150, ry + 2), (150 + int(360 * ratio), ry + 10)], 3, fill=ACCENT_LIGHT)
        text(d, (520, ry), str(int(ratio * 1000)), TEXT_TER, get_font(11))
        ry += 22
    # commit feed (right column of overview)
    cx = 640
    text(d, (cx, oy + 88), "最近提交", TEXT_SEC, get_font(11))
    cy = oy + 110
    commits = ["feat: add knowledge hub", "fix: regroup on level change", "refactor: split note editor", "chore: bump to 1.5.0"]
    for c in commits:
        d.ellipse([(cx, cy + 4), (cx + 7, cy + 11)], fill=ACCENT)
        text(d, (cx + 16, cy), c, TEXT, get_font(11))
        text(d, (cx + 16, cy + 14), "main · jiangcheng", TEXT_TER, get_font(10))
        cy += 32

    # Trend chart
    ty2 = oy + 266
    rrect(d, [(24, ty2), (WIDTH - 24, ty2 + 200)], 10, fill=CARD, outline=BORDER)
    text(d, (44, ty2 + 16), "趋势图", TEXT, get_font(15, bold=True))
    chart_x, chart_y = 60, ty2 + 50
    chart_w, chart_h = WIDTH - 120, 120
    d.rectangle([(chart_x, chart_y), (chart_x + chart_w, chart_y + chart_h)], fill=TERTIARY)
    days = ["07-14", "07-15", "07-16", "07-17", "07-18", "07-19", "07-20"]
    vals = [180, 320, 95, 420, 280, 350, 420]
    mx = max(vals)
    pts = []
    for i, v in enumerate(vals):
        px = chart_x + int(i * chart_w / (len(vals) - 1))
        py = chart_y + chart_h - int(v / mx * chart_h)
        pts.append((px, py))
    poly = [(pts[0][0], chart_y + chart_h)] + pts + [(pts[-1][0], chart_y + chart_h)]
    d.polygon(poly, fill=(74, 125, 74, 40))
    for i in range(len(pts) - 1):
        d.line([pts[i], pts[i + 1]], fill=ACCENT, width=3)
    for px, py in pts:
        d.ellipse([(px - 4, py - 4), (px + 4, py + 4)], fill=ACCENT)
    for i, day in enumerate(days):
        text(d, (pts[i][0] - 14, chart_y + chart_h + 6), day, TEXT_TER, get_font(10))

    img.save(os.path.join(OUTPUT_DIR, "project-detail.png"), "PNG")
    print("Generated project-detail.png")


def draw_settings():
    H = 720
    img = Image.new("RGB", (WIDTH, H), BG)
    d = ImageDraw.Draw(img)
    draw_navbar(d, "设置")
    text(d, (32, 80), "设置", TEXT, get_font(26, bold=True))

    # Tabs
    tabs = ["扫描目录", "代码标准", "作者配置", "外观", "操作"]
    tx = 32
    for i, t in enumerate(tabs):
        active = t == "操作"
        col = TEXT if active else TEXT_TER
        text(d, (tx, 130), t, col, get_font(13, bold=active))
        if active:
            d.line([(tx, 150), (tx + text_w(d, t, get_font(13, bold=True)), 150)], fill=ACCENT, width=2)
        tx += text_w(d, t, get_font(13, bold=True)) + 28
    d.line([(32, 152), (WIDTH - 32, 152)], fill=BORDER)

    # Action section
    rrect(d, [(24, 172), (WIDTH - 24, H - 24)], 10, fill=CARD, outline=BORDER)
    text(d, (44, 196), "操作", TEXT, get_font(17, bold=True))
    text(d, (44, 226), "手动触发全量重新扫描，刷新所有仓库的统计数据。", TEXT_TER, get_font(12))
    rrect(d, [(44, 252), (260, 286)], 8, fill=TEXT)
    text(d, (80, 269), "立即重新扫描所有项目", CARD, get_font(12, bold=True), anchor="lm")

    text(d, (44, 320), "导入 Claude 记忆", TEXT, get_font(17, bold=True))
    text(d, (44, 350), "将 ~/.claude/projects/*/memory/*.md 按项目匹配导入为知识笔记。", TEXT_TER, get_font(12))
    text(d, (44, 370), "重复导入会更新已有笔记而非重复创建。", TEXT_TER, get_font(12))
    rrect(d, [(44, 396), (220, 430)], 8, fill=TEXT)
    text(d, (64, 413), "导入 Claude 记忆", CARD, get_font(12, bold=True), anchor="lm")
    rrect(d, [(236, 396), (380, 430)], 8, fill=TERTIARY, outline=BORDER)
    text(d, (256, 413), "前往知识库查看", TEXT_SEC, get_font(12), anchor="lm")

    img.save(os.path.join(OUTPUT_DIR, "settings.png"), "PNG")
    print("Generated settings.png")


if __name__ == "__main__":
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    draw_dashboard()
    draw_knowledge()
    draw_project_detail()
    draw_settings()
    print(f"Saved to {OUTPUT_DIR}")
