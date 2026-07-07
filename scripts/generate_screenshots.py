#!/usr/bin/env python3
"""Generate mock dashboard screenshots for README."""

from PIL import Image, ImageDraw, ImageFont
import os

OUTPUT_DIR = "/workspace/screenshots"
WIDTH, HEIGHT = 1200, 800

# Colors
NAV_BG = (26, 26, 46)
WHITE = (255, 255, 255)
BG = (245, 247, 250)
CARD_BG = (255, 255, 255)
GREEN = (76, 175, 80)
RED = (244, 67, 54)
GRAY = (136, 136, 136)
DARK = (51, 51, 51)
LIGHT_GRAY = (238, 238, 238)
BLUE = (26, 26, 46)
ORANGE = (255, 152, 0)
ORANGE_BG = (255, 243, 205)
ORANGE_TEXT = (133, 100, 4)
CHART_GREEN = (76, 175, 80)
CHART_BG = (249, 249, 249)

def get_font(size, bold=False):
    try:
        if bold:
            return ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", size)
        return ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", size)
    except:
        return ImageFont.load_default()

def draw_navbar(draw):
    """Draw dark navbar at top"""
    draw.rectangle([(0, 0), (WIDTH, 56)], fill=NAV_BG)
    font_brand = get_font(18, bold=True)
    font_link = get_font(14)
    draw.text((24, 16), "GitBoard", fill=WHITE, font=font_brand)
    draw.text((WIDTH - 180, 18), "仪表盘", fill=(200, 200, 200), font=font_link)
    draw.text((WIDTH - 90, 18), "设置", fill=(200, 200, 200), font=font_link)

def draw_summary_bar(draw):
    """Draw summary stats bar"""
    y = 80
    bar_h = 90
    draw.rounded_rectangle([(24, y), (WIDTH - 24, y + bar_h)], radius=8, fill=CARD_BG)

    stats = [
        ("仓库总数", "12"),
        ("总新增行", "+1,847", GREEN),
        ("总删除行", "-426", RED),
        ("个人新增", "1,203"),
        ("文件变更", "38"),
        ("日期类型", "工作日"),
    ]

    x = 48
    font_label = get_font(12)
    font_value = get_font(22, bold=True)

    for label, value, *color in stats:
        c = color[0] if color else DARK
        draw.text((x, y + 16), label, fill=GRAY, font=font_label)
        draw.text((x, y + 36), value, fill=c, font=font_value)
        x += 180

def draw_controls(draw):
    """Draw date picker and rescan button"""
    y = 190
    font_btn = get_font(13)
    font_active = get_font(13, bold=True)

    # Yesterday button (active)
    draw.rounded_rectangle([(24, y), (68, y + 32)], radius=6, fill=BLUE)
    draw.text((32, y + 6), "昨天", fill=WHITE, font=font_active)

    # Today button
    draw.rounded_rectangle([(76, y), (116, y + 32)], radius=6, fill=None, outline=LIGHT_GRAY)
    draw.text((84, y + 6), "今天", fill=DARK, font=font_btn)

    # Date input
    draw.rounded_rectangle([(124, y), (250, y + 32)], radius=6, fill=None, outline=LIGHT_GRAY)
    draw.text((134, y + 6), "2026-07-05", fill=DARK, font=get_font(14))

    # Rescan button
    draw.rounded_rectangle([(WIDTH - 140, y), (WIDTH - 24, y + 32)], radius=6, fill=BLUE)
    draw.text((WIDTH - 128, y + 6), "重新扫描", fill=WHITE, font=font_btn)

