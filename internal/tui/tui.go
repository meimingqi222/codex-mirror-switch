package tui

import (
	"fmt"
	"os"
	"strings"

	"codex-mirror/internal"

	tea "github.com/charmbracelet/bubbletea"
)

// screen 定义当前屏幕状态.
type screen int

const (
	screenMainMenu     screen = iota // 主菜单屏幕
	screenListMirrors                // 列出镜像源屏幕
	screenSwitchMirror               // 切换镜像源屏幕
	screenAddMirror                  // 添加镜像源屏幕
	screenRemoveMirror               // 删除镜像源屏幕
	screenViewStatus                 // 查看状态屏幕

	// 按键常量.
	keyCtrlC = "ctrl+c"
	keyDown  = "down"
	keyEnter = "enter"
	keyEsc   = "esc"

	// 工具类型常量.
	toolTypeCodex  = "codex"
	toolTypeClaude = "claude"

	// UI 常量.
	errMirrorManagerInit = "错误: 无法初始化镜像管理器\n"
	uiBorderTop          = "╔══════════════════════════════════════╗\n"
	uiBorderBottom       = "╚══════════════════════════════════════╝\n\n"
	uiCursor             = "▶ "
)

// model 是我们的 TUI 应用状态.
type model struct {
	screen       screen                  // 当前屏幕
	choices      []string                // 主菜单选项
	cursor       int                     // 光标位置
	selected     map[int]struct{}        // 选择的项目
	mm           *internal.MirrorManager // 镜像管理器
	mirrors      []internal.MirrorConfig // 镜像源列表
	error        string                  // 错误消息
	message      string                  // 成功消息
	scrollOffset int                     // 滚动偏移量
	// 用于添加/编辑镜像源的字段
	inputStep     int    // 输入步骤
	inputName     string // 输入的名称
	inputURL      string // 输入的URL
	inputAPIKey   string // 输入的API Key
	inputToolType string // 输入的工具类型
	// 用于显示状态的字段
	quitting bool // 是否正在退出
}

// initialModel 返回我们应用的初始状态.
func initialModel() model {
	mm, err := internal.NewMirrorManager()
	var mirrors []internal.MirrorConfig
	if err == nil {
		mirrors = mm.ListActiveMirrors()
	}

	return model{
		screen:   screenMainMenu,
		choices:  []string{"列出镜像源", "切换镜像源", "添加镜像源", "删除镜像源", "查看状态", "退出"},
		selected: make(map[int]struct{}),
		mm:       mm,
		mirrors:  mirrors,
	}
}

// Init 是 Bubble Tea 初始化命令.
func (m model) Init() tea.Cmd {
	return nil
}

// Update 处理消息和更新状态.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// 清除之前的消息和错误
	m.error = ""
	m.message = ""

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch m.screen {
		case screenMainMenu:
			return m.updateMainMenu(msg)
		case screenListMirrors:
			return m.updateListMirrors(msg)
		case screenSwitchMirror:
			return m.updateSwitchMirror(msg)
		case screenAddMirror:
			return m.updateAddMirror(msg)
		case screenRemoveMirror:
			return m.updateRemoveMirror(msg)
		case screenViewStatus:
			return m.updateViewStatus(msg)
		}
	}

	return m, nil
}

// updateMainMenu 处理主菜单更新.
func (m model) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q":
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case keyDown, "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case keyEnter:
		return m.handleMainMenuChoice()
	}

	return m, nil
}

