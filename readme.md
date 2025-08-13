# powpow

**powpow** is a minimal, lightning-fast TUI file manager designed for distraction-free navigation and file management in your terminal. Built with a focus on simplicity and speed - no clutter, just pure file management efficiency.

> Point, navigate, POW POW! Minimal file management at the speed of thought.

---

## ✨ Features

- **Minimal single-pane interface** using full terminal width for file navigation
- **Built-in help system** - Press F1 to see all keyboard shortcuts
- **Advanced fuzzy search** with real-time filtering and typo tolerance
- **Complete file operations** - create, rename, delete files and folders with clean popup dialogs
- **Smart file detection** with text file recognition
- **Seamless editor integration** - opens text files in your `$EDITOR` and exits cleanly
- **Vim-style navigation** (hjkl) plus arrow key support
- **Clean, distraction-free design** focused on productivity
- **Cross-platform** Go implementation with tcell - works everywhere

---

## 📦 Installation

### Using Go
```bash
go install github.com/codinganovel/powpow@latest
```

### Manual Build
```bash
git clone https://github.com/codinganovel/powpow
cd powpow
go build -o powpow
```

---

## 🚀 Usage

```bash
powpow                 # Launch in current directory
powpow -a              # Launch with autocd (inherit final directory on exit)
```

Navigate any directory structure with minimal, distraction-free interface and complete file management capabilities. Press F1 for keyboard shortcuts.

---

## ⌨️ Keyboard Controls

> **Press F1 in the app for a complete help screen with all shortcuts!**

### Navigation
| Key         | Action                        |
|-------------|-------------------------------|
| `↑ ↓` `j k` | Navigate file list            |
| `← →` `h l` | Go up / Enter directory       |
| `Enter`     | Enter directory               |
| `Backspace` | Go to parent directory        |
| `Home/End`  | Jump to first/last item       |
| `PgUp/PgDn` | Jump by page                  |

### File Operations
| Key      | Action                    |
|----------|---------------------------|
| `Ctrl+N` | Create new file           |
| `Ctrl+F` | Create new folder         |
| `Ctrl+O` | Open file in editor       |
| `Ctrl+R` | Rename file/folder        |
| `Ctrl+D` | Delete file/folder        |

### Search & Help
| Key         | Action                          |
|-------------|--------------------------------|
| `/`         | Start fuzzy search             |
| `F1`        | Show help screen               |
| `ESC`       | Exit search/help mode          |
| `q`         | Quit application               |
| `Ctrl+C`    | Force quit                     |

### Search Mode
| Key         | Action                          |
|-------------|--------------------------------|
| `Type`      | Filter files with fuzzy matching |
| `↑ ↓`       | Navigate filtered results       |
| `Enter`     | Select file/directory          |
| `ESC`       | Exit search mode               |
| `Backspace` | Delete search characters       |

---

## 📁 File Management Features

### Smart Text File Detection
- **Automatic recognition** of text files by extension and content analysis
- **Supported formats**: code files, configs, docs, scripts, and many more
- **Safe opening** - only text files can be opened in your editor

### Supported Text Extensions
- **Code**: `.py`, `.js`, `.ts`, `.rs`, `.go`, `.c`, `.cpp`, `.java`, `.php`, `.rb`
- **Web**: `.html`, `.css`, `.scss`, `.jsx`, `.tsx`, `.vue`, `.svelte`
- **Config**: `.yaml`, `.yml`, `.json`, `.toml`, `.ini`, `.conf`, `.cfg`
- **Docs**: `.md`, `.txt`, `.log`, `.csv`, `.xml`, `.sql`
- **Scripts**: `.sh`, `.pl`
- **And many more...**

### File Operations
- **Smart filename sanitization** - spaces become hyphens, invalid chars removed
- **Conflict resolution** - automatic renaming (file-1.txt, file-2.txt, etc.)
- **Safe operations** - clean popup dialogs for destructive actions
- **Clear feedback** - status messages for all operations

---

## ⚙️ Configuration

powpow uses your system's default text editor:

```bash
export EDITOR=micro    # Set your preferred editor
export EDITOR=nano     # or nano  
export EDITOR=vim      # or vim
export EDITOR=code     # or VS Code
```

If `$EDITOR` is not set, powpow will show you how to configure it.

### AutoCD Mode
Enable directory inheritance to stay in the directory when you exit:

```bash
powpow -a                    # Command line flag
export POWPOW_AUTOCD=1       # Environment variable
```

With AutoCD enabled, your shell will change to whatever directory you were browsing when you quit powpow.

---

## 🏗️ Dependencies

powpow is built with:
- **Go** (1.19+)
- **tcell/v2** - Modern terminal interface library
- **fuzzy** - Advanced fuzzy string matching

All dependencies are automatically managed by Go modules.

---

## 🚀 Performance

- **Minimal design** - No visual bloat, maximum efficiency
- **Fast directory scanning** with optimized file operations
- **Responsive UI** - Instant search and smooth navigation
- **Low resource usage** - Designed for speed and simplicity
- **Cross-platform** - Works on Linux, macOS, Windows

---

## 📄 License

under ☕️, check out [the-coffee-license](https://github.com/codinganovel/The-Coffee-License)

I've included both licenses with the repo, do what you know is right. The licensing works by assuming you're operating under good faith.

---

## ✍️ Created by Sam  
Because file management shouldn't slow down your workflow.

*powpow* - The evolved form of *pow*, now with superpowers! 🚀