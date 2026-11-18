// i18n.js — the editor's interface text in Chinese and English
// (spec level-editor / 界面语言).
//
// Every string the page shows goes through t(key, args). Static text in
// index.html is marked data-i18n="key" (or data-i18n-title / -placeholder) and
// filled by applyStatic(). The choice is remembered in localStorage; the first
// time it follows the browser.

const STORE = 'mario-editor-lang';

export const STRINGS = {
  zh: {
    'app.title': '关卡编辑器',
    'loading': '正在载入游戏引擎…',
    'loading.failed': '游戏引擎载入失败：{e}。请运行 make build 重新编译后再启动编辑器。',
    'untitled': '未命名关卡',
    'lang': '语言',

    'btn.new': '新建', 'btn.new.tip': '新建关卡',
    'btn.open': '打开', 'btn.open.tip': '打开关卡',
    'btn.save': '保存', 'btn.save.tip': '保存（Ctrl+S）',
    'btn.play': '▶ 试玩', 'btn.play.tip': '试玩（Enter）',
    'btn.playHere': '▶ 从这里', 'btn.playHere.tip': '从鼠标所在的列开始试玩（Shift+Enter）',
    'btn.options': '选项', 'btn.options.tip': '试玩选项',
    'btn.undo': '撤销', 'btn.undo.tip': '撤销（Ctrl+Z）',
    'btn.redo': '重做', 'btn.redo.tip': '重做（Ctrl+Shift+Z）',
    'btn.help.tip': '快捷键（?）',
    'opt.big': '以大马里奥开始', 'opt.invincible': '无敌（不会死亡）',
    'opt.fixedSeed': '固定种子', 'opt.reseed': '换种子', 'opt.reseed.tip': '换一个种子', 'opt.showDraws': '显示随机结果',

    'sec.tools': '工具', 'sec.elements': '元素', 'sec.stamps': '图章', 'sec.fill': '铺地面',
    'sec.props': '属性', 'sec.level': '关卡', 'sec.look': '外观', 'sec.check': '检查',
    'fill.layers': '层数', 'fill.keep': '保留已有的坑', 'fill.btn': '一键铺地面',
    'fill.btn.tip': '一键铺地面（F）；有选区时只铺选区',
    'stamp.dec': '减小（[）', 'stamp.inc': '加大（]）',
    'playing.hud': '试玩中 · Esc 回到编辑',
    'st.pos': '列 {x} 行 {y}', 'st.posNone': '列 – 行 –', 'st.zoom': '缩放 {z}%', 'st.width': '宽 {w}',
    'st.traceHide': '隐藏轨迹', 'st.traceShow': '显示轨迹',
    'st.draft': '草稿已保存 {t}', 'st.draftFail': '草稿无法保存（浏览器存储不可用）',
    'st.where': '保存到：{p}',

    'tool.pen': '画笔', 'tool.rect': '矩形', 'tool.erase': '橡皮', 'tool.pick': '吸管', 'tool.select': '框选', 'tool.stamp': '图章',
    'tool.hand': '选择', 'tool.hand.tip': '点一个方块、出生点或旗杆，在左边属性区改它（H）',
    'el.ground': '地面', 'el.brick': '砖块', 'el.question': '问号块', 'el.stair': '硬块', 'el.coin': '金币',
    'el.goomba': '蘑菇怪', 'el.koopa': '乌龟', 'el.hidden': '隐藏块', 'el.spawn': '出生点', 'el.flag': '旗杆',
    'el.ground.hint': '土和草皮。铺在关卡底部当路面。',
    'el.brick.hint': '可以顶碎的砖。大个子能顶出里面的东西，小个子顶不碎。',
    'el.question.hint': '顶一下就出东西，可以设定里面是什么。',
    'el.stair.hint': '敲不动的硬块，用来做台阶和墙。',
    'el.coin.hint': '散落在空中的金币，走过去就能拿。',
    'el.goomba.hint': '踩一下就能消灭。它会在平台上左右走，掉下去就没了。',
    'el.koopa.hint': '踩一下缩进壳里，再踢一下壳会滑出去，撞到别的敌人也一起消灭。',
    'el.hidden.hint': '平时看不见，从下面顶到才出现。里面放什么可以在左边属性区设。',
    'el.spawn.hint': '玩家出生的位置。一关只有这一个。',
    'el.flag.hint': '关卡终点。拖动它或者改高度来调整最后一段。',
    'stamp.pipe': '水管', 'stamp.stairs-up': '上行阶梯', 'stamp.stairs-down': '下行阶梯',
    'stamp.gap': '坑', 'stamp.brick-row': '一排砖', 'stamp.coin-arc': '金币阵',
    'unit.pipe': '高', 'unit.stairs-up': '级', 'unit.stairs-down': '级', 'unit.gap': '宽', 'unit.brick-row': '块', 'unit.coin-arc': '枚',
    'spawn.badge': '起点', 'trace.stale': '旧轨迹',

    'kind.coin': '金币', 'kind.mushroom': '红蘑菇', 'kind.star': '星星', 'kind.oneup': '1UP',
    'kind.poison': '毒蘑菇', 'kind.gold': '金币袋', 'kind.random': '随机',

    'props.empty': '什么都没有选中。切到“选择”（H）再点一个方块、出生点或旗杆来改属性；框选多个方块可以一次设置。',
    'props.penBlock': '接下来画的问号块里放什么。已经放下的块用“选择”（H）点它来单独改。',
    'props.stamp': '图章：{name}', 'props.stampHint': '在画布上点一下放下。也可以用 [ / ] 调大小。',
    'props.size': '尺寸',
    'props.noBlocks': '选区里没有问号块或隐藏块。可以 Ctrl+C 复制、Delete 删除，或拖动选区移动。',
    'props.spawn': '出生点', 'props.spawnHint': '拖动可以移动。关卡里只有一个出生点。',
    'props.flag': '旗杆', 'props.flagHeight': '高度', 'props.flagUnit': '格', 'props.flagHint': '拖动可以移动。关卡里只有一根旗杆。',
    'props.at': '第 {x} 列 第 {y} 行', 'props.col': '第 {x} 列',
    'props.nBlocks': '{n} 个方块', 'props.hiddenBlock': '隐藏块', 'props.brick': '砖块', 'props.question': '问号块',
    'props.contents': '内容物', 'props.fixed': '指定', 'props.random': '随机',
    'props.count': '次数', 'props.countHint': '顶几次才变成空块',
    'props.hidden': '隐藏块', 'props.hiddenHint': '平时看不见，顶到才出现',
    'props.weight': '权重', 'props.keepOne': '至少要留一个候选',
    'props.zeroWeights': '权重加起来是 0，游戏会拒绝加载这个关卡。',
    'props.drawNote': '每条命开始时抽一次，这条命里不变；死了重生会重新抽。',

    'lvl.id': '标识', 'lvl.idPh': 'c-作者-名称', 'lvl.author': '作者', 'lvl.desc': '简介', 'lvl.diff': '难度',
    'lvl.diffNone': '未设', 'lvl.time': '时限', 'lvl.width': '宽度', 'lvl.right': '在右侧', 'lvl.left': '在左侧', 'lvl.apply': '应用',
    'lvl.where': '位置', 'lvl.whereBuiltin': '内置', 'lvl.whereLocal': '本地', 'lvl.whereNew': '新关卡',

    'look.reset': '恢复主题默认', 'look.sky': '天空', 'look.solid': '纯色', 'look.grad': '渐变', 'look.custom': '自定义',
    'look.skyTop.tip': '天空颜色（渐变时为顶部）', 'look.skyBottom.tip': '渐变底部',
    'look.turf': '草皮', 'look.ground': '土层', 'look.brick': '砖块', 'look.stair': '硬块', 'look.pipe': '水管',
    'look.customOf': '自定义{x}颜色',

    'check.none': '没有发现问题', 'check.count': '{e} 个错误 · {w} 个提示',
    'check.whole': '整关', 'check.at': '第 {x} 列', 'check.atRow': '第 {x} 列 第 {y} 行',

    'dlg.open': '打开关卡', 'dlg.cancel': '取消', 'dlg.ok': '确定', 'dlg.close': '关闭（Esc）',
    'dlg.new': '新建关卡', 'dlg.newHint': '新关卡带一段铺好的地面、出生点和旗杆，不做任何修改就能试玩通关。',
    'dlg.newBtn': '新建', 'dlg.newWidth': '宽度', 'dlg.newWhere': '保存到：{p}',
    'dlg.help': '快捷键', 'dlg.confirm': '确认',
    'open.official': '官方', 'open.community': '社区', 'open.diff': '难度 {d}',
    'open.builtin': '内置', 'open.local': '本地', 'open.broken': '无法加载：{e}',
    'open.none': '还没有关卡', 'open.dir': '目录：{p}',

    'q.restore': '上次有没保存的改动（{id}，{at}）。要恢复吗？',
    'q.discardOpen': '当前有没保存的改动，打开别的关卡会丢掉它们。继续吗？',
    'q.discardNew': '当前有没保存的改动，新建会丢掉它们。继续吗？',
    'q.overwrite': '{where}已经有一个叫 {id} 的关卡，确认要覆盖吗？',
    'q.resize': '要删掉的列里有 {n} 列有内容（例如第 {x} 列），会一起删掉。继续吗？',
    'q.overwriteTitle': '覆盖关卡', 'q.resizeTitle': '修改宽度',
    'q.askId': '给关卡起一个标识（文件名）。社区关卡用 c-作者-名称：',

    'toast.undo': '撤销：{x}', 'toast.redo': '重做：{x}',
    'toast.notFound': '找不到关卡 {id}', 'toast.pick': '吸取：{x}',
    'toast.copied': '已复制', 'toast.cut': '已剪切', 'toast.selectFirst': '先用框选工具（V）选一块区域',
    'toast.filled': '已铺满整关', 'toast.filledRange': '已铺第 {a}–{b} 列',
    'toast.saveErrors': '有 {n} 个错误，修好之后才能保存：{m}',
    'toast.saved': '已保存到 {p}', 'toast.savedReadOnly': '内置关卡是程序的一部分，已另存到本地：{p}',
    'toast.playErrors': '有错误，不能试玩：{m}', 'toast.noPlay': '不能试玩：{m}',
    'toast.cleared': '通关了！', 'toast.died': '在第 {x} 列附近死亡',
    'toast.playHere': '把鼠标移到画布上要开始的那一列，再按 Shift+Enter',
    'toast.newSeed': '新种子：{s}', 'toast.noFoothold': '这一列没有落脚点，从出生点开始',

    'err.bad-id-chars': '关卡标识只能用小写字母、数字和连字符（例如 c-tom-sky-run），最长 64 个字符',
    'err.bad-id-form': '社区关卡请用 c-<作者>-<名称> 的形式（例如 c-tom-sky-run）；官方关卡用 <世界>-<关>（例如 1-2）',
    'err.read-only': '这里的关卡是程序的一部分，不能修改',
    'err.invalid-level': '游戏无法加载这个关卡：{detail}',
    'err.write-failed': '保存失败：{detail}', 'err.bad-request': '请求格式不对：{detail}',
    'err.bad-source': '未知的关卡位置', 'err.unknown': '保存失败',

    'issue.size': '关卡的宽和高必须大于 0（现在是 {w}×{h}）',
    'issue.rows': '关卡声明高 {h} 行，实际有 {n} 行',
    'issue.spawn': '出生点（{x}, {y}）在关卡外面',
    'issue.flag-x': '旗杆第 {x} 列在关卡外面',
    'issue.flag': '旗杆的行范围 {top}–{bottom} 不对',
    'issue.meta': '关卡信息有误：{detail}', 'issue.style': '外观设置有误：{detail}',
    'issue.row-width': '第 {row} 行有 {n} 列，应该是 {want} 列',
    'issue.symbol': '不认识的地形符号 “{symbol}”',
    'issue.block-outside': '方块（{x}, {y}）在关卡外面',
    'issue.block-kind': '不认识的方块内容 “{kind}”',
    'issue.block-pool': '随机方块设置有误：{detail}',
    'issue.hidden-solid': '隐藏块必须放在空气里，这里是 “{symbol}”',
    'issue.block-tile': '这里是 “{symbol}”，不是问号块或砖块，不能放内容物',
    'issue.entity-outside': '敌人（{x}, {y}）在关卡外面',
    'issue.entity-kind': '不认识的敌人 “{kind}”',
    'issue.block-empty': '这个问号块没有设置内容物',
    'issue.spawn-no-ground': '出生点下方没有地面，玩家一开始就会掉下去',
    'issue.spawn-in-solid': '出生点在实心格里，玩家会卡住',
    'issue.spawn-enemy-near': '出生点前 {n} 格内有敌人，玩家还没准备好就要应付它',
    'issue.block-sealed': '这个方块正下方被挡住，没法从下面顶到',
    'issue.enemy-in-solid': '敌人生成在实心格里，会卡住或被挤出',
    'issue.enemy-floating': '敌人脚下是空的，开局会直接掉下去',
    'issue.contrast': '{nameZh}和天空颜色太接近（色差 {d}，建议至少 {min}），玩家可能看不清',
    'issue.gap-too-wide': '这个坑可能跳不过去：宽 {w} 格，按当前手感奔跑起跳最远约 {reach} 格；从出生点出发可能到不了旗杆',
    'issue.wall-too-high': '这里的障碍可能跳不上去：高 {h} 格，满跳最高约 {rise} 格；从出生点出发可能到不了旗杆',
    'issue.flag-unreachable': '从出生点出发可能到不了旗杆（最远走到第 {x} 列）',

    'help.pen': '画笔', 'help.rect': '矩形填充', 'help.erase': '橡皮（Shift 拖动擦一片）', 'help.pick': '吸管',
    'help.hand': '选择（点一下改属性）',
    'help.select': '框选', 'help.stamp': '图章', 'help.elements': '选择元素', 'help.stampSize': '图章尺寸',    'help.fill': '一键铺地面', 'help.look': '外观面板', 'help.undo': '撤销', 'help.redo': '重做',
    'help.clip': '复制 / 剪切 / 粘贴', 'help.delete': '删除选区', 'help.esc': '取消 / 结束试玩',
    'help.save': '保存', 'help.play': '试玩', 'help.playHere': '从鼠标所在列试玩', 'help.grid': '网格开关',
    'help.zoom': '缩放', 'help.pan': '平移', 'help.rightClick': '擦除', 'help.wheel': '平移 / 缩放', 'help.this': '这个面板',
    'key.space': '空格 + 拖动', 'key.right': '右键', 'key.wheel': '滚轮 / Ctrl+滚轮',
  },

  en: {
    'app.title': 'Level Editor',
    'loading': 'Loading the game engine…',
    'loading.failed': 'The game engine failed to load: {e}. Rebuild with make build and start the editor again.',
    'untitled': 'Untitled level',
    'lang': 'Language',

    'btn.new': 'New', 'btn.new.tip': 'New level',
    'btn.open': 'Open', 'btn.open.tip': 'Open a level',
    'btn.save': 'Save', 'btn.save.tip': 'Save (Ctrl+S)',
    'btn.play': '▶ Play', 'btn.play.tip': 'Play test (Enter)',
    'btn.playHere': '▶ From here', 'btn.playHere.tip': 'Play from the column under the mouse (Shift+Enter)',
    'btn.options': 'Options', 'btn.options.tip': 'Play test options',
    'btn.undo': 'Undo', 'btn.undo.tip': 'Undo (Ctrl+Z)',
    'btn.redo': 'Redo', 'btn.redo.tip': 'Redo (Ctrl+Shift+Z)',
    'btn.help.tip': 'Shortcuts (?)',
    'opt.big': 'Start as big Mario', 'opt.invincible': 'Invincible (cannot die)',
    'opt.fixedSeed': 'Fixed seed', 'opt.reseed': 'New seed', 'opt.reseed.tip': 'Pick another seed', 'opt.showDraws': 'Show random draws',

    'sec.tools': 'Tools', 'sec.elements': 'Elements', 'sec.stamps': 'Stamps', 'sec.fill': 'Ground',
    'sec.props': 'Properties', 'sec.level': 'Level', 'sec.look': 'Look', 'sec.check': 'Checks',
    'fill.layers': 'Layers', 'fill.keep': 'Keep existing gaps', 'fill.btn': 'Fill ground',
    'fill.btn.tip': 'Fill ground (F); only the selection when there is one',
    'stamp.dec': 'Smaller ([)', 'stamp.inc': 'Bigger (])',
    'playing.hud': 'Play testing · Esc to edit',
    'st.pos': 'Col {x} Row {y}', 'st.posNone': 'Col – Row –', 'st.zoom': 'Zoom {z}%', 'st.width': 'Width {w}',
    'st.traceHide': 'Hide trace', 'st.traceShow': 'Show trace',
    'st.draft': 'Draft saved {t}', 'st.draftFail': 'Draft cannot be saved (browser storage unavailable)',
    'st.where': 'Saves to: {p}',

    'tool.pen': 'Pen', 'tool.rect': 'Rectangle', 'tool.erase': 'Eraser', 'tool.pick': 'Picker', 'tool.select': 'Select', 'tool.stamp': 'Stamp',
    'tool.hand': 'Choose', 'tool.hand.tip': 'Click a block, the start or the flagpole, then change it on the left (H)',
    'el.ground': 'Ground', 'el.brick': 'Brick', 'el.question': '? Block', 'el.stair': 'Hard', 'el.coin': 'Coin',
    'el.goomba': 'Goomba', 'el.koopa': 'Koopa', 'el.hidden': 'Hidden', 'el.spawn': 'Start', 'el.flag': 'Flagpole',
    'el.ground.hint': 'Dirt with a grass top. Use it as the floor along the bottom.',
    'el.brick.hint': 'A brick you can break. Big Mario knocks its contents out; small Mario just bumps it.',
    'el.question.hint': 'Hit it from below to get what is inside. You can choose what it holds.',
    'el.stair.hint': 'A solid block that cannot be broken. For steps and walls.',
    'el.coin.hint': 'A coin in the air; walk into it to collect it.',
    'el.goomba.hint': 'Stomp it to defeat it. It walks along the platform and falls off the edge.',
    'el.koopa.hint': 'Stomp it into its shell, then kick the shell; it slides and takes other enemies with it.',
    'el.hidden.hint': 'Invisible until hit from below. Its contents are set on the left.',
    'el.spawn.hint': 'Where the player starts. One per level.',
    'el.flag.hint': 'The end of the level. Drag it, or set its height, to tune the last stretch.',
    'stamp.pipe': 'Pipe', 'stamp.stairs-up': 'Stairs up', 'stamp.stairs-down': 'Stairs down',
    'stamp.gap': 'Gap', 'stamp.brick-row': 'Brick row', 'stamp.coin-arc': 'Coin arc',
    'unit.pipe': 'tall', 'unit.stairs-up': 'steps', 'unit.stairs-down': 'steps', 'unit.gap': 'wide', 'unit.brick-row': 'bricks', 'unit.coin-arc': 'coins',
    'spawn.badge': 'Start', 'trace.stale': 'Old trace',

    'kind.coin': 'Coin', 'kind.mushroom': 'Mushroom', 'kind.star': 'Star', 'kind.oneup': '1UP',
    'kind.poison': 'Poison', 'kind.gold': 'Coin bag', 'kind.random': 'Random',

    'props.empty': 'Nothing is selected. Switch to Choose (H) and click a block, the start or the flagpole; drag over several blocks to set them together.',
    'props.penBlock': 'What the question blocks you draw next will hold. To change one already placed, click it with Choose (H).',
    'props.stamp': 'Stamp: {name}', 'props.stampHint': 'Click the canvas to place it. [ / ] changes the size.',
    'props.size': 'Size',
    'props.noBlocks': 'No question or hidden blocks in the selection. Ctrl+C to copy, Delete to clear, or drag to move it.',
    'props.spawn': 'Start', 'props.spawnHint': 'Drag to move it. A level has one start.',
    'props.flag': 'Flagpole', 'props.flagHeight': 'Height', 'props.flagUnit': 'tiles', 'props.flagHint': 'Drag to move it. A level has one flagpole.',
    'props.at': 'col {x} row {y}', 'props.col': 'col {x}',
    'props.nBlocks': '{n} blocks', 'props.hiddenBlock': 'Hidden block', 'props.brick': 'Brick', 'props.question': '? Block',
    'props.contents': 'Contents', 'props.fixed': 'Fixed', 'props.random': 'Random',
    'props.count': 'Uses', 'props.countHint': 'hits before it is empty',
    'props.hidden': 'Hidden', 'props.hiddenHint': 'invisible until hit',
    'props.weight': 'Weight', 'props.keepOne': 'Keep at least one choice',
    'props.zeroWeights': 'The weights add up to 0; the game will refuse this level.',
    'props.drawNote': 'Drawn once at the start of each life and kept for that life; a respawn draws again.',

    'lvl.id': 'ID', 'lvl.idPh': 'c-author-name', 'lvl.author': 'Author', 'lvl.desc': 'About', 'lvl.diff': 'Difficulty',
    'lvl.diffNone': 'None', 'lvl.time': 'Time', 'lvl.width': 'Width', 'lvl.right': 'on the right', 'lvl.left': 'on the left', 'lvl.apply': 'Apply',
    'lvl.where': 'Kept in', 'lvl.whereBuiltin': 'Built-in', 'lvl.whereLocal': 'Local', 'lvl.whereNew': 'New level',

    'look.reset': 'Reset to theme', 'look.sky': 'Sky', 'look.solid': 'Solid', 'look.grad': 'Gradient', 'look.custom': 'Custom',
    'look.skyTop.tip': 'Sky colour (top of a gradient)', 'look.skyBottom.tip': 'Bottom of the gradient',
    'look.turf': 'Turf', 'look.ground': 'Soil', 'look.brick': 'Bricks', 'look.stair': 'Hard blocks', 'look.pipe': 'Pipes',
    'look.customOf': 'Custom {x} colour',

    'check.none': 'No problems found', 'check.count': '{e} errors · {w} hints',
    'check.whole': 'whole level', 'check.at': 'col {x}', 'check.atRow': 'col {x} row {y}',

    'dlg.open': 'Open a level', 'dlg.cancel': 'Cancel', 'dlg.ok': 'OK', 'dlg.close': 'Close (Esc)',
    'dlg.new': 'New level', 'dlg.newHint': 'A new level comes with ground, a start and a flagpole: it can be played to the end straight away.',
    'dlg.newBtn': 'Create', 'dlg.newWidth': 'Width', 'dlg.newWhere': 'Saves to: {p}',
    'dlg.help': 'Shortcuts', 'dlg.confirm': 'Confirm',
    'open.official': 'Official', 'open.community': 'Community', 'open.diff': 'difficulty {d}',
    'open.builtin': 'Built-in', 'open.local': 'Local', 'open.broken': 'Does not load: {e}',
    'open.none': 'No levels yet', 'open.dir': 'Folder: {p}',

    'q.restore': 'There are unsaved changes from last time ({id}, {at}). Restore them?',
    'q.discardOpen': 'There are unsaved changes; opening another level will lose them. Continue?',
    'q.discardNew': 'There are unsaved changes; a new level will lose them. Continue?',
    'q.overwrite': 'There is already a level called {id} in {where}. Overwrite it?',
    'q.resize': '{n} of the columns to be removed have something in them (column {x}, for one). They will be removed too. Continue?',
    'q.overwriteTitle': 'Overwrite level', 'q.resizeTitle': 'Change width',
    'q.askId': 'Give the level an ID (its file name). Community levels use c-author-name:',

    'toast.undo': 'Undo: {x}', 'toast.redo': 'Redo: {x}',
    'toast.notFound': 'No level called {id}', 'toast.pick': 'Picked: {x}',
    'toast.copied': 'Copied', 'toast.cut': 'Cut', 'toast.selectFirst': 'Select an area with the select tool (V) first',
    'toast.filled': 'Ground filled across the level', 'toast.filledRange': 'Ground filled in columns {a}–{b}',
    'toast.saveErrors': '{n} errors to fix before saving: {m}',
    'toast.saved': 'Saved to {p}', 'toast.savedReadOnly': 'Built-in levels are part of the program; saved a local copy: {p}',
    'toast.playErrors': 'Cannot play with errors: {m}', 'toast.noPlay': 'Cannot play: {m}',
    'toast.cleared': 'Cleared!', 'toast.died': 'Died near column {x}',
    'toast.playHere': 'Move the mouse to the column to start from, then press Shift+Enter',
    'toast.newSeed': 'New seed: {s}', 'toast.noFoothold': 'Nowhere to stand in that column; starting from the start',

    'err.bad-id-chars': 'A level ID can only use lowercase letters, digits and hyphens (like c-tom-sky-run), up to 64 characters',
    'err.bad-id-form': 'Community levels are named c-<author>-<name> (like c-tom-sky-run); official ones <world>-<stage> (like 1-2)',
    'err.read-only': 'Levels here are part of the program and cannot be changed',
    'err.invalid-level': 'The game cannot load this level: {detail}',
    'err.write-failed': 'Saving failed: {detail}', 'err.bad-request': 'Bad request: {detail}',
    'err.bad-source': 'Unknown level folder', 'err.unknown': 'Saving failed',

    'issue.size': 'The level must be wider and taller than 0 (it is {w}×{h})',
    'issue.rows': 'The level says it is {h} rows tall but has {n}',
    'issue.spawn': 'The start ({x}, {y}) is outside the level',
    'issue.flag-x': 'The flagpole at column {x} is outside the level',
    'issue.flag': 'The flagpole rows {top}–{bottom} are not valid',
    'issue.meta': 'The level info is wrong: {detail}', 'issue.style': 'The look settings are wrong: {detail}',
    'issue.row-width': 'Row {row} has {n} columns; it should have {want}',
    'issue.symbol': 'Unknown terrain symbol “{symbol}”',
    'issue.block-outside': 'A block ({x}, {y}) is outside the level',
    'issue.block-kind': 'Unknown block contents “{kind}”',
    'issue.block-pool': 'A random block is set up wrongly: {detail}',
    'issue.hidden-solid': 'A hidden block has to be in open air; this cell is “{symbol}”',
    'issue.block-tile': 'This cell is “{symbol}”, not a question block or brick; it cannot hold contents',
    'issue.entity-outside': 'An enemy ({x}, {y}) is outside the level',
    'issue.entity-kind': 'Unknown enemy “{kind}”',
    'issue.block-empty': 'This question block has no contents set',
    'issue.spawn-no-ground': 'There is no ground under the start; the player falls straight away',
    'issue.spawn-in-solid': 'The start is inside a solid block; the player is stuck',
    'issue.spawn-enemy-near': 'An enemy is within {n} tiles of the start, before the player is ready for it',
    'issue.block-sealed': 'Something is right under this block, so it can never be hit from below',
    'issue.enemy-in-solid': 'This enemy starts inside a solid block and will be stuck or pushed out',
    'issue.enemy-floating': 'Nothing is under this enemy; it falls as soon as the level starts',
    'issue.contrast': '{nameEn} are too close to the sky colour (difference {d}, at least {min} suggested); players may not see them',
    'issue.gap-too-wide': 'This gap may be too wide: {w} tiles, and a running jump reaches about {reach}; the flag may be out of reach',
    'issue.wall-too-high': 'This may be too high to jump: {h} tiles, and a full jump rises about {rise}; the flag may be out of reach',
    'issue.flag-unreachable': 'The flag may be out of reach from the start (the furthest reachable is column {x})',

    'help.pen': 'Pen', 'help.rect': 'Rectangle fill', 'help.erase': 'Eraser (Shift-drag to clear an area)', 'help.pick': 'Picker',
    'help.hand': 'Choose (click a thing to edit it)',
    'help.select': 'Select', 'help.stamp': 'Stamp', 'help.elements': 'Pick an element', 'help.stampSize': 'Stamp size',
    'help.fill': 'Fill ground', 'help.look': 'Look panel', 'help.undo': 'Undo', 'help.redo': 'Redo',
    'help.clip': 'Copy / cut / paste', 'help.delete': 'Clear selection', 'help.esc': 'Cancel / stop play test',
    'help.save': 'Save', 'help.play': 'Play test', 'help.playHere': 'Play from the mouse column', 'help.grid': 'Toggle grid',
    'help.zoom': 'Zoom', 'help.pan': 'Pan', 'help.rightClick': 'Erase', 'help.wheel': 'Pan / zoom', 'help.this': 'This panel',
    'key.space': 'Space + drag', 'key.right': 'Right click', 'key.wheel': 'Wheel / Ctrl+wheel',
  },
};

