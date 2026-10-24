# 设计 004：占位美术规范（票据 13）

> 交付物：`design/tokens.css`、`design/components/placeholder.css`、`design/components/*.svg`、`design/styleguide/index.html` 与本文档。
> 依据：票据 13；docs/design/003 §6（三职业区分规范）；spec 附注（HP 100 等数值语义）。
> 守界声明：本票只产出 `design/` 与本文档，不创建任何工程目录（client/server/judge/proto/db/deploy 归 01 号票）。

## 1. 设计基线

- **深色主题为基线**：编码对抗场景长时间盯屏；同时为正式美术的发光/描边风格升级预留空间。
- **三职业双编码**：剪影轮廓差异（重甲巨剑 / 长袍法杖 / 轻甲短弓）+ 主色差异（赭红 / 紫 / 绿，锚点见 003 §6），二者叠加必须一眼可辨。
- **1 格占位尺寸**：网格格位与职业剪影基准格 = `--cw-space-24`（96px）；剪影绘制坐标系为 64×64 viewBox，缩放适配。
- **血条语义**：满值 = 100HP、25% 分隔刻度（4 段）来自 spec 数值初稿；**低血量态阈值（HP ≤ 30%）是占位组件的 UI 默认语义，spec 未定义该数值**——正式语义如需调整，由后续票据修订此处并同步风格页演示。低血量态用 danger 色覆盖职业色（CSS 中 `--low` 修饰置于职业色修饰之后，同特异性靠源序取胜）。
- **界面引用范围**：MVP 阶段风格页是 token 的唯一引用方（本票守界不建 client）；client 界面接入随 01/06 号票兑现票面"被界面引用"一项。
- **题卡四语义槽位**：题型徽标 / 难度星级 / 题面区 / 编辑器区（对照 01 号票的题面板结构）。

## 2. Token 命名约定

单一来源 = `design/tokens.css`。模式：`--cw-{族}-{槽位}[-{变体}]`，六族：

| 族 | 模式 | 例 |
|---|---|---|
| 颜色 | `--cw-color-{bg\|text\|border\|state\|class\|seat\|accent}-{槽位}` | `--cw-color-bg-inset`、`--cw-color-class-warrior`、`--cw-color-state-danger`、`--cw-color-seat-p1` |
| 字体 | `--cw-font-{sans\|mono}`（族）、`--cw-font-size-{xs…xxl}`、`--cw-font-weight-{regular\|medium\|semibold}` | `--cw-font-size-sm` |
| 间距 | `--cw-space-{N}`（4px 基准，N=步数；`--cw-space-24` 兼作 1 格占位尺寸） | `--cw-space-6` |
| 圆角 | `--cw-radius-{sm\|md\|lg\|full}` | `--cw-radius-md` |
| 动效 | `--cw-motion-{fast\|slow\|pulse}`（150ms / 400ms / 1.6s，14 号票增补） | `--cw-motion-fast` |
| 层级 | `--cw-z-{槽位}`（22 号票增补） | `--cw-z-modal` |

> 22 号票登记：颜色族新增 `seat` 子族（`seat-p1`/`seat-p2`，对局席位识别色——血条填充，
> 与职业主色正交）与 `bg-target`（移动目标格选中底色）、`bg-scrim` 槽位（全屏覆盖层遮罩，
> 以 color-mix 自 `bg-inset` 派生，见使用规则 2）；新增层级族 `z-overlay`/`z-modal`
> （全屏覆盖层 / 模态结算层的 z 阶）。

**使用规则**（风格页 grep 自检项）：

1. 组件样式与占位 SVG **禁止出现 tokens 之外的裸色值与裸字号**；字号一律 `var(--cw-font-size-*)`。
2. 半透明层次统一用 `color-mix(in srgb, var(--cw-*) N%, transparent)` 表达，禁止 `rgba()` 写死。
3. 新增 token 的流程：先在本文档登记语义与槽位 → `tokens.css` 加带注释条目 → 风格页补对照展示 → 组件方可引用。禁止组件私加。
4. 正式美术替换时 **token 名不变、只换值**（见 §6）。

