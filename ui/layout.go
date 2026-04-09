package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"opls/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MainModel struct {
	Sidebar    SidebarModel
	Editor     EditorModel
	Width      int
	Height     int
	SideActive bool
	ZenMode    bool
	Config     *config.Config
	InitCmd    tea.Cmd
}

func (m *MainModel) resizeChildren() {
	if m.ZenMode {
		m.Editor.Width = m.Width
		m.Editor.Height = m.Height
		m.Editor, _ = m.Editor.Update(tea.WindowSizeMsg{Width: m.Editor.Width, Height: m.Editor.Height})
	} else {
		sidebarWidth := 30
		m.Sidebar.Width = sidebarWidth
		m.Sidebar.Height = m.Height

		m.Editor.Width = m.Width - sidebarWidth - 2
		m.Editor.Height = m.Height

		m.Sidebar, _ = m.Sidebar.Update(tea.WindowSizeMsg{Width: m.Sidebar.Width, Height: m.Sidebar.Height})
		m.Editor, _ = m.Editor.Update(tea.WindowSizeMsg{Width: m.Editor.Width, Height: m.Editor.Height})
	}
}

func NewMainModel(startPath string, cfg *config.Config) MainModel {
	absPath, _ := filepath.Abs(startPath)
	info, err := os.Stat(absPath)

	sidebarPath := absPath
	isDir := true
	if err == nil && !info.IsDir() {
		sidebarPath = filepath.Dir(absPath)
		isDir = false
	}

	m := MainModel{
		Sidebar:    NewSidebar(sidebarPath, cfg),
		Editor:     NewEditor(cfg),
		SideActive: isDir,
		Config:     cfg,
	}

	if !isDir {
		if m.Sidebar.IsBinary(absPath) {
			isDir = true
			m.SideActive = true
		} else {
			_, cmd := m.Editor.OpenFile(absPath)
			m.InitCmd = cmd
			m.Editor.SetActive(true)
			m.Sidebar.SetActive(false)
		}
	}

	return m
}

func (m MainModel) Init() tea.Cmd {
	return m.InitCmd
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.resizeChildren()
		return m, nil

	case tea.KeyMsg:
		key := msg.String()
		if m.Config.IsAction("toggle_zen", key) {
			m.ZenMode = !m.ZenMode
			m.Editor.SetZenMode(m.ZenMode)
			m.resizeChildren()
			m.Editor.SetActive(true) // Ensure keyboard focus is maintained
			
			var mouseCmd tea.Cmd
			if m.ZenMode {
				mouseCmd = func() tea.Msg {
					fmt.Print("\x1b[?1002l")
					return nil
				}
			} else {
				mouseCmd = func() tea.Msg {
					fmt.Print("\x1b[?1002h")
					return nil
				}
			}
			return m, mouseCmd
		}
		if m.Config.IsAction("quit", key) {
			return m, tea.Quit
		}
		if m.Config.IsAction("switch_focus", key) {
			m.SideActive = !m.SideActive
			m.Sidebar.SetActive(m.SideActive)
			m.Editor.SetActive(!m.SideActive)
			return m, nil
		}

		if m.SideActive {
			var cmd tea.Cmd
			m.Sidebar, cmd = m.Sidebar.Update(msg)
			return m, cmd
		} else {
			if m.Config.IsAction("save", key) {
				m.Editor.SaveFile()
				return m, nil
			}
			if m.Config.IsAction("root_save", key) {
				return m, m.Editor.SudoSave()
			}
			var cmd tea.Cmd
			m.Editor, cmd = m.Editor.Update(msg)
			return m, cmd
		}

	case tea.MouseMsg:
		sidebarBoundary := m.Sidebar.Width + 2
		if msg.X < sidebarBoundary {
			if !m.SideActive && msg.Action == tea.MouseActionPress {
				m.SideActive = true
				m.Sidebar.SetActive(true)
				m.Editor.SetActive(false)
			}
			var cmd tea.Cmd
			m.Sidebar, cmd = m.Sidebar.Update(msg)
			return m, cmd
		} else {
			if m.SideActive && msg.Action == tea.MouseActionPress {
				m.SideActive = false
				m.Sidebar.SetActive(false)
				m.Editor.SetActive(true)
			}
			adjustedMsg := msg
			adjustedMsg.X = msg.X - sidebarBoundary - 1
			var cmd tea.Cmd
			m.Editor, cmd = m.Editor.Update(adjustedMsg)
			return m, cmd
		}

	case OpenFileMsg:
		_, cmd := m.Editor.OpenFile(msg.Path)
		m.SideActive = false
		m.Sidebar.SetActive(false)
		m.Editor.SetActive(true)
		return m, cmd
	case SudoSaveResultMsg:
		if msg.Error != nil {
			m.Editor.SetStatus(fmt.Sprintf("Sudo Save Failed: %v", msg.Error), true)
		} else {
			m.Editor.SetStatus(fmt.Sprintf("Saved as Root: %s", filepath.Base(msg.Path)), false)
			for i := range m.Editor.Tabs {
				if m.Editor.Tabs[i].FilePath == msg.Path {
					m.Editor.Tabs[i].Modified = false
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m MainModel) View() string {
	if m.ZenMode {
		return m.Editor.View()
	}

	sidebarActiveColor := lipgloss.Color("5")
	editorActiveColor := lipgloss.Color("6")
	sidebarInactiveColor := lipgloss.Color("8")
	editorInactiveColor := lipgloss.Color("8")

	sidebarBorderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(sidebarInactiveColor)
	if m.SideActive {
		sidebarBorderStyle = sidebarBorderStyle.BorderForeground(sidebarActiveColor)
	}

	editorBorderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(editorInactiveColor)
	if !m.SideActive {
		editorBorderStyle = editorBorderStyle.BorderForeground(editorActiveColor)
	}

	sidebar := sidebarBorderStyle.Width(m.Sidebar.Width).Height(m.Sidebar.Height - 2).Render(m.Sidebar.View())
	editor := editorBorderStyle.Width(m.Editor.Width).Height(m.Editor.Height - 2).Render(m.Editor.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, editor)
}
