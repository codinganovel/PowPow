package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test fixtures and helper functions

func createTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
	return path
}

func createTestDir(t *testing.T, parent, name string) string {
	path := filepath.Join(parent, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("Failed to create test directory %s: %v", path, err)
	}
	return path
}

func createTestStructure(t *testing.T) string {
	tempDir := t.TempDir()
	
	// Create various file types
	createTestFile(t, tempDir, "simple.txt", "Hello World")
	createTestFile(t, tempDir, "data.json", `{"key": "value", "number": 42}`)
	createTestFile(t, tempDir, "script.sh", "#!/bin/bash\necho 'test'")
	createTestFile(t, tempDir, "readme.md", "# Test Project\n\nThis is a test.")
	createTestFile(t, tempDir, "app.js", "console.log('Hello, World!');")
	createTestFile(t, tempDir, "style.css", "body { margin: 0; }")
	createTestFile(t, tempDir, "main.py", "print('Hello Python')")
	createTestFile(t, tempDir, "main.rs", "fn main() { println!(\"Hello Rust\"); }")
	createTestFile(t, tempDir, "main.go", "package main\n\nfunc main() { fmt.Println(\"Hello Go\") }")
	createTestFile(t, tempDir, "index.html", "<html><body>Hello HTML</body></html>")
	
	// Create dotfiles
	createTestFile(t, tempDir, ".gitignore", "*.log\ntarget/\nnode_modules/")
	createTestFile(t, tempDir, ".vimrc", "syntax on\nset number")
	createTestFile(t, tempDir, ".bashrc", "alias ll='ls -la'")
	createTestFile(t, tempDir, ".env", "NODE_ENV=test\nDEBUG=true")
	
	// Create files without extensions
	createTestFile(t, tempDir, "CHANGELOG", "v1.0.0 - Initial release")
	createTestFile(t, tempDir, "LICENSE", "MIT License\n\nCopyright (c) 2025")
	createTestFile(t, tempDir, "Makefile", "build:\n\tgo build")
	createTestFile(t, tempDir, "Dockerfile", "FROM alpine\nRUN echo hello")
	
	// Create binary file
	binaryData := make([]byte, 1024)
	for i := range binaryData {
		binaryData[i] = byte(i % 256)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "binary_file"), binaryData, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}
	
	// Create large file (>10MB)
	largeFilePath := filepath.Join(tempDir, "large_file.txt")
	largeFile, err := os.Create(largeFilePath)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}
	// Write 11MB of data
	chunk := make([]byte, 1024*1024) // 1MB chunk
	for i := range chunk {
		chunk[i] = 'A'
	}
	for i := 0; i < 11; i++ {
		if _, err := largeFile.Write(chunk); err != nil {
			largeFile.Close()
			t.Fatalf("Failed to write to large file: %v", err)
		}
	}
	largeFile.Close()
	
	// Create Unicode file
	createTestFile(t, tempDir, "unicode.txt", "Hello 世界! Café naïve résumé 🎉")
	
	// Create nested directories
	subdir1 := createTestDir(t, tempDir, "subdir1")
	createTestFile(t, subdir1, "nested.txt", "This is nested content")
	
	nested := createTestDir(t, subdir1, "nested")
	createTestFile(t, nested, "deep.txt", "Deep nested content")
	
	createTestDir(t, tempDir, "subdir2")
	
	spacedDir := createTestDir(t, tempDir, "dir with spaces")
	createTestFile(t, spacedDir, "spaced.txt", "File in spaced directory")
	
	// Create empty directory
	createTestDir(t, tempDir, "empty_dir")
	
	return tempDir
}

// Tests for FileItem creation and properties

func TestNewFileItem(t *testing.T) {
	testDir := createTestStructure(t)
	
	tests := []struct {
		name     string
		filename string
		wantDir  bool
		wantHidden bool
	}{
		{"regular file", "simple.txt", false, false},
		{"hidden file", ".gitignore", false, true},
		{"directory", "subdir1", true, false},
		{"spaced directory", "dir with spaces", true, false},
		{"no extension", "CHANGELOG", false, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(testDir, tt.filename)
			item, err := NewFileItem(path)
			if err != nil {
				t.Fatalf("NewFileItem() error = %v", err)
			}
			
			if item.IsDir != tt.wantDir {
				t.Errorf("IsDir = %v, want %v", item.IsDir, tt.wantDir)
			}
			
			if item.IsHidden != tt.wantHidden {
				t.Errorf("IsHidden = %v, want %v", item.IsHidden, tt.wantHidden)
			}
			
			if item.Name != tt.filename {
				t.Errorf("Name = %v, want %v", item.Name, tt.filename)
			}
			
			if item.Path != path {
				t.Errorf("Path = %v, want %v", item.Path, path)
			}
		})
	}
}

