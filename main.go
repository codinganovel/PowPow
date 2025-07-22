/*
┌─────────────────────────────────────────────────────────────┐
│                     THE COFFEE LICENSE                      │
├─────────────────────────────────────────────────────────────┤
│ • Free for everyone.                                        │
│ • Coffee money for billionaires.                            │
│                                                             │
│ MIT + Basic Courtesy                                        │
│                                                             │
│ • Free for personal, educational, and small business use    │
│ • $ 50 if your net worth has commas and lawyers        │
│ • Use freely, modify freely                                 │
│   (but buy me a damn latte if you're Google)           │
├─────────────────────────────────────────────────────────────┤
│ Copyright 2025 Sam JamshidiLarijani                                     │
│ Contact: wow@sammakes.art                                            │
├─────────────────────────────────────────────────────────────┤
│ IF YOU'RE ABOVE THE THRESHOLD                              │
│ but would prefer not to pay,                               │
│ restructure your assets                                    │
│ until you're technically not.                              │
├─────────────────────────────────────────────────────────────┤
│ This isn't about money. It's about acknowledgment.         │
│                                                             │
│ ☕ Made with code and good faith                            │
│    by an indie dev who believes in balance                 │
└─────────────────────────────────────────────────────────────┘
*/
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/sahilm/fuzzy"
)

type PaneType int

const (
	FileList PaneType = iota
	Preview
)

type FileItem struct {
	Name     string
	Path     string
	IsDir    bool
	IsHidden bool
	Size     int64
	ModTime  time.Time
	Mode     os.FileMode
}

type PreviewContent struct {
	IsText    bool
	Content   string
	Truncated bool
	ErrorMsg  string
	FileInfo  FileItem
}

type Navigator struct {
	currentPath   string
	items         []FileItem
	filteredItems []FileItem
	selectedIdx   int
	searchMode    bool
	searchQuery   string
	scrollOffset  int
}

type Previewer struct {
	content   PreviewContent
	scrollPos int
	maxLines  int
}

type StatusBar struct {
	message      string
	isPrompt     bool
	promptText   string
	inputBuffer  string
	isError      bool
	messageTime  time.Time
	hasMessage   bool
	defaultMsg   string
}

type App struct {
	screen      tcell.Screen
	navigator   *Navigator
	previewer   *Previewer
	statusBar   *StatusBar
	focusedPane PaneType
	running     bool
	autocd      bool
	width       int
	height      int
}

func NewFileItem(path string) (FileItem, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileItem{}, err
	}

	name := filepath.Base(path)
	isHidden := strings.HasPrefix(name, ".")

	return FileItem{
		Name:     name,
		Path:     path,
		IsDir:    info.IsDir(),
		IsHidden: isHidden,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Mode:     info.Mode(),
	}, nil
}

func NewNavigator(startPath string) *Navigator {
	nav := &Navigator{
		currentPath: startPath,
		selectedIdx: 0,
		searchMode:  false,
		searchQuery: "",
	}
	nav.loadDirectory()
	return nav
}

func (n *Navigator) loadDirectory() error {
	entries, err := os.ReadDir(n.currentPath)
	if err != nil {
		return err
	}

	n.items = make([]FileItem, 0, len(entries))
	for _, entry := range entries {
		fullPath := filepath.Join(n.currentPath, entry.Name())
		item, err := NewFileItem(fullPath)
		if err != nil {
			continue
		}
		n.items = append(n.items, item)
	}

	sort.Slice(n.items, func(i, j int) bool {
		if n.items[i].IsDir != n.items[j].IsDir {
			return n.items[i].IsDir
		}
		return strings.ToLower(n.items[i].Name) < strings.ToLower(n.items[j].Name)
	})

	n.updateFilteredItems()
	n.clampSelection()
	return nil
}

func (n *Navigator) updateFilteredItems() {
	if n.searchQuery == "" {
		n.filteredItems = n.items
		return
	}

	names := make([]string, len(n.items))
	for i, item := range n.items {
		names[i] = item.Name
	}

	matches := fuzzy.Find(n.searchQuery, names)
	n.filteredItems = make([]FileItem, len(matches))
	for i, match := range matches {
		n.filteredItems[i] = n.items[match.Index]
	}
}

