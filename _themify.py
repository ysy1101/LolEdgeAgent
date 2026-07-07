import re

with open('E:\\Desktop\\loledgeagent\\docs\\architecture-diagram.html', 'r', encoding='utf-8') as f:
    content = f.read()

# Dark -> Light background
content = content.replace('background: #0a0a12;', 'background: #f5f5f0;')
content = content.replace('color: #e0e0e0;', 'color: #1a1a1a;')

# Replace the gradient background
old_bg = '''  body::before {
    content: '';
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background:
      radial-gradient(ellipse 80% 60% at 50% 0%, rgba(20, 60, 255, 0.06), transparent),
      radial-gradient(ellipse 60% 50% at 20% 80%, rgba(100, 0, 255, 0.04), transparent),
      radial-gradient(ellipse 60% 50% at 80% 80%, rgba(0, 200, 255, 0.03), transparent);
    pointer-events: none;
    z-index: 0;
  }'''
new_bg = '''  body::before {
    content: '';
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background:
      radial-gradient(ellipse 80% 60% at 50% 0%, rgba(200, 170, 110, 0.05), transparent),
      radial-gradient(ellipse 60% 50% at 20% 80%, rgba(90, 158, 255, 0.03), transparent);
    pointer-events: none;
    z-index: 0;
  }'''
content = content.replace(old_bg, new_bg)

# Header gradient -> solid
old_h = '''    background: linear-gradient(135deg, #f0e6d0 0%, #c8aa6e 50%, #f0e6d0 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    letter-spacing: -0.02em;
    margin-bottom: 8px;'''
new_h = '''    color: #1a1a1a;
    letter-spacing: -0.02em;
    margin-bottom: 8px;'''
content = content.replace(old_h, new_h)

# Colors
content = content.replace('color: #888;', 'color: #666;')
content = content.replace('color: #555;', 'color: #999;')

# Cards
content = content.replace('background: rgba(20, 20, 35, 0.8);', 'background: #ffffff;')
content = content.replace('border: 1px solid rgba(255, 255, 255, 0.06);', 'border: 1px solid #e0dcd4;')
content = content.replace('backdrop-filter: blur(8px);', 'box-shadow: 0 1px 4px rgba(0,0,0,0.04);')
content = content.replace('transition: all 0.3s ease;', 'transition: all 0.25s ease;')
content = content.replace('background: rgba(30, 30, 50, 0.9);', 'background: #fefcf8;')
content = content.replace('rgba(200, 170, 110, 0.3)', '#c8aa6e')
content = content.replace('0 8px 32px rgba(0, 0, 0, 0.4), 0 0 20px rgba(200, 170, 110, 0.05)', '0 8px 24px rgba(0, 0, 0, 0.08)')
content = content.replace('color: #e8dcc8;', 'color: #1a1a1a;')
content = content.replace('.card-body .tech { color: #5a9eff; }', '.card-body .tech { color: #1a73e8; }')
content = content.replace('color: #888; line-height: 1.6; }', 'color: #666; line-height: 1.6; }')

# border-left widths
for old, new in [('3px solid #5a9eff', '4px solid #5a9eff'), ('3px solid #c8aa6e', '4px solid #c8aa6e'),
                 ('3px solid #ff6b6b', '4px solid #e85656'), ('3px solid #4ecdc4', '4px solid #2aa89e'),
                 ('3px solid #a78bfa', '4px solid #8b6fd6'), ('3px solid #f59e0b', '4px solid #d98c0a'),
                 ('3px solid #34d399', '4px solid #28a068')]:
    content = content.replace('border-left: ' + old, 'border-left: ' + new)

# section header
content = content.replace('border-top: 1px solid rgba(255, 255, 255, 0.04);', 'border-top: 1px solid #e0dcd4;')
content = content.replace('background: linear-gradient(90deg, rgba(255,255,255,0.06), transparent);', 'background: linear-gradient(90deg, #ddd8d0, transparent);')

# badges
content = content.replace('background: rgba(255,255,255,0.05);', 'background: #f0ede8;')
content = content.replace('border: 1px solid rgba(255,255,255,0.06);', 'border: 1px solid #e0dcd4;')
content = content.replace('.badge.llm { background: rgba(90, 158, 255, 0.12); color: #5a9eff; }', '.badge.llm { background: #e8f0fe; color: #1a73e8; border-color: #c6dafc; }')
content = content.replace('.badge.go { background: rgba(0, 173, 216, 0.12); color: #00add8; }', '.badge.go { background: #e0f2fe; color: #0284c7; border-color: #b9e6fe; }')
content = content.replace('.badge.ts { background: rgba(49, 120, 198, 0.12); color: #3178c6; }', '.badge.ts { background: #e8edf9; color: #3178c6; border-color: #c5d4ed; }')
content = content.replace('.badge.js { background: rgba(247, 223, 30, 0.12); color: #f7df1e; }', '.badge.js { background: #fef9c3; color: #a16207; border-color: #fde68a; }')

