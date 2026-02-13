export interface Project {
  id: string;
  name: string;
  slug: string;
  description: string | null;
  settings: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Source {
  id: string;
  project_id: string;
  name: string;
  source_type: "git" | "database" | "filesystem" | "upload" | "s3";
  connection_uri: string | null;
  config: Record<string, unknown>;
  last_synced_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface IndexRun {
  id: string;
  project_id: string;
  source_id: string | null;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  started_at: string | null;
  completed_at: string | null;
  files_processed: number;
  symbols_found: number;
  edges_found: number;
  error_message: string | null;
  created_at: string;
}

export interface ListResponse<T> {
  total: number;
  [key: string]: T[] | number;
}

export interface ProjectListResponse {
  projects: Project[];
  total: number;
}

export interface SourceListResponse {
  sources: Source[];
  total: number;
}

export interface IndexRunListResponse {
  index_runs: IndexRun[];
  total: number;
}

export interface CreateProjectInput {
  name: string;
  slug: string;
  description?: string;
}

export interface CreateSourceInput {
  name: string;
  source_type: string;
  connection_uri?: string;
  config?: Record<string, unknown>;
}

export interface UploadResponse {
  source: Source;
  index_run: IndexRun;
  object: string;
}

// Phase 2 types

export interface Symbol {
  id: string;
  project_id: string;
  file_id: string;
  name: string;
  qualified_name: string;
  kind: string;
  language: string;
  start_line: number;
  end_line: number;
  start_col: number | null;
  end_col: number | null;
  signature: string | null;
  doc_comment: string | null;
  created_at: string;
  updated_at: string;
}

export interface SymbolEdge {
  id: string;
  project_id: string;
  source_id: string;
  target_id: string;
  edge_type: string;
  created_at: string;
}

export interface LineageGraph {
  Nodes: LineageNode[];
  Edges: LineageEdge[];
  RootID: string;
}

export interface LineageNode {
  ID: string;
  Name: string;
  QualifiedName: string;
  Kind: string;
  Language: string;
  FileID: string;
}

export interface LineageEdge {
  SourceID: string;
  TargetID: string;
  EdgeType: string;
}

export interface SymbolSearchResponse {
  symbols: Symbol[];
  count: number;
}

export interface SemanticSearchResult {
  symbol: Symbol;
  score: number;
  distance: number;
}

export interface SemanticSearchResponse {
  results: SemanticSearchResult[];
  count: number;
}

// Phase 3 types — Column Lineage

export interface ColumnLineageGraph {
  nodes: ColumnLineageNode[];
  edges: ColumnLineageEdge[];
  root_column_id: string;
}

export interface ColumnLineageNode {
  id: string;
  name: string;
  qualified_name: string;
  table_name: string;
  kind: string;
}

export interface ColumnLineageEdge {
  source_id: string;
  target_id: string;
  derivation_type: string;
  expression: string | null;
}

// Phase 3 types — Impact Analysis

export interface ImpactAnalysisResult {
  root: ImpactSymbol;
  change_type: string;
  direct_impact: ImpactNode[];
  transitive_impact: ImpactNode[];
  total_affected: number;
}

export interface ImpactSymbol {
  id: string;
  name: string;
  qualified_name: string;
  kind: string;
  language: string;
}

export interface ImpactNode {
  symbol: ImpactSymbol;
  depth: number;
  severity: "critical" | "high" | "medium" | "low";
  edge_type: string;
  path: string[];
}