func (n *Navigator) clampSelection() {
	if len(n.filteredItems) == 0 {
		n.selectedIdx = 0
		return
	}
	if n.selectedIdx >= len(n.filteredItems) {
		n.selectedIdx = len(n.filteredItems) - 1
	}
	if n.selectedIdx < 0 {
		n.selectedIdx = 0
	}
}

func (n *Navigator) moveSelection(delta int) {
	n.selectedIdx += delta
	n.clampSelection()
}

func (n *Navigator) getSelectedItem() *FileItem {
	if len(n.filteredItems) == 0 || n.selectedIdx < 0 || n.selectedIdx >= len(n.filteredItems) {
		return nil
	}
	return &n.filteredItems[n.selectedIdx]
}

func (n *Navigator) enterDirectory() error {
	selected := n.getSelectedItem()
	if selected == nil || !selected.IsDir {
		return nil
	}

	newPath := selected.Path
	n.currentPath = newPath
	n.selectedIdx = 0
	n.scrollOffset = 0
	return n.loadDirectory()
}

func (n *Navigator) goUp() error {
	parent := filepath.Dir(n.currentPath)
	if parent == n.currentPath {
		return nil
	}

	oldName := filepath.Base(n.currentPath)
	n.currentPath = parent
	n.selectedIdx = 0
	n.scrollOffset = 0

	err := n.loadDirectory()
	if err != nil {
		return err
	}

	for i, item := range n.filteredItems {
		if item.Name == oldName {
			n.selectedIdx = i
			break
		}
	}

	return nil
}

func (n *Navigator) setSearch(query string) {
	n.searchQuery = query
	n.updateFilteredItems()
	n.selectedIdx = 0
	n.clampSelection()
}

func NewPreviewer() *Previewer {
	return &Previewer{
		scrollPos: 0,
		maxLines:  0,
	}
}

func (p *Previewer) loadFile(item FileItem) {
	if item.IsDir {
		p.content = PreviewContent{
			IsText:   false,
			ErrorMsg: "",
			FileInfo: item,
		}
		return
	}

	if item.Size > 10*1024*1024 {
		p.content = PreviewContent{
			IsText:    false,
			ErrorMsg:  "File too large (> 10MB)",
			FileInfo:  item,
			Truncated: true,
		}
		return
	}

	if !p.isTextFile(item) {
		p.content = PreviewContent{
			IsText:   false,
			ErrorMsg: "",
			FileInfo: item,
		}
		return
	}

	file, err := os.Open(item.Path)
	if err != nil {
		p.content = PreviewContent{
			IsText:   false,
			ErrorMsg: err.Error(),
			FileInfo: item,
		}
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		p.content = PreviewContent{
			IsText:   false,
			ErrorMsg: err.Error(),
			FileInfo: item,
		}
		return
	}

	truncated := false
	if len(content) > 10*1024*1024 {
		content = content[:10*1024*1024]
		truncated = true
	}

	p.content = PreviewContent{
		IsText:    true,
		Content:   string(content),
		Truncated: truncated,
		ErrorMsg:  "",
		FileInfo:  item,
	}
	p.scrollPos = 0
}

func (p *Previewer) isTextFile(item FileItem) bool {
	ext := strings.ToLower(filepath.Ext(item.Name))
	textExts := []string{
		".txt", ".md", ".py", ".js", ".json", ".yaml", ".yml", ".html", ".css",
		".sh", ".conf", ".cfg", ".ini", ".log", ".sql", ".xml", ".csv", ".toml",
		".rs", ".go", ".c", ".cpp", ".h", ".hpp", ".java", ".php", ".rb", ".pl",
		".ts", ".jsx", ".tsx", ".vue", ".svelte", ".scss", ".sass", ".less",
	}

	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}

	return p.detectTextContent(item)
}

func (p *Previewer) detectTextContent(item FileItem) bool {
	file, err := os.Open(item.Path)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	buffer = buffer[:n]
	if !utf8.Valid(buffer) {
		return false
	}

	printable := 0
	for _, b := range buffer {
		if b >= 32 && b <= 126 || b == '\t' || b == '\n' || b == '\r' {
			printable++
		}
	}

	ratio := float64(printable) / float64(len(buffer))
	return ratio > 0.8
}

