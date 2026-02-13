# CodeGraph: Semantic Codebase Indexing & Lineage Engine

## Technical Specification v1.0

**Status:** Living Document — updated as implementation progresses
**Date:** 2026-02-13
**Classification:** Internal — Architecture Specification

---

## 1. Executive Summary

CodeGraph is an enterprise-grade semantic indexing engine for large codebases that extracts, indexes, and exposes rich dependency graphs, data lineage, and structural metadata across heterogeneous technology stacks. It ingests code from GitLab (PAT-authenticated), S3 buckets, and ZIP uploads, builds a unified semantic graph, and serves this graph to LLMs via an MCP (Model Context Protocol) tool layer running on AWS Bedrock AgentCore.

The system is designed for enterprise scale: 1,000+ repositories, 100M+ lines of code, spanning SQL Server, PostgreSQL, ASP Classic, Delphi, .NET (C#/VB.NET/F#), and Java.

---

## 2. Goals & Non-Goals

### 2.1 Goals

- **Semantic indexing** of multi-language codebases with full symbol resolution — functions, procedures, classes, tables, views, stored procedures, columns, and their interrelationships
- **Dependency graph construction** — who calls what, who reads/writes which tables, which services depend on which schemas
- **Data lineage tracking** — trace a column from source table through transformations, stored procedures, views, API layers, and finally to UI rendering or downstream consumers
- **Impact analysis** — given a proposed change to table X or procedure Y, enumerate all affected code paths, services, and downstream consumers
- **LLM-accessible research interface** — expose the full semantic graph through MCP tools so that LLMs on Bedrock AgentCore can autonomously research, query, and reason about the codebase
- **Continuous synchronization** — GitLab webhook-triggered resyncs with incremental indexing
- **Multi-tenant project isolation** — separate indexed projects with RBAC

### 2.2 Non-Goals

- Real-time IDE integration (LSP server) — this is a research/analysis tool, not a code editor
- Code generation or automated refactoring
- Runtime dependency analysis (dynamic call graphs, runtime profiling)
- Source code hosting or version control — we index, we don't store authoritative code

### 2.3 Implementation Status

See **[`docs/STATUS.md`](STATUS.md)** for the current implementation tracker (Phase 1 ✅ complete, Phase 2 planned). The roadmap is in §20.

---

## 3. Architecture Overview

### 3.1 High-Level Component Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                        INGESTION LAYER                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────────────┐  │
│  │  GitLab   │  │   S3     │  │   ZIP    │  │  GitLab Webhooks    │  │
│  │  (PAT)    │  │  Ingest  │  │  Upload  │  │  (push/tag/merge)   │  │
│  └─────┬─────┘  └─────┬────┘  └────┬─────┘  └──────────┬──────────┘  │
│        └───────────────┴───────────┴─────────────────────┘           │
│                                │                                     │
│                    ┌───────────▼───────────┐                         │
│                    │   Ingestion Queue     │                         │
│                    │   (Valkey Streams)    │                         │
│                    └───────────┬───────────┘                         │
└────────────────────────────────┼─────────────────────────────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────────────────┐
│                       PARSING LAYER                                  │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                   Worker Pool (Go)                             │  │
│  │  ┌──────┐ ┌──────┐ ┌───────┐ ┌───────┐ ┌──────┐ ┌──────────┐ │  │
│  │  │T-SQL │ │PgSQL │ │  ASP  │ │Delphi │ │ .NET │ │   Java   │ │  │
│  │  │Parser│ │Parser│ │Parser │ │Parser │ │Parser│ │  Parser  │ │  │
│  │  └──────┘ └──────┘ └───────┘ └───────┘ └──────┘ └──────────┘ │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                │                                     │
│                    ┌───────────▼───────────┐                         │
│                    │   Symbol Resolution   │                         │
│                    │   & Cross-Ref Engine  │                         │
│                    └───────────┬───────────┘                         │
└────────────────────────────────┼─────────────────────────────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────────────────┐
│                       STORAGE LAYER                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────────┐ │
│  │  PostgreSQL   │  │  Graph Store  │  │   Object Store (S3/Minio) │ │
│  │  (metadata,   │  │  (Neo4j)     │  │   (raw files, snapshots,  │ │
│  │   symbols,    │  │              │  │    parse artifacts)       │ │
│  │   projects)   │  │              │  │                            │ │
│  └──────────────┘  └──────────────┘  └────────────────────────────┘ │
│  ┌──────────────┐  ┌──────────────┐                                 │
│  │  Vector Store │  │    Valkey    │                                 │
│  │  (pgvector)   │  │  (cache,     │                                 │
│  │              │  │   queues)    │                                 │
│  └──────────────┘  └──────────────┘                                 │
└──────────────────────────────────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────────────────┐
│                        API LAYER (Go)                                │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────────┐ │
│  │  REST API     │  │  GraphQL     │  │   MCP Server              │ │
│  │  (chi v5)     │  │  (gqlgen)    │  │   (Streamable HTTP)       │ │
│  └──────────────┘  └──────────────┘  └────────────────────────────┘ │
│  ┌──────────────┐  ┌──────────────┐                                 │
│  │  WebSocket    │  │  Webhook     │                                 │
│  │  (live index  │  │  Receiver    │                                 │
│  │   progress)   │  │  (GitLab)    │                                 │
│  └──────────────┘  └──────────────┘                                 │
└──────────────────────────────────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────────────────┐
│                      FRONTEND (React)                                │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────────┐ │
│  │  Project      │  │  Dependency  │  │   Lineage Explorer        │ │
│  │  Management   │  │  Graph Viz   │  │   (column-level)          │ │
│  └──────────────┘  └──────────────┘  └────────────────────────────┘ │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────────┐ │
│  │  Impact       │  │  Index       │  │   Search & Browse         │ │
│  │  Analysis     │  │  Dashboard   │  │                            │ │
│  └──────────────┘  └──────────────┘  └────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.2 Technology Stack

| Layer | Technology | Rationale |
|---|---|---|
| API / Backend | Go 1.25 | Performance, concurrency, static binary deployment |
| API Router | chi v5 | stdlib-compatible, lightweight |
| Ingestion Queue | Valkey Streams (Valkey 8.1+, Linux Foundation, BSD-3) | Lightweight, high-throughput, supports consumer groups |
| Relational Store | PostgreSQL 17+ with pgvector 0.8.0 | Metadata, symbols, RBAC, plus vector embeddings |
| Graph Store | Neo4j 2026.01 | Native graph traversal for dependency/lineage queries |
| Object Store | S3 / MinIO | Raw source files, parse artifacts, snapshots |
| Cache | Valkey 8.1+ | Query cache, session state, rate limiting |
| Frontend | React 19 / TypeScript 5.9 | Component library, graph visualization |
| Graph Viz | Cytoscape.js or D3-force | Interactive dependency/lineage visualization |
| MCP Runtime | Bedrock AgentCore | LLM tool invocation layer |
| MCP Transport | Streamable HTTP | MCP spec 2025-11-25 (SSE deprecated) |
| Container Orchestration | Kubernetes (Helm 4.1+) | Self-hosted enterprise deployment |
| CI/CD | GitLab CI | Matches existing GitLab infrastructure |

### 3.3 Canonical Version Reference

The following tables are the authoritative version reference for all CodeGraph dependencies, finalized during Phase 1 bootstrap.

**Backend (Go)**

| Dependency | Version | Notes |
|---|---|---|
| Go | 1.25.0 | Language runtime |
| chi | v5 | HTTP router (stdlib-compatible) |
| gqlgen | latest | GraphQL code generator |
| sqlc | latest | Type-safe SQL code generator |
| golang-migrate | latest | Database migrations |
| pgx | v5 | PostgreSQL driver |
| neo4j-go-driver | v5 | Neo4j driver |
| valkey-go | latest | Valkey client (Redis-compatible) |
| aws-sdk-go-v2 | latest | AWS SDK (Bedrock, S3) |
| mcp-go | latest | MCP server SDK (Streamable HTTP) |

**Frontend**

| Dependency | Version | Notes |
|---|---|---|
| React | 19 | UI framework |
| TypeScript | 5.9 | Type safety |
| Vite | 7.2 | Build tooling |
| React Router | v7 | Routing |
| Tailwind CSS | v4 | Utility-first CSS |
| Biome | 2.x | Linting + formatting |
| pnpm | latest | Package manager |

**Infrastructure**

| Component | Version | Notes |
|---|---|---|
| PostgreSQL | 17+ | Relational store |
| pgvector | 0.8.0 | Vector embeddings extension |
| Neo4j | 2026.01 | Graph database (calendar versioning) |
| Valkey | 8.1+ | Cache + queue (BSD-3, Linux Foundation) |
| MinIO | latest | S3-compatible object store (dev/self-hosted) |
| Kubernetes | 1.31+ | Container orchestration |
| Helm | 4.1+ | K8s package manager |

---

## 4. Data Model

### 4.1 Core Entities

#### 4.1.1 Project

A logical grouping of one or more repositories/sources that form a coherent system.

```
Project {
  id:             UUID
  name:           string
  slug:           string (unique)
  description:    text
  owner_id:       UUID (→ User)
  created_at:     timestamp
  updated_at:     timestamp
  settings:       JSONB {
    default_branch:    string
    auto_resync:       bool
    resync_schedule:   cron_expression | null
    retention_policy:  { max_snapshots: int, max_age_days: int }
  }
}
```

#### 4.1.2 Source

A code source connected to a project — a GitLab repo, S3 bucket path, or uploaded archive.

```
Source {
  id:             UUID
  project_id:     UUID (→ Project)
  type:           enum(GITLAB, S3, ZIP_UPLOAD)     // see implementation note below
  name:           string
  config:         JSONB {
    // GITLAB
    gitlab_url:        string
    project_id:        int
    pat_secret_ref:    string (reference to Kubernetes Secret or Vault path)
    branch:            string
    webhook_secret:    string

    // S3
    bucket:            string
    prefix:            string
    region:            string
    credentials_ref:   string

    // ZIP_UPLOAD
    original_filename: string
    upload_path:       string (S3 key where ZIP was stored)
  }
  status:          enum(ACTIVE, DISABLED, ERROR)
  last_sync_at:    timestamp
  last_commit_sha: string | null
  created_at:      timestamp
}
```

> **Implementation note:** The actual DB uses a different column name and values:
> - Column is `source_type` (not `type`), with lowercase values: `git`, `database`, `filesystem`, `upload`
> - Has `connection_uri` column (not in spec) for git clone URLs
> - GraphQL maps to uppercase enums: `GIT`, `DATABASE`, `FILESYSTEM`, `UPLOAD`
> - No `status` or `last_commit_sha` columns in current schema

#### 4.1.3 IndexRun

A single indexing pass over a source.

```
IndexRun {
  id:              UUID
  source_id:       UUID (→ Source)
  project_id:      UUID (→ Project)
  trigger:         enum(MANUAL, WEBHOOK, SCHEDULE, UPLOAD)
  status:          enum(QUEUED, CLONING, PARSING, RESOLVING, GRAPH_BUILDING, EMBEDDING, COMPLETE, FAILED)
  started_at:      timestamp
  completed_at:    timestamp | null
  commit_sha:      string | null
  stats:           JSONB {
    files_total:       int
    files_parsed:      int
    files_failed:      int
    symbols_extracted: int
    edges_created:     int
    duration_ms:       int
    errors:            [{ file: string, line: int, message: string }]
  }
  error_message:   text | null
}
```

> **Implementation note:** The actual DB schema (`migrations/postgres/000001_initial_schema.up.sql`) uses a simpler model:
> - **Status values:** `pending`, `running`, `completed`, `failed`, `cancelled` (not the 7-stage enum above)
> - **No `trigger` column** — trigger type is not persisted
> - **Flat stats columns:** `files_processed`, `symbols_found`, `edges_found` (integer columns, not JSONB)
> - The aspirational multi-stage status and detailed JSONB stats are deferred to Phase 2+

#### 4.1.4 File

```
File {
  id:              UUID
  source_id:       UUID (→ Source)
  index_run_id:    UUID (→ IndexRun)
  path:            string (relative path within source)
  language:        enum(TSQL, PGSQL, ASP_CLASSIC, ASP_NET, DELPHI, CSHARP, VB_NET, FSHARP, JAVA, XML_CONFIG, UNKNOWN)
  size_bytes:      int
  line_count:      int
  hash_sha256:     string
  parse_status:    enum(PARSED, FAILED, SKIPPED)
  parse_error:     text | null
}
```

### 4.2 Semantic Symbol Model

The core abstraction — every meaningful named entity in the codebase becomes a `Symbol`.

#### 4.2.1 Symbol

```
Symbol {
  id:              UUID
  project_id:      UUID (→ Project)
  file_id:         UUID (→ File)
  source_id:       UUID (→ Source)
  index_run_id:    UUID (→ IndexRun)

  kind:            enum(
    // Database objects
    DATABASE, SCHEMA, TABLE, VIEW, MATERIALIZED_VIEW,
    COLUMN, INDEX, CONSTRAINT, TRIGGER, SEQUENCE,

    // Routines
    STORED_PROCEDURE, FUNCTION, PACKAGE, PACKAGE_BODY,

    // Application objects
    CLASS, INTERFACE, STRUCT, ENUM, ENUM_VALUE,
    METHOD, PROPERTY, FIELD, CONSTRUCTOR,
    MODULE, NAMESPACE, UNIT,

    // ASP/Web
    ASP_PAGE, ASP_INCLUDE, WEB_FORM, WEB_CONTROL,
    CONTROLLER, ACTION, ROUTE, API_ENDPOINT,

    // Delphi
    DELPHI_UNIT, DELPHI_FORM, DELPHI_DATAMODULE,
    DELPHI_COMPONENT, DELPHI_RECORD, DELPHI_CLASS,

    // Config/mapping
    ORM_MAPPING, CONNECTION_STRING, CONFIG_ENTRY,

    // Queries
    QUERY, CTE, SUBQUERY, TEMP_TABLE
  )

  fqn:             string  // Fully Qualified Name, e.g. "dbo.Customers.Email"
  name:            string  // Short name, e.g. "Email"
  parent_id:       UUID | null (→ Symbol, for nesting)

  location:        JSONB {
    start_line:    int
    end_line:      int
    start_col:     int
    end_col:       int
  }

  metadata:        JSONB {
    // Type-specific metadata examples:
    // TABLE: { engine: "InnoDB", estimated_rows: 1000000 }
    // COLUMN: { data_type: "nvarchar(255)", nullable: true, default: "''" }
    // METHOD: { return_type: "Task<IActionResult>", visibility: "public", is_async: true }
    // STORED_PROCEDURE: { parameters: [{ name: "@CustomerId", type: "int", direction: "IN" }] }
    // API_ENDPOINT: { http_method: "GET", route_template: "/api/v1/customers/{id}" }
    // DELPHI_UNIT: { interface_uses: ["SysUtils", "Classes"], implementation_uses: ["DB"] }
  }

  documentation:   text | null  // Extracted doc comments, XML docs, Javadoc
  signature:       text | null  // Full declaration signature for display
}
```

#### 4.2.2 SymbolEdge (Graph Relationships)

```
SymbolEdge {
  id:              UUID
  source_symbol:   UUID (→ Symbol)
  target_symbol:   UUID (→ Symbol)
  project_id:      UUID (→ Project)
  index_run_id:    UUID (→ IndexRun)

  relationship:    enum(
    // Structural
    CONTAINS,           // Table → Column, Class → Method
    INHERITS,           // Class → Base Class
    IMPLEMENTS,         // Class → Interface
    USES_UNIT,          // Delphi unit uses clause
    IMPORTS,            // Java/C# import/using

    // Call/invocation
    CALLS,              // Method → Method, Proc → Proc
    INSTANTIATES,       // Code → new ClassName()

    // Data access
    READS_TABLE,        // Code reads from table (SELECT)
    WRITES_TABLE,       // Code writes to table (INSERT/UPDATE/DELETE)
    READS_COLUMN,       // Column-level read tracking
    WRITES_COLUMN,      // Column-level write tracking
    JOINS_ON,           // JOIN relationship between tables/columns
    REFERENCES_FK,      // Foreign key reference

    // Data flow / lineage
    TRANSFORMS,         // Column A derived from Column B (e.g., in a VIEW)
    PASSES_TO,          // Data flows from one parameter to another
    RETURNS_FROM,       // Function return value derives from
    ASSIGNS_TO,         // Variable assignment lineage

    // Web/API
    ROUTES_TO,          // URL route → Controller/Action
    RENDERS,            // Controller → View/Page
    INCLUDES,           // ASP include file
    BINDS_TO,           // ORM mapping → DB object
    CONNECTS_VIA,       // Connection string → Database

    // Config
    CONFIGURED_BY,      // Runtime behavior driven by config entry
    DEPENDS_ON          // Generic build/deploy dependency
  )

  metadata:        JSONB {
    // Relationship-specific metadata:
    // READS_TABLE: { operations: ["SELECT"], conditions: "WHERE active = 1" }
    // WRITES_COLUMN: { operation: "UPDATE", is_conditional: true }
    // TRANSFORMS: { expression: "UPPER(first_name) + ' ' + last_name", transform_type: "CONCAT" }
    // CALLS: { is_dynamic: false, call_site_line: 42 }
    // JOINS_ON: { join_type: "LEFT", condition: "a.id = b.customer_id" }
  }

  location:        JSONB {  // Where the reference occurs in source
    file_id:       UUID
    start_line:    int
    end_line:      int
  }

  confidence:      float  // 0.0-1.0, for heuristic/inferred relationships
}
```

### 4.3 Graph Store Schema (Neo4j)

The relational model is mirrored into Neo4j for efficient traversal queries. Each `Symbol` becomes a node labeled with its `kind`, and each `SymbolEdge` becomes a typed relationship.

```cypher
// Node labels match Symbol.kind
(:TABLE {id, fqn, name, project_id, ...})
(:COLUMN {id, fqn, name, data_type, nullable, ...})
(:STORED_PROCEDURE {id, fqn, name, ...})
(:CLASS {id, fqn, name, ...})
(:METHOD {id, fqn, name, return_type, ...})

// Relationships match SymbolEdge.relationship
(:TABLE)-[:CONTAINS]->(:COLUMN)
(:STORED_PROCEDURE)-[:READS_COLUMN]->(:COLUMN)
(:STORED_PROCEDURE)-[:WRITES_TABLE]->(:TABLE)
(:METHOD)-[:CALLS]->(:METHOD)
(:COLUMN)-[:TRANSFORMS]->(:COLUMN)
(:CLASS)-[:INHERITS]->(:CLASS)
(:CONTROLLER)-[:ROUTES_TO {http_method: "GET", route: "/api/customers"}]->(:ACTION)
```

### 4.4 Vector Embeddings (pgvector)

```
SymbolEmbedding {
  symbol_id:       UUID (→ Symbol, PK)
  embedding:       vector(1024)  // Cohere Embed v4
  content_hash:    string        // To detect when re-embedding is needed
  model_version:   string        // Track which embedding model produced this
  created_at:      timestamp
}
```

**Embedding model:** Cohere Embed v4 on Bedrock (multimodal, 100+ languages, 1024 dimensions).

Embedding content is constructed per symbol type:

- **TABLE:** `"Table dbo.Customers: columns [Id int PK, Email nvarchar(255), CreatedAt datetime2]"`
- **METHOD:** `"public async Task<CustomerDto> GetCustomer(int id) — Retrieves a customer by ID from the Customers table. Returns CustomerDto with Email, Name, and OrderCount fields."`
- **STORED_PROCEDURE:** Full body text with parameter signature and inline documentation

---

## 5. Ingestion Pipeline

### 5.1 Source Connectors

#### 5.1.1 GitLab Connector

**Authentication:** Personal Access Token (PAT) stored as Kubernetes Secret, referenced by `pat_secret_ref` in Source config. PATs are never stored in the database.

**Clone Strategy:**
- Initial index: `git clone --depth 1 --single-branch --branch {branch}` for speed, then full clone if history-based analysis is needed
- Resync: `git fetch origin {branch} && git diff --name-only {last_sha}..{new_sha}` to identify changed files, parse only those (incremental indexing)
- Large repos (>1GB): Use `git sparse-checkout` with language-specific patterns

**Webhook Receiver:**

```
POST /api/v1/webhooks/gitlab/{source_id}

Headers:
  X-Gitlab-Token: {webhook_secret}
  X-Gitlab-Event: Push Hook | Tag Push Hook | Merge Request Hook

Payload processing:
  1. Validate X-Gitlab-Token against source.config.webhook_secret
  2. Extract commit SHA, changed file paths, branch
  3. If branch matches source.config.branch:
     a. Enqueue IndexRun with trigger=WEBHOOK
     b. Attach changed_files list for incremental indexing
  4. Respond 200 immediately (async processing)
```

**Resync Triggers:**
1. **Webhook push events** — automatic on every push to tracked branch
2. **Manual trigger** — via API `POST /api/v1/sources/{id}/resync` or UI button
3. **Scheduled** — cron expression in project settings (e.g., `0 2 * * *` for nightly)
4. **Polling fallback** — if webhooks are unreliable, optional polling every N minutes via `GET /api/v4/projects/{id}/repository/commits?ref_name={branch}&since={last_sync}`

**GitLab API Integration:**

```
Required PAT scopes: read_repository, read_api
Base URL: {gitlab_url}/api/v4

Endpoints used:
  GET  /projects/{id}/repository/tree?recursive=true     — file listing
  GET  /projects/{id}/repository/files/{path}/raw         — file content (for targeted fetch)
  GET  /projects/{id}/repository/commits?ref_name={branch} — commit history
  POST /projects/{id}/hooks                                — webhook registration (optional auto-setup)
```

#### 5.1.2 S3 Connector

**Authentication:** IAM Role (preferred for K8s via IRSA/Pod Identity) or access key pair stored in Kubernetes Secret.

**Sync Strategy:**
- List objects under `{bucket}/{prefix}` with pagination
- Compare ETags / LastModified against stored file hashes
- Download changed/new files only
- Support for S3 Event Notifications (SNS/SQS) to trigger reindex on upload

**File Layout Expectations:**
```
s3://{bucket}/{prefix}/
  ├── repo-a/
  │   ├── src/
  │   └── sql/
  ├── repo-b/
  │   └── ...
  └── manifest.json (optional — maps paths to logical repos)
```

#### 5.1.3 ZIP Upload Connector

**Upload Flow:**
1. Frontend uploads ZIP via `POST /api/v1/sources/{project_id}/upload` (multipart/form-data)
2. Backend streams to S3/MinIO with a generated key: `uploads/{project_id}/{source_id}/{timestamp}.zip`
3. Creates Source record with type=ZIP_UPLOAD
4. Enqueues IndexRun
5. Worker extracts ZIP to temp directory, applies language detection, parses

**Constraints:**
- Max upload size: 2GB (configurable)
- Supported archive formats: `.zip`, `.tar.gz`, `.tar.bz2`
- Virus scanning hook (optional, pre-processing step)

### 5.2 Ingestion Queue Design

```
Valkey Stream: codegraph:ingest

Message schema:
{
  "index_run_id": "uuid",
  "source_id": "uuid",
  "project_id": "uuid",
  "trigger": "WEBHOOK | MANUAL | SCHEDULE | UPLOAD",
  "priority": 0-9 (0=highest, used for manual triggers),
  "incremental": true | false,
  "changed_files": ["path/a.cs", "path/b.sql"] | null,
  "commit_sha": "abc123" | null,
  "enqueued_at": "ISO8601"
}

Consumer group: codegraph-workers
  - N worker instances claim messages via XREADGROUP
  - XACK on completion, XCLAIM for stale messages (recovery)
  - Dead letter after 3 retries

Note: Valkey 8.1 is API-compatible with Redis. All commands (XREADGROUP, XACK, XCLAIM, ZADD) work identically.
```

**Priority Queue Implementation:** Use sorted sets alongside streams — high-priority items (manual resync, webhook) are processed before scheduled/background jobs.

### 5.3 Worker Pipeline Stages

Each IndexRun passes through these stages sequentially:

```
QUEUED → CLONING → PARSING → RESOLVING → GRAPH_BUILDING → EMBEDDING → COMPLETE
                                                                        ↓
                                                                      FAILED (at any stage)
```

**Stage 1: CLONING**
- Git clone/fetch or S3 download or ZIP extraction
- Output: local file tree in worker temp directory

**Stage 2: PARSING**
- Language detection (by extension + header heuristics)
- Fan out to language-specific parsers (see §6)
- Output: per-file parse results (symbols + raw references)

**Stage 3: RESOLVING**
- Cross-file symbol resolution
- Fully qualify all references (resolve imports, aliases, schemas)
- Match unresolved references against known symbol FQNs
- Output: resolved symbol graph with SymbolEdge records

**Stage 4: GRAPH_BUILDING**
- Upsert symbols into PostgreSQL (replace for full re-index, merge for incremental)
- Sync graph nodes/edges to Neo4j
- Maintain lineage chains

**Stage 5: EMBEDDING**
- Generate embedding text per symbol
- Batch embed via Bedrock (Titan Embeddings or Cohere)
- Upsert into pgvector
- Output: searchable vector index

---

## 6. Language Parsers

### 6.1 Parser Architecture

Each language parser implements a common Go interface:

```go
type Parser interface {
    // Languages returns the set of file extensions/languages this parser handles
    Languages() []Language

    // Parse extracts symbols and raw references from a single file
    Parse(ctx context.Context, file *FileInput) (*ParseResult, error)

    // ResolveReferences takes raw references and resolves them against the project symbol table
    ResolveReferences(ctx context.Context, refs []RawReference, symbolTable *SymbolTable) ([]ResolvedEdge, error)
}

type FileInput struct {
    Path     string
    Content  []byte
    Language Language
}

type ParseResult struct {
    Symbols    []Symbol
    References []RawReference  // Unresolved — "this code mentions 'dbo.Customers'"
    Errors     []ParseError
}

type RawReference struct {
    FromSymbolID string          // Temp local ID
    TargetName   string          // "dbo.Customers", "CustomerService.GetById"
    Kind         ReferenceKind   // CALL, TABLE_READ, COLUMN_WRITE, etc.
    Location     SourceLocation
    Context      map[string]any  // Parser-specific context
}
```

### 6.2 SQL Server (T-SQL) Parser

**Parsing Strategy:** Custom recursive-descent parser built in Go. T-SQL is complex enough to warrant a dedicated parser rather than relying on tree-sitter (which has limited T-SQL support).

**Extracted Symbols:**
- Tables, Views, Materialized Views (including column definitions with types, nullability, defaults, computed expressions)
- Stored Procedures, Functions (scalar, table-valued, inline)
- Triggers (with trigger events: INSERT, UPDATE, DELETE, and timing: AFTER, INSTEAD OF)
- Indexes, Constraints (PK, FK, UNIQUE, CHECK, DEFAULT)
- Synonyms, Sequences, User-Defined Types
- CTEs (Common Table Expressions), Temp Tables (#temp, ##global)

**Extracted Relationships:**
- `READS_TABLE` / `READS_COLUMN` — from SELECT, JOIN conditions, WHERE clauses, subqueries
- `WRITES_TABLE` / `WRITES_COLUMN` — from INSERT, UPDATE, DELETE, MERGE, SELECT INTO
- `CALLS` — EXEC/EXECUTE procedure calls, function invocations
- `TRANSFORMS` — column lineage through VIEW definitions, CTEs, derived columns (`SELECT a.col1 + b.col2 AS combined`)
- `JOINS_ON` — explicit join conditions with join type (INNER, LEFT, RIGHT, FULL, CROSS)
- `REFERENCES_FK` — foreign key relationships from DDL

**Special Handling:**
- **Dynamic SQL:** Parse `EXEC(@sql)` and `sp_executesql` — extract embedded SQL strings where statically determinable. Flag as `confidence: 0.5` when dynamic. Log as `DYNAMIC_SQL_DETECTED` for manual review.
- **Linked servers:** Detect four-part names (`[Server].[Database].[Schema].[Object]`) and create cross-boundary reference edges
- **Synonyms:** Resolve through to underlying object
- **Schema-qualified names:** Always resolve to FQN. Default schema = `dbo` when unqualified.
- **Cross-database references:** `[OtherDB].[dbo].[Table]` — create placeholder symbols for external databases
- **Conditional DDL:** Handle `IF NOT EXISTS` / `IF OBJECT_ID(...) IS NOT NULL` patterns

### 6.3 PostgreSQL Parser

**Parsing Strategy:** Leverage `pg_query_go` (Go binding for PostgreSQL's own parser, `libpg_query`). This gives us a perfect parse tree using the exact same parser PostgreSQL uses internally.

**Extracted Symbols:** Same categories as T-SQL, plus:
- Extensions, Schemas (CREATE SCHEMA)
- Row-level security policies
- Custom operators, aggregate functions
- Partitioned tables and partition definitions

**Additional Relationships:**
- `TRANSFORMS` through PL/pgSQL function bodies (variable assignments, RETURN expressions)
- `CALLS` for PL/pgSQL internal function calls and `PERFORM` statements
- Inheritance chains (`INHERITS` in PostgreSQL table definitions)

**Special Handling:**
- **PL/pgSQL bodies:** Secondary parse pass inside `$$ ... $$` blocks using a PL/pgSQL-aware sub-parser
- **Schema search path:** Resolve unqualified names using `search_path` configuration (default: `public`)
- **Dollar-quoting:** Properly handle nested `$$` blocks
- **COPY FROM / TO:** Detect bulk data flow paths

### 6.4 ASP Classic Parser

**Parsing Strategy:** Custom parser for VBScript embedded in HTML (`<% ... %>` delimiters). This is a legacy language with no modern tooling.

**Extracted Symbols:**
- ASP Pages (`.asp` files)
- Server-side includes (`<!-- #include file="..." -->`, `<!-- #include virtual="..." -->`)
- VBScript Functions, Subs, Classes
- ADO connection strings, SQL query strings
- Response/Request/Session/Application variable usage

**Extracted Relationships:**
- `INCLUDES` — include file references (resolve relative and virtual paths)
- `READS_TABLE` / `WRITES_TABLE` — extracted from inline SQL strings (ADO `Connection.Execute`, `Recordset.Open`)
- `CALLS` — function/sub calls across includes
- `CONNECTS_VIA` — ADO connection string → database
- `ROUTES_TO` — URL-to-file mapping (if IIS config available)

**Special Handling:**
- **Embedded SQL extraction:** RegEx + heuristic extraction of SQL from string concatenation patterns:
  ```vbscript
  sql = "SELECT * FROM Customers WHERE Id = " & Request("id")
  ```
  Parser must handle multi-line string concatenation (`&` and `_` line continuation) and reconstruct the full SQL, then sub-parse the SQL string with the T-SQL parser.
- **Include resolution:** Build include graph, handle circular includes
- **Mixed HTML/VBScript:** Track `<% ... %>` boundaries accurately
- **Global.asa:** Parse Application_OnStart, Session_OnStart for global state

### 6.5 Delphi (Object Pascal) Parser

**Parsing Strategy:** Custom recursive-descent parser for Delphi's Object Pascal syntax. Tree-sitter has a Pascal grammar that can be used as a foundation, supplemented with Delphi-specific extensions (generics, attributes, anonymous methods).

**Extracted Symbols:**
- Units (with interface/implementation section separation)
- Classes, Records, Interfaces
- Methods (procedures/functions), Properties, Fields
- Forms (`.dfm` files — parse DFM to extract component hierarchy)
- DataModules (critical for database access patterns)
- Published properties and event handlers
- Types (including generic specializations)

**Extracted Relationships:**
- `USES_UNIT` — `uses` clause parsing (separate interface vs implementation uses)
- `INHERITS` — class inheritance (`TMyForm = class(TForm)`)
- `IMPLEMENTS` — interface implementation
- `CALLS` — method calls, event handler wiring
- `READS_TABLE` / `WRITES_TABLE` — via ADO/BDE/FireDAC component analysis in DFM files:
  ```delphi
  // From .dfm:
  object qryCustomers: TADOQuery
    Connection = dmMain.ADOConnection1
    SQL.Strings = ('SELECT * FROM Customers WHERE Active = 1')
  end
  ```
- `BINDS_TO` — DataModule component → database object mappings
- `CONNECTS_VIA` — Connection component → connection string → database

**Special Handling:**
- **DFM parsing:** Binary DFM files must be converted to text (`convert -t`), then parse component tree, extract SQL from query components, map event handlers
- **Component-SQL extraction:** Parse SQL.Strings properties from TADOQuery, TSQLQuery, TFDQuery, TIBQuery, etc.
- **Conditional compilation:** Handle `{$IFDEF}` / `{$IFNDEF}` / `{$IF}` directives
- **Include files:** Handle `{$I filename.inc}` / `{$INCLUDE}`
- **Unit scoping:** Interface section = public API, implementation section = private

### 6.6 .NET Parser (C#, VB.NET, F#)

**Parsing Strategy:** Use Roslyn-based analysis via a sidecar .NET tool invoked from Go. The Go worker shells out to a .NET CLI tool (`dotnet codegraph-analyzer`) that uses Roslyn's semantic model for full type resolution. Output is JSON streamed back to Go.

**Sidecar Architecture:**
```
Go Worker → spawns → dotnet codegraph-analyzer --project {path} --output json
                     (uses Roslyn MSBuildWorkspace for full semantic analysis)
                     → streams JSON results to stdout → Go worker reads and processes
```

**Extracted Symbols:**
- Namespaces, Classes, Interfaces, Structs, Records, Enums
- Methods (with full signature, generics, async markers), Properties, Fields, Events
- Controllers, Actions (MVC/Web API — detected via attributes `[ApiController]`, `[HttpGet]`, etc.)
- Entity Framework DbContext, DbSet mappings, Fluent API configurations
- Dependency injection registrations (from `Startup.cs` / `Program.cs`)
- Configuration bindings (`IOptions<T>`, `appsettings.json`)

**Extracted Relationships:**
- `INHERITS`, `IMPLEMENTS` — full type hierarchy
- `CALLS` — method invocations with full overload resolution
- `READS_TABLE` / `WRITES_TABLE` — via:
  - Entity Framework LINQ queries (detect `.Where()`, `.Include()`, `.Select()` chains → map to table/column access)
  - Dapper queries (extract SQL strings from `connection.Query<T>("SELECT...")`)
  - ADO.NET `SqlCommand` (extract SQL from `CommandText` assignments)
  - Raw SQL in EF (`FromSqlRaw`, `ExecuteSqlRaw`)
- `ROUTES_TO` — attribute-based routing (`[Route("api/[controller]")]`, `[HttpGet("{id}")]`)
- `BINDS_TO` — EF entity class ↔ table mapping (convention-based and Fluent API)
- `CONFIGURED_BY` — `IConfiguration` / `IOptions<T>` bindings to `appsettings.json` keys
- `DEPENDS_ON` — NuGet package references, project references

**Special Handling:**
- **Solution/Project loading:** MSBuildWorkspace loads `.sln` → resolves all `.csproj` references → full cross-project type resolution
- **EF Migrations:** Parse migration files to reconstruct schema evolution history
- **Source Generators:** Handle generated code (mark as `source: "generated"`)
- **Blazor components:** Parse `.razor` files — extract `@inject` dependencies, `@code` blocks, component references
- **Minimal APIs:** Parse `MapGet`, `MapPost` patterns in `Program.cs`
- **Auto-properties and records:** Extract underlying field semantics

### 6.7 Java Parser

**Parsing Strategy:** Use tree-sitter-java for AST extraction (fast, Go-native via CGo bindings), supplemented with custom semantic analysis for framework-specific patterns.

**Extracted Symbols:**
- Packages, Classes, Interfaces, Enums, Records, Annotations
- Methods, Fields, Constructors
- Spring Controllers, Services, Repositories (detected via annotations)
- JPA/Hibernate entity mappings
- JDBC/MyBatis/jOOQ query patterns

**Extracted Relationships:**
- `INHERITS`, `IMPLEMENTS` — class hierarchy
- `CALLS` — method invocations (best-effort without full classpath resolution; use import statements to resolve)
- `IMPORTS` — import statements
- `READS_TABLE` / `WRITES_TABLE` — via:
  - JPA `@Entity` + `@Table(name="...")` mappings
  - JPA `@Query("SELECT...")` annotations — sub-parse SQL
  - JDBC `PreparedStatement` / `Statement` SQL strings
  - MyBatis XML mapper files — parse SQL in `<select>`, `<insert>`, `<update>`, `<delete>` tags
  - jOOQ DSL patterns
  - Spring Data repository method name conventions (`findByEmailAndActive` → SELECT with WHERE email AND active)
- `ROUTES_TO` — Spring `@RequestMapping`, `@GetMapping`, `@PostMapping`, etc.
- `CONFIGURED_BY` — `@Value`, `@ConfigurationProperties` → `application.yml` / `application.properties` keys
- `DEPENDS_ON` — Maven `pom.xml` / Gradle `build.gradle` dependencies

**Special Handling:**
- **Multi-module Maven/Gradle projects:** Resolve inter-module dependencies from build files
- **Spring dependency injection:** `@Autowired`, `@Inject`, constructor injection — build dependency graph between Spring beans
- **Lombok:** Detect `@Data`, `@Getter`, `@Setter`, `@Builder` and synthesize the implied methods/fields
- **MyBatis XML parsing:** Dedicated XML parser for mapper files with dynamic SQL (`<if>`, `<choose>`, `<foreach>`) handling
- **`application.yml` parsing:** Resolve property placeholders (`${db.url}`)

---

## 7. Symbol Resolution Engine

### 7.1 Resolution Strategy

Symbol resolution runs after all files in a source are parsed, operating on the full project symbol table.

```
Phase 1: Build Symbol Table
  - Collect all Symbols from all parse results
  - Index by FQN, by short name, by file

Phase 2: Resolve Imports/Uses
  - For each file, resolve import/using/uses statements to actual Symbol FQNs
  - Build per-file "visible symbols" scope

Phase 3: Resolve References
  - For each RawReference, attempt resolution:
    1. Check file-local scope (same file symbols)
    2. Check imported scope (resolved imports)
    3. Check project-wide scope (FQN match)
    4. Check cross-source scope (other sources in same project)
    5. If unresolved: create placeholder Symbol with kind=UNKNOWN and flag for review

Phase 4: Confidence Scoring
  - Exact FQN match: 1.0
  - Unambiguous short name in imported scope: 0.95
  - Inferred from naming convention: 0.7
  - Dynamic SQL / string concatenation: 0.5
  - Heuristic pattern match: 0.3
```

### 7.2 Cross-Language Resolution

Critical for codebases where Delphi calls stored procedures, .NET reads views, Java services call .NET APIs, etc.

```
Resolution bridge rules:
  1. Application code (Delphi/.NET/Java/ASP) referencing SQL objects:
     - Extract table/procedure names from SQL strings
     - Resolve against DATABASE → SCHEMA → OBJECT hierarchy
     - Default schema mapping per source (configurable: "dbo", "public", etc.)

  2. Cross-service references:
     - Match API_ENDPOINT FQNs across sources
     - HTTP client calls matching API route patterns
     - Service discovery configuration parsing

  3. Shared database:
     - Multiple applications referencing same database
     - Unified table/column symbol set per database
     - Track which applications READ vs WRITE each table
```

---

## 8. Lineage Engine

### 8.1 Column-Level Lineage

The lineage engine traces data flow at the column level through the full stack:

```
Source Table Column
       ↓ (SELECT in VIEW)
View Column
       ↓ (SELECT in Stored Procedure)
Stored Procedure Output Parameter
       ↓ (ADO.NET SqlDataReader / EF mapping)
.NET DTO Property
       ↓ (API Controller serialization)
JSON Response Field
       ↓ (consumed by downstream service)
Downstream Table Column
```

### 8.2 Lineage Graph Construction

```go
type LineageNode struct {
    SymbolID    uuid.UUID
    SymbolFQN   string
    SymbolKind  SymbolKind
    SourceID    uuid.UUID
    Transform   *string  // e.g., "UPPER(first_name) + ' ' + last_name"
}

type LineageEdge struct {
    From       LineageNode
    To         LineageNode
    Derivation DerivationType  // DIRECT_COPY, TRANSFORM, AGGREGATE, FILTER, JOIN, CONDITIONAL
    Expression *string         // The transformation expression
}

type LineageChain struct {
    Roots    []LineageNode  // Ultimate source columns
    Leaves   []LineageNode  // Final consumer columns
    Edges    []LineageEdge
    Depth    int
}
```

**Derivation Types:**
- `DIRECT_COPY` — `SELECT column FROM table` (no transformation)
- `TRANSFORM` — `SELECT UPPER(column)`, `SELECT a + b AS combined`
- `AGGREGATE` — `SELECT SUM(amount)`, `SELECT COUNT(*)`
- `FILTER` — column used in WHERE/HAVING (affects which rows)
- `JOIN` — column used in JOIN condition
- `CONDITIONAL` — column used in CASE/IF logic

### 8.3 Lineage Queries (Neo4j Cypher)

```cypher
// Forward lineage: Where does this column's data flow TO?
MATCH path = (source:COLUMN {fqn: $fqn})-[:TRANSFORMS|PASSES_TO|RETURNS_FROM*1..10]->(downstream)
RETURN path

// Backward lineage: Where does this column's data come FROM?
MATCH path = (upstream)-[:TRANSFORMS|PASSES_TO|RETURNS_FROM*1..10]->(target:COLUMN {fqn: $fqn})
RETURN path

// Impact analysis: What breaks if I change this column?
MATCH path = (source:COLUMN {fqn: $fqn})<-[:READS_COLUMN|TRANSFORMS|JOINS_ON*1..10]-(dependent)
RETURN DISTINCT dependent, length(path) AS distance
ORDER BY distance
```

---

## 9. API Layer

### 9.1 REST API

**Base URL:** `/api/v1`

**Authentication:** JWT (issued by external IdP — Keycloak, Okta, Azure AD) validated by middleware. Service accounts use API keys with scoped permissions.

#### 9.1.1 Project Management

```
POST   /projects                          — Create project
GET    /projects                          — List projects (paginated, filtered by RBAC)
GET    /projects/{slug}                   — Get project details
PUT    /projects/{slug}                   — Update project
DELETE /projects/{slug}                   — Delete project and all associated data

GET    /projects/{slug}/stats             — Aggregated statistics (symbol counts, edge counts, coverage)
```

#### 9.1.2 Source Management

```
POST   /projects/{slug}/sources           — Add source (GitLab/S3 config)
GET    /projects/{slug}/sources           — List sources
GET    /sources/{id}                      — Get source details
PUT    /sources/{id}                      — Update source config
DELETE /sources/{id}                      — Remove source

POST   /sources/{id}/resync              — Trigger manual resync
POST   /projects/{slug}/upload           — Upload ZIP archive (multipart)

GET    /sources/{id}/sync-history        — List IndexRuns for this source
GET    /index-runs/{id}                  — Get IndexRun details + live status
GET    /index-runs/{id}/errors           — List parse errors for an IndexRun
```

#### 9.1.3 Symbol & Dependency API

```
GET    /projects/{slug}/symbols           — Search/list symbols
  Query params:
    q=<search term>             — Full-text search on name/FQN
    kind=TABLE,VIEW             — Filter by kind(s)
    source_id=<uuid>            — Filter by source
    file_path=<glob>            — Filter by file path pattern
    page=1&per_page=50          — Pagination
    sort=fqn|kind|name          — Sort

GET    /symbols/{id}                     — Get symbol details
GET    /symbols/{id}/references          — List all edges involving this symbol
  Query params:
    direction=inbound|outbound|both
    relationship=CALLS,READS_TABLE       — Filter by relationship type(s)
    depth=1                              — Traversal depth (1=direct only)

GET    /symbols/{id}/lineage             — Get lineage chain
  Query params:
    direction=forward|backward|both
    max_depth=10
    include_transforms=true              — Include transformation expressions

GET    /symbols/{id}/impact              — Impact analysis
  Query params:
    max_depth=5
    include_indirect=true
```

#### 9.1.4 Graph Query API

```
POST   /projects/{slug}/query/cypher     — Execute read-only Cypher query
  Body: { "query": "MATCH ...", "params": {} }
  (Restricted to read-only queries, query timeout enforced)

POST   /projects/{slug}/query/dependencies — High-level dependency query
  Body: {
    "from": { "kind": "TABLE", "name_pattern": "dbo.Customer*" },
    "to":   { "kind": "STORED_PROCEDURE" },
    "relationship": ["READS_TABLE", "WRITES_TABLE"],
    "max_depth": 3
  }
```

#### 9.1.5 Semantic Search API

```
POST   /projects/{slug}/search/semantic  — Vector similarity search
  Body: {
    "query": "customer email validation logic",
    "top_k": 20,
    "filters": {
      "kinds": ["METHOD", "STORED_PROCEDURE", "FUNCTION"],
      "sources": ["uuid1", "uuid2"]
    },
    "include_context": true  — Include surrounding code snippet
  }
  Response: [{ symbol: {...}, score: 0.92, context: "..." }]
```

#### 9.1.6 Webhook Receiver

```
POST   /api/v1/webhooks/gitlab/{source_id}  — GitLab push/MR webhooks
```

#### 9.1.7 Error Response Format (✅ Implemented)

All REST API errors use a structured format provided by `pkg/apierr`:

```json
{
  "error": {
    "code": "PROJECT_NOT_FOUND",
    "message": "project not found"
  }
}
```

**Common error codes:**

| Code | HTTP Status | Description |
|---|---|---|
| `INVALID_REQUEST_BODY` | 400 | Malformed or undecodable JSON body |
| `SLUG_REQUIRED` / `SLUG_INVALID` | 400 | Missing or invalid project slug |
| `NAME_REQUIRED` / `NAME_TOO_LONG` | 400 | Missing or overly long name field |
| `INVALID_SOURCE_TYPE` | 400 | Unrecognized source type |
| `PROJECT_NOT_FOUND` | 404 | No project with the given slug |
| `SOURCE_NOT_FOUND` | 404 | No source with the given ID |
| `INDEX_RUN_NOT_FOUND` | 404 | No index run with the given ID |
| `MISSING_AUTH_TOKEN` / `INVALID_AUTH_TOKEN` | 401 | Webhook authentication failure |
| `INTERNAL_ERROR` | 500 | Unexpected server error |
| `NOT_IMPLEMENTED` | 501 | Feature not yet available |
| `DATABASE_NOT_READY` | 503 | Health check failure |

GraphQL errors use the standard `errors` array with extension codes:

```json
{
  "errors": [{
    "message": "project not found",
    "path": ["project"],
    "extensions": { "code": "PROJECT_NOT_FOUND" }
  }]
}
```

See `pkg/apierr/code.go` for the full error catalog.

### 9.2 GraphQL API

Optional secondary API for the frontend — provides flexible querying for the graph visualization layer.

```graphql
type Query {
  project(slug: String!): Project
  symbol(id: ID!): Symbol
  searchSymbols(projectSlug: String!, query: String!, filters: SymbolFilters): SymbolConnection!
  lineage(symbolId: ID!, direction: LineageDirection!, maxDepth: Int): LineageGraph!
  impact(symbolId: ID!, maxDepth: Int): ImpactGraph!
}

type Symbol {
  id: ID!
  fqn: String!
  name: String!
  kind: SymbolKind!
  file: File!
  parent: Symbol
  children: [Symbol!]!
  location: SourceLocation!
  metadata: JSON
  documentation: String
  signature: String
  references(direction: Direction, relationship: [RelationshipType!]): [SymbolEdge!]!
  lineage(direction: LineageDirection!, maxDepth: Int): LineageGraph!
}

type SymbolEdge {
  id: ID!
  source: Symbol!
  target: Symbol!
  relationship: RelationshipType!
  location: SourceLocation
  confidence: Float!
  metadata: JSON
}

type LineageGraph {
  nodes: [LineageNode!]!
  edges: [LineageEdge!]!
}
```

### 9.3 WebSocket API

Real-time progress updates for indexing operations.

```
WS /api/v1/ws/index-progress/{index_run_id}

Messages (server → client):
{
  "type": "progress",
  "stage": "PARSING",
  "progress": 0.45,
  "current_file": "src/Data/CustomerRepository.cs",
  "stats": { "files_parsed": 234, "files_total": 520, "symbols_found": 4521 }
}

{
  "type": "stage_change",
  "from": "PARSING",
  "to": "RESOLVING"
}

{
  "type": "complete",
  "stats": { ... }
}

{
  "type": "error",
  "message": "Failed to parse ...",
  "fatal": false
}
```

---

## 10. MCP Server — Bedrock AgentCore Integration

### 10.1 Overview

The MCP (Model Context Protocol) server exposes CodeGraph's capabilities as tools that LLMs running on AWS Bedrock AgentCore can invoke. This enables natural language research queries against the codebase.

### 10.2 MCP Transport

```
Transport: Streamable HTTP (MCP spec 2025-11-25)
Endpoint:  /mcp/v1

Note: SSE transport was DEPRECATED in March 2025.
Streamable HTTP is the current MCP standard, required by Bedrock AgentCore.
Alternative: stdio transport for local development / testing
```

The MCP server runs as a separate Go binary (`codegraph-mcp`) deployed as a Kubernetes Deployment, registered as a tool provider in Bedrock AgentCore configuration.

### 10.3 Tool Definitions

#### 10.3.1 `search_symbols`

Search for code symbols by name, kind, or natural language description.

```json
{
  "name": "search_symbols",
  "description": "Search for code symbols (tables, procedures, classes, methods, etc.) across the indexed codebase. Use this to find specific code elements by name or to discover what exists. Supports fuzzy matching and semantic search.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project": {
        "type": "string",
        "description": "Project slug to search within"
      },
      "query": {
        "type": "string",
        "description": "Search query — can be a name like 'CustomerRepository', a pattern like 'dbo.Customer*', or a natural language description like 'email validation logic'"
      },
      "kinds": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Filter by symbol kinds: TABLE, VIEW, STORED_PROCEDURE, FUNCTION, CLASS, METHOD, COLUMN, API_ENDPOINT, etc."
      },
      "search_mode": {
        "type": "string",
        "enum": ["exact", "fuzzy", "semantic"],
        "description": "exact: FQN match. fuzzy: name similarity. semantic: natural language vector search."
      },
      "limit": {
        "type": "integer",
        "default": 20
      }
    },
    "required": ["project", "query"]
  }
}
```

#### 10.3.2 `get_symbol_details`

Get full details about a specific symbol including its source code, metadata, and documentation.

```json
{
  "name": "get_symbol_details",
  "description": "Retrieve detailed information about a specific code symbol, including its full source code, documentation, parameter list, type information, and file location. Use this after finding a symbol via search to examine it closely.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "symbol_id": {
        "type": "string",
        "description": "UUID of the symbol to examine"
      },
      "include_source": {
        "type": "boolean",
        "default": true,
        "description": "Include the source code of this symbol"
      },
      "include_children": {
        "type": "boolean",
        "default": true,
        "description": "Include child symbols (e.g., columns for a table, methods for a class)"
      }
    },
    "required": ["symbol_id"]
  }
}
```

#### 10.3.3 `get_dependencies`

Find what a symbol depends on or what depends on it.

```json
{
  "name": "get_dependencies",
  "description": "Find dependencies for a symbol. 'outbound' shows what this symbol depends on (calls, reads from, inherits). 'inbound' shows what depends on this symbol (callers, writers, subclasses). Use for understanding coupling and impact.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "symbol_id": {
        "type": "string",
        "description": "UUID of the symbol to analyze"
      },
      "direction": {
        "type": "string",
        "enum": ["inbound", "outbound", "both"],
        "description": "inbound: who depends on me. outbound: what do I depend on. both: all connections."
      },
      "relationship_types": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Filter by relationship types: CALLS, READS_TABLE, WRITES_TABLE, INHERITS, IMPLEMENTS, TRANSFORMS, etc. Omit for all."
      },
      "depth": {
        "type": "integer",
        "default": 1,
        "description": "Traversal depth. 1 = direct dependencies only. Higher values follow chains."
      },
      "limit": {
        "type": "integer",
        "default": 50
      }
    },
    "required": ["symbol_id", "direction"]
  }
}
```

#### 10.3.4 `trace_lineage`

Trace data lineage for a specific column or field.

```json
{
  "name": "trace_lineage",
  "description": "Trace the data lineage of a column or field — where does this data originate from (backward) or where does it flow to (forward). Returns a chain of transformations showing how data moves through views, procedures, application code, and APIs.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "symbol_id": {
        "type": "string",
        "description": "UUID of the column, field, or property to trace"
      },
      "direction": {
        "type": "string",
        "enum": ["forward", "backward", "both"],
        "description": "forward: where does this data go. backward: where does this data come from."
      },
      "max_depth": {
        "type": "integer",
        "default": 10,
        "description": "Maximum chain length to follow"
      },
      "include_transforms": {
        "type": "boolean",
        "default": true,
        "description": "Include transformation expressions at each step"
      }
    },
    "required": ["symbol_id", "direction"]
  }
}
```

#### 10.3.5 `analyze_impact`

Assess the impact of changing a specific symbol.

```json
{
  "name": "analyze_impact",
  "description": "Analyze the potential impact of modifying a symbol. Returns all directly and transitively affected symbols, grouped by severity (direct dependency vs. transitive). Useful for change planning, risk assessment, and migration analysis.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "symbol_id": {
        "type": "string",
        "description": "UUID of the symbol that would change"
      },
      "change_type": {
        "type": "string",
        "enum": ["modify", "delete", "rename"],
        "description": "Type of change being considered"
      },
      "max_depth": {
        "type": "integer",
        "default": 5
      }
    },
    "required": ["symbol_id", "change_type"]
  }
}
```

#### 10.3.6 `get_file_contents`

Read source code from an indexed file.

```json
{
  "name": "get_file_contents",
  "description": "Read the source code of an indexed file. Can retrieve the full file or a specific line range. Use this to examine code in context after finding symbols.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project": {
        "type": "string",
        "description": "Project slug"
      },
      "file_path": {
        "type": "string",
        "description": "Relative file path within the source"
      },
      "source_id": {
        "type": "string",
        "description": "Source UUID (required if file_path is ambiguous across sources)"
      },
      "start_line": {
        "type": "integer",
        "description": "Start line (1-indexed). Omit for full file."
      },
      "end_line": {
        "type": "integer",
        "description": "End line (inclusive). Omit for full file."
      }
    },
    "required": ["project", "file_path"]
  }
}
```

#### 10.3.7 `query_graph`

Execute a structured graph query (abstraction over Cypher).

```json
{
  "name": "query_graph",
  "description": "Execute a structured dependency graph query. Find paths between symbols, discover clusters, or run pattern-matching queries across the codebase graph. More powerful than simple dependency lookups for complex questions.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project": {
        "type": "string",
        "description": "Project slug"
      },
      "query_type": {
        "type": "string",
        "enum": ["shortest_path", "all_paths", "pattern_match", "neighbors", "cluster"],
        "description": "shortest_path: find shortest connection between two symbols. all_paths: all paths up to max_depth. pattern_match: find subgraph patterns. neighbors: N-hop neighborhood. cluster: find tightly-coupled symbol clusters."
      },
      "from_symbol": {
        "type": "string",
        "description": "Starting symbol ID or FQN"
      },
      "to_symbol": {
        "type": "string",
        "description": "Target symbol ID or FQN (for path queries)"
      },
      "relationship_filter": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Limit traversal to these relationship types"
      },
      "max_depth": {
        "type": "integer",
        "default": 5
      }
    },
    "required": ["project", "query_type"]
  }
}
```

#### 10.3.8 `list_project_overview`

Get a high-level overview of a project's structure and statistics.

```json
{
  "name": "list_project_overview",
  "description": "Get a structural overview of an indexed project — sources, languages, symbol counts by kind, top-level namespaces/schemas, and recent index status. Use this as a starting point to orient yourself in a codebase.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project": {
        "type": "string",
        "description": "Project slug"
      }
    },
    "required": ["project"]
  }
}
```

#### 10.3.9 `find_usages`

Find all usages of a symbol across the entire codebase.

```json
{
  "name": "find_usages",
  "description": "Find every location in the codebase where a symbol is referenced — all call sites, all queries that read/write a table, all classes that implement an interface, etc. Returns file locations with surrounding context.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "symbol_id": {
        "type": "string",
        "description": "UUID of the symbol to find usages of"
      },
      "usage_types": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Filter usage types: READ, WRITE, CALL, INHERIT, IMPLEMENT, REFERENCE. Omit for all."
      },
      "include_context": {
        "type": "boolean",
        "default": true,
        "description": "Include surrounding source code lines for each usage"
      },
      "context_lines": {
        "type": "integer",
        "default": 5,
        "description": "Number of lines of context above and below each usage"
      },
      "limit": {
        "type": "integer",
        "default": 50
      }
    },
    "required": ["symbol_id"]
  }
}
```

#### 10.3.10 `compare_snapshots`

Compare two index snapshots to find what changed.

```json
{
  "name": "compare_snapshots",
  "description": "Compare two index runs to see what changed — new symbols, removed symbols, modified signatures, changed dependencies. Useful for code review assistance and change tracking.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project": {
        "type": "string",
        "description": "Project slug"
      },
      "from_run_id": {
        "type": "string",
        "description": "Earlier IndexRun UUID"
      },
      "to_run_id": {
        "type": "string",
        "description": "Later IndexRun UUID"
      },
      "change_types": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Filter: ADDED, REMOVED, MODIFIED, MOVED. Omit for all."
      }
    },
    "required": ["project", "from_run_id", "to_run_id"]
  }
}
```

### 10.4 Bedrock AgentCore Registration

```yaml
# agentcore-tool-config.yaml
tool_provider:
  name: codegraph
  description: "Semantic codebase indexing engine. Search code, trace data lineage, analyze dependencies, and assess change impact across enterprise codebases spanning SQL Server, PostgreSQL, ASP, Delphi, .NET, and Java."
  endpoint:
    type: streamable-http
    url: "https://codegraph-mcp.internal.company.com/mcp/v1"
    auth:
      type: iam  # IAM-based auth for AgentCore → MCP server
  tools:
    - search_symbols
    - get_symbol_details
    - get_dependencies
    - trace_lineage
    - analyze_impact
    - get_file_contents
    - query_graph
    - list_project_overview
    - find_usages
    - compare_snapshots
```

### 10.5 MCP Response Formatting

All MCP tool responses follow a consistent structure optimized for LLM consumption:

```json
{
  "content": [
    {
      "type": "text",
      "text": "Found 3 symbols matching 'CustomerRepository':\n\n1. **CustomerRepository** (CLASS)\n   FQN: `MyApp.Data.Repositories.CustomerRepository`\n   File: `src/Data/Repositories/CustomerRepository.cs:15-89`\n   Methods: GetById, GetByEmail, Create, Update, Delete\n   Reads: dbo.Customers, dbo.CustomerAddresses\n   ID: `a1b2c3d4-...`\n\n2. ..."
    }
  ]
}
```

Responses are formatted in Markdown with code blocks for source code, making them natural for LLMs to reason about. Symbol IDs are always included to enable follow-up tool calls.

---

## 11. Frontend (React)

### 11.1 Technology Stack

| Library | Version | Purpose |
|---|---|---|
| React | 19 | Core framework |
| TypeScript | 5.9 | Type safety |
| Vite | 7.2 | Build tooling |
| TanStack Query | latest | Server state management |
| Zustand | v5 | Client state management (toast store) |
| React Router | v7 | Routing |
| Cytoscape.js | latest | Graph visualization |
| Monaco Editor | latest | Source code viewer with syntax highlighting |
| shadcn/ui + Tailwind | v4 | Component library + styling |
| Biome | 2.x | Linting + formatting |
| Vitest + Testing Library | latest | Testing |
| pnpm | latest | Package manager |

> **✅ Implemented error handling infrastructure (Phase 1):**
>
> - `ApiError` class (`frontend/src/api/errors.ts`) — typed errors with `code`, `status`, `isNotFound`, `isClientError`, `isServerError` getters
> - `ToastContainer` component — global notification system for mutation errors (auto-dismiss, color-coded by severity)
> - `ErrorState` component — inline query error display with optional retry button
> - Zustand toast store — manages toast lifecycle (add, auto-remove after 5s)

### 11.2 Page Architecture

```
/
├── /login                          — SSO redirect
├── /projects                       — Project list
│   ├── /projects/new               — Create project
│   └── /projects/:slug             — Project dashboard
│       ├── /sources                — Source management
│       │   ├── /sources/new        — Add source (GitLab/S3/Upload)
│       │   └── /sources/:id        — Source details + sync history
│       ├── /explorer               — Symbol browser (tree + search)
│       │   └── /explorer/:symbolId — Symbol detail view
│       ├── /graph                  — Interactive dependency graph
│       ├── /lineage                — Column lineage explorer
│       │   └── /lineage/:symbolId  — Lineage for specific column
│       ├── /impact                 — Impact analysis tool
│       ├── /search                 — Full-text + semantic search
│       ├── /index-runs             — Index run history + logs
│       └── /settings               — Project settings, RBAC
└── /admin                          — System administration
    ├── /users                      — User management
    └── /system                     — System health, queue status
```

### 11.3 Key UI Components

#### 11.3.1 Dependency Graph Viewer

Interactive force-directed or hierarchical graph rendered with Cytoscape.js.

**Features:**
- Pan, zoom, search-to-focus
- Color-coded nodes by symbol kind (tables=blue, procedures=green, classes=orange, etc.)
- Edge labels showing relationship type
- Click node to expand neighborhood (lazy-load adjacent nodes)
- Right-click context menu: "Show Lineage", "Impact Analysis", "View Source", "Find Usages"
- Filter panel: show/hide symbol kinds, relationship types, sources
- Layout modes: force-directed, hierarchical (top-down), circular, dagre
- Export: PNG, SVG, DOT format

#### 11.3.2 Lineage Explorer

Sankey-style or left-to-right DAG visualization for data lineage.

**Features:**
- Select a column/field as the starting point
- Toggle forward/backward/both directions
- Each node shows: symbol name, kind, source, transformation expression
- Highlight path on hover
- Expand/collapse intermediate steps
- Filter by source/language
- Side panel: show full details for selected node

#### 11.3.3 Impact Analysis View

Tree/list view showing blast radius of a proposed change.

**Features:**
- Select symbol + change type (modify/delete/rename)
- Tiered results: Direct dependents → Transitive dependents
- Severity indicators: high (direct callers/readers), medium (2-hop), low (3+ hops)
- Group by source/language/kind
- Export as CSV/JSON for ticketing

#### 11.3.4 Symbol Explorer

Tree + detail panel for browsing the symbol hierarchy.

**Features:**
- Left panel: tree view organized by Source → File → Symbol hierarchy
- Filtering by kind, language, search
- Right panel: symbol details, source code (Monaco with syntax highlighting), metadata, documentation
- Breadcrumb navigation via FQN
- Quick actions: "Find Usages", "Show Dependencies", "Trace Lineage"

#### 11.3.5 Source Code Viewer

Monaco Editor integration for reading indexed source code.

**Features:**
- Syntax highlighting for all supported languages
- Click-to-navigate: click on a symbol name to jump to its definition
- Gutter annotations: show dependency/usage counts per line
- Highlight all references to selected symbol
- Inline lineage markers: show where data flows in/out

#### 11.3.6 Index Dashboard

Real-time monitoring of indexing operations.

**Features:**
- Active index runs with live progress bars (WebSocket)
- Per-stage progress (CLONING → PARSING → RESOLVING → GRAPH_BUILDING → EMBEDDING)
- Current file being parsed
- Error log with file/line links
- Historical run list with stats (files, symbols, duration, error rate)
- Trigger manual resync button

---

## 12. Security

### 12.1 Authentication & Authorization

```
Authentication:
  - OIDC/SAML via external IdP (Keycloak, Azure AD, Okta)
  - JWT tokens validated by API middleware
  - API keys for service accounts (MCP server, CI/CD)
  - mTLS for internal service communication

Authorization (RBAC):
  Roles:
    - admin:       Full access, user management, system config
    - project_owner: Full access to owned projects
    - editor:      Read + trigger resyncs + manage sources
    - viewer:      Read-only access to indexed data
    - mcp_service: API access scoped to MCP tool operations

  Permissions are scoped per-project:
    User → Role → Project(s)
```

### 12.2 Secrets Management

```
PATs (GitLab):
  - Stored as Kubernetes Secrets or HashiCorp Vault entries
  - Referenced by path in Source.config.pat_secret_ref
  - Never stored in PostgreSQL
  - Rotated on schedule (configurable, alert on expiry)

S3 Credentials:
  - Prefer IRSA (IAM Roles for Service Accounts) for EKS
  - Fallback: Kubernetes Secrets

Database Credentials:
  - Kubernetes Secrets or Vault dynamic credentials
  - Connection pooling via PgBouncer

Webhook Secrets:
  - Generated per-source, stored in Source.config (hashed in DB)
  - Validated on every incoming webhook
```

### 12.3 Network Security

```
External-facing:
  - API server behind Ingress with TLS termination
  - Rate limiting: 100 req/s per API key, 1000 req/s global
  - CORS: configurable allowed origins
  - CSP headers for frontend

Internal:
  - Pod-to-pod communication via ClusterIP services
  - Network policies restricting inter-namespace traffic
  - Neo4j, PostgreSQL, Valkey not exposed externally

Data at rest:
  - PostgreSQL: encrypted tablespace (or EBS encryption)
  - S3/MinIO: server-side encryption (SSE-S3 or SSE-KMS)
  - Valkey: AUTH required, optional TLS

Data in transit:
  - TLS 1.3 for all external connections
  - mTLS for internal gRPC/HTTP between services
```

### 12.4 Audit Logging

```
Logged events:
  - Authentication (login, logout, token refresh, failed attempts)
  - Authorization (permission denied)
  - Source management (create, update, delete, resync trigger)
  - MCP tool invocations (tool name, project, query parameters, response size)
  - Data export (search results, lineage exports)
  - Admin actions (user management, system config changes)

Format: structured JSON → shipped to centralized logging (ELK, CloudWatch, Splunk)
Retention: configurable, default 90 days
```

---

## 13. Deployment

### 13.1 Kubernetes Architecture

```yaml
# Namespace: codegraph
Services:
  codegraph-api:
    replicas: 3 (HPA: 3-10 based on CPU/request rate)
    resources: { requests: { cpu: 500m, memory: 1Gi }, limits: { cpu: 2, memory: 4Gi } }
    ports: 8080 (HTTP), 8443 (gRPC)

  codegraph-mcp:
    replicas: 2 (HPA: 2-6)
    resources: { requests: { cpu: 500m, memory: 1Gi }, limits: { cpu: 2, memory: 4Gi } }
    ports: 9090 (Streamable HTTP)

  codegraph-worker:
    replicas: 5 (HPA: 5-20 based on queue depth)
    resources: { requests: { cpu: 2, memory: 4Gi }, limits: { cpu: 4, memory: 8Gi } }
    # Workers need more CPU/memory for parsing large codebases

  codegraph-scheduler:
    replicas: 1 (leader election)
    resources: { requests: { cpu: 100m, memory: 256Mi } }
    # Cron-based resync scheduler

  codegraph-frontend:
    replicas: 2
    resources: { requests: { cpu: 100m, memory: 128Mi } }
    # Nginx serving static React build

StatefulSets:
  postgresql:
    replicas: 3 (primary + 2 replicas)
    storage: 500Gi SSD (adjust for codebase size)
    # Or use managed PostgreSQL (RDS, CloudNativePG)

  neo4j:
    replicas: 3 (core cluster)
    storage: 200Gi SSD
    # Neo4j Enterprise for clustering, or Community for single-node

  valkey:
    replicas: 3 (Sentinel or Cluster)
    storage: 50Gi

  minio:  # If not using cloud S3
    replicas: 4 (erasure coding)
    storage: 1Ti per node
```

### 13.2 Helm Chart Structure

```
codegraph-helm/
├── Chart.yaml
├── values.yaml                 — Default configuration
├── values-production.yaml      — Production overrides
├── templates/
│   ├── api-deployment.yaml
│   ├── api-service.yaml
│   ├── api-ingress.yaml
│   ├── mcp-deployment.yaml
│   ├── mcp-service.yaml
│   ├── worker-deployment.yaml
│   ├── worker-hpa.yaml
│   ├── scheduler-deployment.yaml
│   ├── frontend-deployment.yaml
│   ├── frontend-service.yaml
│   ├── postgresql-statefulset.yaml  (or subchart)
│   ├── neo4j-statefulset.yaml       (or subchart)
│   ├── redis-statefulset.yaml       (or subchart)
│   ├── minio-statefulset.yaml       (or subchart)
│   ├── configmap.yaml
│   ├── secrets.yaml
│   ├── networkpolicy.yaml
│   ├── serviceaccount.yaml
│   ├── rbac.yaml
│   └── jobs/
│       ├── db-migrate.yaml          — Schema migration Job
│       └── neo4j-init.yaml          — Graph DB initialization
└── charts/                          — Subchart dependencies
```

### 13.3 Configuration

```yaml
# values.yaml (excerpt)
global:
  image:
    registry: registry.company.com
    tag: "1.0.0"

api:
  replicas: 3
  config:
    log_level: info
    cors_origins: ["https://codegraph.company.com"]
    jwt_issuer: "https://keycloak.company.com/realms/codegraph"
    jwt_audience: "codegraph-api"
    max_request_body_size: "2GB"

worker:
  replicas: 5
  config:
    concurrency: 4  # parallel files per worker
    parse_timeout: "5m"
    max_file_size: "50MB"
    dotnet_analyzer_image: "registry.company.com/codegraph-dotnet-analyzer:1.0.0"

mcp:
  replicas: 2
  config:
    max_query_depth: 10
    query_timeout: "30s"
    max_results_per_tool: 100

postgresql:
  storage: 500Gi
  resources:
    requests: { cpu: 2, memory: 8Gi }
  pgvector:
    dimensions: 1024

neo4j:
  edition: enterprise  # or community
  storage: 200Gi
  memory:
    heap: 4G
    pagecache: 4G

valkey:
  maxmemory: "4gb"
  maxmemory_policy: "allkeys-lru"

embedding:
  provider: bedrock
  model: "cohere.embed-v4"
  batch_size: 100
  dimensions: 1024
  rate_limit_rps: 50

ingestion:
  gitlab:
    clone_depth: 1  # shallow clone for initial speed
    max_repo_size: "10GB"
    webhook_timeout: "5s"
  s3:
    max_file_size: "50MB"
    polling_interval: "15m"  # fallback polling
  upload:
    max_size: "2GB"
    allowed_extensions: [".zip", ".tar.gz", ".tar.bz2"]
```

---

## 14. Observability

### 14.1 Metrics (Prometheus)

```
# Ingestion metrics
codegraph_index_runs_total{trigger, status}
codegraph_index_run_duration_seconds{source_type, stage}
codegraph_files_parsed_total{language, status}
codegraph_symbols_extracted_total{kind, language}
codegraph_parse_errors_total{language, error_type}
codegraph_queue_depth{queue}
codegraph_queue_processing_time_seconds{queue}

# API metrics
codegraph_api_requests_total{method, path, status}
codegraph_api_request_duration_seconds{method, path}
codegraph_api_request_size_bytes{method, path}

# MCP metrics
codegraph_mcp_tool_invocations_total{tool, project}
codegraph_mcp_tool_duration_seconds{tool}
codegraph_mcp_tool_result_size_bytes{tool}
codegraph_mcp_errors_total{tool, error_type}

# Storage metrics
codegraph_symbols_total{project, kind}
codegraph_edges_total{project, relationship}
codegraph_pg_connections_active
codegraph_neo4j_queries_total{type}
codegraph_neo4j_query_duration_seconds{type}
codegraph_embedding_requests_total{status}
codegraph_embedding_latency_seconds

# System metrics
codegraph_worker_utilization_ratio
codegraph_git_clone_duration_seconds
codegraph_s3_download_bytes_total
```

### 14.2 Logging

Structured JSON logging via `slog` (Go stdlib).

```json
{
  "timestamp": "2026-02-13T10:30:00Z",
  "level": "INFO",
  "service": "codegraph-worker",
  "message": "File parsed successfully",
  "index_run_id": "...",
  "source_id": "...",
  "file_path": "src/Data/CustomerRepo.cs",
  "language": "CSHARP",
  "symbols_found": 12,
  "duration_ms": 45,
  "trace_id": "abc123"
}
```

### 14.3 Distributed Tracing

OpenTelemetry integration (OTLP exporter → Jaeger/Tempo).

Trace spans:
- `index_run` (root span for entire indexing pipeline)
  - `git_clone`
  - `parse_file` (per file)
  - `resolve_references`
  - `build_graph`
  - `generate_embeddings`
- `api_request` (HTTP request lifecycle)
- `mcp_tool_call` (MCP tool invocation)
- `neo4j_query` (graph query)
- `embedding_batch` (Bedrock call)

### 14.4 Alerting Rules

```yaml
# Critical
- alert: IndexRunFailureRate
  expr: rate(codegraph_index_runs_total{status="FAILED"}[1h]) / rate(codegraph_index_runs_total[1h]) > 0.1
  for: 10m

- alert: QueueBacklog
  expr: codegraph_queue_depth{queue="ingest"} > 100
  for: 30m

- alert: Neo4jDown
  expr: up{job="neo4j"} == 0
  for: 2m

# Warning
- alert: HighParseErrorRate
  expr: rate(codegraph_parse_errors_total[1h]) / rate(codegraph_files_parsed_total[1h]) > 0.05
  for: 30m

- alert: SlowEmbedding
  expr: histogram_quantile(0.99, codegraph_embedding_latency_seconds) > 10
  for: 15m

- alert: PATExpiryWarning
  expr: codegraph_gitlab_pat_expiry_days < 14
  for: 1h
```

---

## 15. Incremental Indexing Strategy

### 15.1 Change Detection

```
Git sources:
  1. git diff --name-only {last_sha}..{new_sha} → changed files list
  2. For each changed file:
     a. If DELETED: remove all symbols from that file, recompute affected edges
     b. If MODIFIED: re-parse file, diff symbols (add/remove/update), recompute edges
     c. If ADDED: parse file, add symbols, resolve new references
  3. Re-run resolution only for symbols that changed or reference changed symbols
  4. Update graph store (incremental upsert/delete)
  5. Re-embed only changed/new symbols

S3 sources:
  1. Compare ETags / LastModified against stored file records
  2. Same file-level diff strategy as Git

ZIP uploads:
  1. Always full re-index (no incremental for uploads)
  2. Old snapshot retained for comparison via compare_snapshots tool
```

### 15.2 Cascade Invalidation

When a symbol changes, downstream symbols may need re-evaluation:

```
Symbol X changed (e.g., table column renamed):
  1. Find all edges where X is target
  2. For each upstream symbol:
     a. If edge.confidence < 1.0: re-validate reference
     b. If edge relationship is TRANSFORMS: recalculate lineage chain
  3. Mark affected symbols as "stale" until re-validated
  4. Background job: re-resolve stale symbols
```

### 15.3 Snapshot Management

```
Retention policy per project:
  - max_snapshots: 10 (keep last 10 full index runs per source)
  - max_age_days: 90

Snapshot = IndexRun + all associated symbols/edges

Snapshots enable:
  - compare_snapshots MCP tool
  - Rollback to previous index state
  - Historical lineage tracking (when did this dependency appear?)
```

---

## 16. Performance Targets

### 16.1 Indexing Performance

| Metric | Target |
|---|---|
| Files/second (parsing) | 100+ files/second per worker (simple files), 20+ files/second (complex stored procs) |
| Incremental re-index (100 changed files) | < 60 seconds end-to-end |
| Full re-index (1M LOC repo) | < 15 minutes |
| Full re-index (100M+ LOC, all sources) | < 4 hours with 20 workers |
| Embedding throughput | 1,000 symbols/minute (bottlenecked by Bedrock API rate) |

### 16.2 Query Performance

| Metric | Target |
|---|---|
| Symbol search (text) | < 100ms (p95) |
| Symbol search (semantic/vector) | < 200ms (p95) |
| Direct dependencies (depth=1) | < 50ms (p95) |
| Transitive dependencies (depth=5) | < 500ms (p95) |
| Column lineage (depth=10) | < 1s (p95) |
| Impact analysis (depth=5) | < 2s (p95) |
| MCP tool response (any tool) | < 3s (p95) |

### 16.3 Scale Targets

| Metric | Target |
|---|---|
| Total symbols indexed | 50M+ |
| Total edges | 200M+ |
| Concurrent index runs | 50+ |
| Concurrent API requests | 500+ |
| MCP tool calls/minute | 1,000+ |

---

## 17. Testing Strategy

### 17.1 Parser Testing

```
Per-language parser test suites:
  - Golden file tests: input source file → expected parse result (symbols + references)
  - Edge case tests: malformed SQL, deeply nested queries, encoding issues
  - Regression tests: real-world code samples from each supported language
  - Fuzz testing: generated source files to find parser crashes

Test corpus: curated set of representative code patterns per language
  - T-SQL: CTEs, dynamic SQL, cursors, MERGE, cross-database queries, linked servers
  - PostgreSQL: PL/pgSQL, dollar-quoting, inheritance, partitions, JSON operators
  - ASP Classic: include chains, ADO patterns, string-concatenated SQL
  - Delphi: DFM files, generic types, interface/implementation uses, conditional compilation
  - .NET: EF LINQ, Dapper, async/await, DI patterns, Blazor, Minimal API
  - Java: Spring Boot, JPA, MyBatis XML, Lombok, multi-module Maven
```

### 17.2 Integration Testing

```
Docker Compose test environment:
  - PostgreSQL + pgvector
  - Neo4j
  - Valkey
  - MinIO
  - Mock GitLab API (WireMock)
  - Mock Bedrock endpoint (for embedding)

Test scenarios:
  - End-to-end: Upload ZIP → index completes → query symbols → verify graph
  - Incremental: Full index → modify files → resync → verify diff
  - Cross-language: .NET app calling stored procedures → verify edges cross languages
  - Lineage: INSERT → VIEW → stored proc → .NET method → API endpoint → verify full chain
  - MCP: Invoke each MCP tool → verify response format and correctness
  - Webhook: Simulate GitLab push event → verify index triggered and completed
```

### 17.3 Performance Testing

```
k6 / Locust load tests:
  - API endpoint throughput under load
  - MCP tool response times under concurrent invocation
  - Indexing pipeline throughput with varying worker counts
  - Neo4j query performance with graph at scale (50M nodes, 200M edges)
```

---

## 18. Project Structure (Go Backend)

```
codegraph/
├── cmd/
│   ├── api/              — API server entrypoint
│   │   └── main.go
│   ├── worker/           — Indexing worker entrypoint
│   │   └── main.go
│   ├── mcp/              — MCP server entrypoint
│   │   └── main.go
│   └── scheduler/        — Cron scheduler entrypoint
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handler/      — HTTP handlers (project, source, symbol, webhook)
│   │   ├── middleware/    — Auth, logging, rate limiting, CORS
│   │   ├── graphql/      — GraphQL schema and resolvers (gqlgen)
│   │   └── websocket/    — WebSocket handlers (progress updates)
│   ├── mcp/
│   │   ├── server.go     — MCP server implementation (Streamable HTTP transport)
│   │   ├── tools/        — One file per MCP tool implementation
│   │   └── formatter.go  — Response formatting for LLM consumption
│   ├── ingestion/
│   │   ├── queue.go      — Valkey Streams consumer/producer
│   │   ├── pipeline.go   — Orchestrates ingestion stages
│   │   ├── connectors/
│   │   │   ├── gitlab.go
│   │   │   ├── s3.go
│   │   │   └── zip.go
│   │   └── scheduler.go  — Cron-based resync scheduling
│   ├── parser/
│   │   ├── parser.go     — Parser interface definition
│   │   ├── registry.go   — Parser registry (language → parser mapping)
│   │   ├── tsql/         — T-SQL parser
│   │   ├── pgsql/        — PostgreSQL parser (pg_query_go)
│   │   ├── asp/          — ASP Classic parser
│   │   ├── delphi/       — Delphi/Object Pascal parser
│   │   ├── dotnet/       — .NET parser (Roslyn sidecar orchestration)
│   │   ├── java/         — Java parser (tree-sitter)
│   │   └── common/       — Shared parsing utilities (SQL extraction, string interpolation)
│   ├── resolver/
│   │   ├── resolver.go   — Cross-file/cross-language symbol resolution
│   │   ├── scope.go      — Scope/import resolution
│   │   └── crosslang.go  — Cross-language bridge resolution
│   ├── lineage/
│   │   ├── engine.go     — Lineage computation engine
│   │   ├── column.go     — Column-level lineage extraction
│   │   └── chain.go      — Lineage chain construction
│   ├── graph/
│   │   ├── neo4j.go      — Neo4j client (read/write)
│   │   ├── queries.go    — Pre-built Cypher queries
│   │   └── sync.go       — PostgreSQL → Neo4j synchronization
│   ├── embedding/
│   │   ├── bedrock.go    — Bedrock Titan embedding client
│   │   ├── content.go    — Embedding content generation per symbol type
│   │   └── batch.go      — Batched embedding with rate limiting
│   ├── store/
│   │   ├── postgres/     — PostgreSQL repositories (SQLC generated)
│   │   │   ├── queries/  — SQL query files
│   │   │   ├── models.go — Generated models
│   │   │   └── db.go     — Generated query functions
│   │   └── valkey/        — Valkey client (cache, queue, pub/sub)
│   ├── auth/
│   │   ├── jwt.go        — JWT validation
│   │   ├── rbac.go       — Role-based access control
│   │   └── apikey.go     — API key management
│   └── config/
│       └── config.go     — Configuration loading (env, file, defaults)
├── pkg/
│   ├── apierr/          — Structured API error system (codes, catalog, wire format)
│   ├── models/           — Shared domain models (Symbol, Edge, etc.)
│   └── client/           — Go client library for CodeGraph API
├── tools/
│   └── dotnet-analyzer/  — .NET Roslyn sidecar project
│       ├── CodeGraph.Analyzer.sln
│       ├── src/
│       │   ├── Program.cs
│       │   ├── RoslynAnalyzer.cs
│       │   └── OutputFormatter.cs
│       └── Dockerfile
├── migrations/
│   ├── postgres/         — SQL migration files (golang-migrate)
│   └── neo4j/            — Cypher constraint/index setup scripts
├── deploy/
│   ├── helm/             — Helm chart (see §13.2)
│   ├── docker/
│   │   ├── Dockerfile.api
│   │   ├── Dockerfile.worker
│   │   ├── Dockerfile.mcp
│   │   └── Dockerfile.frontend
│   └── docker-compose.yml — Local development
├── frontend/             — React application
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── hooks/
│   │   ├── api/          — API client + error handling (ApiError, parseApiError)
│   │   └── stores/       — Zustand stores (toast notifications)
│   ├── package.json
│   └── vite.config.ts
├── test/
│   ├── testdata/         — Golden file test data per language
│   ├── integration/      — Integration test suites
│   └── fixtures/         — Docker Compose for test dependencies
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 19. Migration & Database Schema

### 19.1 PostgreSQL Migrations

> **Note:** The **authoritative schema** is in `migrations/postgres/000001_initial_schema.up.sql`. The SQL below is the original spec draft and has diverged from the implemented schema in several ways:
>
> - `projects.created_by` (nullable FK to `users`) — spec has `owner_id` (NOT NULL, no FK)
> - `sources.source_type` uses lowercase values (`git`, `database`, `filesystem`, `upload`) with `connection_uri` column — spec has uppercase `type` column (`GITLAB`, `S3`, `ZIP_UPLOAD`) with `status` and `last_commit_sha`
> - `index_runs` has flat integer columns (`files_processed`, `symbols_found`, `edges_found`) — spec has JSONB `stats` and a `trigger` column
> - `symbols.qualified_name` with `language` column, location as integer columns (`start_line`, `end_line`, `start_col`, `end_col`) — spec has `fqn` with JSONB `location`
> - `symbol_edges.source_id`/`target_id` — spec has `source_symbol`/`target_symbol` with `confidence` and `location` columns

```sql
-- 001_initial_schema.up.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- trigram index for fuzzy search

-- Projects
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT,
    owner_id    UUID NOT NULL,
    settings    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sources
CREATE TABLE sources (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    type            TEXT NOT NULL CHECK (type IN ('GITLAB', 'S3', 'ZIP_UPLOAD')),
    name            TEXT NOT NULL,
    config          JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED', 'ERROR')),
    last_sync_at    TIMESTAMPTZ,
    last_commit_sha TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sources_project ON sources(project_id);

-- Index Runs
CREATE TABLE index_runs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id     UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    trigger       TEXT NOT NULL CHECK (trigger IN ('MANUAL', 'WEBHOOK', 'SCHEDULE', 'UPLOAD')),
    status        TEXT NOT NULL DEFAULT 'QUEUED',
    started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,
    commit_sha    TEXT,
    stats         JSONB NOT NULL DEFAULT '{}',
    error_message TEXT
);

CREATE INDEX idx_index_runs_source ON index_runs(source_id, started_at DESC);
CREATE INDEX idx_index_runs_project ON index_runs(project_id, started_at DESC);
CREATE INDEX idx_index_runs_status ON index_runs(status) WHERE status NOT IN ('COMPLETE', 'FAILED');

-- Files
CREATE TABLE files (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id     UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    index_run_id  UUID NOT NULL REFERENCES index_runs(id) ON DELETE CASCADE,
    path          TEXT NOT NULL,
    language      TEXT NOT NULL,
    size_bytes    INT NOT NULL,
    line_count    INT NOT NULL,
    hash_sha256   TEXT NOT NULL,
    parse_status  TEXT NOT NULL DEFAULT 'PARSED',
    parse_error   TEXT
);

CREATE INDEX idx_files_source ON files(source_id);
CREATE INDEX idx_files_run ON files(index_run_id);
CREATE INDEX idx_files_path ON files(source_id, path);

-- Symbols
CREATE TABLE symbols (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    file_id       UUID REFERENCES files(id) ON DELETE CASCADE,
    source_id     UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    index_run_id  UUID NOT NULL REFERENCES index_runs(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,
    fqn           TEXT NOT NULL,
    name          TEXT NOT NULL,
    parent_id     UUID REFERENCES symbols(id) ON DELETE SET NULL,
    location      JSONB,
    metadata      JSONB NOT NULL DEFAULT '{}',
    documentation TEXT,
    signature     TEXT
);

CREATE INDEX idx_symbols_project ON symbols(project_id);
CREATE INDEX idx_symbols_fqn ON symbols(project_id, fqn);
CREATE INDEX idx_symbols_name ON symbols(project_id, name);
CREATE INDEX idx_symbols_kind ON symbols(project_id, kind);
CREATE INDEX idx_symbols_source ON symbols(source_id);
CREATE INDEX idx_symbols_file ON symbols(file_id);
CREATE INDEX idx_symbols_parent ON symbols(parent_id);
CREATE INDEX idx_symbols_name_trgm ON symbols USING gin (name gin_trgm_ops);
CREATE INDEX idx_symbols_fqn_trgm ON symbols USING gin (fqn gin_trgm_ops);

-- Symbol Edges
CREATE TABLE symbol_edges (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_symbol   UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    target_symbol   UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    index_run_id    UUID NOT NULL REFERENCES index_runs(id) ON DELETE CASCADE,
    relationship    TEXT NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    location        JSONB,
    confidence      REAL NOT NULL DEFAULT 1.0
);

CREATE INDEX idx_edges_source ON symbol_edges(source_symbol);
CREATE INDEX idx_edges_target ON symbol_edges(target_symbol);
CREATE INDEX idx_edges_project ON symbol_edges(project_id);
CREATE INDEX idx_edges_relationship ON symbol_edges(project_id, relationship);
CREATE INDEX idx_edges_pair ON symbol_edges(source_symbol, target_symbol, relationship);

-- Symbol Embeddings (pgvector)
CREATE TABLE symbol_embeddings (
    symbol_id     UUID PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE,
    embedding     vector(1024) NOT NULL,
    content_hash  TEXT NOT NULL,
    model_version TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_embeddings_vector ON symbol_embeddings USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

-- RBAC
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id TEXT NOT NULL UNIQUE,  -- from IdP
    email       TEXT NOT NULL,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE project_members (
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, user_id)
);

CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    key_hash    TEXT NOT NULL UNIQUE,
    scopes      TEXT[] NOT NULL DEFAULT '{}',
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 19.2 Neo4j Constraints & Indexes

```cypher
// Unique constraints
CREATE CONSTRAINT symbol_id IF NOT EXISTS FOR (s:Symbol) REQUIRE s.id IS UNIQUE;

// Composite indexes for common queries
CREATE INDEX symbol_project_fqn IF NOT EXISTS FOR (s:Symbol) ON (s.project_id, s.fqn);
CREATE INDEX symbol_project_kind IF NOT EXISTS FOR (s:Symbol) ON (s.project_id, s.kind);
CREATE INDEX symbol_name IF NOT EXISTS FOR (s:Symbol) ON (s.name);

// Full-text index for search
CREATE FULLTEXT INDEX symbol_search IF NOT EXISTS
FOR (s:Symbol)
ON EACH [s.name, s.fqn, s.documentation];
```

---

## 20. Development Roadmap

### Phase 1: Foundation — ✅ COMPLETE

- ✅ Project, Source, IndexRun CRUD APIs (REST + GraphQL mutations/queries)
- ✅ GitLab connector (PAT auth, shallow clone, webhook receiver)
- ✅ ZIP upload connector (MinIO storage, zip-slip protection)
- ✅ PostgreSQL schema + migrations (all tables incl. symbols, RBAC, embeddings)
- ✅ T-SQL parser (custom recursive-descent: tables, views, stored procedures, basic lineage)
- ✅ PostgreSQL parser (pg_query_go/v6)
- ✅ Centralized error handling (`pkg/apierr` — codes, catalog, structured wire format)
- ✅ Ingestion queue (Valkey Streams with consumer groups)
- ✅ React frontend (project management, sources, index runs, upload, error states, toasts)
- ✅ Docker Compose dev environment (PostgreSQL, Neo4j, Valkey, MinIO)
- ⚠️ Ingestion pipeline stages (framework exists, stages are no-ops pending parser wiring)
- ❌ GraphQL symbol/lineage queries (schema defined, resolvers return `NOT_IMPLEMENTED`)

### Phase 2: Core Parsers & Graph (Weeks 7–12)

- Symbol search API + wiring
- Neo4j integration + graph sync
- Embeddings pipeline (Cohere Embed v4 on Bedrock)
- Parse stage wiring (connect parsers to ingestion pipeline)
- Cross-file symbol resolution engine
- .NET parser (Roslyn sidecar — classes, methods, EF mappings)
- Java parser (tree-sitter — classes, methods, Spring/JPA)
- ASP Classic parser (includes, embedded SQL)
- Delphi parser (units, DFM files, component-SQL extraction)
- Dependency graph API endpoints
- Graph visualization (Cytoscape.js)

### Phase 3: Lineage & Graph (Weeks 13–18)

- Column-level lineage engine
- Cross-language resolution bridges
- Neo4j Cypher query API
- Graph visualization (Cytoscape.js)
- Lineage explorer UI
- Impact analysis engine + UI
- S3 connector
- Incremental indexing for Git sources

### Phase 4: MCP & Intelligence (Weeks 19–24)

- MCP server implementation (all 10 tools)
- Bedrock AgentCore registration + testing
- Vector embeddings (Bedrock Titan)
- Semantic search API + UI
- Snapshot comparison tool
- WebSocket progress updates
- Advanced graph queries (shortest path, clustering)

### Phase 5: Production Hardening (Weeks 25–30)

- Helm chart + Kubernetes deployment
- RBAC implementation
- Audit logging
- Observability (metrics, tracing, dashboards)
- Performance optimization (query caching, batch processing)
- Load testing + scale validation
- Security hardening (mTLS, network policies, secret rotation)
- Documentation (API docs, user guide, operations runbook)

---

## 21. Open Questions & Decisions

| # | Question | Options | Status |
|---|---|---|---|
| 1 | Graph database choice | Neo4j Enterprise vs ArangoDB vs Dgraph | **DECIDED: Neo4j 2026.01** (calendar versioning, GQL conformance) |
| 2 | .NET parser approach | Roslyn sidecar vs OmniSharp vs tree-sitter-c-sharp | **DECIDED: Roslyn sidecar** — only option for full semantic analysis (type resolution, overload resolution) |
| 3 | Embedding model | Bedrock Titan v2 vs Cohere Embed v4 | **DECIDED: Cohere Embed v4** on Bedrock (multimodal, 1024 dims, 100+ languages) |
| 4 | T-SQL parser approach | Custom Go parser vs ANTLR grammar vs `sqlparser` lib | **DECIDED: Custom Go** — most control, T-SQL grammars in ANTLR are incomplete for our needs. ✅ Implemented in Phase 1. |
| 5 | PostgreSQL parser | `pg_query_go` vs tree-sitter-sql | **DECIDED: `pg_query_go`** — uses PostgreSQL's own parser, zero ambiguity. ✅ Implemented in Phase 1. |
| 6 | Frontend graph library | Cytoscape.js vs D3-force vs vis.js vs React Flow | **DECIDED: Cytoscape.js** — best combination of features, performance at scale, and React integration |
| 7 | Delphi DFM binary support | Shell out to Delphi `convert` tool vs custom binary parser | **DECIDED: Custom Go binary DFM parser** — avoid Delphi toolchain dependency |
| 8 | Queue system | Valkey Streams vs NATS JetStream vs RabbitMQ | **DECIDED: Valkey Streams** (Valkey 8.1, BSD-3, Linux Foundation). ✅ Implemented in Phase 1. |
| 9 | API framework (Go) | gin vs echo vs chi vs stdlib | **DECIDED: chi v5** (stdlib-compatible). ✅ Implemented in Phase 1. |
| 10 | SQL generation | SQLC vs sqlx vs GORM | **DECIDED: SQLC** — compile-time safety, no ORM overhead, excellent for complex queries. ✅ Implemented in Phase 1. |

---

## Appendix A: Glossary

| Term | Definition |
|---|---|
| **FQN** | Fully Qualified Name — unique identifier for a symbol (e.g., `MyApp.Data.Repositories.CustomerRepository.GetById`) |
| **Symbol** | Any named code entity: table, column, class, method, procedure, etc. |
| **Edge** | A directional relationship between two symbols |
| **Lineage** | The chain of data transformations from source to destination |
| **Impact** | The set of symbols affected by a change to a given symbol |
| **MCP** | Model Context Protocol — standard for exposing tools to LLMs |
| **PAT** | Personal Access Token (GitLab authentication) |
| **Resync** | Re-indexing a source to capture latest changes |
| **Snapshot** | A point-in-time capture of all symbols/edges from an IndexRun |
| **Sidecar** | A separate process spawned by the worker for language-specific analysis |

## Appendix B: Supported File Extensions

| Language | Extensions |
|---|---|
| T-SQL | `.sql` (with SQL Server dialect detection) |
| PostgreSQL | `.sql` (with PostgreSQL dialect detection), `.pgsql` |
| ASP Classic | `.asp`, `.asa`, `.inc` |
| Delphi | `.pas`, `.dpr`, `.dpk`, `.dfm`, `.inc` |
| C# | `.cs`, `.csx` |
| VB.NET | `.vb` |
| F# | `.fs`, `.fsx` |
| Java | `.java` |
| Config/Mapping | `.csproj`, `.vbproj`, `.fsproj`, `.sln`, `pom.xml`, `build.gradle`, `web.config`, `app.config`, `appsettings.json`, `appsettings.*.json`, `application.yml`, `application.properties`, `.mybatis.xml` |