// handleMainMenuChoice 处理主菜单选择.
func (m model) handleMainMenuChoice() (tea.Model, tea.Cmd) {
	switch m.choices[m.cursor] {
	case "退出":
		m.quitting = true
		return m, tea.Quit
	case "列出镜像源":
		if m.mm != nil {
			m.mirrors = m.mm.ListActiveMirrors()
		}
		m.screen = screenListMirrors
		m.cursor = 0
	case "切换镜像源":
		if m.mm != nil {
			m.mirrors = m.mm.ListActiveMirrors()
		}
		m.screen = screenSwitchMirror
		m.cursor = 0
	case "添加镜像源":
		m.screen = screenAddMirror
		m.inputStep = 0
		m.inputName = ""
		m.inputURL = ""
		m.inputAPIKey = ""
		m.inputToolType = toolTypeCodex
		m.cursor = 0
	case "删除镜像源":
		if m.mm != nil {
			m.mirrors = m.mm.ListActiveMirrors()
		}
		m.screen = screenRemoveMirror
		m.cursor = 0
	case "查看状态":
		m.screen = screenViewStatus
		m.cursor = 0
	}

	return m, nil
}

// updateListMirrors 处理列出镜像源屏幕更新.
func (m model) updateListMirrors(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q", keyEsc, "b":
		m.screen = screenMainMenu
		m.cursor = 0
		m.scrollOffset = 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			// 调整滚动偏移
			if m.cursor < m.scrollOffset {
				m.scrollOffset = m.cursor
			}
		}
	case keyDown, "j":
		if m.cursor < len(m.mirrors)-1 {
			m.cursor++
			// 调整滚动偏移
			visibleItems := 5 // 假设一次显示5个镜像源
			if m.cursor >= m.scrollOffset+visibleItems {
				m.scrollOffset = m.cursor - visibleItems + 1
			}
		}
	}

	return m, nil
}

// updateSwitchMirror 处理切换镜像源屏幕更新.
func (m model) updateSwitchMirror(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q", keyEsc, "b":
		m.screen = screenMainMenu
		m.cursor = 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case keyDown, "j":
		if m.cursor < len(m.mirrors)-1 {
			m.cursor++
		}
	case keyEnter:
		if m.mm != nil && len(m.mirrors) > 0 {
			mirror := m.mirrors[m.cursor]
			if mirror.Name != internal.DefaultMirrorName || m.canDeleteOfficial() {
				err := m.mm.SwitchMirror(mirror.Name)
				if err != nil {
					m.error = fmt.Sprintf("切换失败: %v", err)
				} else {
					m.message = fmt.Sprintf("已成功切换到: %s", mirror.Name)
				}
			}
		}
	}

	return m, nil
}

// updateAddMirror 处理添加镜像源屏幕更新.
func (m model) updateAddMirror(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q", keyEsc, "b":
		m.screen = screenMainMenu
		m.cursor = 0
	case keyEnter:
		return m.handleAddMirrorInput()
	case "backspace":
		return m.handleAddMirrorBackspace()
	case "up", "k":
		if m.inputStep == 3 { // 工具类型选择
			if m.inputToolType == toolTypeClaude {
				m.inputToolType = toolTypeCodex
			}
		}
	case keyDown, "j":
		if m.inputStep == 3 { // 工具类型选择
			if m.inputToolType == toolTypeCodex {
				m.inputToolType = toolTypeClaude
			}
		}
	default:
		// 处理字符输入
		if len(msg.String()) == 1 {
			return m.handleAddMirrorChar(msg.String())
		}
	}

	return m, nil
}

// handleAddMirrorInput 处理添加镜像源的输入确认.
func (m model) handleAddMirrorInput() (tea.Model, tea.Cmd) {
	switch m.inputStep {
	case 0: // 输入名称
		if m.inputName != "" {
			m.inputStep++
		}
	case 1: // 输入URL
		if m.inputURL != "" {
			m.inputStep++
		}
	case 2: // 输入API Key
		m.inputStep++
	case 3: // 选择工具类型
		if m.mm != nil {
			var toolType internal.ToolType
			if m.inputToolType == toolTypeCodex {
				toolType = internal.ToolTypeCodex
			} else {
				toolType = internal.ToolTypeClaude
			}

			err := m.mm.AddMirrorWithType(m.inputName, m.inputURL, m.inputAPIKey, toolType)
			if err != nil {
				m.error = fmt.Sprintf("添加失败: %v", err)
			} else {
				m.message = fmt.Sprintf("成功添加镜像源: %s", m.inputName)
				m.mirrors = m.mm.ListActiveMirrors()
				m.screen = screenMainMenu
				m.cursor = 0
			}
		}
	}

	return m, nil
}