func (p *Previewer) scroll(delta int) {
	lines := strings.Count(p.content.Content, "\n") + 1
	p.scrollPos += delta
	if p.scrollPos < 0 {
		p.scrollPos = 0
	}
	if p.scrollPos >= lines-p.maxLines {
		p.scrollPos = lines - p.maxLines
		if p.scrollPos < 0 {
			p.scrollPos = 0
		}
	}
}

func NewStatusBar(autocd bool) *StatusBar {
	var defaultMsg string
	if autocd {
		defaultMsg = "[Ready - AutoCD] • q:inherit directory /:search Ctrl+n:new file Ctrl+o:open"
	} else {
		defaultMsg = "[Ready] • q:quit /:search Ctrl+n:new file Ctrl+o:open"
	}
	return &StatusBar{
		message:     defaultMsg,
		isPrompt:    false,
		promptText:  "",
		inputBuffer: "",
		isError:     false,
		messageTime: time.Time{},
		hasMessage:  false,
		defaultMsg:  defaultMsg,
	}
}

func (s *StatusBar) showMessage(msg string) {
	s.message = msg
	s.isPrompt = false
	s.isError = false
	s.messageTime = time.Now()
	s.hasMessage = true
}

func (s *StatusBar) showError(msg string) {
	s.message = msg
	s.isPrompt = false
	s.isError = true
	s.messageTime = time.Now()
	s.hasMessage = true
}

func (s *StatusBar) startPrompt(prompt string) {
	s.promptText = prompt
	s.inputBuffer = ""
	s.isPrompt = true
	s.isError = false
}

func (s *StatusBar) addToInput(ch rune) {
	s.inputBuffer += string(ch)
}

func (s *StatusBar) backspace() {
	if len(s.inputBuffer) > 0 {
		s.inputBuffer = s.inputBuffer[:len(s.inputBuffer)-1]
	}
}

func (s *StatusBar) getInput() string {
	return s.inputBuffer
}

func (s *StatusBar) endPrompt() {
	s.isPrompt = false
	s.message = s.defaultMsg
	s.hasMessage = false
}

func (s *StatusBar) updateMessage() {
	if s.hasMessage && !s.isPrompt {
		elapsed := time.Since(s.messageTime)
		if elapsed >= 2*time.Second {
			s.message = s.defaultMsg
			s.hasMessage = false
			s.isError = false
		}
	}
}

func NewApp(autocd bool) (*App, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	err = screen.Init()
	if err != nil {
		return nil, err
	}

	screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset))
	screen.Clear()

	width, height := screen.Size()

	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	app := &App{
		screen:      screen,
		navigator:   NewNavigator(wd),
		previewer:   NewPreviewer(),
		statusBar:   NewStatusBar(autocd),
		focusedPane: FileList,
		running:     true,
		autocd:      autocd,
		width:       width,
		height:      height,
	}

	app.updatePreview()
	return app, nil
}

func (app *App) updatePreview() {
	selected := app.navigator.getSelectedItem()
	if selected != nil {
		app.previewer.loadFile(*selected)
		app.previewer.maxLines = app.height - 3
	}
}

func (app *App) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		if x+i >= app.width {
			break
		}
		app.screen.SetContent(x+i, y, r, nil, style)
	}
}


func (app *App) render() {
	app.screen.Clear()
	app.statusBar.updateMessage()

	fileListWidth := app.width * 40 / 100
	previewWidth := app.width - fileListWidth - 1

	app.drawBreadcrumbs()
	app.drawFileList(fileListWidth)
	app.drawPreview(fileListWidth+1, previewWidth)
	app.drawStatusBar()

	app.screen.Show()
}

func (app *App) drawBreadcrumbs() {
	style := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite)
	breadcrumb := app.navigator.currentPath
	if len(breadcrumb) > app.width-2 {
		breadcrumb = "..." + breadcrumb[len(breadcrumb)-(app.width-5):]
	}
	
	for i := 0; i < app.width; i++ {
		app.screen.SetContent(i, 0, ' ', nil, style)
	}
	app.drawText(1, 0, breadcrumb, style)
}

