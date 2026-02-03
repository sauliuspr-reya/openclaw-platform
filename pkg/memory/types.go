// Package memory provides hierarchical memory management for OpenClaw agents.
// Memory is organized in four tiers: Company, Group, Swarm, and Individual.
package memory

import (
	"context"
	"time"
)

// Tier represents the memory hierarchy level
type Tier string

const (
	TierCompany    Tier = "company"
	TierGroup      Tier = "group"
	TierSwarm      Tier = "swarm"
	TierIndividual Tier = "individual"
)

// BucketConfig defines NATS bucket naming conventions
type BucketConfig struct {
	// Company-level buckets
	CompanyKnowledge string // memory.company.knowledge
	CompanyTools     string // memory.company.tools
	CompanyPolicies  string // memory.company.policies

	// Group bucket pattern: memory.group.{name}.*
	GroupPrefix string // memory.group.

	// Swarm bucket pattern: memory.swarm.{id}.*
	SwarmPrefix string // memory.swarm.

	// Agent bucket pattern: memory.agent.{id}.*
	AgentPrefix string // memory.agent.
}

// DefaultBucketConfig returns the standard bucket naming configuration
func DefaultBucketConfig() *BucketConfig {
	return &BucketConfig{
		CompanyKnowledge: "memory.company.knowledge",
		CompanyTools:     "memory.company.tools",
		CompanyPolicies:  "memory.company.policies",
		GroupPrefix:      "memory.group.",
		SwarmPrefix:      "memory.swarm.",
		AgentPrefix:      "memory.agent.",
	}
}

// Entry represents a single memory entry
type Entry struct {
	Key       string            `json:"key"`
	Value     []byte            `json:"value"`
	Revision  uint64            `json:"revision"`
	Created   time.Time         `json:"created"`
	Modified  time.Time         `json:"modified"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	TTL       time.Duration     `json:"ttl,omitempty"`
}

// PromotionRequest represents a request to promote memory to a higher tier
type PromotionRequest struct {
	SourceTier   Tier              `json:"source_tier"`
	TargetTier   Tier              `json:"target_tier"`
	SourceBucket string            `json:"source_bucket"`
	TargetBucket string            `json:"target_bucket"`
	Key          string            `json:"key"`
	Reason       string            `json:"reason,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	RequestedBy  string            `json:"requested_by"` // agent ID
	RequestedAt  time.Time         `json:"requested_at"`
}

// AccessMode defines read/write permissions
type AccessMode string

const (
	AccessModeReadOnly  AccessMode = "readonly"
	AccessModeReadWrite AccessMode = "readwrite"
)

// TierAccess defines access configuration for a memory tier
type TierAccess struct {
	Tier     Tier       `json:"tier"`
	Enabled  bool       `json:"enabled"`
	Mode     AccessMode `json:"mode"`
	Bucket   string     `json:"bucket,omitempty"`   // specific bucket for individual/swarm
	Group    string     `json:"group,omitempty"`    // group name for group tier
	SwarmID  string     `json:"swarm_id,omitempty"` // swarm ID for swarm tier
}

// AgentMemoryConfig defines the complete memory configuration for an agent
type AgentMemoryConfig struct {
	NATSUrl    string       `json:"nats_url"`
	AgentID    string       `json:"agent_id"`
	Company    *TierAccess  `json:"company,omitempty"`
	Group      *TierAccess  `json:"group,omitempty"`
	Swarm      *TierAccess  `json:"swarm,omitempty"`
	Individual *TierAccess  `json:"individual,omitempty"`
}

// Store defines the interface for memory operations
type Store interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) (*Entry, error)
	
	// Put stores a value
	Put(ctx context.Context, key string, value []byte, opts ...PutOption) (*Entry, error)
	
	// Delete removes a key
	Delete(ctx context.Context, key string) error
	
	// List returns all keys matching a pattern
	List(ctx context.Context, pattern string) ([]string, error)
	
	// Watch monitors changes to keys matching a pattern
	Watch(ctx context.Context, pattern string) (<-chan WatchEvent, error)
	
	// History returns previous revisions of a key
	History(ctx context.Context, key string, limit int) ([]*Entry, error)
}

// PutOption configures a Put operation
type PutOption func(*putOptions)

type putOptions struct {
	ttl      time.Duration
	revision uint64 // for CAS operations
	metadata map[string]string
}

// WithTTL sets a time-to-live for the entry
func WithTTL(ttl time.Duration) PutOption {
	return func(o *putOptions) {
		o.ttl = ttl
	}
}

// WithRevision sets the expected revision for CAS operations
func WithRevision(rev uint64) PutOption {
	return func(o *putOptions) {
		o.revision = rev
	}
}

// WithMetadata adds metadata to the entry
func WithMetadata(meta map[string]string) PutOption {
	return func(o *putOptions) {
		o.metadata = meta
	}
}

// WatchEventType indicates the type of watch event
type WatchEventType string

const (
	WatchEventPut    WatchEventType = "put"
	WatchEventDelete WatchEventType = "delete"
)

// WatchEvent represents a change notification
type WatchEvent struct {
	Type  WatchEventType `json:"type"`
	Entry *Entry         `json:"entry,omitempty"`
	Key   string         `json:"key"`
}

// CoordinationSignal represents swarm coordination messages
type CoordinationSignal struct {
	Type      string            `json:"type"` // progress, blocker, complete, etc.
	AgentID   string            `json:"agent_id"`
	SwarmID   string            `json:"swarm_id"`
	ShardID   string            `json:"shard_id,omitempty"`
	Progress  float64           `json:"progress,omitempty"` // 0.0 - 1.0
	Message   string            `json:"message,omitempty"`
	Data      map[string]any    `json:"data,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ShardAssignment represents a work shard assigned to an agent
type ShardAssignment struct {
	SwarmID     string            `json:"swarm_id"`
	ShardID     string            `json:"shard_id"`
	AgentID     string            `json:"agent_id"`
	Description string            `json:"description"`
	Input       map[string]any    `json:"input,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	AssignedAt  time.Time         `json:"assigned_at"`
	Status      ShardStatus       `json:"status"`
}

// ShardStatus represents the status of a shard
type ShardStatus string

const (
	ShardStatusPending    ShardStatus = "pending"
	ShardStatusAssigned   ShardStatus = "assigned"
	ShardStatusInProgress ShardStatus = "in_progress"
	ShardStatusComplete   ShardStatus = "complete"
	ShardStatusFailed     ShardStatus = "failed"
)

// ShardResult represents the output of a completed shard
type ShardResult struct {
	SwarmID     string         `json:"swarm_id"`
	ShardID     string         `json:"shard_id"`
	AgentID     string         `json:"agent_id"`
	Output      map[string]any `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	CompletedAt time.Time      `json:"completed_at"`
}

// SwarmGoal represents the objective of a swarm
type SwarmGoal struct {
	SwarmID     string            `json:"swarm_id"`
	Description string            `json:"description"`
	Constraints map[string]string `json:"constraints,omitempty"`
	CreatedBy   string            `json:"created_by"` // orchestrator agent ID
	CreatedAt   time.Time         `json:"created_at"`
	Deadline    *time.Time        `json:"deadline,omitempty"`
}