// handleAddMirrorBackspace 处理添加镜像源的退格键.
func (m model) handleAddMirrorBackspace() (tea.Model, tea.Cmd) {
	switch m.inputStep {
	case 0: // 输入名称
		if m.inputName != "" {
			m.inputName = m.inputName[:len(m.inputName)-1]
		}
	case 1: // 输入URL
		if m.inputURL != "" {
			m.inputURL = m.inputURL[:len(m.inputURL)-1]
		} else {
			m.inputStep--
		}
	case 2: // 输入API Key
		if m.inputAPIKey != "" {
			m.inputAPIKey = m.inputAPIKey[:len(m.inputAPIKey)-1]
		} else {
			m.inputStep--
		}
	case 3: // 选择工具类型
		m.inputStep--
	}

	return m, nil
}

// handleAddMirrorChar 处理添加镜像源的字符输入.
func (m model) handleAddMirrorChar(char string) (tea.Model, tea.Cmd) {
	switch m.inputStep {
	case 0: // 输入名称
		m.inputName += char
	case 1: // 输入URL
		m.inputURL += char
	case 2: // 输入API Key
		m.inputAPIKey += char
	}

	return m, nil
}

// updateRemoveMirror 处理删除镜像源屏幕更新.
func (m model) updateRemoveMirror(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q", keyEsc, "b":
		m.screen = screenMainMenu
		m.cursor = 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case keyDown, "j":
		if m.cursor < len(m.mirrors)-1 {
			m.cursor++
		}
	case keyEnter:
		if m.mm != nil && len(m.mirrors) > 0 {
			mirror := m.mirrors[m.cursor]
			if mirror.Name != internal.DefaultMirrorName {
				err := m.mm.RemoveMirror(mirror.Name)
				if err != nil {
					m.error = fmt.Sprintf("删除失败: %v", err)
				} else {
					m.message = fmt.Sprintf("已成功删除: %s", mirror.Name)
					m.mirrors = m.mm.ListActiveMirrors()
					if m.cursor >= len(m.mirrors) && len(m.mirrors) > 0 {
						m.cursor = len(m.mirrors) - 1
					}
				}
			} else {
				m.error = "不能删除官方镜像源"
			}
		}
	}

	return m, nil
}

// updateViewStatus 处理查看状态屏幕更新.
func (m model) updateViewStatus(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q", keyEsc, "b":
		m.screen = screenMainMenu
		m.cursor = 0
	}

	return m, nil
}

// canDeleteOfficial 判断是否可以删除官方镜像源（始终返回false）.
func (m model) canDeleteOfficial() bool {
	return false
}

// View 只是根据模型中的数据返回一个字符串.
func (m model) View() string {
	if m.quitting {
		return "再见！\n"
	}

	var s string

	switch m.screen {
	case screenMainMenu:
		s = m.viewMainMenu()
	case screenListMirrors:
		s = m.viewListMirrors()
	case screenSwitchMirror:
		s = m.viewSwitchMirror()
	case screenAddMirror:
		s = m.viewAddMirror()
	case screenRemoveMirror:
		s = m.viewRemoveMirror()
	case screenViewStatus:
		s = m.viewViewStatus()
	}

	// 显示消息和错误
	if m.error != "" {
		s += fmt.Sprintf("\n✗ 错误: %s\n", m.error)
	}
	if m.message != "" {
		s += fmt.Sprintf("\n✓ %s\n", m.message)
	}

	return s
}

