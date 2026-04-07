package ui

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"opls/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FileItem struct {
	Name  string
	Path  string
	IsDir bool
}

type OpenFileMsg struct {
	Path string
}

type SidebarModel struct {
	Items         []FileItem
	Cursor        int
	Width         int
	Height        int
	CurrentPath   string
	Active        bool
	LastClickTime time.Time
	LastClickIdx  int
	Config        *config.Config
	ShowHidden    bool
	Naming        bool
	TextInput     textinput.Model
}

func NewSidebar(startPath string, cfg *config.Config) SidebarModel {
	m := SidebarModel{
		CurrentPath:  startPath,
		Width:        30,
		Active:       true,
		LastClickIdx: -1,
		Config:       cfg,
		ShowHidden:   false,
		Naming:       false,
	}

	ti := textinput.New()
	ti.Placeholder = "New filename..."
	ti.Prompt = " > "
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 20
	m.TextInput = ti

	m.LoadDirectory(startPath)
	return m
}

func (m *SidebarModel) LoadDirectory(path string) {
	m.Cursor = 0
	m.Items = []FileItem{}
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	if path != "/" && path != "." {
		m.Items = append(m.Items, FileItem{Name: "..", Path: filepath.Dir(path), IsDir: true})
	}

	var dirs []FileItem
	var files []FileItem

	for _, e := range entries {
		if !m.ShowHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		item := FileItem{Name: e.Name(), Path: filepath.Join(path, e.Name()), IsDir: e.IsDir()}
		if e.IsDir() {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	m.Items = append(m.Items, dirs...)
	m.Items = append(m.Items, files...)
}

func (m *SidebarModel) SetActive(active bool) {
	m.Active = active
}

func (m SidebarModel) getVisibleSlice() (int, int) {
	start := 0
	end := len(m.Items)
	maxVisible := m.Height - 5
	if end > maxVisible {
		start = m.Cursor - (maxVisible / 2)
		if start < 0 {
			start = 0
		}
		end = start + maxVisible
		if end > len(m.Items) {
			end = len(m.Items)
			start = end - maxVisible
		}
	}
	return start, end
}

func (m SidebarModel) IsBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 1024)
	n, _ := f.Read(buf)
	return bytes.Contains(buf[:n], []byte{0})
}

func (m SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	if m.Naming {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			if key == "enter" {
				name := strings.TrimSpace(m.TextInput.Value())
				if name != "" {
					path := filepath.Join(m.CurrentPath, name)
					if _, err := os.Stat(path); os.IsNotExist(err) {
						os.WriteFile(path, []byte(""), 0644)
						m.Naming = false
						m.TextInput.SetValue("")
						m.LoadDirectory(m.CurrentPath)
						return m, func() tea.Msg {
							return OpenFileMsg{Path: path}
						}
					}
				}
				m.Naming = false
				m.TextInput.SetValue("")
				return m, nil
			} else if key == "esc" {
				m.Naming = false
				m.TextInput.SetValue("")
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.TextInput, cmd = m.TextInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.MouseMsg:
		if !m.Active {
			return m, nil
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if msg.Y >= 3 {
				start, _ := m.getVisibleSlice()
				clickedIdx := start + (msg.Y - 3)
				if clickedIdx >= 0 && clickedIdx < len(m.Items) {
					now := time.Now()
					isDoubleClick := clickedIdx == m.LastClickIdx && now.Sub(m.LastClickTime) < 400*time.Millisecond

					m.Cursor = clickedIdx
					m.LastClickIdx = clickedIdx
					m.LastClickTime = now

					if isDoubleClick {
						selected := m.Items[m.Cursor]
						if selected.IsDir {
							m.CurrentPath = selected.Path
							m.LoadDirectory(m.CurrentPath)
						} else {
							if m.IsBinary(selected.Path) {
								return m, nil
							}
							return m, func() tea.Msg {
								return OpenFileMsg{Path: selected.Path}
							}
						}
					}
				}
			}
		}

	case tea.KeyMsg:
		if !m.Active {
			return m, nil
		}
		key := msg.String()
		if m.Config.IsAction("up", key) {
			if m.Cursor > 0 {
				m.Cursor--
			}
		} else if m.Config.IsAction("down", key) {
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
		} else if m.Config.IsAction("new_file", key) {
			m.Naming = true
			m.TextInput.Focus()
			return m, nil
		} else if m.Config.IsAction("enter", key) {
			if len(m.Items) == 0 {
				return m, nil
			}
			selected := m.Items[m.Cursor]
			if selected.IsDir {
				m.CurrentPath = selected.Path
				m.LoadDirectory(m.CurrentPath)
			} else {
				if m.IsBinary(selected.Path) {
					return m, nil
				}
				return m, func() tea.Msg {
					return OpenFileMsg{Path: selected.Path}
				}
			}
		} else if m.Config.IsAction("toggle_hidden", key) {
			oldCursor := m.Cursor
			m.ShowHidden = !m.ShowHidden
			m.LoadDirectory(m.CurrentPath)
			m.Cursor = oldCursor
			if m.Cursor >= len(m.Items) {
				m.Cursor = len(m.Items) - 1
			}
			if m.Cursor < 0 {
				m.Cursor = 0
			}
		}
	}
	return m, nil
}

func (m SidebarModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).MarginLeft(1) // Cyan Title
	title := titleStyle.Render(" FILES")

	start, end := m.getVisibleSlice()
	var items []string
	for i := start; i < end; i++ {
		item := m.Items[i]
		prefix := "  "
		if i == m.Cursor {
			prefix = "> "
		}

		name := item.Name
		if item.IsDir {
			name += "/"
		}

		style := lipgloss.NewStyle()
		if i == m.Cursor {
			style = style.Foreground(lipgloss.Color("2")).Bold(true)
		} else if item.IsDir {
			style = style.Foreground(lipgloss.Color("4"))
		} else {
			style = style.Foreground(lipgloss.Color("7"))
		}

		items = append(items, prefix+style.Render(name))
	}

	if m.Naming {
		return title + "\n\n  NEW FILE:\n" + "  " + m.TextInput.View()
	}

	return title + "\n\n" + strings.Join(items, "\n")
}
