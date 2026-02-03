// Package swarm provides swarm memory and coordination primitives
package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openclaw/openclaw-platform/pkg/memory"
	"github.com/openclaw/openclaw-platform/pkg/memory/client"
)

// Manager handles swarm memory and coordination
type Manager struct {
	client   *client.Client
	swarmID  string
	agentID  string
	role     Role
	store    memory.Store
	mu       sync.RWMutex
}

// Role defines the agent's role in the swarm
type Role string

const (
	RoleWorker       Role = "worker"
	RoleOrchestrator Role = "orchestrator"
)

// NewManager creates a new swarm memory manager
func NewManager(ctx context.Context, c *client.Client, swarmID, agentID string, role Role) (*Manager, error) {
	store, err := c.Swarm(ctx, swarmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swarm store: %w", err)
	}

	return &Manager{
		client:  c,
		swarmID: swarmID,
		agentID: agentID,
		role:    role,
		store:   store,
	}, nil
}

// Well-known swarm memory keys
const (
	KeyGoal        = "goal"
	KeyState       = "state"
	KeyShards      = "shards"
	KeyResults     = "results"
	KeyCoordination = "coordination"
	KeyContext     = "context"
)

// Goal represents the swarm's objective
type Goal struct {
	SwarmID     string            `json:"swarm_id"`
	Description string            `json:"description"`
	Constraints map[string]string `json:"constraints,omitempty"`
	Input       map[string]any    `json:"input,omitempty"`
	CreatedBy   string            `json:"created_by"`
	CreatedAt   time.Time         `json:"created_at"`
	Deadline    *time.Time        `json:"deadline,omitempty"`
}

// State represents the current swarm state
type State struct {
	Phase           Phase             `json:"phase"`
	TotalShards     int               `json:"total_shards"`
	CompletedShards int               `json:"completed_shards"`
	FailedShards    int               `json:"failed_shards"`
	Workers         []string          `json:"workers"`
	StartedAt       time.Time         `json:"started_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// Phase represents the swarm lifecycle phase
type Phase string

const (
	PhasePending    Phase = "pending"
	PhaseAssigning  Phase = "assigning"
	PhaseProcessing Phase = "processing"
	PhaseCollecting Phase = "collecting"
	PhaseMerging    Phase = "merging"
	PhaseCompleted  Phase = "completed"
	PhaseFailed     Phase = "failed"
)

// Shard represents a unit of work
type Shard struct {
	ID          string         `json:"id"`
	Description string         `json:"description"`
	Input       map[string]any `json:"input,omitempty"`
	AssignedTo  string         `json:"assigned_to,omitempty"`
	Status      ShardStatus    `json:"status"`
	Priority    int            `json:"priority"`
	Timeout     time.Duration  `json:"timeout,omitempty"`
	AssignedAt  *time.Time     `json:"assigned_at,omitempty"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Retries     int            `json:"retries"`
	MaxRetries  int            `json:"max_retries"`
}

// ShardStatus represents shard processing status
type ShardStatus string

const (
	ShardStatusPending    ShardStatus = "pending"
	ShardStatusAssigned   ShardStatus = "assigned"
	ShardStatusInProgress ShardStatus = "in_progress"
	ShardStatusCompleted  ShardStatus = "completed"
	ShardStatusFailed     ShardStatus = "failed"
	ShardStatusRetrying   ShardStatus = "retrying"
)

// Result represents a shard's output
type Result struct {
	ShardID     string         `json:"shard_id"`
	AgentID     string         `json:"agent_id"`
	Output      map[string]any `json:"output,omitempty"`
	Artifacts   []Artifact     `json:"artifacts,omitempty"`
	Error       string         `json:"error,omitempty"`
	Duration    time.Duration  `json:"duration"`
	CompletedAt time.Time      `json:"completed_at"`
}

// Artifact represents a produced artifact
type Artifact struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // file, data, reference
	Location string `json:"location"`
	Size     int64  `json:"size,omitempty"`
}

