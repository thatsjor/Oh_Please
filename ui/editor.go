package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"opls/config"
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
	
	ta.SetWidth(m.Width)
	taHeight := m.Height - 4
	if taHeight < 1 { taHeight = 1 }
	ta.SetHeight(taHeight)

	// CRITICAL: Temporarily focus so the Update loop accepts our message.
	// Send KeyCtrlHome to force the internal row and col to 0.
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
	}
	return err
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
	
	// We need to mirror the View() logic here to get accurate widths
	activeTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("2")).Padding(0, 1)
	inactiveTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("8")).Padding(0, 1)
	modifiedDotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)

	for i, t := range m.Tabs {
		bg := inactiveTabStyle.GetBackground()
		fg := inactiveTabStyle.GetForeground()
		if i == m.ActiveIndex {
			bg = activeTabStyle.GetBackground()
			fg = activeTabStyle.GetForeground()
		}
		
		base := lipgloss.NewStyle().Background(bg).Foreground(fg).Padding(0, 1)
		indicator := " "
		if t.Modified {
			indicator = modifiedDotStyle.Copy().Background(bg).Render("•")
		}
		
		name := fmt.Sprintf(" %s%s [x] ", indicator, filepath.Base(t.FilePath))
		tabWidth := lipgloss.Width(base.Render(name))
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
				taHeight = m.Height - 4 // Account for tabs bar only if not in zen mode
			}
			if taHeight < 1 { taHeight = 1 }
			m.Tabs[i].TextArea.SetHeight(taHeight)
		}
		return m, nil

	case tea.MouseMsg:
		if !m.Active {
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
					// The [x] is at the end of the tab. 
					// Tab looks like: "[ ] [•] [filename] [[x]] [ ]"
					// It's roughly the last 5-6 characters of the tab width.
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
			if taMsg.Y < 0 { return m, nil }

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
		oldValue := m.Tabs[m.ActiveIndex].TextArea.Value()
		var cmd tea.Cmd
		m.Tabs[m.ActiveIndex].TextArea, cmd = m.Tabs[m.ActiveIndex].TextArea.Update(msg)
		if m.Tabs[m.ActiveIndex].TextArea.Value() != oldValue {
			m.Tabs[m.ActiveIndex].Modified = true
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
	activeTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("2")).Padding(0, 1) // Green active background
	inactiveTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("8")).Padding(0, 1) // Muted Grey inactive background
	modifiedDotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // Red dot

	var tabsStr []string
	for i, t := range m.Tabs {
		bg := inactiveTabStyle.GetBackground()
		fg := inactiveTabStyle.GetForeground()
		if i == m.ActiveIndex {
			bg = activeTabStyle.GetBackground()
			fg = activeTabStyle.GetForeground()
		}

		base := lipgloss.NewStyle().Background(bg).Foreground(fg).Padding(0, 1)
		
		var indicator string
		if t.Modified {
			indicator = modifiedDotStyle.Copy().Background(bg).Render("•")
		} else {
			indicator = " "
		}

		fileName := filepath.Base(t.FilePath)
		// Render the whole tab as one string to ensure consistent padding and width
		tabContent := fmt.Sprintf(" %s%s [x] ", indicator, fileName)
		tabsStr = append(tabsStr, base.Render(tabContent))
	}

	tabBar := tabBarStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, tabsStr...))
	
	// Ensure the line numbers are slightly colored if we want, but textarea defaults to TTY styles appropriately for them.
	// We drop the heavy syntax highlighting and just render the actual text so it uses standard TTY colors.
	
	return lipgloss.JoinVertical(lipgloss.Left, tabBar, originalView)
}
