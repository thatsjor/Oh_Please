# ohplease
(opls)

A minimal, tabbed terminal text editor built with Go and Bubble Tea. It is designed to be lightweight, easy to navigate, and visually consistent with your terminal's native color palette.

## Features

- **Tabbed Interface**: Manage multiple open files with mouse-clickable tabs.
- **Sidebar File Browser**: Navigate your filesystem and open files directly from within the editor.
- **Zen Mode**: Press `ctrl+z` to hide all UI elements (tabs, sidebar, line numbers) for a clean editing experience and distraction-free copy/pasting.
- **Terminal Native Colors**: Uses your terminal's own 16-color ANSI palette for a consistent look.
- **Mouse Support**: Click to switch tabs, select files, or use the scroll wheel for navigation.
- **Customizable**: Keybindings are configurable via a simple INI-style config file.

## Configuration

Configuration is stored at `~/.config/opls/config.conf`. You can customize keybindings using an INI-style format.

### Example Configuration

```ini
[keys]
save = ctrl+s
close_tab = ctrl+w
next_tab = ctrl+f, alt+right
prev_tab = ctrl+b, alt+left
quit = ctrl+q, ctrl+c, esc
switch_focus = tab
up = up, k
down = down, j
enter = enter
toggle_hidden = .
toggle_zen = ctrl+z
new_file = ctrl+n
```

### Default Keybindings

- **Tab**: Switch focus between the sidebar and the editor.
- **Ctrl+s**: Save the current file.
- **Ctrl+n**: (In Sidebar) Create a new file.
- **Ctrl+z**: Toggle Zen Mode.
- **Ctrl+w**: Close current tab.
- **Ctrl+f / Alt+Right**: Next tab.
- **Ctrl+b / Alt+Left**: Previous tab.
- **Ctrl+q / Esc**: Quit.
- **Ctrl+r**: Save as Root (Elevated).
- **Enter**: Open selected file or directory in the sidebar.

## Installation

Ensure you have Go installed, then clone the repository and build:

step 1
```bash
go build -o opls
```
step 2
```bash
sudo mv ./opls /usr/local/bin/opls
```
step 3
```bash
sudo chmod +x /usr/local/bin/opls
```
step 4
```bash
cp opls.desktop /usr/share/applications/
sudo update-desktop-database
```
Installing to `/usr/local/bin/` ensures that `sudo opls` works correctly across your system.

Works with file paths in your launch command.

TO OPEN A FILE
```bash
opls /mnt/projects/file.txt
```

TO OPEN A DIRECTORY
```bash
opls /mnt/projects/
```