func (app *App) drawFileList(width int) {
	startY := 2
	maxItems := app.height - 3

	if app.navigator.selectedIdx >= app.navigator.scrollOffset+maxItems {
		app.navigator.scrollOffset = app.navigator.selectedIdx - maxItems + 1
	}
	if app.navigator.selectedIdx < app.navigator.scrollOffset {
		app.navigator.scrollOffset = app.navigator.selectedIdx
	}

	for i := 0; i < maxItems && i+app.navigator.scrollOffset < len(app.navigator.filteredItems); i++ {
		itemIdx := i + app.navigator.scrollOffset
		item := app.navigator.filteredItems[itemIdx]
		y := startY + i

		var style tcell.Style
		var prefix string

		if itemIdx == app.navigator.selectedIdx {
			if app.focusedPane == FileList {
				style = tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
			} else {
				style = tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite)
			}
			prefix = "► "
		} else {
			style = tcell.StyleDefault
			prefix = "  "
		}

		if item.IsDir {
			prefix += "├── "
		} else {
			prefix += "└── "
		}

		displayName := item.Name
		if item.IsDir {
			displayName += "/"
		}

		text := prefix + displayName
		if len(text) > width-1 {
			text = text[:width-4] + "..."
		}

		for j := 0; j < width; j++ {
			app.screen.SetContent(j, y, ' ', nil, style)
		}
		app.drawText(0, y, text, style)

		if itemIdx == app.navigator.selectedIdx && app.focusedPane == FileList {
			app.screen.SetContent(width-2, y, '◀', nil, style)
		}
	}

	style := tcell.StyleDefault
	app.screen.SetContent(width, 1, '┬', nil, style)
	for y := startY; y < app.height-1; y++ {
		app.screen.SetContent(width, y, '│', nil, style)
	}
	app.screen.SetContent(width, app.height-1, '┴', nil, style)
}

func (app *App) drawPreview(startX, width int) {
	startY := 2
	maxLines := app.height - 3

	var style tcell.Style
	if app.focusedPane == Preview {
		style = tcell.StyleDefault.Background(tcell.ColorDarkBlue)
	} else {
		style = tcell.StyleDefault
	}

	for y := startY; y < app.height-1; y++ {
		for x := startX; x < startX+width; x++ {
			app.screen.SetContent(x, y, ' ', nil, style)
		}
	}

	if app.previewer.content.ErrorMsg != "" {
		app.drawText(startX+1, startY, "Error: "+app.previewer.content.ErrorMsg, tcell.StyleDefault.Foreground(tcell.ColorRed))
		return
	}

	if !app.previewer.content.IsText {
		item := app.previewer.content.FileInfo
		info := []string{
			"File: " + item.Name,
			"Size: " + app.formatSize(item.Size),
			"Modified: " + item.ModTime.Format("2006-01-02 15:04:05"),
			"Permissions: " + item.Mode.String(),
			"",
			"No preview available",
		}

		if app.previewer.content.Truncated {
			info = append(info, "", "File too large to preview (> 10MB)")
		}

		for i, line := range info {
			if i >= maxLines {
				break
			}
			app.drawText(startX+1, startY+i, line, tcell.StyleDefault)
		}
		return
	}

	lines := strings.Split(app.previewer.content.Content, "\n")
	for i := 0; i < maxLines && i+app.previewer.scrollPos < len(lines); i++ {
		lineIdx := i + app.previewer.scrollPos
		line := lines[lineIdx]
		if len(line) > width-2 {
			line = line[:width-5] + "..."
		}
		app.drawText(startX+1, startY+i, line, tcell.StyleDefault)
	}

	if app.previewer.content.Truncated && app.previewer.scrollPos+maxLines >= len(lines) {
		msg := "... [Content truncated - file exceeds 10MB limit]"
		app.drawText(startX+1, startY+maxLines-1, msg, tcell.StyleDefault.Foreground(tcell.ColorYellow))
	}
}

func (app *App) drawStatusBar() {
	y := app.height - 1
	var style tcell.Style
	var text string

	if app.statusBar.isError {
		style = tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite)
		text = app.statusBar.message
	} else if app.statusBar.isPrompt {
		style = tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack)
		text = app.statusBar.promptText + app.statusBar.inputBuffer
	} else if app.navigator.searchMode {
		style = tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack)
		text = "Search: " + app.navigator.searchQuery
	} else {
		style = tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite)
		text = app.statusBar.message
	}

	for i := 0; i < app.width; i++ {
		app.screen.SetContent(i, y, ' ', nil, style)
	}

	if len(text) > app.width-1 {
		text = text[:app.width-4] + "..."
	}
	app.drawText(0, y, text, style)
}

