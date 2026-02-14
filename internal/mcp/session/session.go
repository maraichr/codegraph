package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

const (
	sessionKeyPrefix = "mcp:session:"
	sessionTTL       = 30 * time.Minute
	maxQueryHistory  = 20
	maxFocusArea     = 10
	maxRecapTokens   = 500
)

// Session tracks agent state across MCP tool calls within an investigation.
// Stored in Valkey with a 30-minute TTL, keyed by mcp:session:{session_id}.
type Session struct {
	ID           string            `json:"id"`
	SeenSymbols  map[string]bool   `json:"seen_symbols"`
	QueryHistory []string          `json:"query_history"`
	FocusArea    []string          `json:"focus_area"`
	Waypoints    []Waypoint        `json:"waypoints"`
	Recap        []string          `json:"recap"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Waypoint is a agent-bookmarked symbol for later reference.
type Waypoint struct {
	SymbolID uuid.UUID `json:"symbol_id"`
	Label    string    `json:"label"`
	AddedAt  time.Time `json:"added_at"`
}

// Manager handles loading and saving sessions to Valkey.
type Manager struct {
	client valkey.Client
}

// NewManager creates a session manager backed by the given Valkey client.
func NewManager(client valkey.Client) *Manager {
	return &Manager{client: client}
}

// Load retrieves a session from Valkey. If the session doesn't exist, a new one is created.
func (m *Manager) Load(ctx context.Context, sessionID string) (*Session, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	key := sessionKeyPrefix + sessionID
	resp := m.client.Do(ctx, m.client.B().Get().Key(key).Build())
	data, err := resp.AsBytes()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return newSession(sessionID), nil
		}
		return nil, fmt.Errorf("load session %s: %w", sessionID, err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return newSession(sessionID), nil
	}
	return &s, nil
}

// Save persists a session to Valkey with a 30-minute TTL.
func (m *Manager) Save(ctx context.Context, s *Session) error {
	s.UpdatedAt = time.Now()
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	key := sessionKeyPrefix + s.ID
	resp := m.client.Do(ctx, m.client.B().Set().Key(key).Value(string(data)).Ex(sessionTTL).Build())
	if err := resp.Error(); err != nil {
		return fmt.Errorf("save session %s: %w", s.ID, err)
	}
	return nil
}

func newSession(id string) *Session {
	now := time.Now()
	return &Session{
		ID:          id,
		SeenSymbols: make(map[string]bool),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MarkSeen records that the given symbol IDs have been returned to the agent.
func (s *Session) MarkSeen(ids ...uuid.UUID) {
	if s.SeenSymbols == nil {
		s.SeenSymbols = make(map[string]bool)
	}
	for _, id := range ids {
		s.SeenSymbols[id.String()] = true
	}
}

// IsSeen returns true if the symbol was previously returned in this session.
func (s *Session) IsSeen(id uuid.UUID) bool {
	if s.SeenSymbols == nil {
		return false
	}
	return s.SeenSymbols[id.String()]
}

// SeenCount returns the number of symbols seen in this session.
func (s *Session) SeenCount() int {
	return len(s.SeenSymbols)
}

// AddQuery appends a query to history, keeping the last maxQueryHistory entries.
func (s *Session) AddQuery(query string) {
	s.QueryHistory = append(s.QueryHistory, query)
	if len(s.QueryHistory) > maxQueryHistory {
		s.QueryHistory = s.QueryHistory[len(s.QueryHistory)-maxQueryHistory:]
	}
}

// UpdateFocus adds symbol IDs to the focus area (most recently examined symbols).
func (s *Session) UpdateFocus(ids ...uuid.UUID) {
	for _, id := range ids {
		s.FocusArea = append(s.FocusArea, id.String())
	}
	if len(s.FocusArea) > maxFocusArea {
		s.FocusArea = s.FocusArea[len(s.FocusArea)-maxFocusArea:]
	}
}

// FocusAreaUUIDs returns the focus area as parsed UUIDs.
func (s *Session) FocusAreaUUIDs() []uuid.UUID {
	result := make([]uuid.UUID, 0, len(s.FocusArea))
	for _, idStr := range s.FocusArea {
		if id, err := uuid.Parse(idStr); err == nil {
			result = append(result, id)
		}
	}
	return result
}

// AddWaypoint bookmarks a symbol for later reference.
func (s *Session) AddWaypoint(symbolID uuid.UUID, label string) {
	s.Waypoints = append(s.Waypoints, Waypoint{
		SymbolID: symbolID,
		Label:    label,
		AddedAt:  time.Now(),
	})
}

// AddRecap appends a one-line investigation finding.
func (s *Session) AddRecap(finding string) {
	s.Recap = append(s.Recap, finding)
	for estimateTokens(s.Recap) > maxRecapTokens && len(s.Recap) > 1 {
		s.Recap = s.Recap[1:]
	}
}

// RecapText returns the investigation recap as a formatted string.
func (s *Session) RecapText() string {
	if len(s.Recap) == 0 {
		return ""
	}
	var b strings.Builder
	for i, r := range s.Recap {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%d. %s", i+1, r)
	}
	return b.String()
}

func estimateTokens(lines []string) int {
	total := 0
	for _, l := range lines {
		total += len(l) / 4
	}
	return total
}
