# MCP Tool Design — Context-Frugal Principles

CodeGraph's MCP tools are the primary interface for LLM agents investigating enterprise
codebases. Agents operate with finite context windows against codebases with millions of
symbols. Every design decision must respect this asymmetry.

## Table of Contents

1. [Design philosophy](#design-philosophy)
2. [Progressive disclosure](#progressive-disclosure)
3. [Token budget control](#token-budget-control)
4. [Session awareness](#session-awareness)
5. [Response ranking and pagination](#response-ranking-and-pagination)
6. [Traversal pruning](#traversal-pruning)
7. [Structural summaries](#structural-summaries)
8. [Response formatting](#response-formatting)
9. [Tool catalog](#tool-catalog)
10. [Implementation patterns](#implementation-patterns)

---

## Design Philosophy

Three principles govern MCP tool design:

1. **The agent controls context consumption.** Tools accept parameters that let agents
   manage how much data enters their context. Never force-feed large payloads.

2. **Summaries first, details on demand.** Default responses are compact. Agents drill
   deeper with follow-up calls using symbol IDs from the summary.

3. **The server knows the graph; the agent knows the question.** Push expensive filtering,
   ranking, and truncation to the server. The agent shouldn't receive 500 results to
   find the 5 that matter.

## Progressive Disclosure

Every tool that returns symbol data should support a `verbosity` parameter:

```
verbosity: "summary" | "standard" | "full"
```

**summary** — Compact card: FQN, kind, signature (one line), edge counts, symbol ID.
Ideal for search results, dependency lists, and initial exploration.

```
CustomerRepository (CLASS) — MyApp.Data.Repositories.CustomerRepository
  Methods: 5 | Reads: 2 tables | Callers: 12
  ID: a1b2c3d4-...
```

**standard** — Card plus key metadata: parameter lists, column types, documentation
snippet (first 2 sentences), direct dependency summary. Default for most tools.

**full** — Everything: complete source code, all metadata, full documentation,
all edges with metadata. Use sparingly — a single stored procedure at `full` verbosity
can consume thousands of tokens.

Implement this as a server-side projection. The database query is the same; only the
response serialization changes.

## Token Budget Control

Tools should accept an optional `max_response_tokens` parameter:

```json
{
  "symbol_id": "...",
  "direction": "outbound",
  "max_response_tokens": 2000
}
```

The server estimates token count during response construction and truncates intelligently:
- Prioritize higher-ranked items
- Include a `truncated: true` flag and `remaining_count` in the response
- Never cut mid-symbol — either include a symbol fully or omit it

Token estimation: rough heuristic of 4 chars per token is sufficient. Don't overthink this;
the goal is order-of-magnitude control, not precision.

## Session Awareness

Even though agents are per-request stateless, the MCP server can maintain lightweight
session state to improve responses across a multi-turn investigation.

**Implementation**: Use Valkey with a session key (passed as `session_id` parameter,
auto-generated if not provided). TTL of 30 minutes.

**What to track per session**:
- `seen_symbols`: Set of symbol IDs already returned to this agent
- `query_history`: Last N queries (for deduplication detection)
- `focus_area`: The symbols/files the agent has been investigating (for relevance boosting)

**How session state improves responses**:
- **Deduplication**: When returning dependency lists, collapse already-seen symbols to
  one-line references (`CustomerRepository (already examined) — ID: ...`)
- **Focus boosting**: Rank results closer to the agent's recent focus area higher
- **Breadcrumb trail**: The `list_project_overview` tool can include "you've been
  investigating: dbo.Customers → CustomerRepository → OrderService" context

Session is optional. Tools must work correctly without it. Session state is a quality
improvement, not a correctness requirement.

## Response Ranking and Pagination

All list-returning tools must:

1. **Rank results by relevance**, not alphabetically. Relevance signals:
   - Edge distance from query focus (closer = more relevant)
   - In-degree / out-degree (highly-connected symbols are often more important)
   - Symbol kind priority (tables and classes before individual columns and fields)
   - Recency of last index (fresher data ranked higher for duplicate symbols)

2. **Return counts with truncated results**:
   ```json
   {
     "results": [...],
     "total_count": 247,
     "returned_count": 20,
     "has_more": true
   }
   ```

3. **Support cursor-based pagination** via `offset` and `limit` parameters.

4. **Group when helpful**: Impact analysis should group by severity tier
   (direct → transitive 2-hop → transitive 3+). Dependency lists should
   group by relationship type.

## Traversal Pruning

Graph traversal tools (`trace_lineage`, `analyze_impact`, `get_dependencies`) must support
filters that prune the traversal to what the agent actually needs:

**`stop_at_kinds`**: Stop traversal when encountering these symbol kinds.
Example: "trace lineage backward, but stop when you hit a TABLE — I don't need intra-database
column-to-column lineage right now." This is the single most effective context reducer.

**`cross_language_only`**: Only return nodes where data crosses a language boundary
(e.g., .NET → T-SQL, Java → PostgreSQL). Collapses same-language chains to single edges.

**`relationship_filter`**: Only traverse edges of these types. Already in the spec —
make sure all traversal tools actually implement it.

**`exclude_kinds`**: Exclude these symbol kinds from results (but still traverse through them).
Example: "show me dependencies but exclude individual COLUMNs — just show TABLEs."

**`min_confidence`**: Only follow edges above this confidence threshold. Useful for
filtering out heuristic/inferred relationships.

## Structural Summaries

Pre-compute natural-language summaries at multiple granularities during the embedding phase:

**Schema-level**: "The `dbo` schema contains 142 tables centered around customer management
(Customers, CustomerAddresses, CustomerPreferences), order processing (Orders, OrderItems,
OrderStatus), and product catalog (Products, Categories, ProductVariants)."

**Namespace-level**: "MyApp.Data.Repositories contains 23 repository classes implementing
the repository pattern over Entity Framework. Primary data access layer for the customer
and order domains."

**Source-level**: "This source (main-api) is a .NET 8 Web API with 45 controllers,
12 services, and 23 repositories. It reads from 89 tables and writes to 34."

**Service boundary**: "The OrderService reads from dbo.Customers, dbo.Products, and
dbo.Inventory, and writes to dbo.Orders, dbo.OrderItems, and dbo.OrderAuditLog.
It calls the NotificationService and InventoryService."

Store these as metadata on the relevant container symbols (SCHEMA, NAMESPACE, Source records).
Generate them with an LLM during the embedding phase and cache them.

These summaries power the `list_project_overview` tool and give agents fast orientation
without loading thousands of symbols into context.

## Response Formatting

MCP tool responses are consumed by LLMs. Format for parseability:

- Use Markdown with code blocks for source code
- Include symbol IDs in every response (agents need them for follow-up)
- Use consistent structure: agents learn the format and parse more efficiently over time
- For graph data, prefer flat lists with indentation over nested JSON
- Include FQNs — they're the most information-dense identifier

Example response format for `search_symbols`:

```
Found 3 symbols matching 'CustomerRepository' (47 total, showing top 3):

1. **CustomerRepository** (CLASS)
   FQN: `MyApp.Data.Repositories.CustomerRepository`
   File: `src/Data/Repositories/CustomerRepository.cs:15-89`
   Methods: GetById, GetByEmail, Create, Update, Delete
   Reads: dbo.Customers, dbo.CustomerAddresses
   ID: `a1b2c3d4-...`

2. **ICustomerRepository** (INTERFACE)
   FQN: `MyApp.Data.Interfaces.ICustomerRepository`
   File: `src/Data/Interfaces/ICustomerRepository.cs:8-24`
   Implementors: 1
   ID: `e5f6a7b8-...`

3. **CustomerRepositoryTests** (CLASS)
   FQN: `MyApp.Tests.Data.CustomerRepositoryTests`
   File: `tests/Data/CustomerRepositoryTests.cs:12-156`
   Test methods: 8
   ID: `c9d0e1f2-...`
```

## Tool Catalog

Current tools (10 defined in spec, implement with context-frugal parameters):

| Tool | Purpose | Key frugal parameters |
|---|---|---|
| `search_symbols` | Find symbols by name/kind/description | `verbosity`, `limit`, `search_mode` |
| `get_symbol_details` | Deep-dive into one symbol | `include_source`, `include_children`, `verbosity` |
| `get_dependencies` | What depends on / is depended by | `depth`, `relationship_types`, `stop_at_kinds`, `verbosity`, `max_response_tokens` |
| `trace_lineage` | Column-level data flow | `direction`, `max_depth`, `stop_at_kinds`, `cross_language_only` |
| `analyze_impact` | Blast radius of a change | `change_type`, `max_depth`, `stop_at_kinds`, `severity_threshold` |
| `get_file_contents` | Read source code | `start_line`, `end_line` (always support ranges) |
| `query_graph` | Structured graph queries | `max_depth`, `relationship_filter`, `limit` |
| `list_project_overview` | Orient in a codebase | `verbosity` (summary vs detailed breakdown) |
| `find_usages` | All references to a symbol | `usage_types`, `include_context`, `context_lines`, `limit` |
| `compare_snapshots` | Diff between index runs | `change_types`, `limit` |

## Implementation Patterns

### Tool handler structure

Each tool lives in `internal/mcp/tools/{tool_name}.go`:

```go
type SearchSymbolsParams struct {
    Project           string   `json:"project"`
    Query             string   `json:"query"`
    Kinds             []string `json:"kinds,omitempty"`
    SearchMode        string   `json:"search_mode,omitempty"`
    Limit             int      `json:"limit,omitempty"`
    Verbosity         string   `json:"verbosity,omitempty"`
    MaxResponseTokens int      `json:"max_response_tokens,omitempty"`
    SessionID         string   `json:"session_id,omitempty"`
}

func (h *SearchSymbolsHandler) Handle(ctx context.Context, params SearchSymbolsParams) (*mcp.ToolResponse, error) {
    // 1. Validate + apply defaults
    // 2. Execute query (PG text search, pgvector, or FQN match)
    // 3. Rank results
    // 4. Apply session deduplication (if session_id provided)
    // 5. Project to requested verbosity
    // 6. Truncate to token budget
    // 7. Format as Markdown
    return &mcp.ToolResponse{Content: []mcp.Content{{Type: "text", Text: formatted}}}, nil
}
```

### Markdown response builder

Create a shared response builder in `internal/mcp/formatter.go`:

```go
type ResponseBuilder struct {
    buf           strings.Builder
    tokenEstimate int
    maxTokens     int
    truncated     bool
}

func (rb *ResponseBuilder) AddSymbolCard(s *Symbol, verbosity string) bool {
    // Returns false if adding this card would exceed token budget
}

func (rb *ResponseBuilder) Finalize(totalCount, returnedCount int) string {
    // Appends truncation notice if needed
}
```

### Session management

```go
// internal/mcp/session.go
type Session struct {
    ID           string
    SeenSymbols  map[uuid.UUID]bool
    QueryHistory []string
    FocusArea    []uuid.UUID
    TTL          time.Duration
}

func (s *Session) MarkSeen(ids ...uuid.UUID)
func (s *Session) IsSeen(id uuid.UUID) bool
func (s *Session) UpdateFocus(ids ...uuid.UUID)
```

Store in Valkey with key `mcp:session:{session_id}`, TTL 30 minutes.
