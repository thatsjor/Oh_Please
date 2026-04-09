package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"opls/config"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Tab struct {
	FilePath string
	TextArea textarea.Model
	Modified bool
}

type EditorModel struct {
	Tabs        []Tab
	ActiveIndex int
	Active      bool
	Width       int
	Height      int
	ZenMode     bool
	Config      *config.Config
	Status      string
	StatusErr   bool
	StatusTime  time.Time
}

func NewEditor(cfg *config.Config) EditorModel {
	return EditorModel{
		Tabs:        []Tab{},
		ActiveIndex: -1,
		Active:      false,
		Config:      cfg,
	}
}

func (m *EditorModel) OpenFile(path string) (error, tea.Cmd) {
	for i, t := range m.Tabs {
		if t.FilePath == path {
			m.ActiveIndex = i
			if m.Active {
				return nil, m.Tabs[m.ActiveIndex].TextArea.Focus()
			}
			return nil, nil
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err, nil
	}

	ta := textarea.New()
	ta.Placeholder = "Empty file..."
	ta.ShowLineNumbers = !m.ZenMode
	if m.ZenMode {
		ta.Prompt = ""
	} else {
		ta.Prompt = "┃ "
	}
	ta.CharLimit = 0
	ta.SetValue(string(content))

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.CursorLineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Bold(true)

	// Use standard word jumps for alt keys
	ta.KeyMap = textarea.DefaultKeyMap

	ta.SetWidth(m.Width)
	taHeight := m.Height - 4
	if taHeight < 1 {
		taHeight = 1
	}
	ta.SetHeight(taHeight)

	ta.Focus()
	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyCtrlHome})
	ta.Blur()

	newTab := Tab{
		FilePath: path,
		TextArea: ta,
	}

	m.Tabs = append(m.Tabs, newTab)
	m.ActiveIndex = len(m.Tabs) - 1
	var cmd tea.Cmd
	if m.Active {
		cmd = m.Tabs[m.ActiveIndex].TextArea.Focus()
	}

	return nil, cmd
}

func (m *EditorModel) SaveFile() error {
	if m.ActiveIndex < 0 || m.ActiveIndex >= len(m.Tabs) {
		return nil
	}
	tab := &m.Tabs[m.ActiveIndex]
	err := os.WriteFile(tab.FilePath, []byte(tab.TextArea.Value()), 0644)
	if err == nil {
		tab.Modified = false
		m.SetStatus(fmt.Sprintf("Saved: %s", filepath.Base(tab.FilePath)), false)
	} else {
		if os.IsPermission(err) {
			m.SetStatus("Permission Denied (Use Ctrl+R to save as root)", true)
		} else {
			m.SetStatus(fmt.Sprintf("Error: %v", err), true)
		}
	}
	return err
}

func (m *EditorModel) SetStatus(msg string, isErr bool) {
	m.Status = msg
	m.StatusErr = isErr
	m.StatusTime = time.Now()
}

func (m *EditorModel) SudoSave() tea.Cmd {
	if m.ActiveIndex < 0 || m.ActiveIndex >= len(m.Tabs) {
		return nil
	}
	tab := &m.Tabs[m.ActiveIndex]
	content := tab.TextArea.Value()

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "opls_sudo_*")
	if err != nil {
		m.SetStatus(fmt.Sprintf("Temp file error: %v", err), true)
		return nil
	}
	tmpPath := tmpFile.Name()
	tmpFile.Write([]byte(content))
	tmpFile.Close()

	// Prepare the command
	c := exec.Command("sudo", "cp", tmpPath, tab.FilePath)
	
	return tea.ExecProcess(c, func(err error) tea.Msg {
		os.Remove(tmpPath)
		if err != nil {
			return SudoSaveResultMsg{Error: err}
		}
		return SudoSaveResultMsg{Path: tab.FilePath}
	})
}

type SudoSaveResultMsg struct {
	Path  string
	Error error
}

func (m *EditorModel) CloseTab(index int) {
	if index < 0 || index >= len(m.Tabs) {
		return
	}
	m.Tabs = append(m.Tabs[:index], m.Tabs[index+1:]...)
	if m.ActiveIndex >= len(m.Tabs) {
		m.ActiveIndex = len(m.Tabs) - 1
	}
	if m.Active && m.ActiveIndex >= 0 {
		m.Tabs[m.ActiveIndex].TextArea.Focus()
	}
}

