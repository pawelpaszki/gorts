package coverage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeFunctionChecksums(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		wantFuncs   []string
		wantErr     bool
	}{
		{
			name: "compute checksum for simple function",
			fileContent: `package model

func NewBook(title, author string) *Book {
	return &Book{Title: title, Author: author}
}
`,
			wantFuncs: []string{"NewBook"},
			wantErr:   false,
		},
		{
			name: "compute checksum for multiple functions",
			fileContent: `package stringutil

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
`,
			wantFuncs: []string{"Truncate", "Capitalize"},
			wantErr:   false,
		},
		{
			name: "handle method with pointer receiver",
			fileContent: `package model

type Book struct {
	Title string
}

func (b *Book) Validate() error {
	if b.Title == "" {
		return errors.New("title required")
	}
	return nil
}
`,
			wantFuncs: []string{"(*Book).Validate"},
			wantErr:   false,
		},
		{
			name: "handle method with value receiver",
			fileContent: `package model

type Author struct {
	Name string
}

func (a Author) FullName() string {
	return a.Name
}
`,
			wantFuncs: []string{"(Author).FullName"},
			wantErr:   false,
		},
		{
			name: "skip function without body",
			fileContent: `package handler

type BookHandler interface {
	GetBook(id string) (*Book, error)
	CreateBook(book *Book) error
}
`,
			wantFuncs: []string{},
			wantErr:   false,
		},
		{
			name: "handle empty function body",
			fileContent: `package middleware

func NoOp() {}
`,
			wantFuncs: []string{"NoOp"},
			wantErr:   false,
		},
		{
			name: "return error for invalid Go syntax",
			fileContent: `package broken

func HandleRequest( {
`,
			wantFuncs: nil,
			wantErr:   true,
		},
		{
			name: "handle init function",
			fileContent: `package config

func init() {
	loadDefaults()
}
`,
			wantFuncs: []string{"init"},
			wantErr:   false,
		},
		{
			name: "handle multiple methods on same type",
			fileContent: `package model

type ReadingList struct {
	Books []string
}

func (r *ReadingList) AddBook(bookID string) bool {
	r.Books = append(r.Books, bookID)
	return true
}

func (r *ReadingList) RemoveBook(bookID string) bool {
	return false
}

func (r *ReadingList) ContainsBook(bookID string) bool {
	return false
}
`,
			wantFuncs: []string{"(*ReadingList).AddBook", "(*ReadingList).RemoveBook", "(*ReadingList).ContainsBook"},
			wantErr:   false,
		},
		{
			name: "handle mixed functions and methods",
			fileContent: `package service

type BookService struct {
	repo BookRepository
}

func NewBookService(repo BookRepository) *BookService {
	return &BookService{repo: repo}
}

func (s *BookService) GetBook(id string) (*Book, error) {
	return s.repo.FindByID(id)
}

func (s BookService) ListBooks() ([]*Book, error) {
	return s.repo.FindAll()
}

func validateISBN(isbn string) bool {
	return len(isbn) == 13
}
`,
			wantFuncs: []string{"NewBookService", "(*BookService).GetBook", "(BookService).ListBooks", "validateISBN"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test.go")
			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			checksums, err := ComputeFunctionChecksums(tmpFile)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			for _, fn := range tt.wantFuncs {
				assert.Contains(t, checksums, fn, "expected function %s not found", fn)
				assert.NotEmpty(t, checksums[fn], "checksum for %s should not be empty", fn)
			}

			assert.Len(t, checksums, len(tt.wantFuncs))
		})
	}
}

func TestComputeFunctionChecksums_NonExistentFile(t *testing.T) {
	_, err := ComputeFunctionChecksums("/nonexistent/path/book_handler.go")
	assert.Error(t, err)
}

