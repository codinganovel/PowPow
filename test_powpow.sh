#!/bin/bash

# PowPow Test Script
# Tests all major functionality of the powpow file explorer

# set -e  # Exit on any error - disabled for debugging

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TEST_DIR="/tmp/powpow_test_$$"

print_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Setup test environment
setup_test_env() {
    print_test "Setting up test environment"
    
    # Create test directory
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Create various file types for testing
    echo "Hello World" > "test.txt"
    echo '{"key": "value"}' > "data.json"
    echo "#!/bin/bash" > "script.sh"
    echo "# Markdown file" > "readme.md"
    echo "console.log('test');" > "app.js"
    echo "body { color: red; }" > "style.css"
    echo "import os" > "main.py"
    echo "fn main() {}" > "main.rs"
    echo "package main" > "main.go"
    echo "<html></html>" > "index.html"
    
    # Create dotfiles (should be detected as text after our fix)
    echo "*.log" > ".gitignore"
    echo "syntax on" > ".vimrc"
    echo "alias ll='ls -la'" > ".bashrc"
    echo "NODE_ENV=test" > ".env"
    echo "root = true" > ".editorconfig"
    
    # Create files without extensions
    echo "This is a text file" > "CHANGELOG"
    echo "This is another text file" > "LICENSE"
    echo "Installation instructions" > "INSTALL"
    
    # Create binary file (should not be detected as text)
    dd if=/dev/urandom of=binary_file bs=1024 count=1 2>/dev/null
    
    # Create directories
    mkdir -p subdir1/nested
    mkdir -p subdir2
    mkdir -p "dir with spaces"
    
    # Files in subdirectories
    echo "nested file" > "subdir1/nested.txt"
    echo "another nested file" > "subdir1/nested/deep.txt"
    echo "spaced file" > "dir with spaces/spaced.txt"
    
    # Create large file (>10MB) for size limit testing
    print_test "Creating large file for size limit testing..."
    dd if=/dev/zero of=large_file.txt bs=1M count=11 2>/dev/null
    
    # Create file with unicode content
    echo "Café naïve résumé 中文 🎉" > "unicode.txt"
    
    print_pass "Test environment created at $TEST_DIR"
}

# Test file detection
test_file_detection() {
    print_test "Testing text file detection"
    
    # Build a simple test program that uses the same logic
    cat > test_detection.go << 'EOF'
package main

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "unicode/utf8"
)

type FileItem struct {
    Name string
    Path string
    Size int64
    IsDir bool
}

func isTextFile(item FileItem) bool {
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

    return detectTextContent(item)
}

func detectTextContent(item FileItem) bool {
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

func main() {
    if len(os.Args) != 2 {
        fmt.Println("Usage: test_detection <file>")
        os.Exit(1)
    }
    
    filename := os.Args[1]
    stat, err := os.Stat(filename)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    
    item := FileItem{
        Name: filepath.Base(filename),
        Path: filename,
        Size: stat.Size(),
        IsDir: stat.IsDir(),
    }
    
    if isTextFile(item) {
        fmt.Println("TEXT")
    } else {
        fmt.Println("BINARY")
    }
}
EOF

    go build -o test_detection test_detection.go
    
    # Test known text files
    test_files=(
        "test.txt:TEXT"
        "data.json:TEXT"
        "script.sh:TEXT"
        "readme.md:TEXT"
        "app.js:TEXT"
        "style.css:TEXT"
        "main.py:TEXT"
        "main.rs:TEXT"
        "main.go:TEXT"
        "index.html:TEXT"
        ".gitignore:TEXT"
        ".vimrc:TEXT"
        ".bashrc:TEXT"
        ".env:TEXT"
        ".editorconfig:TEXT"
        "CHANGELOG:TEXT"
        "LICENSE:TEXT"
        "INSTALL:TEXT"
        "unicode.txt:TEXT"
        "binary_file:BINARY"
    )
    
    for test_case in "${test_files[@]}"; do
        file="${test_case%:*}"
        expected="${test_case#*:}"
        
        if [ -f "$file" ]; then
            result=$(./test_detection "$file")
            if [ "$result" = "$expected" ]; then
                print_pass "File detection: $file -> $result"
            else
                print_fail "File detection: $file -> expected $expected, got $result"
            fi
        else
            print_warning "File not found: $file"
        fi
    done
    
    # Test large file handling
    result=$(./test_detection "large_file.txt")
    if [ "$result" = "TEXT" ]; then
        print_pass "Large file detection: large_file.txt -> TEXT"
    else
        print_fail "Large file detection: large_file.txt -> expected TEXT, got $result"
    fi
    
    rm test_detection test_detection.go
}

# Test filename sanitization
test_filename_sanitization() {
    print_test "Testing filename sanitization"
    
    cat > test_sanitize.go << 'EOF'
package main

import (
    "fmt"
    "os"
    "strings"
)

func sanitizeFilename(name string) string {
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
    return name
}

func main() {
    if len(os.Args) != 2 {
        fmt.Println("Usage: test_sanitize <filename>")
        os.Exit(1)
    }
    
    fmt.Println(sanitizeFilename(os.Args[1]))
}
EOF

    go build -o test_sanitize test_sanitize.go
    
    # Test sanitization cases
    sanitize_tests=(
        "hello world.txt:hello-world.txt"
        "file@#$%.txt:file.txt"
        "my--file.txt:my-file.txt"
        "normal_file.txt:normal_file.txt"
        "file   with   spaces.txt:file-with-spaces.txt"
        "café.txt:caf.txt"  # Unicode gets stripped (known limitation)
    )
    
    for test_case in "${sanitize_tests[@]}"; do
        input="${test_case%:*}"
        expected="${test_case#*:}"
        result=$(./test_sanitize "$input")
        
        if [ "$result" = "$expected" ]; then
            print_pass "Filename sanitization: '$input' -> '$result'"
        else
            print_fail "Filename sanitization: '$input' -> expected '$expected', got '$result'"
        fi
    done
    
    rm test_sanitize test_sanitize.go
}