func (app *App) formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func (app *App) handleKey(ev *tcell.EventKey) {
	if app.statusBar.isPrompt {
		app.handlePromptKey(ev)
		return
	}

	if app.navigator.searchMode {
		app.handleSearchKey(ev)
		return
	}

	switch ev.Key() {
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q':
			if app.autocd {
				app.exitWithDirectoryInheritance(app.navigator.currentPath)
			} else {
				app.running = false
			}
		case '/':
			app.navigator.searchMode = true
			app.navigator.searchQuery = ""
			app.navigator.setSearch("")
		case 'h':
			if app.focusedPane == FileList {
				err := app.navigator.goUp()
				if err != nil {
					app.statusBar.showError("Cannot access parent directory: " + err.Error())
				} else {
					app.updatePreview()
				}
			}
		case 'j':
			if app.focusedPane == FileList {
				app.navigator.moveSelection(1)
				app.updatePreview()
			} else {
				app.previewer.scroll(1)
			}
		case 'k':
			if app.focusedPane == FileList {
				app.navigator.moveSelection(-1)
				app.updatePreview()
			} else {
				app.previewer.scroll(-1)
			}
		case 'l':
			if app.focusedPane == FileList {
				selected := app.navigator.getSelectedItem()
				if selected != nil && selected.IsDir {
					err := app.navigator.enterDirectory()
					if err != nil {
						app.statusBar.showError("Cannot read directory: " + err.Error())
					} else {
						app.updatePreview()
					}
				} else {
					app.focusedPane = Preview
				}
			}
		}

	case tcell.KeyUp:
		if app.focusedPane == FileList {
			app.navigator.moveSelection(-1)
			app.updatePreview()
		} else {
			app.previewer.scroll(-1)
		}

	case tcell.KeyDown:
		if app.focusedPane == FileList {
			app.navigator.moveSelection(1)
			app.updatePreview()
		} else {
			app.previewer.scroll(1)
		}

	case tcell.KeyEnter:
		if app.focusedPane == FileList {
			selected := app.navigator.getSelectedItem()
			if selected != nil && selected.IsDir {
				err := app.navigator.enterDirectory()
				if err != nil {
					app.statusBar.showError("Cannot read directory: " + err.Error())
				} else {
					app.updatePreview()
				}
			} else {
				app.focusedPane = Preview
			}
		}

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if app.focusedPane == FileList {
			err := app.navigator.goUp()
			if err != nil {
				app.statusBar.showError("Cannot access parent directory: " + err.Error())
			} else {
				app.updatePreview()
			}
		}

	case tcell.KeyTab:
		if app.focusedPane == FileList {
			app.focusedPane = Preview
		} else {
			app.focusedPane = FileList
		}

	case tcell.KeyEscape:
		if app.focusedPane == Preview {
			app.focusedPane = FileList
		}

	case tcell.KeyHome:
		if app.focusedPane == FileList {
			app.navigator.selectedIdx = 0
			app.updatePreview()
		} else {
			app.previewer.scrollPos = 0
		}

	case tcell.KeyEnd:
		if app.focusedPane == FileList {
			app.navigator.selectedIdx = len(app.navigator.filteredItems) - 1
			app.navigator.clampSelection()
			app.updatePreview()
		}

	case tcell.KeyPgUp:
		if app.focusedPane == FileList {
			app.navigator.moveSelection(-10)
			app.updatePreview()
		} else {
			app.previewer.scroll(-10)
		}

	case tcell.KeyPgDn:
		if app.focusedPane == FileList {
			app.navigator.moveSelection(10)
			app.updatePreview()
		} else {
			app.previewer.scroll(10)
		}

	case tcell.KeyCtrlN:
		app.statusBar.startPrompt("New file name: ")

	case tcell.KeyCtrlF:
		app.statusBar.startPrompt("New folder name: ")

	case tcell.KeyCtrlO:
		app.openFile()

	case tcell.KeyCtrlR:
		selected := app.navigator.getSelectedItem()
		if selected != nil {
			app.statusBar.startPrompt("Rename to: ")
			app.statusBar.inputBuffer = selected.Name
		}

	case tcell.KeyCtrlD:
		selected := app.navigator.getSelectedItem()
		if selected != nil {
			prompt := fmt.Sprintf("Delete '%s'? (y/N): ", selected.Name)
			app.statusBar.startPrompt(prompt)
		}

	case tcell.KeyCtrlC:
		if app.autocd {
			app.exitWithDirectoryInheritance(app.navigator.currentPath)
		} else {
			app.running = false
		}
	}
}