## 3. 资产命名约定

### 3.1 SVG 文件

模式：`cw-{类别}-{名称}[-{状态}].svg`，全小写连字符。类别 ∈ `class`（职业剪影）/ `icon`（图标）/ `tile`（格位纹理）。

| 文件 | 内容 | viewBox |
|---|---|---|
| `cw-class-warrior.svg` | 剑士剪影：重甲宽肩 + 巨剑 | 0 0 64 64 |
| `cw-class-mage.svg` | 法师剪影：长袍 + 法杖顶球 | 0 0 64 64 |
| `cw-class-ranger.svg` | 游侠剪影：轻甲窄身 + 短弓搭箭 | 0 0 64 64 |
| `cw-tile-floor.svg` | 格位底纹（边框 + 斜纹 + 角标） | 0 0 64 64 |
| `cw-icon-attack/move/defend/skill.svg` | 行动图标（题卡与技能槽） | 0 0 24 24 |
| `cw-icon-star.svg` | 难度星级（实心数量 = L1-L5） | 0 0 24 24 |

**着色规则**：全部单色 `currentColor` + `opacity` 分层，不携带任何具体颜色——颜色由宿主 CSS 注入。引用方式（14 号票修订：mask 优先）：

- **CSS mask（推荐）**：`mask-image: url(cw-*.svg)` + `background-color: currentColor`（或具体 token 色）——单一来源、零复制、宿主自由染色；风格页与客户端组件的默认方式。
- **内联 SVG**：仅当资产需要**多种颜色**时使用（当前无此类资产）。
- **禁止 `<img>` 引用**：`<img>` 上下文中 `currentColor` 无法被页面染色（会退化为黑）。
- **⚠️ XML 注释禁用连续连字符**：SVG 是 XML 文档，注释中出现 `--`（如引用 BEM 类名 `.cw-avatar--warrior`）会导致整个文件解析失败、作为图像/蒙版时静默变空白。注释里提及类名时写"宿主的 warrior 修饰类"这类文字。入库前用 `python -c "import xml.etree.ElementTree as ET; ET.parse(文件)"` 验证。

### 3.2 CSS 类名

BEM：`.cw-{块}__{元素}--{修饰符}`。块 = 组件（`cw-grid` / `cw-hp` / `cw-card` / `cw-skill-slot` / `cw-panel` / `cw-btn` / `cw-badge` / `cw-avatar` / `cw-stars`）；修饰符表达语义态（`--hero-mage`、`--coverage`、`--cooldown`、`--fill--low`）。

## 4. 资产清单与替换映射清单

| 占位资产 | 正式资产槽位名 | 建议交付格式 |
|---|---|---|
| `design/tokens.css` | `art.slots.theme` | 同名 token 的正式主题（CSS 或 Figma Tokens JSON），只换值 |
| `cw-class-warrior.svg` | `art.slots.character.warrior` | 分层 SVG（idle/attack/move/defend 姿态集待 06 号票定）或 ≥96px 透明 PNG 序列帧 |
| `cw-class-mage.svg` | `art.slots.character.mage` | 同上 |
| `cw-class-ranger.svg` | `art.slots.character.ranger` | 同上 |
| `cw-tile-floor.svg` | `art.slots.tile.floor` | 可无缝平铺纹理（SVG/PNG），保留 alpha 通道以兼容蒙版用法 |
| `cw-icon-attack.svg` | `art.slots.icon.action.attack` | 单色 SVG，保持 `currentColor` 继承语义 |
| `cw-icon-move.svg` | `art.slots.icon.action.move` | 同上 |
| `cw-icon-defend.svg` | `art.slots.icon.action.defend` | 同上 |
| `cw-icon-skill.svg` | `art.slots.icon.action.skill` | 同上 |
| `cw-icon-star.svg` | `art.slots.icon.difficulty` | 同上 |
| `design/components/placeholder.css` | `art.slots.ui-components` | 组件样式视觉升级（类名与 token 引用不变，随主题换肤） |

