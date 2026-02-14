# Parser Development Guide

Parsers are the heart of CodeGraph. Every language-specific parser extracts symbols and
references from source files, which are then resolved, graphed, and exposed via MCP tools.

## Table of Contents

1. [Parser interface](#parser-interface)
2. [Core types](#core-types)
3. [Symbol kinds](#symbol-kinds)
4. [Relationship types](#relationship-types)
5. [Adding a new parser](#adding-a-new-parser)
6. [Migration file handling](#migration-file-handling)
7. [Cross-language resolution](#cross-language-resolution)
8. [Testing requirements](#testing-requirements)
9. [Parser-specific notes](#parser-specific-notes)

---

## Parser Interface

Every parser implements this Go interface in `internal/parser/parser.go`:

```go
type Parser interface {
    // Languages returns the file extensions/languages this parser handles
    Languages() []Language

    // Parse extracts symbols and raw (unresolved) references from a single file
    Parse(ctx context.Context, file *FileInput) (*ParseResult, error)

    // ResolveReferences resolves raw references against the project symbol table
    ResolveReferences(ctx context.Context, refs []RawReference, symbolTable *SymbolTable) ([]ResolvedEdge, error)
}
```

Register parsers in `internal/parser/registry.go`. The registry maps file extensions to
parser instances. A file's language is detected by extension first, then by header heuristics
(e.g., SQL dialect detection for `.sql` files).

## Core Types

```go
type FileInput struct {
    Path              string
    Content           []byte
    Language          Language
    SkipColumnLineage bool    // Set for migration/schema files (see ADR-001)
}

type ParseResult struct {
    Symbols    []Symbol        // Extracted named entities
    References []RawReference  // Unresolved references to other symbols
    Errors     []ParseError    // Non-fatal parse errors (file still partially parsed)
}

type RawReference struct {
    FromSymbolID string          // Temporary local ID (scoped to this parse)
    TargetName   string          // What's being referenced: "dbo.Customers", "GetById"
    Kind         ReferenceKind   // CALL, TABLE_READ, COLUMN_WRITE, etc.
    Location     SourceLocation  // Where in the file this reference occurs
    Context      map[string]any  // Parser-specific metadata for resolution
}
```

Important: `RawReference.FromSymbolID` is a temporary ID assigned during parsing. It maps to
a `Symbol` in the same `ParseResult`. The resolution engine later replaces these with real UUIDs.

## Symbol Kinds

Use the correct kind for each entity. The full enum is in `pkg/models/symbol.go`. Common kinds:

**Database objects**: `TABLE`, `VIEW`, `MATERIALIZED_VIEW`, `COLUMN`, `INDEX`, `CONSTRAINT`,
`TRIGGER`, `SEQUENCE`, `SCHEMA`, `DATABASE`

**Routines**: `STORED_PROCEDURE`, `FUNCTION`, `PACKAGE`, `PACKAGE_BODY`

**Application code**: `CLASS`, `INTERFACE`, `STRUCT`, `ENUM`, `ENUM_VALUE`, `METHOD`,
`PROPERTY`, `FIELD`, `CONSTRUCTOR`, `MODULE`, `NAMESPACE`

**Web/API**: `ASP_PAGE`, `ASP_INCLUDE`, `CONTROLLER`, `ACTION`, `ROUTE`, `API_ENDPOINT`

**Delphi**: `DELPHI_UNIT`, `DELPHI_FORM`, `DELPHI_DATAMODULE`, `DELPHI_COMPONENT`,
`DELPHI_RECORD`, `DELPHI_CLASS`

**Queries**: `QUERY`, `CTE`, `SUBQUERY`, `TEMP_TABLE`

**Config**: `ORM_MAPPING`, `CONNECTION_STRING`, `CONFIG_ENTRY`

## Relationship Types

Edges between symbols. Full enum in `pkg/models/edge.go`.

**Structural**: `CONTAINS`, `INHERITS`, `IMPLEMENTS`, `USES_UNIT`, `IMPORTS`
**Call/invocation**: `CALLS`, `INSTANTIATES`
**Data access**: `READS_TABLE`, `WRITES_TABLE`, `READS_COLUMN`, `WRITES_COLUMN`, `JOINS_ON`, `REFERENCES_FK`
**Data flow / lineage**: `TRANSFORMS`, `PASSES_TO`, `RETURNS_FROM`, `ASSIGNS_TO`
**Web/API**: `ROUTES_TO`, `RENDERS`, `INCLUDES`, `BINDS_TO`, `CONNECTS_VIA`
**Config**: `CONFIGURED_BY`, `DEPENDS_ON`

## Adding a New Parser

1. Create directory: `internal/parser/{language}/`
2. Implement the `Parser` interface
3. Register in `internal/parser/registry.go`
4. Create golden test data: `test/testdata/{language}/`
5. Write tests in `internal/parser/{language}/{language}_test.go`
6. Add file extension mappings to the language detection logic

Directory structure for a parser:

```
internal/parser/{language}/
├── {language}.go           # Main parser implementation
├── {language}_test.go      # Unit tests
├── ast.go                  # AST node types (if custom parser)
├── lexer.go                # Lexer (if custom recursive-descent)
└── sql_extraction.go       # SQL string extraction from host language (if applicable)
```

## Migration File Handling (ADR-001)

Files classified as migrations skip column-level lineage extraction. This avoids flooding
the graph with low-value `direct_copy` edges from `INSERT...SELECT` in schema scripts.

**Classification criteria** (set `FileInput.SkipColumnLineage = true`):
- Paths containing: `Database/`, `Migrations/`, `Scripts/`
- Suffixes: `.Install.sql`, `.Upgrade.sql`
- DNN-style paths: `DNN Platform/`, `Dnn.AdminExperience/`, `Providers/`
- Project `settings.lineage_exclude_paths` (glob or substring)

**Parser behavior when `SkipColumnLineage` is true**:
- Still extract all symbols (tables, views, procedures)
- Skip appending to `colRefs` in SELECT, INSERT, UPDATE, SET, MERGE handling
- The T-SQL parser implements this — replicate the pattern in other SQL parsers

## Cross-Language Resolution

The resolver (`internal/resolver/`) runs after all files are parsed. It operates in phases:

**Phase 1 — Build symbol table**: Collect all symbols, index by FQN, short name, file.

**Phase 2 — Resolve imports**: For each file, resolve import/using/uses to actual FQNs.

**Phase 3 — Resolve references**: For each `RawReference`, attempt resolution:
  1. File-local scope (same file)
  2. Imported scope (resolved imports)
  3. Project-wide scope (FQN match)
  4. Cross-source scope (other sources in same project)
  5. If unresolved: create placeholder with `kind=UNKNOWN`

**Phase 4 — Confidence scoring**:
  - Exact FQN match: 1.0
  - Unambiguous short name in imported scope: 0.95
  - Naming convention inference: 0.7
  - Dynamic SQL / string concatenation: 0.5
  - Heuristic: 0.3

**Cross-language bridge rules** (`internal/resolver/crosslang.go`):
- App code → SQL: extract table/procedure names from SQL strings, resolve against
  `DATABASE → SCHEMA → OBJECT` hierarchy, apply `schema_qualified` and `case_insensitive` matching
- C# `[Table("...")]` and `DbSet<T>` → T-SQL tables (default schema `dbo`)
- Empty `FromSymbol` in attribute-only refs → falls back to first symbol in file

## Testing Requirements

**Mandatory for every parser:**

### Golden File Tests

```
test/testdata/{language}/
├── basic_table.sql           # Input file
├── basic_table.expected.json # Expected ParseResult (symbols + references)
├── complex_proc.sql
├── complex_proc.expected.json
├── ...
```

Each test reads the input, runs the parser, and compares against the expected JSON output.
Use `go test -update` flag to regenerate expected files when intentionally changing output.

### Edge Case Tests

Test at minimum:
- Malformed/incomplete syntax (parser should extract what it can, not crash)
- Deeply nested constructs (subqueries, CTEs, nested classes)
- Encoding issues (BOM, mixed encodings)
- Empty files, files with only comments
- Maximum nesting depth (guard against stack overflow)

### Regression Tests

Curated real-world code samples from each language. When a bug is found in production,
add the triggering code as a regression test before fixing.

### Language-Specific Test Scenarios

**T-SQL**: CTEs, dynamic SQL, cursors, MERGE, cross-database queries, linked servers,
synonyms, temp tables (#local, ##global), conditional DDL

**PostgreSQL**: PL/pgSQL bodies, dollar-quoting, inheritance, partitions, JSON operators,
row-level security, custom operators

**ASP Classic**: Include chains, ADO patterns, string-concatenated SQL, Global.asa,
circular includes, mixed HTML/VBScript boundaries

**Delphi**: DFM files (text and binary), generics, interface/implementation uses,
conditional compilation ({$IFDEF}), include files ({$I})

**.NET**: EF LINQ chains, Dapper queries, ADO.NET SqlCommand, attribute-based routing,
Minimal APIs, Blazor components, DI registration, NuGet references

**Java**: Spring annotations, JPA @Query, MyBatis XML, Lombok, multi-module Maven,
Spring Data method name conventions, application.yml property resolution

## Parser-Specific Notes

### T-SQL (✅ Implemented)

Custom recursive-descent parser in Go. Located in `internal/parser/tsql/`.

Key implementation details:
- Default schema is `dbo` when unqualified
- Dynamic SQL (`EXEC(@sql)`, `sp_executesql`) is parsed when statically determinable,
  flagged with confidence 0.5 when dynamic
- Four-part names (`[Server].[DB].[Schema].[Object]`) create cross-boundary edges
- Synonyms resolve through to underlying object
- Handles `SkipColumnLineage` flag for migration files

### PostgreSQL (✅ Implemented)

Uses `pg_query_go/v6` — Go binding for PostgreSQL's own parser. Located in
`internal/parser/pgsql/`.

Key details:
- Perfect parse tree from the actual PG parser — zero ambiguity
- PL/pgSQL bodies (`$$ ... $$`) need a secondary parse pass
- Default schema search path is `public`
- Dollar-quoting must be handled correctly for nested blocks

### .NET (Phase 2)

Current: tree-sitter C# for AST extraction. Produces `uses_table` references from
`[Table("...")]`, `DbSet<T>`, and inline SQL patterns.

Future: Roslyn sidecar (`tools/dotnet-analyzer/`) for full semantic analysis.
The sidecar is a separate .NET process:
```
Go Worker → spawns → dotnet codegraph-analyzer --project {path} --output json
```

Bridge rules: `schema_qualified` + `case_insensitive` to link C# types → T-SQL tables.

### Java (Phase 2)

tree-sitter-java for AST extraction. Key framework patterns to handle:
- Spring: `@RequestMapping`, `@GetMapping`, `@Autowired`, `@Service`, `@Repository`
- JPA: `@Entity`, `@Table`, `@Query`, `@Column`
- MyBatis: XML mapper files with `<select>`, `<insert>` tags — dedicated XML sub-parser
- Spring Data: method name → query derivation (`findByEmailAndActive`)
- Lombok: synthesize implied methods from `@Data`, `@Getter`, etc.

### ASP Classic (Phase 2)

Custom parser for VBScript in HTML (`<% ... %>`). Key challenges:
- Embedded SQL extraction from string concatenation with `&` and `_` line continuation
- Include resolution: `<!-- #include file="..." -->` and `<!-- #include virtual="..." -->`
- Circular include detection
- Reconstructed SQL strings are sub-parsed with the T-SQL parser

### Delphi (Phase 2)

Custom recursive-descent parser. Key challenges:
- DFM parsing (binary → text conversion, component tree extraction)
- SQL in query component `SQL.Strings` properties (TADOQuery, TSQLQuery, etc.)
- Interface vs implementation `uses` clauses (different visibility)
- Conditional compilation (`{$IFDEF}`) — track both branches