def draw_project_card(draw, x, y, w, name, repo_count, added, deleted, net_added, files, below_standard=False):
    """Draw a single project card"""
    card_h = 130
    draw.rounded_rectangle([(x, y), (x + w, y + card_h)], radius=8, fill=CARD_BG)

    # Header
    font_name = get_font(16, bold=True)
    font_badge = get_font(11, bold=True)
    draw.text((x + 16, y + 12), name, fill=DARK, font=font_name)

    if below_standard:
        badge_w = 52
        draw.rounded_rectangle([(x + w - badge_w - 16, y + 12), (x + w - 16, y + 30)], radius=10, fill=ORANGE_BG)
        draw.text((x + w - badge_w - 8, y + 14), "未达标", fill=ORANGE_TEXT, font=font_badge)

    # Divider
    draw.line([(x + 16, y + 40), (x + w - 16, y + 40)], fill=LIGHT_GRAY)

    # Stats rows
    font_label = get_font(13)
    font_val = get_font(13, bold=True)
    row_labels = ["仓库数", "新增行数", "删除行数", "净增行数", "文件变更"]
    row_values = [str(repo_count), f"+{added}", f"-{deleted}", f"+{net_added}" if net_added >= 0 else str(net_added), str(files)]
    row_colors = [DARK, GREEN, RED, DARK, DARK]

    for i, (label, val, color) in enumerate(zip(row_labels, row_values, row_colors)):
        ry = y + 48 + i * 16
        draw.text((x + 16, ry), label, fill=GRAY, font=font_label)
        # right-align value
        val_w = draw.textlength(val, font=font_val)
        draw.text((x + w - 16 - val_w, ry), val, fill=color, font=font_val)

def draw_dashboard():
    """Generate full dashboard screenshot"""
    img = Image.new('RGB', (WIDTH, HEIGHT), BG)
    draw = ImageDraw.Draw(img)

    draw_navbar(draw)
    draw_summary_bar(draw)
    draw_controls(draw)

    # Project cards grid
    mock_projects = [
        ("business-toolkit", 3, 420, 85, 335, 12, True),
        ("CodeStat", 1, 180, 30, 150, 5, False),
        ("user-service", 2, 0, 0, 0, 0, True),
        ("api-gateway", 1, 95, 12, 83, 3, False),
        ("data-platform", 4, 0, 0, 0, 0, True),
        ("frontend-app", 1, 310, 45, 265, 8, False),
        ("infra-tools", 2, 0, 0, 0, 0, True),
        ("monorepo-root", 1, 156, 28, 128, 4, False),
    ]

    card_w = 270
    card_h = 130
    gap = 16
    start_x, start_y = 24, 240

    for i, (name, repos, added, deleted, net_added, files, below) in enumerate(mock_projects):
        col = i % 4
        row = i // 4
        x = start_x + col * (card_w + gap)
        y = start_y + row * (card_h + gap)
        draw_project_card(draw, x, y, card_w, name, repos, added, deleted, net_added, files, below)

    img.save(os.path.join(OUTPUT_DIR, "dashboard.png"), "PNG")
    print("Generated dashboard.png")