func TestComputeFunctionChecksums_IgnoresDocComments(t *testing.T) {
	dir := t.TempDir()

	withoutDocComment := `package model

func (b *Book) IsPublished() bool {
	return b.PublishedAt != nil
}
`
	withDocComment := `package model

// IsPublished returns true if the book has been published
func (b *Book) IsPublished() bool {
	return b.PublishedAt != nil
}
`

	file1 := filepath.Join(dir, "without_doc.go")
	file2 := filepath.Join(dir, "with_doc.go")

	err := os.WriteFile(file1, []byte(withoutDocComment), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte(withDocComment), 0644)
	require.NoError(t, err)

	checksums1, err := ComputeFunctionChecksums(file1)
	require.NoError(t, err)

	checksums2, err := ComputeFunctionChecksums(file2)
	require.NoError(t, err)

	assert.Equal(t, checksums1["(*Book).IsPublished"], checksums2["(*Book).IsPublished"],
		"doc comments above function should not affect checksum")
}

func TestComputeFunctionChecksums_InlineCommentsAffectChecksum(t *testing.T) {
	dir := t.TempDir()

	withoutInline := `package validator

func ValidateISBN(isbn string) bool {
	return len(isbn) == 13
}
`
	withInline := `package validator

func ValidateISBN(isbn string) bool {
	// Check if ISBN-13 format
	return len(isbn) == 13
}
`

	file1 := filepath.Join(dir, "without_inline.go")
	file2 := filepath.Join(dir, "with_inline.go")

	err := os.WriteFile(file1, []byte(withoutInline), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte(withInline), 0644)
	require.NoError(t, err)

	checksums1, err := ComputeFunctionChecksums(file1)
	require.NoError(t, err)

	checksums2, err := ComputeFunctionChecksums(file2)
	require.NoError(t, err)

	assert.NotEqual(t, checksums1["ValidateISBN"], checksums2["ValidateISBN"],
		"inline comments within body affect checksum due to AST position changes")
}

func TestComputeFunctionChecksums_DifferentBodyProducesDifferentChecksum(t *testing.T) {
	dir := t.TempDir()

	version1 := `package stringutil

func Slugify(s string) string {
	return strings.ToLower(s)
}
`
	version2 := `package stringutil

func Slugify(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), " ", "-")
}
`

	file1 := filepath.Join(dir, "version1.go")
	file2 := filepath.Join(dir, "version2.go")

	err := os.WriteFile(file1, []byte(version1), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte(version2), 0644)
	require.NoError(t, err)

	checksums1, err := ComputeFunctionChecksums(file1)
	require.NoError(t, err)

	checksums2, err := ComputeFunctionChecksums(file2)
	require.NoError(t, err)

	assert.NotEqual(t, checksums1["Slugify"], checksums2["Slugify"],
		"different function bodies should produce different checksums")
}