> 血条 / 题卡 / 结算面板等纯 CSS 组件没有独立图形资产，其"替换"即 `art.slots.theme` 换值 + 组件样式视觉升级。

## 5. 霓虹质感规范（14 号票增补）

风格语言：**霓虹终端**——深底、单色辉光、垂直渐变、细扫描线；一切质感由 token 推导，零新颜色、零位图。

**辉光分级**（`box-shadow` / `text-shadow` 的颜色强度档位，一律 `color-mix(in srgb, var(--cw-*) N%, transparent)`）：

| 档位 | 强度 | 用途 |
|---|---|---|
| subtle | 8~12% | 容器氛围（网格外晕、面板底光） |
| normal | 18~30% | 语义色条、已装备技能槽、主按钮 |
| accent | 40~55% | 悬停增强、胜利/失败标题、英雄格内晕 |

**渐变约定**：单色系垂直渐变（顶亮底暗）。SVG 内 = `stop-color="currentColor"` + `stop-opacity 1 → 0.55`；CSS 面板 = `linear-gradient(180deg, var(--cw-color-bg-raised), var(--cw-color-bg-surface))`。禁止跨色相渐变。

**纹理密度**：扫描线周期 4px、强度 ≤3%；格位底纹角标 opacity 0.5、斜纹 0.15；中心电路节点 0.5。

**动效四件套**：低血量脉动（`cw-pulse`，`--cw-motion-pulse`）、冷却 conic 扫描（`cw-sweep`）、悬停位移+辉光反馈（`--cw-motion-fast`）、面板入场（`cw-enter`，`--cw-motion-slow`）。全部时长走 `--cw-motion-*`，且必须被页尾的 `prefers-reduced-motion: reduce` 块静止兜底。

**SVG 内部结构约定**（辉光实现）：形体集合定义一次（`<g id>`），经 `<use>` 三次引用——辉光晕（`feGaussianBlur` + opacity 0.55）、渐变主体、1px 轮廓描边；职业细节线（装甲缝/法杖环/弓臂副线）置于形体集之外、opacity ≤ 0.4。

## 6. 整包替换指南

正式美术接入时按以下顺序执行（预期由后续票据执行，本文档为操作依据）：

1. **落位**：新资产按 §4 槽位名放入正式资产目录（建议 `design/art/` 或独立资源仓，届时建目录的票据负责）。
2. **换肤**：主题类只替换 `tokens.css` 中的值（或新增正式主题文件覆盖 `:root`），token 名与五族结构不变——所有组件自动换肤，无需改组件代码。
3. **换图**：图形类同名替换 SVG 文件，或把宿主引用指向新文件；客户端组件只认槽位名与 `currentColor` 契约。
4. **复核**：跑 §7 的 grep 自检（新资产同样不得引入裸色值）+ 打开风格页复查三职业可辨与组件完整性。
5. **禁区**：正式资产不得把色值写进组件 CSS；不得为单个组件绕过 token 引入一次性颜色；暗色基线保留（未来亮色主题 = 新增一份 token 值文件，而非改组件）。

## 7. 风格页与自检

**查看**（零构建零依赖，端口遵循 AGENTS.md 的 30NN 约定）：

```bash
cd design && python -m http.server 3013
# 打开 http://localhost:3013/styleguide/
```

**自检命令**：

```bash
# 1. 裸色值检查（应无输出；tokens.css 是唯一允许持有色值的文件）
grep -RnE '#[0-9a-fA-F]{3,8}\b|rgba?\(|hsla?\(' design/components/

# 2. 裸字号检查（应全部为 var(--cw-font-size-*)）
grep -n 'font-size' design/components/placeholder.css | grep -v 'var(--cw-'

# 3. SVG 着色契约检查（所有 svg 应只含 currentColor，无 hex）
grep -L 'currentColor' design/components/*.svg
```
