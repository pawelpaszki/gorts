package coverage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func ComputeFunctionChecksums(filePath string) (map[string]string, error) {
	fileSet := token.NewFileSet()
	node, err := parser.ParseFile(fileSet, filePath, nil, 0) // Don't include comments in hash
	if err != nil {
		return nil, err
	}

	checksums := make(map[string]string)

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}

		// Get function name (with receiver for methods)
		name := getFunctionName(fn)

		// Print function body to string and hash it
		var buf bytes.Buffer
		printer.Fprint(&buf, fileSet, fn.Body)
		hash := sha256.Sum256(buf.Bytes())
		checksums[name] = hex.EncodeToString(hash[:])

		return true
	})

	return checksums, nil
}

// ComputeAllChecksums computes checksums for all Go files in the given directories
// test files are ignored
func ComputeAllChecksums(repoPath string, files []string) (map[string]string, error) {
	allChecksums := make(map[string]string)

	for _, relFile := range files {
		fullPath := filepath.Join(repoPath, relFile)

		// Skip test files - we don't need checksums for them
		if strings.HasSuffix(relFile, "_test.go") {
			continue
		}

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}

		fileChecksums, err := ComputeFunctionChecksums(fullPath)
		if err != nil {
			continue // Skip files that can't be parsed
		}

		for funcName, hash := range fileChecksums {
			qualifiedName := relFile + "::" + funcName
			allChecksums[qualifiedName] = hash
		}
	}

	return allChecksums, nil
}

func getFunctionName(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return fn.Name.Name
	}

	// Method - include receiver type
	// Format must match `go tool covdata func` output: "*Type.Method"
	var recvType string
	if len(fn.Recv.List) > 0 {
		switch t := fn.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				recvType = "*" + ident.Name
			}
		case *ast.Ident:
			recvType = t.Name
		}
	}

	if recvType != "" {
		return recvType + "." + fn.Name.Name
	}
	return fn.Name.Name
}