// Tests for Navigator functionality

func TestNavigatorLoadDirectory(t *testing.T) {
	testDir := createTestStructure(t)
	nav := NewNavigator(testDir)
	
	if len(nav.items) == 0 {
		t.Error("Navigator should have loaded items")
	}
	
	// Check that directories come first (sorted)
	foundFirstFile := false
	for _, item := range nav.items {
		if !item.IsDir && !foundFirstFile {
			foundFirstFile = true
		}
		if item.IsDir && foundFirstFile {
			t.Error("Directories should come before files in sorted order")
		}
	}
	
	// Check specific items exist
	expectedItems := []string{"subdir1", "subdir2", "dir with spaces", "empty_dir", ".gitignore", "simple.txt", "CHANGELOG"}
	found := make(map[string]bool)
	for _, item := range nav.items {
		found[item.Name] = true
	}
	
	for _, expected := range expectedItems {
		if !found[expected] {
			t.Errorf("Expected item %s not found in loaded directory", expected)
		}
	}
}

func TestNavigatorEnterDirectory(t *testing.T) {
	testDir := createTestStructure(t)
	nav := NewNavigator(testDir)
	
	// Find subdir1 and enter it
	for i, item := range nav.filteredItems {
		if item.Name == "subdir1" && item.IsDir {
			nav.selectedIdx = i
			break
		}
	}
	
	err := nav.enterDirectory()
	if err != nil {
		t.Fatalf("enterDirectory() error = %v", err)
	}
	
	expectedPath := filepath.Join(testDir, "subdir1")
	if nav.currentPath != expectedPath {
		t.Errorf("currentPath = %v, want %v", nav.currentPath, expectedPath)
	}
	
	// Check that nested.txt is found
	found := false
	for _, item := range nav.items {
		if item.Name == "nested.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected nested.txt to be found in subdir1")
	}
}

func TestNavigatorGoUp(t *testing.T) {
	testDir := createTestStructure(t)
	subdir := filepath.Join(testDir, "subdir1")
	nav := NewNavigator(subdir)
	
	err := nav.goUp()
	if err != nil {
		t.Fatalf("goUp() error = %v", err)
	}
	
	if nav.currentPath != testDir {
		t.Errorf("currentPath = %v, want %v", nav.currentPath, testDir)
	}
	
	// Check that selection is on subdir1 (the directory we came from)
	if nav.selectedIdx >= len(nav.filteredItems) {
		t.Fatal("selectedIdx out of range")
	}
	
	selectedItem := nav.filteredItems[nav.selectedIdx]
	if selectedItem.Name != "subdir1" {
		t.Errorf("After going up, selected item = %v, want subdir1", selectedItem.Name)
	}
}

func TestNavigatorSearch(t *testing.T) {
	testDir := createTestStructure(t)
	nav := NewNavigator(testDir)
	
	// Search for "git"
	nav.setSearch("git")
	
	if len(nav.filteredItems) == 0 {
		t.Error("Search should find items")
	}
	
	// Should find .gitignore
	found := false
	for _, item := range nav.filteredItems {
		if item.Name == ".gitignore" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Search for 'git' should find .gitignore")
	}
	
	// Clear search
	nav.setSearch("")
	if len(nav.filteredItems) != len(nav.items) {
		t.Error("Clearing search should show all items")
	}
}

// Tests for text file detection (simplified without Previewer)

func TestTextFileDetection(t *testing.T) {
	testDir := createTestStructure(t)
	app := &App{} // Create minimal app instance for text detection
	
	tests := []struct {
		filename string
		wantText bool
	}{
		{"simple.txt", true},
		{"data.json", true},
		{"script.sh", true},
		{"readme.md", true},
		{"app.js", true},
		{"style.css", true},
		{"main.py", true},
		{"main.rs", true},
		{"main.go", true},
		{"index.html", true},
		{".gitignore", true},
		{".vimrc", true},
		{"CHANGELOG", true},
		{"LICENSE", true},
		{"Makefile", true},
		{"Dockerfile", true},
		{"unicode.txt", true},
		{"binary_file", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			path := filepath.Join(testDir, tt.filename)
			item, err := NewFileItem(path)
			if err != nil {
				t.Fatalf("NewFileItem() error = %v", err)
			}
			
			isText := app.isTextFile(item)
			if isText != tt.wantText {
				t.Errorf("isTextFile(%s) = %v, want %v", tt.filename, isText, tt.wantText)
			}
		})
	}
}