func TestGetFunctionName(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		want        string
	}{
		{
			name: "plain function",
			fileContent: `package handler

func HandleHealth() {}
`,
			want: "HandleHealth",
		},
		{
			name: "pointer receiver method",
			fileContent: `package service

type AuthorService struct{}

func (s *AuthorService) GetAuthor() {}
`,
			want: "(*AuthorService).GetAuthor",
		},
		{
			name: "value receiver method",
			fileContent: `package model

type Book struct{}

func (b Book) GetTitle() string { return "" }
`,
			want: "(Book).GetTitle",
		},
		{
			name: "single letter receiver name",
			fileContent: `package repository

type BookRepository struct{}

func (r *BookRepository) FindByID() {}
`,
			want: "(*BookRepository).FindByID",
		},
		{
			name: "longer receiver name",
			fileContent: `package middleware

type AuthMiddleware struct{}

func (middleware *AuthMiddleware) Authenticate() {}
`,
			want: "(*AuthMiddleware).Authenticate",
		},
		{
			name: "unexported function",
			fileContent: `package validator

func validateStringField() {}
`,
			want: "validateStringField",
		},
		{
			name: "unexported method",
			fileContent: `package service

type bookService struct{}

func (s *bookService) findByISBN() {}
`,
			want: "(*bookService).findByISBN",
		},
		{
			name: "init function",
			fileContent: `package config

func init() {}
`,
			want: "init",
		},
		{
			name: "main function",
			fileContent: `package main

func main() {}
`,
			want: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "test.go", tt.fileContent, 0)
			require.NoError(t, err)

			var funcDecl *ast.FuncDecl
			ast.Inspect(node, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					funcDecl = fn
					return false
				}
				return true
			})
			require.NotNil(t, funcDecl, "no function found in test input")

			got := getFunctionName(funcDecl)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeAllChecksums(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, repoPath string) []string
		wantFuncs []string
		wantEmpty bool
	}{
		{
			name: "compute checksums for single file",
			setup: func(t *testing.T, repoPath string) []string {
				content := `package stringutil

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func Capitalize(s string) string {
	return s
}
`
				pkgDir := filepath.Join(repoPath, "pkg", "stringutil")
				err := os.MkdirAll(pkgDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(pkgDir, "stringutil.go"), []byte(content), 0644)
				require.NoError(t, err)
				return []string{"pkg/stringutil/stringutil.go"}
			},
			wantFuncs: []string{"pkg/stringutil/stringutil.go::Truncate", "pkg/stringutil/stringutil.go::Capitalize"},
		},
		{
			name: "compute checksums for multiple files",
			setup: func(t *testing.T, repoPath string) []string {
				handlerContent := `package handler

func HandleGetBook() {}
`
				serviceContent := `package service

func ProcessOrder() {}
`
				err := os.MkdirAll(filepath.Join(repoPath, "internal", "handler"), 0755)
				require.NoError(t, err)
				err = os.MkdirAll(filepath.Join(repoPath, "internal", "service"), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "internal", "handler", "book_handler.go"), []byte(handlerContent), 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "internal", "service", "book_service.go"), []byte(serviceContent), 0644)
				require.NoError(t, err)
				return []string{"internal/handler/book_handler.go", "internal/service/book_service.go"}
			},
			wantFuncs: []string{"internal/handler/book_handler.go::HandleGetBook", "internal/service/book_service.go::ProcessOrder"},
		},
		{
			name: "skip test files",
			setup: func(t *testing.T, repoPath string) []string {
				sourceContent := `package model

func (b *Book) Validate() error { return nil }
`
				testContent := `package model

func TestBook_Validate() {}
`
				err := os.MkdirAll(filepath.Join(repoPath, "internal", "model"), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "internal", "model", "book.go"), []byte(sourceContent), 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "internal", "model", "book_test.go"), []byte(testContent), 0644)
				require.NoError(t, err)
				return []string{"internal/model/book.go", "internal/model/book_test.go"}
			},
			wantFuncs: []string{"internal/model/book.go::(*Book).Validate"},
		},
		{
			name: "skip non-existent files",
			setup: func(t *testing.T, repoPath string) []string {
				content := `package config

func LoadConfig() {}
`
				err := os.MkdirAll(filepath.Join(repoPath, "internal", "config"), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "internal", "config", "config.go"), []byte(content), 0644)
				require.NoError(t, err)
				return []string{"internal/config/config.go", "internal/config/nonexistent.go"}
			},
			wantFuncs: []string{"internal/config/config.go::LoadConfig"},
		},
		{
			name: "skip files that fail to parse",
			setup: func(t *testing.T, repoPath string) []string {
				validContent := `package middleware

func BasicAuth() {}
`
				invalidContent := `package middleware

func broken( {
`
				err := os.MkdirAll(filepath.Join(repoPath, "internal", "middleware"), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "internal", "middleware", "auth.go"), []byte(validContent), 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "internal", "middleware", "invalid.go"), []byte(invalidContent), 0644)
				require.NoError(t, err)
				return []string{"internal/middleware/auth.go", "internal/middleware/invalid.go"}
			},
			wantFuncs: []string{"internal/middleware/auth.go::BasicAuth"},
		},
		{
			name: "return empty map for empty file list",
			setup: func(t *testing.T, repoPath string) []string {
				return []string{}
			},
			wantEmpty: true,
		},
		{
			name: "handle nested directory paths",
			setup: func(t *testing.T, repoPath string) []string {
				content := `package repo

func (r *BookRepository) FindByID() {}
`
				deepPath := filepath.Join(repoPath, "internal", "repository")
				err := os.MkdirAll(deepPath, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(deepPath, "book_repo.go"), []byte(content), 0644)
				require.NoError(t, err)
				return []string{"internal/repository/book_repo.go"}
			},
			wantFuncs: []string{"internal/repository/book_repo.go::(*BookRepository).FindByID"},
		},
		{
			name: "qualify function names with file path",
			setup: func(t *testing.T, repoPath string) []string {
				content := `package service

type AuthorService struct{}

func NewAuthorService() *AuthorService { return nil }
func (s *AuthorService) GetAuthor() {}
`
				err := os.MkdirAll(filepath.Join(repoPath, "internal", "service"), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "internal", "service", "author_service.go"), []byte(content), 0644)
				require.NoError(t, err)
				return []string{"internal/service/author_service.go"}
			},
			wantFuncs: []string{"internal/service/author_service.go::NewAuthorService", "internal/service/author_service.go::(*AuthorService).GetAuthor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := t.TempDir()
			files := tt.setup(t, repoPath)

			checksums, err := ComputeAllChecksums(repoPath, files)
			require.NoError(t, err)

			if tt.wantEmpty {
				assert.Empty(t, checksums)
				return
			}

			for _, fn := range tt.wantFuncs {
				assert.Contains(t, checksums, fn, "expected function %s not found", fn)
				assert.NotEmpty(t, checksums[fn], "checksum for %s should not be empty", fn)
			}

			assert.Len(t, checksums, len(tt.wantFuncs))
		})
	}
}