def draw_project_detail():
    """Generate project detail page screenshot"""
    img = Image.new('RGB', (WIDTH, 900), BG)
    draw = ImageDraw.Draw(img)

    draw_navbar(draw)
    font_back = get_font(14)
    font_h1 = get_font(24, bold=True)
    font_path = get_font(13)
    font_label = get_font(14)
    font_btn = get_font(13)
    font_section = get_font(16, bold=True)

    # Back button
    y = 80
    draw.text((24, y), "< 返回仪表盘", fill=GRAY, font=font_back)

    # Project header
    y = 110
    draw.text((24, y), "business-toolkit", fill=DARK, font=font_h1)
    draw.text((24, y + 32), "/home/user/projects/business-toolkit", fill=GRAY, font=font_path)

    # Level controls
    y = 160
    draw.text((24, y), "项目级别调整：", fill=DARK, font=font_label)
    draw.rounded_rectangle([(160, y - 2), (230, y + 26)], radius=6, fill=None, outline=LIGHT_GRAY)
    draw.text((168, y + 2), "向上合并", fill=DARK, font=font_btn)
    draw.rounded_rectangle([(238, y - 2), (308, y + 26)], radius=6, fill=None, outline=LIGHT_GRAY)
    draw.text((246, y + 2), "向下拆分", fill=DARK, font=font_btn)
    draw.text((320, y + 2), "自动分组 (偏移: 0)", fill=GRAY, font=font_path)

    # Trend Chart Section
    y = 200
    section_w = WIDTH - 48
    draw.rounded_rectangle([(24, y), (24 + section_w, y + 350)], radius=8, fill=CARD_BG)
    draw.text((44, y + 16), "趋势图", fill=DARK, font=font_section)

    # Draw simulated chart
    chart_x, chart_y = 60, y + 50
    chart_w, chart_h = section_w - 72, 260

    # Chart background grid
    draw.rectangle([(chart_x, chart_y), (chart_x + chart_w, chart_y + chart_h)], fill=CHART_BG)

    # Grid lines
    for gy in range(chart_y + 40, chart_y + chart_h, 40):
        draw.line([(chart_x, gy), (chart_x + chart_w, gy)], fill=(230, 230, 230))

    # Y-axis labels
    for val, gy in [(0, chart_y + chart_h), (100, chart_y + chart_h - 40), (200, chart_y + chart_h - 80), (300, chart_y + chart_h - 120), (400, chart_y + chart_h - 160), (500, chart_y + chart_h - 200)]:
        draw.text((chart_x - 35, gy - 8), str(val), fill=GRAY, font=get_font(11))

    # X-axis - last 7 days
    days = ["06-30", "07-01", "07-02", "07-03", "07-04", "07-05", "07-06"]
    day_values = [180, 320, 95, 420, 280, 350, 156]
    max_val = 500
    point_spacing = chart_w // len(days)

    # Plot line and fill
    points = []
    for i, (day, val) in enumerate(zip(days, day_values)):
        px = chart_x + i * point_spacing + point_spacing // 2
        py = chart_y + chart_h - int(val / max_val * chart_h)
        points.append((px, py))

    # Fill area under line
    fill_points = [(points[0][0], chart_y + chart_h)] + points + [(points[-1][0], chart_y + chart_h)]
    draw.polygon(fill_points, fill=(76, 175, 80, 50))

    # Draw line
    for i in range(len(points) - 1):
        draw.line([points[i], points[i + 1]], fill=CHART_GREEN, width=3)

    # Draw dots and labels
    for i, (px, py) in enumerate(points):
        draw.ellipse([(px - 4, py - 4), (px + 4, py + 4)], fill=CHART_GREEN)
        draw.text((px - 15, chart_y + chart_h + 4), days[i], fill=GRAY, font=get_font(10))

    # Repositories Section
    y = 570
    draw.rounded_rectangle([(24, y), (24 + section_w, y + 300)], radius=8, fill=CARD_BG)
    draw.text((44, y + 16), "子仓库 (3)", fill=DARK, font=font_section)

    mock_repos = [
        ("/home/user/projects/business-toolkit/frontend", [
            ("2026-07-06: +156 -28 (sky-jiangcheng)", GREEN),
            ("2026-07-05: +320 -45 (sky-jiangcheng)", GREEN),
        ]),
        ("/home/user/projects/business-toolkit/backend", [
            ("2026-07-06: +210 -30 (sky-jiangcheng)", GREEN),
            ("2026-07-05: +0 -0 (无提交)", GRAY),
        ]),
        ("/home/user/projects/business-toolkit/shared", [
            ("2026-07-06: +54 -12 (sky-jiangcheng)", GREEN),
        ]),
    ]

    ry = y + 40
    for repo_path, stats in mock_repos:
        # Repo box
        box_h = 70
        draw.rounded_rectangle([(44, ry), (24 + section_w - 20, ry + box_h)], radius=6, fill=CHART_BG)
        draw.text((56, ry + 10), repo_path, fill=GRAY, font=get_font(12))

        tag_y = ry + 32
        for stat_text, color in stats:
            tag_w = draw.textlength(stat_text, font=get_font(11)) + 20
            draw.rounded_rectangle([(56, tag_y), (56 + tag_w, tag_y + 22)], radius=4, fill=(232, 232, 232))
            draw.text((66, tag_y + 3), stat_text, fill=DARK, font=get_font(11))
            tag_y += 28

        ry += box_h + 12

    img.save(os.path.join(OUTPUT_DIR, "project-detail.png"), "PNG")
    print("Generated project-detail.png")

if __name__ == "__main__":
    draw_dashboard()
    draw_project_detail()
