# powpow

**powpow** is a lightning-fast TUI file manager with split-pane preview for navigating, viewing, and managing files in your terminal. See file contents instantly, perform file operations seamlessly, and launch text files directly in your configured editor.

> Point, preview, POW POW! File management at the speed of thought.

---

## ✨ Features

- **Split-pane interface** with file list (40%) and preview pane (60%)
- **Instant text file preview** with syntax highlighting-ready display
- **Advanced fuzzy search** with real-time filtering and typo tolerance
- **Complete file operations** - create, rename, delete files and folders
- **Smart file detection** for all file types with detailed info display
- **Seamless editor integration** - opens text files in your `$EDITOR` and exits cleanly
- **Vim-style navigation** (hjkl) plus arrow key support
- **Focus switching** between file list and preview pane
- **Tree-style directory display** with clean visual hierarchy
- **Cross-platform** Go implementation with tcell - works everywhere

---

## 📦 Installation

### Using Go
```bash
go install github.com/yourusername/powpow@latest
```

### Manual Build
```bash
git clone https://github.com/yourusername/powpow
cd powpow
go build -o powpow
```

---

## 🚀 Usage

```bash
powpow                 # Launch in current directory
```

Navigate any directory structure with instant preview of text files, complete file information for binaries, and full file management capabilities.

---

## ⌨️ Keyboard Controls

### Navigation
| Key         | Action                              |
|-------------|-------------------------------------|
| `↑ ↓` `j k` | Navigate file list                  |
| `← →` `h l` | Go up directory / Enter directory   |
| `Enter`     | Enter directory or focus preview    |
| `Tab`       | Switch focus (file list ↔ preview) |
| `Backspace` | Go to parent directory              |
| `Home/End`  | Jump to first/last item             |
| `PgUp/PgDn` | Jump by page                        |

### File Operations
| Key      | Action                    |
|----------|---------------------------|
| `Ctrl+N` | Create new file           |
| `Ctrl+F` | Create new folder         |
| `Ctrl+O` | Open file in editor       |
| `Ctrl+R` | Rename file/folder        |
| `Ctrl+D` | Delete file/folder        |

### Search & Navigation
| Key         | Action                          |
|-------------|--------------------------------|
| `/`         | Start fuzzy search             |
| `ESC`       | Exit search/return to file list|
| `q`         | Quit application               |
| `Ctrl+C`    | Force quit                     |

### Preview Pane (when focused)
| Key      | Action                    |
|----------|---------------------------|
| `↑ ↓`    | Scroll preview content    |
| `PgUp/Dn`| Scroll by page            |
| `Home/End`| Jump to start/end        |
| `ESC`    | Return focus to file list |

### Search Mode
| Key         | Action                          |
|-------------|--------------------------------|
| `Type`      | Filter files with fuzzy matching |
| `↑ ↓`       | Navigate filtered results       |
| `Enter`     | Select file                     |
| `ESC`       | Exit search mode               |
| `Backspace` | Delete search characters       |

---

## 🔍 Preview Features

### Text File Preview
- **Instant content display** for supported text files
- **10MB size limit** with truncation warnings for performance
- **Encoding detection** with fallback for binary files
- **Scrollable content** with independent pane navigation

### Supported Text Extensions
- **Code**: `.py`, `.js`, `.ts`, `.rs`, `.go`, `.c`, `.cpp`, `.java`, `.php`, `.rb`
- **Web**: `.html`, `.css`, `.scss`, `.jsx`, `.tsx`, `.vue`, `.svelte`
- **Config**: `.yaml`, `.yml`, `.json`, `.toml`, `.ini`, `.conf`, `.cfg`
- **Docs**: `.md`, `.txt`, `.log`, `.csv`, `.xml`, `.sql`
- **Scripts**: `.sh`, `.pl`
- **And many more...**

### Non-Text File Info
For images, binaries, and other non-text files, powpow displays:
- File size (human-readable)
- Last modified date
- File permissions
- File type indication

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

### Smart File Operations
- **Auto-sanitization** of filenames (spaces → hyphens, invalid chars removed)
- **Conflict resolution** with automatic renaming (file-1.txt, file-2.txt, etc.)
- **Safe deletion** with confirmation prompts
- **Error handling** with clear status messages

---

## 🏗️ Dependencies

powpow is built with:
- **Go** (1.19+)
- **tcell/v2** - Modern terminal interface library
- **fuzzy** - Advanced fuzzy string matching

All dependencies are automatically managed by Go modules.

---

## 🚀 Performance

- **Memory efficient** - Large files are handled gracefully
- **Fast directory scanning** with lazy loading
- **Responsive UI** - Smooth scrolling and instant search
- **Cross-platform** - Works on Linux, macOS, Windows

---

## 📄 License

under ☕️, check out [the-coffee-license](https://github.com/codinganovel/The-Coffee-License)

I've included both licenses with the repo, do what you know is right. The licensing works by assuming you're operating under good faith.

---

## ✍️ Created by Sam  
Because file management shouldn't slow down your workflow.

*powpow* - The evolved form of *pow*, now with superpowers! 🚀