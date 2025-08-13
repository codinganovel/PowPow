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
	"time"
	"unicode/utf8"

	"github.com/codinganovel/autocd-go"
	"github.com/gdamore/tcell/v2"
	"github.com/sahilm/fuzzy"
)

// Removed PaneType - single pane file explorer only

type FileItem struct {
	Name     string
	Path     string
	IsDir    bool
	IsHidden bool
	Size     int64
	ModTime  time.Time
	Mode     os.FileMode
}

// Removed PreviewContent - no preview functionality

type Navigator struct {
	currentPath   string
	items         []FileItem
	filteredItems []FileItem
	selectedIdx   int
	searchMode    bool
	searchQuery   string
	scrollOffset  int
}

// Removed Previewer - no preview functionality

type StatusBar struct {
	message      string
	isError      bool
	messageTime  time.Time
	hasMessage   bool
	defaultMsg   string
	// Removed animation fields for minimal design
}

type PopupType int

const (
	PopupNone PopupType = iota
	PopupCreateFile
	PopupCreateFolder
	PopupRename
	PopupDelete
)

type PopupState struct {
	active      bool
	popupType   PopupType
	title       string
	prompt      string
	inputBuffer string
	prefilledText string
	targetItem  *FileItem
}

type App struct {
	screen    tcell.Screen
	navigator *Navigator
	statusBar *StatusBar
	running   bool
	autocd    bool
	width     int
	height    int
	helpMode  bool
	popup     PopupState
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

// Simple text file detection for file opening
func (app *App) isTextFile(item FileItem) bool {
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

	return app.detectTextContent(item)
}

func (app *App) detectTextContent(item FileItem) bool {
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

func NewStatusBar(autocd bool) *StatusBar {
	var defaultMsg string
	if autocd {
		defaultMsg = "Ready - AutoCD | F1=Help q:inherit directory /:search Ctrl+n:new file Ctrl+o:open"
	} else {
		defaultMsg = "Ready | F1=Help q:quit /:search Ctrl+n:new file Ctrl+o:open"
	}
	return &StatusBar{
		message:     defaultMsg,
		isError:     false,
		messageTime: time.Time{},
		hasMessage:  false,
		defaultMsg:  defaultMsg,
	}
}

func (s *StatusBar) showMessage(msg string) {
	s.message = msg // Clean message without symbols
	s.isError = false
	s.messageTime = time.Now()
	s.hasMessage = true
}

func (s *StatusBar) showError(msg string) {
	s.message = "Error: " + msg // Simple error prefix
	s.isError = true
	s.messageTime = time.Now()
	s.hasMessage = true
}






func (s *StatusBar) updateMessage() {
	if s.hasMessage {
		elapsed := time.Since(s.messageTime)
		if elapsed >= 2*time.Second {
			s.message = s.defaultMsg
			s.hasMessage = false
			s.isError = false
		}
	}
	// Removed animation for minimal design
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
		screen:    screen,
		navigator: NewNavigator(wd),
		statusBar: NewStatusBar(autocd),
		running:   true,
		autocd:    autocd,
		width:     width,
		height:    height,
	}

	return app, nil
}

// Popup helper methods
func (app *App) showPopup(popupType PopupType, title, prompt string, prefilled string, targetItem *FileItem) {
	app.popup = PopupState{
		active:        true,
		popupType:     popupType,
		title:         title,
		prompt:        prompt,
		inputBuffer:   prefilled,
		prefilledText: prefilled,
		targetItem:    targetItem,
	}
}

func (app *App) hidePopup() {
	app.popup = PopupState{active: false}
}

func (app *App) addToPopupInput(ch rune) {
	app.popup.inputBuffer += string(ch)
}

func (app *App) backspacePopupInput() {
	if len(app.popup.inputBuffer) > 0 {
		app.popup.inputBuffer = app.popup.inputBuffer[:len(app.popup.inputBuffer)-1]
	}
}

func (app *App) getPopupInput() string {
	return app.popup.inputBuffer
}

// Removed updatePreview - no preview functionality

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

	if app.helpMode {
		app.drawHelp()
	} else {
		// Simple minimal rendering - full width file list
		app.drawBreadcrumbs()
		app.drawFileList()
		app.drawStatusBar()
		
		// Draw popup on top if active
		app.drawPopup()
	}

	app.screen.Show()
}

func (app *App) drawBreadcrumbs() {
	// Simple breadcrumbs without decorations
	style := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite)
	breadcrumb := app.navigator.currentPath
	if len(breadcrumb) > app.width-4 {
		breadcrumb = "..." + breadcrumb[len(breadcrumb)-(app.width-7):]
	}
	
	// Simple background fill
	for i := 0; i < app.width; i++ {
		app.screen.SetContent(i, 0, ' ', nil, style)
	}
	
	app.drawText(1, 0, breadcrumb, style)
}