func (app *App) handleSearchKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		app.navigator.searchMode = false
		app.navigator.setSearch("")
		app.statusBar.message = app.statusBar.defaultMsg
		app.statusBar.hasMessage = false

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(app.navigator.searchQuery) > 0 {
			app.navigator.searchQuery = app.navigator.searchQuery[:len(app.navigator.searchQuery)-1]
			app.navigator.setSearch(app.navigator.searchQuery)
		}

	case tcell.KeyUp:
		app.navigator.selectedIdx--
		app.navigator.clampSelection()

	case tcell.KeyDown:
		app.navigator.selectedIdx++
		app.navigator.clampSelection()

	case tcell.KeyEnter:
		selected := app.navigator.getSelectedItem()
		if selected != nil {
			if selected.IsDir {
				app.navigator.enterDirectory()
				app.navigator.searchMode = false
				app.navigator.setSearch("")
			} else {
				app.openFileWithEditor(selected.Path)
			}
		}

	case tcell.KeyRune:
		app.navigator.searchQuery += string(ev.Rune())
		app.navigator.setSearch(app.navigator.searchQuery)
	}
	app.updatePreview()
}

func (app *App) handlePromptKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		app.statusBar.endPrompt()

	case tcell.KeyEnter:
		input := app.statusBar.getInput()
		prompt := app.statusBar.promptText

		app.statusBar.endPrompt()

		switch {
		case strings.HasPrefix(prompt, "New file name:"):
			app.createFile(input)
		case strings.HasPrefix(prompt, "New folder name:"):
			app.createFolder(input)
		case strings.HasPrefix(prompt, "Rename to:"):
			app.renameItem(input)
		case strings.Contains(prompt, "Delete"):
			if input == "y" || input == "Y" {
				app.deleteItem()
			}
		}

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		app.statusBar.backspace()

	case tcell.KeyRune:
		app.statusBar.addToInput(ev.Rune())
	}
}

func (app *App) sanitizeFilename(name string) string {
	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	// Remove special characters, keep alphanumeric, hyphens, underscores, and dots
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			result.WriteRune(r)
		}
	}
	name = result.String()
	// Remove consecutive hyphens
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")
	// Ensure it's not empty
	if name == "" {
		name = "untitled"
	}
	return name
}

func (app *App) getUniqueFilePath(basePath string) string {
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return basePath
	}

	dir := filepath.Dir(basePath)
	ext := filepath.Ext(basePath)
	nameWithoutExt := strings.TrimSuffix(filepath.Base(basePath), ext)

	counter := 1
	for {
		newPath := filepath.Join(dir, fmt.Sprintf("%s-%d%s", nameWithoutExt, counter, ext))
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

func (app *App) createFile(name string) {
	if name == "" {
		app.statusBar.showError("File name cannot be empty")
		return
	}

	sanitizedName := app.sanitizeFilename(name)
	basePath := filepath.Join(app.navigator.currentPath, sanitizedName)
	filePath := app.getUniqueFilePath(basePath)

	file, err := os.Create(filePath)
	if err != nil {
		app.statusBar.showError("Cannot create file: " + err.Error())
		return
	}
	file.Close()

	app.navigator.loadDirectory()
	app.updatePreview()
	finalName := filepath.Base(filePath)
	if finalName != sanitizedName {
		app.statusBar.showMessage(fmt.Sprintf("Created file: %s (auto-renamed)", finalName))
	} else {
		app.statusBar.showMessage("Created file: " + finalName)
	}

	editor := os.Getenv("EDITOR")
	if editor != "" {
		app.openFileWithEditor(filePath)
	}
}

func (app *App) createFolder(name string) {
	if name == "" {
		app.statusBar.showError("Folder name cannot be empty")
		return
	}

	sanitizedName := app.sanitizeFilename(name)
	basePath := filepath.Join(app.navigator.currentPath, sanitizedName)
	folderPath := app.getUniqueFilePath(basePath)

	err := os.Mkdir(folderPath, 0755)
	if err != nil {
		app.statusBar.showError("Cannot create folder: " + err.Error())
		return
	}

	app.navigator.loadDirectory()
	app.updatePreview()
	finalName := filepath.Base(folderPath)
	if finalName != sanitizedName {
		app.statusBar.showMessage(fmt.Sprintf("Created folder: %s (auto-renamed)", finalName))
	} else {
		app.statusBar.showMessage("Created folder: " + finalName)
	}
}

func (app *App) openFile() {
	selected := app.navigator.getSelectedItem()
	if selected == nil || selected.IsDir {
		return
	}

	if !app.previewer.content.IsText {
		app.statusBar.showError("Cannot open non-text file")
		return
	}

	app.openFileWithEditor(selected.Path)
}

func (app *App) openFileWithEditor(filePath string) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		app.statusBar.showError("No editor configured. Set with: export EDITOR=nano")
		return
	}

	app.screen.Fini()
	
	cmd := []string{editor, filePath}
	if err := execCommand(cmd[0], cmd[1:]...); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to launch editor: %v\n", err)
		os.Exit(1)
	}
	
	// If autocd is enabled, use directory inheritance; otherwise clean exit
	if app.autocd {
		app.exitWithDirectoryInheritance(app.navigator.currentPath)
	} else {
		os.Exit(0)
	}
}