// Tests for filename sanitization

func TestSanitizeFilename(t *testing.T) {
	// Create a minimal app instance without terminal
	app := &App{}
	
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world.txt", "hello-world.txt"},
		{"file@#$%.txt", "file.txt"},
		{"my--file.txt", "my-file.txt"},
		{"normal_file.txt", "normal_file.txt"},
		{"file   with   spaces.txt", "file-with-spaces.txt"},
		{"café.txt", "caf.txt"}, // Unicode gets stripped
		{"", "untitled"},
		{"---", "untitled"},
		{"file-name.txt", "file-name.txt"},
		{"FILE_NAME.TXT", "FILE_NAME.TXT"},
		{"123.txt", "123.txt"},
		{"a!b@c#d$e%f^g&h*i()j", "abcdefghij"},
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("input_%s", tt.input), func(t *testing.T) {
			result := app.sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Tests for unique file path generation

func TestGetUniqueFilePath(t *testing.T) {
	testDir := createTestStructure(t)
	
	// Create a minimal app instance without terminal
	app := &App{}
	
	// Test with existing file
	existingPath := filepath.Join(testDir, "simple.txt")
	uniquePath := app.getUniqueFilePath(existingPath)
	
	expectedPath := filepath.Join(testDir, "simple-1.txt")
	if uniquePath != expectedPath {
		t.Errorf("getUniqueFilePath() = %v, want %v", uniquePath, expectedPath)
	}
	
	// Test with non-existing file
	nonExistingPath := filepath.Join(testDir, "nonexistent.txt")
	uniquePath = app.getUniqueFilePath(nonExistingPath)
	
	if uniquePath != nonExistingPath {
		t.Errorf("getUniqueFilePath() for non-existing = %v, want %v", uniquePath, nonExistingPath)
	}
}

// Tests for StatusBar functionality

func TestStatusBarMessages(t *testing.T) {
	statusBar := NewStatusBar(false)
	
	// Test default message
	if statusBar.message == "" {
		t.Error("StatusBar should have default message")
	}
	
	// Test regular message
	statusBar.showMessage("Test message")
	if statusBar.message != "Test message" {
		t.Errorf("message = %v, want 'Test message'", statusBar.message)
	}
	if statusBar.isError {
		t.Error("Regular message should not be error")
	}
	
	// Test error message
	statusBar.showError("Error message")
	if statusBar.message != "Error: Error message" {
		t.Errorf("message = %v, want 'Error: Error message'", statusBar.message)
	}
	if !statusBar.isError {
		t.Error("Error message should be marked as error")
	}
}

// Integration tests

func TestNavigatorBasicIntegration(t *testing.T) {
	testDir := createTestStructure(t)
	nav := NewNavigator(testDir)
	
	// Find a text file
	var textItem *FileItem
	for _, item := range nav.items {
		if item.Name == "simple.txt" {
			textItem = &item
			break
		}
	}
	
	if textItem == nil {
		t.Fatal("Could not find simple.txt")
	}
	
	// Test that we can detect text files
	app := &App{}
	if !app.isTextFile(*textItem) {
		t.Error("simple.txt should be detected as text")
	}
}

// Performance tests

func TestNavigatorPerformanceWithManyFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create many files
	numFiles := 1000
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file_%04d.txt", i)
		createTestFile(t, tempDir, filename, fmt.Sprintf("Content of file %d", i))
	}
	
	start := time.Now()
	nav := NewNavigator(tempDir)
	loadTime := time.Since(start)
	
	if len(nav.items) != numFiles {
		t.Errorf("Expected %d items, got %d", numFiles, len(nav.items))
	}
	
	// Should load reasonably quickly (adjust threshold as needed)
	if loadTime > 100*time.Millisecond {
		t.Errorf("Loading %d files took %v, expected < 100ms", numFiles, loadTime)
	}
	
	// Test search performance
	start = time.Now()
	nav.setSearch("file_05")
	searchTime := time.Since(start)
	
	if searchTime > 10*time.Millisecond {
		t.Errorf("Search took %v, expected < 10ms", searchTime)
	}
	
	// Should find relevant files
	if len(nav.filteredItems) == 0 {
		t.Error("Search should find matching files")
	}
}

// Benchmark tests

func BenchmarkNavigatorLoadDirectory(b *testing.B) {
	testDir := createTestStructure(&testing.T{})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nav := NewNavigator(testDir)
		_ = nav.items
	}
}