# Test directory structure and file operations
test_directory_operations() {
    print_test "Testing directory structure"
    
    # Test that all expected files exist
    expected_files=(
        "test.txt"
        "data.json"
        "script.sh"
        "readme.md"
        ".gitignore"
        ".vimrc"
        "CHANGELOG"
        "subdir1/nested.txt"
        "subdir1/nested/deep.txt"
        "dir with spaces/spaced.txt"
        "large_file.txt"
        "binary_file"
    )
    
    for file in "${expected_files[@]}"; do
        if [ -e "$file" ]; then
            print_pass "File exists: $file"
        else
            print_fail "File missing: $file"
        fi
    done
    
    # Test directory structure
    expected_dirs=(
        "subdir1"
        "subdir2"
        "dir with spaces"
        "subdir1/nested"
    )
    
    for dir in "${expected_dirs[@]}"; do
        if [ -d "$dir" ]; then
            print_pass "Directory exists: $dir"
        else
            print_fail "Directory missing: $dir"
        fi
    done
}

# Test file size limits
test_file_sizes() {
    print_test "Testing file size handling"
    
    # Check large file size
    if [ -f "large_file.txt" ]; then
        size=$(stat -c%s "large_file.txt" 2>/dev/null || stat -f%z "large_file.txt" 2>/dev/null || echo "0")
        if [ "$size" -gt 10485760 ]; then  # 10MB
            print_pass "Large file created: $(($size / 1024 / 1024))MB"
        else
            print_fail "Large file too small: $(($size / 1024 / 1024))MB"
        fi
    else
        print_fail "Large file not created"
    fi
    
    # Test small files
    small_files=("test.txt" "data.json" ".gitignore")
    for file in "${small_files[@]}"; do
        if [ -f "$file" ]; then
            size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo "0")
            if [ "$size" -lt 1024 ]; then  # Less than 1KB
                print_pass "Small file size OK: $file (${size}B)"
            else
                print_warning "Small file unexpectedly large: $file (${size}B)"
            fi
        fi
    done
}

# Test powpow executable exists and basic functionality
test_powpow_executable() {
    print_test "Testing powpow executable"
    
    if [ ! -f "../powpow" ]; then
        print_fail "powpow executable not found at ../powpow"
        return
    fi
    
    print_pass "powpow executable found"
    
    # Test that it can be executed (this will exit immediately in non-interactive mode)
    if timeout 2s ../powpow 2>/dev/null || [ $? -eq 124 ]; then
        print_pass "powpow executable can be launched"
    else
        print_fail "powpow executable failed to launch"
    fi
}

# Interactive test guidance
print_interactive_tests() {
    echo ""
    echo -e "${BLUE}=== MANUAL TESTING INSTRUCTIONS ===${NC}"
    echo "Run the following manual tests in the test directory:"
    echo ""
    echo "1. Basic Navigation:"
    echo "   cd $TEST_DIR && ../powpow"
    echo "   - Use ↑/↓ or j/k to navigate files"
    echo "   - Press Tab to switch between file list and preview"
    echo "   - Press Enter to open files in \$EDITOR"
    echo "   - Press h to go up directories"
    echo "   - Press l to enter directories"
    echo ""
    echo "2. Search Functionality:"
    echo "   - Press / to enter search mode"
    echo "   - Type 'test' and verify search results"
    echo "   - Use ↑/↓ or j/k to navigate search results"
    echo "   - Press Enter to open selected file"
    echo "   - Press ESC to exit search mode"
    echo ""
    echo "3. File Operations:"
    echo "   - Press n to create new file"
    echo "   - Press N to create new folder"
    echo "   - Press r to rename files"
    echo "   - Press d to delete files (with confirmation)"
    echo ""
    echo "4. Text File Preview:"
    echo "   - Verify text files show content in preview pane"
    echo "   - Verify binary files show file info instead"
    echo "   - Check that large_file.txt shows size warning"
    echo "   - Verify dotfiles (.gitignore, .vimrc) show as text"
    echo ""
    echo "5. Unicode and Edge Cases:"
    echo "   - Check unicode.txt displays correctly"
    echo "   - Navigate to 'dir with spaces' directory"
    echo "   - Test files without extensions (CHANGELOG, LICENSE)"
    echo ""
    echo "6. Exit:"
    echo "   - Press q to quit"
    echo ""
}

# Cleanup function
cleanup() {
    if [ -d "$TEST_DIR" ]; then
        print_test "Cleaning up test directory: $TEST_DIR"
        rm -rf "$TEST_DIR"
        print_pass "Cleanup completed"
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}PowPow Test Suite${NC}"
    echo "=================="
    echo ""
    
    # Set trap for cleanup on exit
    trap cleanup EXIT
    
    # Run tests
    setup_test_env
    test_file_detection
    test_filename_sanitization
    test_directory_operations
    test_file_sizes
    test_powpow_executable
    
    # Print results
    echo ""
    echo -e "${BLUE}=== TEST RESULTS ===${NC}"
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All automated tests passed!${NC}"
    else
        echo -e "${RED}Some tests failed. Check output above.${NC}"
    fi
    
    print_interactive_tests
    
    echo ""
    echo "Test directory will remain at: $TEST_DIR"
    echo "Run 'rm -rf $TEST_DIR' to clean up manually if needed."
}

# Run main function
main "$@"