# pipeline
content = content.replace('background: rgba(20, 20, 35, 0.5);', 'background: #ffffff;')
content = content.replace('border: 1px solid rgba(255,255,255,0.05);', 'border: 1px solid #e0dcd4;')
content = content.replace('border-right: 1px solid rgba(255,255,255,0.04);', 'border-right: 1px solid #ece8e0;')
content = content.replace('background: rgba(40, 40, 70, 0.6);', 'background: #faf8f5;')
content = content.replace('color: #c8c0b0;', 'color: #333;')
content = content.replace('.step-desc { font-size: 0.62rem; color: #666; }', '.step-desc { font-size: 0.62rem; color: #888; }')
content = content.replace('font-size: 0.58rem; color: #444;', 'font-size: 0.58rem; color: #aaa;')
content = content.replace('color: #3a3a5a;', 'color: #ccc;')

# tooltip
content = content.replace('background: #1a1a2e;', 'background: #ffffff;')
content = content.replace('border: 1px solid rgba(200, 170, 110, 0.2);', 'border: 1px solid #d0c8b8;')
content = content.replace('border-top-color: rgba(200, 170, 110, 0.2);', 'border-top-color: #d0c8b8;')
content = content.replace('0 12px 48px rgba(0,0,0,0.6)', '0 8px 32px rgba(0,0,0,0.12)')

# particles
content = content.replace('width: 2px; height: 2px;', 'width: 3px;\n    height: 3px;')
content = content.replace('background: rgba(200, 170, 110, 0.3);', 'background: rgba(200, 170, 110, 0.2);')
content = content.replace('animation: float 15s infinite linear;', 'animation: float 18s infinite ease-in-out;')

# section accents
for old, new in [('color:#34d399', 'color:#28a068'), ('color:#c8aa6e', 'color:#b8962a'),
                 ('color:#f59e0b', 'color:#d98c0a'), ('color:#4ecdc4', 'color:#2aa89e'),
                 ('color:#ff6b6b', 'color:#e85656'), ('color:#a78bfa', 'color:#8b6fd6')]:
    content = content.replace(old, new)

content = content.replace('.footer span { color: #c8aa6e; }', '.footer span { color: #b8962a; }')

# flow section
content = content.replace('background:rgba(20,20,35,0.4);border:1px solid rgba(255,255,255,0.04);border-radius:14px;padding:24px 20px', 'background:#ffffff;border:1px solid #e0dcd4;border-radius:14px;padding:24px 20px;box-shadow:0 1px 4px rgba(0,0,0,0.04)')

flow_styles = {
    'background:rgba(255,107,107,0.1);padding:6px 14px;border-radius:8px;border:1px solid rgba(255,107,107,0.15);color:#ff6b6b;font-weight:600': 'background:#fef2f2;padding:6px 14px;border-radius:8px;border:1px solid #fecaca;color:#dc2626;font-weight:600',
    'background:rgba(167,139,250,0.1);padding:6px 14px;border-radius:8px;border:1px solid rgba(167,139,250,0.15);color:#a78bfa;font-weight:600': 'background:#f5f3ff;padding:6px 14px;border-radius:8px;border:1px solid #d6d0f5;color:#7c3aed;font-weight:600',
    'background:rgba(90,158,255,0.1);padding:6px 14px;border-radius:8px;border:1px solid rgba(90,158,255,0.15);color:#5a9eff;font-weight:600': 'background:#eff6ff;padding:6px 14px;border-radius:8px;border:1px solid #bfdbfe;color:#2563eb;font-weight:600',
    'background:rgba(255,255,255,0.05);padding:6px 14px;border-radius:8px;border:1px solid rgba(255,255,255,0.08);color:#888;font-weight:500': 'background:#f8f8f5;padding:6px 14px;border-radius:8px;border:1px solid #e0dcd4;color:#666;font-weight:500',
    'background:rgba(200,170,110,0.1);padding:6px 14px;border-radius:8px;border:1px solid rgba(200,170,110,0.15);color:#c8aa6e;font-weight:600': 'background:#fefce8;padding:6px 14px;border-radius:8px;border:1px solid #fde68a;color:#a16207;font-weight:600',
    'background:rgba(78,205,196,0.1);padding:6px 14px;border-radius:8px;border:1px solid rgba(78,205,196,0.15);color:#4ecdc4;font-weight:600': 'background:#f0fdfa;padding:6px 14px;border-radius:8px;border:1px solid #a7f3d0;color:#0f766e;font-weight:600',
    'background:rgba(245,158,11,0.1);padding:6px 14px;border-radius:8px;border:1px solid rgba(245,158,11,0.15);color:#f59e0b;font-weight:600': 'background:#fffbeb;padding:6px 14px;border-radius:8px;border:1px solid #fde68a;color:#d97706;font-weight:600',
}
for old, new in flow_styles.items():
    content = content.replace(old, new)

content = content.replace('color:#555;text-align:center', 'color:#999;text-align:center')
content = content.replace('color:#777', 'color:#555')
content = content.replace('border-top:1px solid #3a3a5a', 'border-top:1px solid #ddd8d0')

with open('E:\\Desktop\\loledgeagent\\docs\\architecture-diagram.html', 'w', encoding='utf-8') as f:
    f.write(content)

print('Done - theme updated to light')