func (app *App) drawFileList() {
	startY := 1
	maxItems := app.height - 2

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
			// Selected item - simple highlight
			style = tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
			prefix = "> "
		} else {
			// Unselected item - minimal styling
			if item.IsDir {
				style = tcell.StyleDefault.Foreground(tcell.ColorBlue)
			} else if item.IsHidden {
				style = tcell.StyleDefault.Foreground(tcell.ColorGray)
			} else {
				style = tcell.StyleDefault.Foreground(tcell.ColorWhite)
			}
			prefix = "  "
		}

		// Create simple display name
		displayName := item.Name
		if item.IsDir {
			displayName += "/"
		}

		text := prefix + displayName
		if len(text) > app.width-1 {
			text = text[:app.width-4] + "..."
		}

		// Fill background for selected items
		if itemIdx == app.navigator.selectedIdx {
			for j := 0; j < app.width; j++ {
				app.screen.SetContent(j, y, ' ', nil, style)
			}
		}
		
		// Draw the text
		app.drawText(0, y, text, style)
	}
}

// Removed drawPreview function - no preview functionality

func (app *App) drawHelp() {
	// Help content with clean, minimal styling
	helpText := []string{
		"PowPow - Keyboard Shortcuts",
		"",
		"Navigation:",
		"  hjkl, arrow keys    Navigate file list",
		"  Enter               Enter directory / Open file",
		"  Backspace           Go to parent directory",
		"  Home / End          Jump to first / last item",
		"  Page Up / Down      Jump by page",
		"",
		"File Operations:",
		"  Ctrl+N              Create new file",
		"  Ctrl+F              Create new folder",
		"  Ctrl+O              Open file in editor",
		"  Ctrl+R              Rename file/folder",
		"  Ctrl+D              Delete file/folder",
		"",
		"Search & General:",
		"  /                   Start fuzzy search",
		"  ESC                 Exit search mode",
		"  q                   Quit application",
		"  F1                  Show this help",
		"",
		"Press ESC to return to file explorer",
	}

	// Use a slightly different background for help mode
	helpStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	headerStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorYellow)
	sectionStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorBlue)
	footerStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorGreen)

	// Clear the screen with help background
	for y := 0; y < app.height; y++ {
		for x := 0; x < app.width; x++ {
			app.screen.SetContent(x, y, ' ', nil, helpStyle)
		}
	}

	// Center the help content
	startY := (app.height - len(helpText)) / 2
	if startY < 0 {
		startY = 0
	}

	// Draw help text
	for i, line := range helpText {
		y := startY + i
		if y >= app.height {
			break
		}

		// Center the text horizontally
		startX := (app.width - len(line)) / 2
		if startX < 0 {
			startX = 2 // Left margin if text is too wide
		}

		// Choose style based on content
		var style tcell.Style
		switch {
		case i == 0: // Title
			style = headerStyle
		case strings.HasSuffix(line, ":") && !strings.HasPrefix(line, " "): // Section headers
			style = sectionStyle
		case strings.Contains(line, "Press ESC"): // Footer instruction
			style = footerStyle
		default:
			style = helpStyle
		}

		app.drawText(startX, y, line, style)
	}
}