// Signal represents a coordination signal
type Signal struct {
	Type      SignalType     `json:"type"`
	AgentID   string         `json:"agent_id"`
	ShardID   string         `json:"shard_id,omitempty"`
	Message   string         `json:"message,omitempty"`
	Progress  float64        `json:"progress,omitempty"` // 0.0 - 1.0
	Data      map[string]any `json:"data,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// SignalType defines coordination signal types
type SignalType string

const (
	SignalTypeHeartbeat  SignalType = "heartbeat"
	SignalTypeProgress   SignalType = "progress"
	SignalTypeBlocker    SignalType = "blocker"
	SignalTypeDiscovery  SignalType = "discovery"  // Found something useful
	SignalTypeRequest    SignalType = "request"    // Requesting help/info
	SignalTypeResponse   SignalType = "response"   // Response to request
	SignalTypeComplete   SignalType = "complete"
	SignalTypeFailed     SignalType = "failed"
)

// SharedContext represents context shared across the swarm
type SharedContext struct {
	Files       []string          `json:"files,omitempty"`
	Decisions   []Decision        `json:"decisions,omitempty"`
	Discoveries []Discovery       `json:"discoveries,omitempty"`
	Constraints []string          `json:"constraints,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Decision represents a decision made during processing
type Decision struct {
	Topic       string    `json:"topic"`
	Decision    string    `json:"decision"`
	Rationale   string    `json:"rationale,omitempty"`
	MadeBy      string    `json:"made_by"`
	MadeAt      time.Time `json:"made_at"`
}

// Discovery represents something discovered during processing
type Discovery struct {
	Type        string         `json:"type"` // pattern, issue, opportunity, etc.
	Description string         `json:"description"`
	Details     map[string]any `json:"details,omitempty"`
	DiscoveredBy string        `json:"discovered_by"`
	DiscoveredAt time.Time     `json:"discovered_at"`
	Relevance   float64        `json:"relevance"` // 0.0 - 1.0
}

// === Orchestrator Methods ===

// InitializeSwarm initializes a new swarm (orchestrator only)
func (m *Manager) InitializeSwarm(ctx context.Context, goal *Goal, shards []*Shard) error {
	if m.role != RoleOrchestrator {
		return fmt.Errorf("only orchestrator can initialize swarm")
	}

	goal.SwarmID = m.swarmID
	goal.CreatedBy = m.agentID
	goal.CreatedAt = time.Now()

	// Store goal
	goalData, err := json.Marshal(goal)
	if err != nil {
		return fmt.Errorf("failed to marshal goal: %w", err)
	}
	if _, err := m.store.Put(ctx, KeyGoal, goalData); err != nil {
		return fmt.Errorf("failed to store goal: %w", err)
	}

	// Store initial state
	state := &State{
		Phase:       PhasePending,
		TotalShards: len(shards),
		Workers:     []string{},
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	if _, err := m.store.Put(ctx, KeyState, stateData); err != nil {
		return fmt.Errorf("failed to store state: %w", err)
	}

	// Store shards
	for _, shard := range shards {
		shard.Status = ShardStatusPending
		if err := m.storeShard(ctx, shard); err != nil {
			return fmt.Errorf("failed to store shard %s: %w", shard.ID, err)
		}
	}

	// Initialize shared context
	sharedCtx := &SharedContext{UpdatedAt: time.Now()}
	ctxData, err := json.Marshal(sharedCtx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}
	if _, err := m.store.Put(ctx, KeyContext, ctxData); err != nil {
		return fmt.Errorf("failed to store context: %w", err)
	}

	return nil
}

// AssignShard assigns a shard to a worker (orchestrator only)
func (m *Manager) AssignShard(ctx context.Context, shardID, workerID string) error {
	if m.role != RoleOrchestrator {
		return fmt.Errorf("only orchestrator can assign shards")
	}

	shard, err := m.GetShard(ctx, shardID)
	if err != nil {
		return err
	}

	now := time.Now()
	shard.AssignedTo = workerID
	shard.AssignedAt = &now
	shard.Status = ShardStatusAssigned

	return m.storeShard(ctx, shard)
}

// GetPendingShards returns shards awaiting assignment
func (m *Manager) GetPendingShards(ctx context.Context) ([]*Shard, error) {
	return m.getShardsByStatus(ctx, ShardStatusPending)
}

// CollectResults collects all completed results
func (m *Manager) CollectResults(ctx context.Context) ([]*Result, error) {
	keys, err := m.store.List(ctx, KeyResults+".*")
	if err != nil {
		return nil, err
	}

	var results []*Result
	for _, key := range keys {
		entry, err := m.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var result Result
		if err := json.Unmarshal(entry.Value, &result); err != nil {
			continue
		}
		results = append(results, &result)
	}

	return results, nil
}

// UpdatePhase updates the swarm phase (orchestrator only)
func (m *Manager) UpdatePhase(ctx context.Context, phase Phase) error {
	if m.role != RoleOrchestrator {
		return fmt.Errorf("only orchestrator can update phase")
	}

	state, err := m.GetState(ctx)
	if err != nil {
		return err
	}

	state.Phase = phase
	state.UpdatedAt = time.Now()

	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	_, err = m.store.Put(ctx, KeyState, stateData)
	return err
}

// === Worker Methods ===

// JoinSwarm joins the swarm as a worker
func (m *Manager) JoinSwarm(ctx context.Context) error {
	state, err := m.GetState(ctx)
	if err != nil {
		return err
	}

	// Add to workers list if not already present
	for _, w := range state.Workers {
		if w == m.agentID {
			return nil // Already joined
		}
	}

	state.Workers = append(state.Workers, m.agentID)
	state.UpdatedAt = time.Now()

	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	_, err = m.store.Put(ctx, KeyState, stateData)
	return err
}

// ClaimShard claims an assigned shard for processing
func (m *Manager) ClaimShard(ctx context.Context, shardID string) (*Shard, error) {
	shard, err := m.GetShard(ctx, shardID)
	if err != nil {
		return nil, err
	}

	if shard.AssignedTo != m.agentID {
		return nil, fmt.Errorf("shard %s is not assigned to this agent", shardID)
	}

	if shard.Status != ShardStatusAssigned {
		return nil, fmt.Errorf("shard %s is not in assigned status", shardID)
	}

	now := time.Now()
	shard.StartedAt = &now
	shard.Status = ShardStatusInProgress

	if err := m.storeShard(ctx, shard); err != nil {
		return nil, err
	}

	return shard, nil
}

// CompleteShard marks a shard as completed with result
func (m *Manager) CompleteShard(ctx context.Context, shardID string, result *Result) error {
	shard, err := m.GetShard(ctx, shardID)
	if err != nil {
		return err
	}

	now := time.Now()
	shard.CompletedAt = &now
	shard.Status = ShardStatusCompleted

	if err := m.storeShard(ctx, shard); err != nil {
		return err
	}

	// Store result
	result.ShardID = shardID
	result.AgentID = m.agentID
	result.CompletedAt = now
	if shard.StartedAt != nil {
		result.Duration = now.Sub(*shard.StartedAt)
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	key := fmt.Sprintf("%s.%s", KeyResults, shardID)
	_, err = m.store.Put(ctx, key, resultData)
	if err != nil {
		return err
	}

	// Update state
	return m.updateShardCounts(ctx)
}

// FailShard marks a shard as failed
func (m *Manager) FailShard(ctx context.Context, shardID string, errorMsg string) error {
	shard, err := m.GetShard(ctx, shardID)
	if err != nil {
		return err
	}

	shard.Retries++
	if shard.Retries < shard.MaxRetries {
		// Allow retry
		shard.Status = ShardStatusRetrying
		shard.AssignedTo = ""
		shard.AssignedAt = nil
		shard.StartedAt = nil
	} else {
		// Max retries exceeded
		now := time.Now()
		shard.CompletedAt = &now
		shard.Status = ShardStatusFailed

		// Store failure result
		result := &Result{
			ShardID:     shardID,
			AgentID:     m.agentID,
			Error:       errorMsg,
			CompletedAt: now,
		}
		if shard.StartedAt != nil {
			result.Duration = now.Sub(*shard.StartedAt)
		}

		resultData, _ := json.Marshal(result)
		key := fmt.Sprintf("%s.%s", KeyResults, shardID)
		m.store.Put(ctx, key, resultData)
	}

	if err := m.storeShard(ctx, shard); err != nil {
		return err
	}

	return m.updateShardCounts(ctx)
}

// === Shared Methods ===

// GetGoal retrieves the swarm goal
func (m *Manager) GetGoal(ctx context.Context) (*Goal, error) {
	entry, err := m.store.Get(ctx, KeyGoal)
	if err != nil {
		return nil, err
	}

	var goal Goal
	if err := json.Unmarshal(entry.Value, &goal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal goal: %w", err)
	}

	return &goal, nil
}

// GetState retrieves the current swarm state
func (m *Manager) GetState(ctx context.Context) (*State, error) {
	entry, err := m.store.Get(ctx, KeyState)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(entry.Value, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// GetShard retrieves a shard by ID
func (m *Manager) GetShard(ctx context.Context, shardID string) (*Shard, error) {
	key := fmt.Sprintf("%s.%s", KeyShards, shardID)
	entry, err := m.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var shard Shard
	if err := json.Unmarshal(entry.Value, &shard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal shard: %w", err)
	}

	return &shard, nil
}

// GetMyShards returns shards assigned to this agent
func (m *Manager) GetMyShards(ctx context.Context) ([]*Shard, error) {
	keys, err := m.store.List(ctx, KeyShards+".*")
	if err != nil {
		return nil, err
	}

	var shards []*Shard
	for _, key := range keys {
		entry, err := m.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var shard Shard
		if err := json.Unmarshal(entry.Value, &shard); err != nil {
			continue
		}

		if shard.AssignedTo == m.agentID {
			shards = append(shards, &shard)
		}
	}

	return shards, nil
}

// GetResult retrieves a result by shard ID
func (m *Manager) GetResult(ctx context.Context, shardID string) (*Result, error) {
	key := fmt.Sprintf("%s.%s", KeyResults, shardID)
	entry, err := m.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var result Result
	if err := json.Unmarshal(entry.Value, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// === Coordination Methods ===

// SendSignal sends a coordination signal
func (m *Manager) SendSignal(ctx context.Context, signal *Signal) error {
	signal.AgentID = m.agentID
	signal.Timestamp = time.Now()
	return m.client.PublishSignal(ctx, &memory.CoordinationSignal{
		Type:      string(signal.Type),
		AgentID:   signal.AgentID,
		SwarmID:   m.swarmID,
		ShardID:   signal.ShardID,
		Progress:  signal.Progress,
		Message:   signal.Message,
		Data:      signal.Data,
		Timestamp: signal.Timestamp,
	})
}

// WatchSignals watches for coordination signals
func (m *Manager) WatchSignals(ctx context.Context) (<-chan *Signal, error) {
	natsSignals, err := m.client.SubscribeSignals(ctx, m.swarmID)
	if err != nil {
		return nil, err
	}

	signals := make(chan *Signal, 100)
	go func() {
		defer close(signals)
		for ns := range natsSignals {
			signals <- &Signal{
				Type:      SignalType(ns.Type),
				AgentID:   ns.AgentID,
				ShardID:   ns.ShardID,
				Progress:  ns.Progress,
				Message:   ns.Message,
				Data:      ns.Data,
				Timestamp: ns.Timestamp,
			}
		}
	}()

	return signals, nil
}

// ReportProgress reports progress on current shard
func (m *Manager) ReportProgress(ctx context.Context, shardID string, progress float64, message string) error {
	return m.SendSignal(ctx, &Signal{
		Type:     SignalTypeProgress,
		ShardID:  shardID,
		Progress: progress,
		Message:  message,
	})
}

// ReportBlocker reports a blocker
func (m *Manager) ReportBlocker(ctx context.Context, shardID, message string, data map[string]any) error {
	return m.SendSignal(ctx, &Signal{
		Type:    SignalTypeBlocker,
		ShardID: shardID,
		Message: message,
		Data:    data,
	})
}

// ShareDiscovery shares a discovery with the swarm
func (m *Manager) ShareDiscovery(ctx context.Context, discovery *Discovery) error {
	discovery.DiscoveredBy = m.agentID
	discovery.DiscoveredAt = time.Now()

	// Add to shared context
	sharedCtx, err := m.GetSharedContext(ctx)
	if err != nil {
		sharedCtx = &SharedContext{}
	}

	sharedCtx.Discoveries = append(sharedCtx.Discoveries, *discovery)
	sharedCtx.UpdatedAt = time.Now()

	ctxData, err := json.Marshal(sharedCtx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	if _, err := m.store.Put(ctx, KeyContext, ctxData); err != nil {
		return err
	}

	// Also send as signal for immediate notification
	return m.SendSignal(ctx, &Signal{
		Type:    SignalTypeDiscovery,
		Message: discovery.Description,
		Data: map[string]any{
			"type":      discovery.Type,
			"details":   discovery.Details,
			"relevance": discovery.Relevance,
		},
	})
}

// RecordDecision records a decision in the shared context
func (m *Manager) RecordDecision(ctx context.Context, decision *Decision) error {
	decision.MadeBy = m.agentID
	decision.MadeAt = time.Now()

	sharedCtx, err := m.GetSharedContext(ctx)
	if err != nil {
		sharedCtx = &SharedContext{}
	}

	sharedCtx.Decisions = append(sharedCtx.Decisions, *decision)
	sharedCtx.UpdatedAt = time.Now()

	ctxData, err := json.Marshal(sharedCtx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	_, err = m.store.Put(ctx, KeyContext, ctxData)
	return err
}

// GetSharedContext retrieves the shared context
func (m *Manager) GetSharedContext(ctx context.Context) (*SharedContext, error) {
	entry, err := m.store.Get(ctx, KeyContext)
	if err != nil {
		return nil, err
	}

	var sharedCtx SharedContext
	if err := json.Unmarshal(entry.Value, &sharedCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	return &sharedCtx, nil
}

// === Internal Methods ===

func (m *Manager) storeShard(ctx context.Context, shard *Shard) error {
	data, err := json.Marshal(shard)
	if err != nil {
		return fmt.Errorf("failed to marshal shard: %w", err)
	}

	key := fmt.Sprintf("%s.%s", KeyShards, shard.ID)
	_, err = m.store.Put(ctx, key, data)
	return err
}

func (m *Manager) getShardsByStatus(ctx context.Context, status ShardStatus) ([]*Shard, error) {
	keys, err := m.store.List(ctx, KeyShards+".*")
	if err != nil {
		return nil, err
	}

	var shards []*Shard
	for _, key := range keys {
		entry, err := m.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var shard Shard
		if err := json.Unmarshal(entry.Value, &shard); err != nil {
			continue
		}

		if shard.Status == status {
			shards = append(shards, &shard)
		}
	}

	return shards, nil
}

func (m *Manager) updateShardCounts(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.GetState(ctx)
	if err != nil {
		return err
	}

	keys, err := m.store.List(ctx, KeyShards+".*")
	if err != nil {
		return err
	}

	completed := 0
	failed := 0
	for _, key := range keys {
		entry, err := m.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var shard Shard
		if err := json.Unmarshal(entry.Value, &shard); err != nil {
			continue
		}

		switch shard.Status {
		case ShardStatusCompleted:
			completed++
		case ShardStatusFailed:
			failed++
		}
	}

	state.CompletedShards = completed
	state.FailedShards = failed
	state.UpdatedAt = time.Now()

	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	_, err = m.store.Put(ctx, KeyState, stateData)
	return err
}