func (m *EditorModel) SetZenMode(zen bool) {
	m.ZenMode = zen
	for i := range m.Tabs {
		m.Tabs[i].TextArea.ShowLineNumbers = !zen
		if zen {
			m.Tabs[i].TextArea.Prompt = ""
		} else {
			m.Tabs[i].TextArea.Prompt = "┃ "
		}
	}
}

func (m *EditorModel) SetActive(active bool) {
	m.Active = active
	if m.ActiveIndex >= 0 && m.ActiveIndex < len(m.Tabs) {
		if active {
			m.Tabs[m.ActiveIndex].TextArea.Focus()
		} else {
			m.Tabs[m.ActiveIndex].TextArea.Blur()
		}
	}
}

func (m *EditorModel) getTabBounds() []int {
	bounds := []int{}
	currentX := 0

	activeTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("4")).Padding(0, 1)
	inactiveTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("8")).Padding(0, 1)
	modifiedDotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Bold(true)

	for i, t := range m.Tabs {
		bg := inactiveTabStyle.GetBackground()
		fg := inactiveTabStyle.GetForeground()
		if i == m.ActiveIndex {
			bg = activeTabStyle.GetBackground()
			fg = activeTabStyle.GetForeground()
		}

		style := lipgloss.NewStyle().Background(bg).Foreground(fg)
		
		var indicator string
		if t.Modified {
			indicator = modifiedDotStyle.Copy().Background(bg).Render("•")
		} else {
			indicator = style.Render(" ")
		}

		name := style.Render(fmt.Sprintf("%s [x] ", filepath.Base(t.FilePath)))
		tabContent := lipgloss.JoinHorizontal(lipgloss.Top, " ", indicator, name)
		tabWidth := lipgloss.Width(tabContent)
		currentX += tabWidth
		bounds = append(bounds, currentX)
	}
	return bounds
}