func (app *App) drawPopup() {
	if !app.popup.active {
		return
	}

	// Calculate popup dimensions
	var popupWidth, popupHeight int
	var lines []string

	switch app.popup.popupType {
	case PopupCreateFile:
		lines = []string{
			app.popup.title,
			"",
			app.popup.prompt + app.popup.inputBuffer + "█",
			"",
			"ESC: Cancel  Enter: OK",
		}
	case PopupCreateFolder:
		lines = []string{
			app.popup.title,
			"",
			app.popup.prompt + app.popup.inputBuffer + "█",
			"",
			"ESC: Cancel  Enter: OK",
		}
	case PopupRename:
		lines = []string{
			app.popup.title,
			"",
			app.popup.prompt + app.popup.inputBuffer + "█",
			"",
			"ESC: Cancel  Enter: OK",
		}
	case PopupDelete:
		var filename string
		if app.popup.targetItem != nil {
			filename = app.popup.targetItem.Name
		}
		lines = []string{
			"Delete Confirmation",
			"",
			"Delete '" + filename + "'?",
			"",
			"y: Yes  n: No  ESC: Cancel",
		}
	}

	// Calculate popup size with padding
	popupHeight = len(lines) + 2 // 2 for borders
	popupWidth = 0
	for _, line := range lines {
		if len(line) > popupWidth {
			popupWidth = len(line)
		}
	}
	popupWidth += 4 // 2 for borders + 2 for padding

	// Ensure minimum width
	if popupWidth < 25 {
		popupWidth = 25
	}

	// Center the popup
	startX := (app.width - popupWidth) / 2
	startY := (app.height - popupHeight) / 2

	// Ensure popup fits on screen
	if startX < 0 {
		startX = 1
	}
	if startY < 0 {
		startY = 1
	}

	// Define popup style
	borderStyle := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	contentStyle := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	titleStyle := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlue)
	
	// Draw popup background
	for y := startY; y < startY+popupHeight; y++ {
		for x := startX; x < startX+popupWidth; x++ {
			app.screen.SetContent(x, y, ' ', nil, contentStyle)
		}
	}

	// Draw border
	// Top border
	app.screen.SetContent(startX, startY, '┌', nil, borderStyle)
	for x := startX + 1; x < startX+popupWidth-1; x++ {
		app.screen.SetContent(x, startY, '─', nil, borderStyle)
	}
	app.screen.SetContent(startX+popupWidth-1, startY, '┐', nil, borderStyle)

	// Side borders and content
	for y := startY + 1; y < startY+popupHeight-1; y++ {
		app.screen.SetContent(startX, y, '│', nil, borderStyle)
		app.screen.SetContent(startX+popupWidth-1, y, '│', nil, borderStyle)
	}

	// Bottom border
	app.screen.SetContent(startX, startY+popupHeight-1, '└', nil, borderStyle)
	for x := startX + 1; x < startX+popupWidth-1; x++ {
		app.screen.SetContent(x, startY+popupHeight-1, '─', nil, borderStyle)
	}
	app.screen.SetContent(startX+popupWidth-1, startY+popupHeight-1, '┘', nil, borderStyle)

	// Draw content
	for i, line := range lines {
		y := startY + 1 + i
		x := startX + 2 // Left padding

		// Center the text within the popup
		if line != "" {
			textX := startX + (popupWidth-len(line))/2
			if textX < x {
				textX = x
			}

			var style tcell.Style
			if i == 0 { // Title line
				style = titleStyle
			} else {
				style = contentStyle
			}

			app.drawText(textX, y, line, style)
		}
	}
}

func (app *App) drawStatusBar() {
	y := app.height - 1
	var style tcell.Style
	var text string

	if app.statusBar.isError {
		style = tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite)
		text = app.statusBar.message
	} else if app.navigator.searchMode {
		style = tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack)
		text = "Search: " + app.navigator.searchQuery
	} else {
		style = tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite)
		text = app.statusBar.message
	}

	// Simple status bar fill
	for i := 0; i < app.width; i++ {
		app.screen.SetContent(i, y, ' ', nil, style)
	}

	if len(text) > app.width-1 {
		text = text[:app.width-4] + "..."
	}
	app.drawText(0, y, text, style)
}

// Removed formatSize function - not needed in minimal design

// Removed drawBackground function - minimal design

// Removed getFileIcon function - no icons in minimal design

func (app *App) handleKey(ev *tcell.EventKey) {
	if app.helpMode {
		app.handleHelpKey(ev)
		return
	}

	if app.popup.active {
		app.handlePopupKey(ev)
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
			err := app.navigator.goUp()
			if err != nil {
				app.statusBar.showError("Cannot access parent directory: " + err.Error())
			}
		case 'j':
			app.navigator.moveSelection(1)
		case 'k':
			app.navigator.moveSelection(-1)
		case 'l':
			selected := app.navigator.getSelectedItem()
			if selected != nil && selected.IsDir {
				err := app.navigator.enterDirectory()
				if err != nil {
					app.statusBar.showError("Cannot read directory: " + err.Error())
				}
			} else if selected != nil {
				// Open file with editor
				app.openFile()
			}
		}

	case tcell.KeyUp:
		app.navigator.moveSelection(-1)

	case tcell.KeyDown:
		app.navigator.moveSelection(1)

	case tcell.KeyEnter:
		selected := app.navigator.getSelectedItem()
		if selected != nil && selected.IsDir {
			err := app.navigator.enterDirectory()
			if err != nil {
				app.statusBar.showError("Cannot read directory: " + err.Error())
			}
		} else if selected != nil {
			// Open file with editor
			app.openFile()
		}

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		err := app.navigator.goUp()
		if err != nil {
			app.statusBar.showError("Cannot access parent directory: " + err.Error())
		}

	case tcell.KeyHome:
		app.navigator.selectedIdx = 0

	case tcell.KeyEnd:
		app.navigator.selectedIdx = len(app.navigator.filteredItems) - 1
		app.navigator.clampSelection()

	case tcell.KeyPgUp:
		app.navigator.moveSelection(-10)

	case tcell.KeyPgDn:
		app.navigator.moveSelection(10)

	case tcell.KeyCtrlN:
		app.showPopup(PopupCreateFile, "Create new file", "Name: ", "", nil)

	case tcell.KeyCtrlF:
		app.showPopup(PopupCreateFolder, "Create new folder", "Name: ", "", nil)

	case tcell.KeyCtrlO:
		app.openFile()

	case tcell.KeyCtrlR:
		selected := app.navigator.getSelectedItem()
		if selected != nil {
			app.showPopup(PopupRename, "Rename item", "New name: ", selected.Name, selected)
		}

	case tcell.KeyCtrlD:
		selected := app.navigator.getSelectedItem()
		if selected != nil {
			app.showPopup(PopupDelete, "Delete Confirmation", "", "", selected)
		}

	case tcell.KeyCtrlC:
		if app.autocd {
			app.exitWithDirectoryInheritance(app.navigator.currentPath)
		} else {
			app.running = false
		}

	case tcell.KeyF1:
		app.helpMode = true
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
}