export const LANGS = [{ id: 'zh', label: '中文' }, { id: 'en', label: 'English' }];

let lang = initial();

function initial() {
  try {
    const saved = localStorage.getItem(STORE);
    if (saved && STRINGS[saved]) return saved;
  } catch { /* storage unavailable: fall back to the browser */ }
  const nav = (globalThis.navigator?.language || 'zh').toLowerCase();
  return nav.startsWith('zh') ? 'zh' : 'en';
}

export const current = () => lang;

export function setLang(l) {
  if (!STRINGS[l]) return;
  lang = l;
  try { localStorage.setItem(STORE, l); } catch { /* not remembered, still switched */ }
  if (globalThis.document) document.documentElement.lang = l === 'zh' ? 'zh-CN' : 'en';
}

/** The text for key in the current language, with {name} filled from args.
 * A key missing from the current language falls back to Chinese, then to the
 * key itself, so a gap shows up instead of breaking the page. */
export function t(key, args = {}) {
  let s = STRINGS[lang][key] ?? STRINGS.zh[key] ?? key;
  for (const [k, v] of Object.entries(args)) s = s.split('{' + k + '}').join(String(v));
  return s;
}

/** A check result in the current language: the template for its code, filled
 * from the args Go sent. Codes without a template show Go's own message. */
export function issueText(is) {
  const code = is.code.startsWith('contrast-') ? 'contrast' : is.code;
  const key = 'issue.' + code;
  if (!STRINGS.zh[key]) return is.message;
  return t(key, is.args || {});
}

/** Fill every element marked data-i18n, data-i18n-title or
 * data-i18n-placeholder. */
export function applyStatic(root = document) {
  root.querySelectorAll('[data-i18n]').forEach(el => { el.textContent = t(el.dataset.i18n); });
  root.querySelectorAll('[data-i18n-title]').forEach(el => { el.title = t(el.dataset.i18nTitle); });
  root.querySelectorAll('[data-i18n-placeholder]').forEach(el => { el.placeholder = t(el.dataset.i18nPlaceholder); });
}