func BenchmarkTextDetection(b *testing.B) {
	testDir := createTestStructure(&testing.T{})
	app := &App{}
	textPath := filepath.Join(testDir, "simple.txt")
	textItem, _ := NewFileItem(textPath)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = app.isTextFile(textItem)
	}
}

// Error handling tests

func TestNavigatorErrorHandling(t *testing.T) {
	// Test with non-existent directory
	nav := NewNavigator("/non/existent/path")
	
	// Should handle gracefully without crashing
	if len(nav.items) != 0 {
		t.Error("Non-existent directory should result in empty items")
	}
	
	// Test entering non-existent directory
	nav.currentPath = t.TempDir()
	nav.items = []FileItem{{Name: "fake", Path: "/fake/path", IsDir: true}}
	nav.filteredItems = nav.items
	nav.selectedIdx = 0
	
	err := nav.enterDirectory()
	if err == nil {
		t.Error("Entering non-existent directory should return error")
	}
}

func TestTextFileErrorHandling(t *testing.T) {
	app := &App{}
	
	// Test with non-existent file (no extension to avoid extension-based detection)
	fakeItem := FileItem{
		Name: "fakefile",
		Path: "/non/existent/fakefile",
		IsDir: false,
		Size: 100,
	}
	
	// Should handle gracefully without crashing
	isText := app.isTextFile(fakeItem)
	if isText {
		t.Error("Non-existent file should not be detected as text")
	}
}

// Edge case tests

func TestNavigatorEmptyDirectory(t *testing.T) {
	emptyDir := t.TempDir()
	nav := NewNavigator(emptyDir)
	
	if len(nav.items) != 0 {
		t.Error("Empty directory should have no items")
	}
	
	if len(nav.filteredItems) != 0 {
		t.Error("Empty directory should have no filtered items")
	}
}

func TestNavigatorSingleFile(t *testing.T) {
	testDir := t.TempDir()
	createTestFile(&testing.T{}, testDir, "only.txt", "single file")
	
	nav := NewNavigator(testDir)
	
	if len(nav.items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(nav.items))
	}
	
	if nav.items[0].Name != "only.txt" {
		t.Errorf("Expected only.txt, got %s", nav.items[0].Name)
	}
}

func TestEmptyFileTextDetection(t *testing.T) {
	testDir := t.TempDir()
	emptyPath := createTestFile(&testing.T{}, testDir, "empty.txt", "")
	emptyItem, _ := NewFileItem(emptyPath)
	
	app := &App{}
	isText := app.isTextFile(emptyItem)
	
	if !isText {
		t.Error("Empty text file should be detected as text")
	}
}

// Test helper to verify file operations don't modify filesystem unexpectedly
func TestFileSystemSafety(t *testing.T) {
	testDir := createTestStructure(t)
	
	// Record initial state
	initialFiles := make(map[string]os.FileInfo)
	err := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(testDir, path)
		initialFiles[relPath] = info
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk initial directory: %v", err)
	}
	
	// Run navigator operations
	nav := NewNavigator(testDir)
	nav.setSearch("txt")
	nav.setSearch("")
	
	// Enter and exit directories
	for _, item := range nav.items {
		if item.IsDir {
			nav.selectedIdx = 0
			for i, filteredItem := range nav.filteredItems {
				if filteredItem.Name == item.Name {
					nav.selectedIdx = i
					break
				}
			}
			nav.enterDirectory()
			nav.goUp()
			break
		}
	}
	
	// Run text detection operations
	app := &App{}
	for _, item := range nav.items {
		if !item.IsDir {
			app.isTextFile(item)
			break
		}
	}
	
	// Verify filesystem is unchanged
	finalFiles := make(map[string]os.FileInfo)
	err = filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(testDir, path)
		finalFiles[relPath] = info
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk final directory: %v", err)
	}
	
	if len(initialFiles) != len(finalFiles) {
		t.Errorf("File count changed: initial %d, final %d", len(initialFiles), len(finalFiles))
	}
	
	for path, initialInfo := range initialFiles {
		finalInfo, exists := finalFiles[path]
		if !exists {
			t.Errorf("File disappeared: %s", path)
			continue
		}
		
		if initialInfo.Size() != finalInfo.Size() {
			t.Errorf("File size changed for %s: %d -> %d", path, initialInfo.Size(), finalInfo.Size())
		}
		
		if initialInfo.ModTime() != finalInfo.ModTime() {
			t.Errorf("File modification time changed for %s", path)
		}
	}
}