// exitWithDirectoryInheritance implements the autocd pattern from the autocd documentation.
// It creates a self-cleaning transition script that changes to the target directory
// and spawns a new shell, then replaces the current process with that script.
func (app *App) exitWithDirectoryInheritance(targetDir string) {
	// Validate target directory
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		app.statusBar.showError("Target directory does not exist: " + targetDir)
		app.running = false // fallback to normal exit
		return
	}
	
	// Get user's shell, fallback to bash
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	
	// Create transition script with self-cleanup
	scriptContent := fmt.Sprintf(`#!/bin/bash
# Auto-generated transition script for PowPow directory inheritance
# This script will self-destruct after execution

# Set up self-cleanup trap
trap 'rm -f "$0" 2>/dev/null || true' EXIT INT TERM

# Change to target directory with error handling  
if cd "%s" 2>/dev/null; then
    echo "PowPow: Inheriting directory: %s"
else
    echo "PowPow: Warning - Could not change to %s, staying in current directory" >&2
fi

# Replace this script process with user's shell
exec "%s"
`, targetDir, targetDir, targetDir, shell)

	// Create temporary script file
	tmpFile, err := os.CreateTemp("", "powpow-autocd-*.sh")
	if err != nil {
		app.statusBar.showError("Failed to create transition script: " + err.Error())
		app.running = false // fallback to normal exit
		return
	}
	scriptPath := tmpFile.Name()
	
	// Write script content
	if _, err := tmpFile.WriteString(scriptContent); err != nil {
		tmpFile.Close()
		os.Remove(scriptPath)
		app.statusBar.showError("Failed to write transition script: " + err.Error())
		app.running = false // fallback to normal exit
		return
	}
	tmpFile.Close()
	
	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		os.Remove(scriptPath)
		app.statusBar.showError("Failed to make script executable: " + err.Error())
		app.running = false // fallback to normal exit
		return
	}
	
	// Clean up tcell before process replacement
	app.screen.Fini()
	
	// Replace current process with the transition script
	if err := syscall.Exec(scriptPath, []string{scriptPath}, os.Environ()); err != nil {
		// If exec fails, restore terminal and show error
		fmt.Fprintf(os.Stderr, "PowPow: Failed to execute transition script: %v\n", err)
		os.Remove(scriptPath)
		os.Exit(1)
	}
	// This point should never be reached if exec succeeds
}

func (app *App) renameItem(newName string) {
	if newName == "" {
		app.statusBar.showError("Name cannot be empty")
		return
	}

	selected := app.navigator.getSelectedItem()
	if selected == nil {
		return
	}

	oldPath := selected.Path
	newPath := filepath.Join(filepath.Dir(oldPath), newName)

	err := os.Rename(oldPath, newPath)
	if err != nil {
		app.statusBar.showError("Cannot rename: " + err.Error())
		return
	}

	app.navigator.loadDirectory()
	app.updatePreview()
	app.statusBar.showMessage("Renamed to: " + newName)
}

