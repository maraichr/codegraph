// Neo4j initialization script for CodeGraph
// Run via: cat init.cypher | cypher-shell -u neo4j -p codegraph

// Constraints
CREATE CONSTRAINT symbol_id IF NOT EXISTS FOR (s:Symbol) REQUIRE s.id IS UNIQUE;
CREATE CONSTRAINT file_id IF NOT EXISTS FOR (f:File) REQUIRE f.id IS UNIQUE;
CREATE CONSTRAINT project_id IF NOT EXISTS FOR (p:Project) REQUIRE p.id IS UNIQUE;

// Indexes
CREATE INDEX symbol_name IF NOT EXISTS FOR (s:Symbol) ON (s.name);
CREATE INDEX symbol_kind IF NOT EXISTS FOR (s:Symbol) ON (s.kind);
CREATE INDEX symbol_language IF NOT EXISTS FOR (s:Symbol) ON (s.language);
CREATE INDEX symbol_qualified_name IF NOT EXISTS FOR (s:Symbol) ON (s.qualifiedName);
CREATE INDEX file_path IF NOT EXISTS FOR (f:File) ON (f.path);

// Full-text search indexes
CREATE FULLTEXT INDEX symbol_search IF NOT EXISTS FOR (s:Symbol) ON EACH [s.name, s.qualifiedName];