func (app *App) handleHelpKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		app.helpMode = false
	case tcell.KeyF1:
		app.helpMode = false
	}
}

func (app *App) handlePopupKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		app.hidePopup()

	case tcell.KeyEnter:
		input := app.getPopupInput()
		
		switch app.popup.popupType {
		case PopupCreateFile:
			app.hidePopup()
			app.createFile(input)
		case PopupCreateFolder:
			app.hidePopup()
			app.createFolder(input)
		case PopupRename:
			app.hidePopup()
			app.renameItem(input)
		case PopupDelete:
			app.hidePopup()
			// For delete confirmation, Enter means yes
			app.deleteItem()
		}

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if app.popup.popupType != PopupDelete {
			app.backspacePopupInput()
		}

	case tcell.KeyRune:
		switch app.popup.popupType {
		case PopupDelete:
			// Handle y/n for delete confirmation
			if ev.Rune() == 'y' || ev.Rune() == 'Y' {
				app.hidePopup()
				app.deleteItem()
			} else if ev.Rune() == 'n' || ev.Rune() == 'N' {
				app.hidePopup()
			}
		default:
			// For text input popups
			app.addToPopupInput(ev.Rune())
		}
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

	// Simple text file detection for opening
	if !app.isTextFile(*selected) {
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

// exitWithDirectoryInheritance uses the autocd-go library for directory inheritance.
func (app *App) exitWithDirectoryInheritance(targetDir string) {
	// Clean up tcell before process replacement
	app.screen.Fini()
	
	// Use autocd-go library with fallback handling
	autocd.ExitWithDirectoryOrFallback(targetDir, func() {
		fmt.Printf("PowPow: AutoCD failed, but final directory was: %s\n", targetDir)
		os.Exit(0)
	})
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
	app.statusBar.showMessage("Deleted: " + selected.Name)
}

func (app *App) handleResize() {
	app.width, app.height = app.screen.Size()
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
	fmt.Println(`powpow - Minimal Terminal File Explorer

USAGE:
    powpow [OPTIONS]

OPTIONS:
    -a, --autocd       Enable directory inheritance (stay in final directory after quit)
    -h, --help         Show this help message

ENVIRONMENT:
    POWPOW_AUTOCD=1   Enable autocd mode via environment variable
    EDITOR            Your preferred text editor (nano, vim, code, etc.)

## Keyboard Controls

### Navigation
| Key         | Action                        |
|-------------|-------------------------------|
| ↑ ↓  j k    | Navigate file list            |
| ← →  h l    | Go up / Enter directory       |
| Enter       | Enter directory               |
| Backspace   | Go to parent directory        |
| Home/End    | Jump to first/last item       |
| PgUp/PgDn   | Jump by page                  |

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
| ESC         | Exit search mode               |
| q           | Quit application               |
| Ctrl+C      | Force quit                     |

### Search Mode
| Key         | Action                          |
|-------------|--------------------------------|
| Type        | Filter files with fuzzy matching |
| ↑ ↓         | Navigate filtered results       |
| Enter       | Select file/directory          |
| ESC         | Exit search mode               |
| Backspace   | Delete search characters       |

## Features

- Minimal, distraction-free interface
- Fast file navigation with vim-style keys
- Fuzzy search for quick file finding
- Smart filename sanitization with auto-incrementing
- Directory inheritance support for seamless workflow`)
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