// viewMainMenu 渲染主菜单.
func (m model) viewMainMenu() string {
	s := uiBorderTop
	s += "║   Codex Mirror Switch TUI            ║\n"
	s += uiBorderBottom

	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = uiCursor
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\n按 q 或选择“退出”退出. 按 Enter 选择."
	return s
}

// viewListMirrors 渲染镜像源列表.
func (m model) viewListMirrors() string {
	s := uiBorderTop
	s += "║         镜像源列表                   ║\n"
	s += uiBorderBottom

	switch {
	case m.mm == nil:
		s += errMirrorManagerInit
	case len(m.mirrors) == 0:
		s += "没有配置的镜像源\n"
	default:
		visibleItems := 5 // 假设一次显示5个镜像源
		start := m.scrollOffset
		end := start + visibleItems
		if end > len(m.mirrors) {
			end = len(m.mirrors)
		}

		for i := start; i < end; i++ {
			mirror := m.mirrors[i]
			cursor := "  "
			if m.cursor == i {
				cursor = uiCursor
			}
			current := "  "
			if m.mm.GetConfig().CurrentCodex == mirror.Name || m.mm.GetConfig().CurrentClaude == mirror.Name {
				current = "★ "
			}

			s += fmt.Sprintf("%s%s%s\n", cursor, current, mirror.Name)
			s += fmt.Sprintf("  类型: %s\n", mirror.ToolType)
			s += fmt.Sprintf("  URL: %s\n", mirror.BaseURL)
			if mirror.APIKey != "" {
				// 只显示API Key的前4位和后4位
				maskedKey := maskAPIKey(mirror.APIKey)
				s += fmt.Sprintf("  API Key: %s\n", maskedKey)
			}
			if mirror.ModelName != "" {
				s += fmt.Sprintf("  模型: %s\n", mirror.ModelName)
			}
			s += "\n"
		}

		// 显示滚动指示
		if start > 0 || end < len(m.mirrors) {
			s += fmt.Sprintf("... 显示 %d-%d 共 %d 个 ...\n", start+1, end, len(m.mirrors))
		}
	}

	s += "按 q 或 b 返回主菜单, 按 j/k 或 方向键滚动."
	return s
}

// viewSwitchMirror 渲染切换镜像源屏幕.
func (m model) viewSwitchMirror() string {
	s := uiBorderTop
	s += "║       选择要切换的镜像源             ║\n"
	s += uiBorderBottom

	switch {
	case m.mm == nil:
		s += errMirrorManagerInit
	case len(m.mirrors) == 0:
		s += "没有可切换的镜像源\n"
	default:
		for i := range m.mirrors {
			mirror := &m.mirrors[i]
			cursor := "  "
			if m.cursor == i {
				cursor = uiCursor
			}
			current := "  "
			if m.mm.GetConfig().CurrentCodex == mirror.Name || m.mm.GetConfig().CurrentClaude == mirror.Name {
				current = "★ "
			}

			s += fmt.Sprintf("%s%s%s [%s]\n", cursor, current, mirror.Name, mirror.ToolType)
		}
	}

	s += "\n按 Enter 切换, 按 q 或 b 返回主菜单."
	return s
}

