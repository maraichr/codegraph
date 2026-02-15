package embedding

import (
	"fmt"
	"strings"

	"github.com/maraichr/lattice/internal/store/postgres"
)

// BuildEmbeddingText creates the text representation of a symbol for embedding.
// Different symbol kinds get different text formats to maximize semantic quality.
func BuildEmbeddingText(sym postgres.Symbol) string {
	switch strings.ToLower(sym.Kind) {
	case "table":
		return fmt.Sprintf("Table %s", sym.QualifiedName)

	case "stored_procedure", "procedure":
		text := fmt.Sprintf("Stored procedure %s", sym.QualifiedName)
		if sym.Signature != nil && *sym.Signature != "" {
			text += fmt.Sprintf(" %s", *sym.Signature)
		}
		if sym.DocComment != nil && *sym.DocComment != "" {
			text += fmt.Sprintf(" — %s", *sym.DocComment)
		}
		return text

	case "function":
		text := fmt.Sprintf("Function %s", sym.QualifiedName)
		if sym.Signature != nil && *sym.Signature != "" {
			text += fmt.Sprintf(" %s", *sym.Signature)
		}
		if sym.DocComment != nil && *sym.DocComment != "" {
			text += fmt.Sprintf(" — %s", *sym.DocComment)
		}
		return text

	case "view":
		text := fmt.Sprintf("View %s", sym.QualifiedName)
		if sym.DocComment != nil && *sym.DocComment != "" {
			text += fmt.Sprintf(" — %s", *sym.DocComment)
		}
		return text

	case "trigger":
		text := fmt.Sprintf("Trigger %s", sym.QualifiedName)
		if sym.Signature != nil && *sym.Signature != "" {
			text += fmt.Sprintf(" %s", *sym.Signature)
		}
		return text

	case "column":
		text := fmt.Sprintf("Column %s", sym.QualifiedName)
		if sym.Signature != nil && *sym.Signature != "" {
			text += fmt.Sprintf(" type %s", *sym.Signature)
		}
		return text

	case "class":
		text := fmt.Sprintf("Class %s", sym.QualifiedName)
		if sym.DocComment != nil && *sym.DocComment != "" {
			text += fmt.Sprintf(" — %s", *sym.DocComment)
		}
		return text

	case "method":
		text := fmt.Sprintf("Method %s", sym.QualifiedName)
		if sym.Signature != nil && *sym.Signature != "" {
			text += fmt.Sprintf(" %s", *sym.Signature)
		}
		return text

	default:
		text := fmt.Sprintf("%s %s", sym.Kind, sym.QualifiedName)
		if sym.Signature != nil && *sym.Signature != "" {
			text += fmt.Sprintf(": %s", *sym.Signature)
		}
		return text
	}
}
