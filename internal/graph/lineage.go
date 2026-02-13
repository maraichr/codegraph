package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// LineageNode represents a symbol node in the lineage graph.
type LineageNode struct {
	ID            string
	Name          string
	QualifiedName string
	Kind          string
	Language      string
	FileID        string
}

// LineageEdge represents a relationship in the lineage graph.
type LineageEdge struct {
	SourceID string
	TargetID string
	EdgeType string
}

// LineageResult contains the result of a lineage query.
type LineageResult struct {
	Nodes  []LineageNode
	Edges  []LineageEdge
	RootID string
}

// Lineage queries the Neo4j graph for upstream/downstream dependencies.
func (c *Client) Lineage(ctx context.Context, symbolID uuid.UUID, direction string, maxDepth int) (*LineageResult, error) {
	if maxDepth <= 0 || maxDepth > 10 {
		maxDepth = 3
	}

	session := c.Session(ctx)
	defer session.Close(ctx)

	var query string
	switch direction {
	case "upstream":
		query = fmt.Sprintf(LineageUpstream, maxDepth)
	case "downstream":
		query = fmt.Sprintf(LineageDownstream, maxDepth)
	default:
		query = fmt.Sprintf(LineageBoth, maxDepth, maxDepth)
	}

	result, err := neo4j.ExecuteRead(ctx, session, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{
			"symbolId": symbolID.String(),
		})
		if err != nil {
			return nil, err
		}

		nodeMap := make(map[string]LineageNode)
		var edges []LineageEdge

		for records.Next(ctx) {
			record := records.Record()
			pathVal, ok := record.Get("path")
			if !ok {
				continue
			}
			path, ok := pathVal.(dbtype.Path)
			if !ok {
				continue
			}

			// Extract nodes from path
			for _, node := range path.Nodes {
				id, _ := node.Props["id"].(string)
				if id == "" {
					continue
				}
				if _, exists := nodeMap[id]; exists {
					continue
				}
				name, _ := node.Props["name"].(string)
				qname, _ := node.Props["qualifiedName"].(string)
				kind, _ := node.Props["kind"].(string)
				lang, _ := node.Props["language"].(string)
				fileID, _ := node.Props["fileId"].(string)

				nodeMap[id] = LineageNode{
					ID:            id,
					Name:          name,
					QualifiedName: qname,
					Kind:          kind,
					Language:      lang,
					FileID:        fileID,
				}
			}

			// Extract relationships from path
			// Build element ID → symbol ID map
			elemToSymbol := make(map[string]string)
			for _, node := range path.Nodes {
				symID, _ := node.Props["id"].(string)
				elemToSymbol[node.ElementId] = symID
			}

			for _, rel := range path.Relationships {
				edgeType, _ := rel.Props["edgeType"].(string)
				if edgeType == "" {
					edgeType = rel.Type
				}

				startID := elemToSymbol[rel.StartElementId]
				endID := elemToSymbol[rel.EndElementId]

				if startID != "" && endID != "" {
					edges = append(edges, LineageEdge{
						SourceID: startID,
						TargetID: endID,
						EdgeType: edgeType,
					})
				}
			}
		}

		if err := records.Err(); err != nil {
			return nil, err
		}

		nodes := make([]LineageNode, 0, len(nodeMap))
		for _, n := range nodeMap {
			nodes = append(nodes, n)
		}

		return &LineageResult{
			Nodes:  nodes,
			Edges:  edges,
			RootID: symbolID.String(),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("lineage query: %w", err)
	}

	return result.(*LineageResult), nil
}

// ColumnLineageNode represents a column in the column lineage graph.
type ColumnLineageNode struct {
	ID            string
	Name          string
	QualifiedName string
	TableName     string
	Kind          string
}

// ColumnLineageEdge represents a column-level data flow relationship.
type ColumnLineageEdge struct {
	SourceID       string
	TargetID       string
	DerivationType string
	Expression     string
}

// ColumnLineageResult contains the result of a column-level lineage query.
type ColumnLineageResult struct {
	Nodes  []ColumnLineageNode
	Edges  []ColumnLineageEdge
	RootID string
}

// ColumnLineage queries Neo4j for column-level lineage via COLUMN_FLOW relationships.
func (c *Client) ColumnLineage(ctx context.Context, symbolID uuid.UUID, direction string, maxDepth int) (*ColumnLineageResult, error) {
	if maxDepth <= 0 || maxDepth > 10 {
		maxDepth = 5
	}

	session := c.Session(ctx)
	defer session.Close(ctx)

	var query string
	switch direction {
	case "upstream":
		query = fmt.Sprintf(ColumnLineageUpstream, maxDepth)
	case "downstream":
		query = fmt.Sprintf(ColumnLineageDownstream, maxDepth)
	default:
		query = fmt.Sprintf(ColumnLineageBoth, maxDepth, maxDepth)
	}

	result, err := neo4j.ExecuteRead(ctx, session, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{
			"symbolId": symbolID.String(),
		})
		if err != nil {
			return nil, err
		}

		nodeMap := make(map[string]ColumnLineageNode)
		var edges []ColumnLineageEdge

		for records.Next(ctx) {
			record := records.Record()
			pathVal, ok := record.Get("path")
			if !ok {
				continue
			}
			path, ok := pathVal.(dbtype.Path)
			if !ok {
				continue
			}

			for _, node := range path.Nodes {
				id, _ := node.Props["id"].(string)
				if id == "" {
					continue
				}
				if _, exists := nodeMap[id]; exists {
					continue
				}
				name, _ := node.Props["name"].(string)
				qname, _ := node.Props["qualifiedName"].(string)
				kind, _ := node.Props["kind"].(string)

				tableName := ""
				if parts := strings.SplitN(qname, ".", -1); len(parts) > 1 {
					tableName = strings.Join(parts[:len(parts)-1], ".")
				}

				nodeMap[id] = ColumnLineageNode{
					ID:            id,
					Name:          name,
					QualifiedName: qname,
					TableName:     tableName,
					Kind:          kind,
				}
			}

			elemToSymbol := make(map[string]string)
			for _, node := range path.Nodes {
				symID, _ := node.Props["id"].(string)
				elemToSymbol[node.ElementId] = symID
			}

			for _, rel := range path.Relationships {
				derivationType, _ := rel.Props["derivationType"].(string)
				expression, _ := rel.Props["expression"].(string)

				startID := elemToSymbol[rel.StartElementId]
				endID := elemToSymbol[rel.EndElementId]

				if startID != "" && endID != "" {
					edges = append(edges, ColumnLineageEdge{
						SourceID:       startID,
						TargetID:       endID,
						DerivationType: derivationType,
						Expression:     expression,
					})
				}
			}
		}

		if err := records.Err(); err != nil {
			return nil, err
		}

		nodes := make([]ColumnLineageNode, 0, len(nodeMap))
		for _, n := range nodeMap {
			nodes = append(nodes, n)
		}

		return &ColumnLineageResult{
			Nodes:  nodes,
			Edges:  edges,
			RootID: symbolID.String(),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("column lineage query: %w", err)
	}

	return result.(*ColumnLineageResult), nil
}
