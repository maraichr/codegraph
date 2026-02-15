package oracle

import "encoding/json"

// Response is the structured response from an Oracle query.
type Response struct {
	SessionID string       `json:"session_id"`
	Tool      string       `json:"tool"`
	Blocks    []Block      `json:"blocks"`
	Hints     []Hint       `json:"hints"`
	Meta      ResponseMeta `json:"meta"`
}

// Block is a typed content block in an Oracle response.
type Block struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Hint is a suggested follow-up question.
type Hint struct {
	Label    string `json:"label"`
	Question string `json:"question"`
}

// ResponseMeta contains metadata about the Oracle response.
type ResponseMeta struct {
	ToolSelected string `json:"tool_selected"`
	TokensUsed   int    `json:"tokens_used,omitempty"`
	TotalResults int    `json:"total_results"`
	Shown        int    `json:"shown"`
}

// Block data types

// HeaderData is the payload for a "header" block.
type HeaderData struct {
	Text string `json:"text"`
}

// SymbolItem represents a symbol in a "symbol_list" block.
type SymbolItem struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	QualifiedName string  `json:"qualified_name"`
	Kind          string  `json:"kind"`
	Language      string  `json:"language"`
	Signature     *string `json:"signature,omitempty"`
	InDegree      int32   `json:"in_degree,omitempty"`
	PageRank      float64 `json:"pagerank,omitempty"`
}

// SymbolListData is the payload for a "symbol_list" block.
type SymbolListData struct {
	Symbols []SymbolItem `json:"symbols"`
}

// GraphNode is a node in a "graph" block.
type GraphNode struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Label string `json:"label,omitempty"`
}

// GraphEdge is an edge in a "graph" block.
type GraphEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	EdgeType string `json:"edge_type"`
}

// GraphData is the payload for a "graph" block.
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// TableData is the payload for a "table" block.
type TableData struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// TextData is the payload for a "text" block.
type TextData struct {
	Content string `json:"content"`
}

// TruncationData is the payload for a "truncation" block.
type TruncationData struct {
	Shown int `json:"shown"`
	Total int `json:"total"`
}

// Helper constructors

func headerBlock(text string) Block {
	data, _ := json.Marshal(HeaderData{Text: text})
	return Block{Type: "header", Data: data}
}

func symbolListBlock(symbols []SymbolItem) Block {
	data, _ := json.Marshal(SymbolListData{Symbols: symbols})
	return Block{Type: "symbol_list", Data: data}
}

func graphBlock(nodes []GraphNode, edges []GraphEdge) Block {
	data, _ := json.Marshal(GraphData{Nodes: nodes, Edges: edges})
	return Block{Type: "graph", Data: data}
}

func tableBlock(headers []string, rows [][]string) Block {
	data, _ := json.Marshal(TableData{Headers: headers, Rows: rows})
	return Block{Type: "table", Data: data}
}

func textBlock(content string) Block {
	data, _ := json.Marshal(TextData{Content: content})
	return Block{Type: "text", Data: data}
}

func truncationBlock(shown, total int) Block {
	data, _ := json.Marshal(TruncationData{Shown: shown, Total: total})
	return Block{Type: "truncation", Data: data}
}