func TestChecksumDeterminism_SameCodeProducesSameHash(t *testing.T) {
	content := `package model

func (b *Book) Validate() error {
	if b.Title == "" {
		return errors.New("title is required")
	}
	return nil
}

func (b *Book) IsPublished() bool {
	return b.PublishedAt != nil
}
`
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "book.go")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	checksums1, err := ComputeFunctionChecksums(tmpFile)
	require.NoError(t, err)

	checksums2, err := ComputeFunctionChecksums(tmpFile)
	require.NoError(t, err)

	checksums3, err := ComputeFunctionChecksums(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, checksums1, checksums2, "checksums should be identical across runs")
	assert.Equal(t, checksums2, checksums3, "checksums should be identical across runs")
}

func TestChecksumDeterminism_DifferentFileSameCodeProducesSameHash(t *testing.T) {
	content := `package handler

func HandleHealthCheck() error {
	return nil
}
`
	dir := t.TempDir()

	file1 := filepath.Join(dir, "health_handler.go")
	file2 := filepath.Join(dir, "health_handler_copy.go")

	err := os.WriteFile(file1, []byte(content), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte(content), 0644)
	require.NoError(t, err)

	checksums1, err := ComputeFunctionChecksums(file1)
	require.NoError(t, err)

	checksums2, err := ComputeFunctionChecksums(file2)
	require.NoError(t, err)

	assert.Equal(t, checksums1["HandleHealthCheck"], checksums2["HandleHealthCheck"],
		"same code in different files should produce same checksum")
}

func TestChecksumDeterminism_HashFormat(t *testing.T) {
	content := `package middleware

func LogRequest() {}
`
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "logging.go")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	checksums, err := ComputeFunctionChecksums(tmpFile)
	require.NoError(t, err)

	hash := checksums["LogRequest"]

	assert.Len(t, hash, 64, "SHA256 hash should be 64 hex characters")

	for _, c := range hash {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"hash should only contain lowercase hex characters, got: %c", c)
	}
}