func (app *App) deleteItem() {
	selected := app.navigator.getSelectedItem()
	if selected == nil {
		return
	}

	var err error
	if selected.IsDir {
		err = os.RemoveAll(selected.Path)
	} else {
		err = os.Remove(selected.Path)
	}

	if err != nil {
		app.statusBar.showError("Cannot delete: " + err.Error())
		return
	}

	app.navigator.loadDirectory()
	app.updatePreview()
	app.statusBar.showMessage("Deleted: " + selected.Name)
}

func (app *App) handleResize() {
	app.width, app.height = app.screen.Size()
	app.previewer.maxLines = app.height - 3
}

func (app *App) run() {
	for app.running {
		app.render()

		ev := app.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			app.handleKey(ev)
		case *tcell.EventResize:
			app.handleResize()
		}
	}

	app.screen.Fini()
}


func printHelp() {
	fmt.Println(`powpow - Terminal File Explorer

USAGE:
    powpow [OPTIONS]

OPTIONS:
    -a, --autocd       Enable directory inheritance (stay in final directory after quit)
    -h, --help         Show this help message

ENVIRONMENT:
    POWPOW_AUTOCD=1   Enable autocd mode via environment variable
    EDITOR            Your preferred text editor (nano, vim, code, etc.)

## ⌨️ Keyboard Controls

### Navigation
| Key         | Action                              |
|-------------|-------------------------------------|
| ↑ ↓  j k    | Navigate file list                  |
| ← →  h l    | Go up directory / Enter directory   |
| Enter       | Enter directory or focus preview    |
| Tab         | Switch focus (file list ↔ preview) |
| Backspace   | Go to parent directory              |
| Home/End    | Jump to first/last item             |
| PgUp/PgDn   | Jump by page                        |

### File Operations
| Key      | Action                    |
|----------|---------------------------|
| Ctrl+N   | Create new file           |
| Ctrl+F   | Create new folder         |
| Ctrl+O   | Open file in editor       |
| Ctrl+R   | Rename file/folder        |
| Ctrl+D   | Delete file/folder        |

### Search & Navigation
| Key         | Action                          |
|-------------|--------------------------------|
| /           | Start fuzzy search             |
| ESC         | Exit search/return to file list|
| q           | Quit application               |
| Ctrl+C      | Force quit                     |

### Preview Pane (when focused)
| Key      | Action                    |
|----------|---------------------------|
| ↑ ↓      | Scroll preview content    |
| PgUp/Dn  | Scroll by page            |
| Home/End | Jump to start/end        |
| ESC      | Return focus to file list |

### Search Mode
| Key         | Action                          |
|-------------|--------------------------------|
| Type        | Filter files with fuzzy matching |
| ↑ ↓         | Navigate filtered results       |
| Enter       | Select file                     |
| ESC         | Exit search mode               |
| Backspace   | Delete search characters       |

---

## 🔍 Preview Features

### Text File Preview
- Instant content display for supported text files
- 10MB size limit with truncation warnings for performance
- Encoding detection with fallback for binary files
- Scrollable content with independent pane navigation

### Supported Text Extensions
- Code: .py, .js, .ts, .rs, .go, .c, .cpp, .java, .php, .rb
- Web: .html, .css, .scss, .jsx, .tsx, .vue, .svelte
- Config: .yaml, .yml, .json, .toml, .ini, .conf, .cfg
- Docs: .md, .txt, .log, .csv, .xml, .sql
- Scripts: .sh, .pl
- And many more...

### Non-Text File Info
For images, binaries, and other non-text files, powpow displays:
- File size (human-readable)
- Last modified date
- File permissions
- File type indication`)
}

func main() {
	// Parse command-line arguments
	autocd := false
	
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--help", "-h":
			printHelp()
			return
		case "--autocd", "-a":
			autocd = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", arg)
				fmt.Fprintf(os.Stderr, "Use --help for usage information\n")
				os.Exit(1)
			}
		}
	}
	
	// Check environment variable as fallback
	if !autocd && os.Getenv("POWPOW_AUTOCD") == "1" {
		autocd = true
	}

	app, err := NewApp(autocd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}

	defer app.screen.Fini()
	app.run()
}