func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		for i := range m.Tabs {
			m.Tabs[i].TextArea.SetWidth(m.Width)
			taHeight := m.Height
			if !m.ZenMode {
				taHeight = m.Height - 4
			}
			if taHeight < 1 {
				taHeight = 1
			}
			m.Tabs[i].TextArea.SetHeight(taHeight)
		}
		return m, nil

	case tea.MouseMsg:
		if !m.Active || m.ZenMode {
			return m, nil
		}
		if msg.Y == 1 && msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			bounds := m.getTabBounds()
			for i, rightBound := range bounds {
				leftBound := 0
				if i > 0 {
					leftBound = bounds[i-1]
				}
				if msg.X >= leftBound && msg.X < rightBound {
					if msg.X >= rightBound-6 && msg.X <= rightBound-3 {
						m.CloseTab(i)
					} else {
						if m.ActiveIndex >= 0 && m.ActiveIndex < len(m.Tabs) {
							m.Tabs[m.ActiveIndex].TextArea.Blur()
						}
						m.ActiveIndex = i
						if m.ActiveIndex >= 0 {
							m.Tabs[m.ActiveIndex].TextArea.Focus()
						}
					}
					return m, nil
				}
			}
			return m, nil
		}

		if m.ActiveIndex >= 0 && m.ActiveIndex < len(m.Tabs) {
			if msg.Action == tea.MouseActionPress {
				if msg.Button == tea.MouseButtonWheelUp {
					var cmd tea.Cmd
					m.Tabs[m.ActiveIndex].TextArea, cmd = m.Tabs[m.ActiveIndex].TextArea.Update(tea.KeyMsg{Type: tea.KeyUp})
					return m, cmd
				} else if msg.Button == tea.MouseButtonWheelDown {
					var cmd tea.Cmd
					m.Tabs[m.ActiveIndex].TextArea, cmd = m.Tabs[m.ActiveIndex].TextArea.Update(tea.KeyMsg{Type: tea.KeyDown})
					return m, cmd
				}
			}

			taMsg := msg
			taMsg.Y -= 3
			if taMsg.Y < 0 {
				return m, nil
			}

			var cmd tea.Cmd
			m.Tabs[m.ActiveIndex].TextArea, cmd = m.Tabs[m.ActiveIndex].TextArea.Update(taMsg)
			return m, cmd
		}

	case tea.KeyMsg:
		if !m.Active {
			return m, nil
		}
		key := msg.String()
		if m.Config.IsAction("close_tab", key) {
			m.CloseTab(m.ActiveIndex)
			return m, nil
		} else if m.Config.IsAction("next_tab", key) {
			if len(m.Tabs) > 1 {
				m.Tabs[m.ActiveIndex].TextArea.Blur()
				m.ActiveIndex = (m.ActiveIndex + 1) % len(m.Tabs)
				m.Tabs[m.ActiveIndex].TextArea.Focus()
			}
			return m, nil
		} else if m.Config.IsAction("prev_tab", key) {
			if len(m.Tabs) > 1 {
				m.Tabs[m.ActiveIndex].TextArea.Blur()
				m.ActiveIndex = (m.ActiveIndex - 1 + len(m.Tabs)) % len(m.Tabs)
				m.Tabs[m.ActiveIndex].TextArea.Focus()
			}
			return m, nil
		}
	}

	if m.ActiveIndex >= 0 && m.ActiveIndex < len(m.Tabs) {
		tab := &m.Tabs[m.ActiveIndex]
		oldValue := tab.TextArea.Value()
		
		if km, ok := msg.(tea.KeyMsg); ok {
			ks := km.String()
			
			// Map Shift+Arrows to fast movement
			if strings.Contains(ks, "shift+") {
				baseKey := strings.Replace(ks, "shift+", "", 1)
				switch baseKey {
				case "right":
					msg = tea.KeyMsg{Type: tea.KeyRight, Alt: true} // Word Right
				case "left":
					msg = tea.KeyMsg{Type: tea.KeyLeft, Alt: true} // Word Left
				case "up":
					for i := 0; i < 8; i++ {
						tab.TextArea, _ = tab.TextArea.Update(tea.KeyMsg{Type: tea.KeyUp})
					}
					return m, nil
				case "down":
					for i := 0; i < 8; i++ {
						tab.TextArea, _ = tab.TextArea.Update(tea.KeyMsg{Type: tea.KeyDown})
					}
					return m, nil
				}
			}
		}

		var cmd tea.Cmd
		tab.TextArea, cmd = tab.TextArea.Update(msg)
		if tab.TextArea.Value() != oldValue {
			tab.Modified = true
		}
		return m, cmd
	}

	return m, nil
}

func (m EditorModel) View() string {
	if len(m.Tabs) == 0 {
		return "\n\n   No files open. Select a file from the sidebar."
	}

	activeTab := m.Tabs[m.ActiveIndex]
	originalView := activeTab.TextArea.View()

	if m.ZenMode {
		return originalView
	}

	tabBarStyle := lipgloss.NewStyle().MarginBottom(1)
	activeTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("4")).Padding(0, 1)
	inactiveTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("8")).Padding(0, 1)
	modifiedDotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Bold(true)

	var tabsStr []string
	for i, t := range m.Tabs {
		bg := inactiveTabStyle.GetBackground()
		fg := inactiveTabStyle.GetForeground()
		if i == m.ActiveIndex {
			bg = activeTabStyle.GetBackground()
			fg = activeTabStyle.GetForeground()
		}

		style := lipgloss.NewStyle().Background(bg).Foreground(fg)

		var indicator string
		if t.Modified {
			indicator = modifiedDotStyle.Copy().Background(bg).Render("•")
		} else {
			indicator = style.Render(" ")
		}

		fileName := filepath.Base(t.FilePath)
		name := style.Render(fmt.Sprintf("%s [x] ", fileName))
		tabContent := lipgloss.JoinHorizontal(lipgloss.Top, style.Render(" "), indicator, name)
		tabsStr = append(tabsStr, tabContent)
	}

	tabBar := tabBarStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, tabsStr...))

	// Note: We avoid JoinVertical for status to keep the layout stable.
	// Status/Selections can be displayed in a non-disruptive way if needed.
	
	return lipgloss.JoinVertical(lipgloss.Left, tabBar, originalView)
}