// viewAddMirror 渲染添加镜像源屏幕.
func (m model) viewAddMirror() string {
	s := uiBorderTop
	s += "║         添加新的镜像源               ║\n"
	s += uiBorderBottom

	switch m.inputStep {
	case 0:
		s += fmt.Sprintf("镜像源名称: %s█\n", m.inputName)
		s += "\n输入镜像源名称，然后按 Enter 继续."
	case 1:
		s += fmt.Sprintf("镜像源名称: %s\n", m.inputName)
		s += fmt.Sprintf("API 基础 URL: %s█\n", m.inputURL)
		s += "\n输入 API 基础 URL (例如: https://api.example.com)，然后按 Enter 继续."
	case 2:
		s += fmt.Sprintf("镜像源名称: %s\n", m.inputName)
		s += fmt.Sprintf("API 基础 URL: %s\n", m.inputURL)
		s += fmt.Sprintf("API Key (可选): %s█\n", m.inputAPIKey)
		s += "\n输入 API Key (可选)，然后按 Enter 继续."
	case 3:
		s += fmt.Sprintf("镜像源名称: %s\n", m.inputName)
		s += fmt.Sprintf("API 基础 URL: %s\n", m.inputURL)
		if m.inputAPIKey != "" {
			s += fmt.Sprintf("API Key: %s\n", maskAPIKey(m.inputAPIKey))
		}
		s += "\n选择工具类型:\n"
		cursorCodex := "  "
		cursorClaude := "  "
		if m.inputToolType == toolTypeCodex {
			cursorCodex = uiCursor
		} else {
			cursorClaude = uiCursor
		}
		s += fmt.Sprintf("%sCodex (OpenAI 兼容)\n", cursorCodex)
		s += fmt.Sprintf("%sClaude\n", cursorClaude)
		s += "\n使用 ↑/↓ 选择，按 Enter 确认添加."
	}

	s += "\n按 Esc 或 b 返回主菜单."
	return s
}

// viewRemoveMirror 渲染删除镜像源屏幕.
func (m model) viewRemoveMirror() string {
	s := uiBorderTop
	s += "║         选择要删除的镜像源           ║\n"
	s += uiBorderBottom

	switch {
	case m.mm == nil:
		s += errMirrorManagerInit
	case len(m.mirrors) == 0:
		s += "没有可删除的镜像源\n"
	default:
		for i := range m.mirrors {
			mirror := &m.mirrors[i]
			cursor := "  "
			if m.cursor == i {
				cursor = uiCursor
			}
			locked := "  "
			if mirror.Name == internal.DefaultMirrorName {
				locked = "🔒 "
			}

			s += fmt.Sprintf("%s%s%s [%s]\n", cursor, locked, mirror.Name, mirror.ToolType)
		}
		s += "\n🔒 表示不能删除的官方镜像源."
	}

	s += "\n按 Enter 删除, 按 q 或 b 返回主菜单."
	return s
}

// viewViewStatus 渲染查看状态屏幕.
func (m model) viewViewStatus() string {
	s := uiBorderTop
	s += "║           当前状态                   ║\n"
	s += uiBorderBottom

	if m.mm == nil {
		s += errMirrorManagerInit
	} else {
		config := m.mm.GetConfig()

		s += fmt.Sprintf("配置文件路径: %s\n", m.mm.GetConfigPath())
		s += "\n"

		if config.CurrentCodex != "" {
			s += fmt.Sprintf("当前 Codex 镜像源: %s\n", config.CurrentCodex)
		}
		if config.CurrentClaude != "" {
			s += fmt.Sprintf("当前 Claude 镜像源: %s\n", config.CurrentClaude)
		}

		activeMirrors := m.mm.ListActiveMirrors()
		s += fmt.Sprintf("\n镜像源总数: %d\n", len(activeMirrors))

		// 按类型统计
		codexCount := 0
		claudeCount := 0
		for i := range activeMirrors {
			mirror := &activeMirrors[i]
			switch mirror.ToolType {
			case internal.ToolTypeCodex:
				codexCount++
			case internal.ToolTypeClaude:
				claudeCount++
			}
		}

		s += fmt.Sprintf("  - Codex 类型: %d\n", codexCount)
		s += fmt.Sprintf("  - Claude 类型: %d\n", claudeCount)
	}

	s += "\n按 q 或 b 返回主菜单."
	return s
}

// maskAPIKey 掩码显示 API Key.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

// Start 启动 TUI 应用.
func Start() error {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("哎呀，出错了：%v", err)
		os.Exit(1)
		return err
	}
	return nil
}
