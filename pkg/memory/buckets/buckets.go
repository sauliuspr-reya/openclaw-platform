// Package buckets provides utilities for managing NATS KV buckets for OpenClaw memory.
package buckets

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/openclaw/openclaw-platform/pkg/memory"
)

// Manager handles bucket lifecycle operations
type Manager struct {
	js     jetstream.JetStream
	config *memory.BucketConfig
}

// NewManager creates a new bucket manager
func NewManager(js jetstream.JetStream, config *memory.BucketConfig) *Manager {
	if config == nil {
		config = memory.DefaultBucketConfig()
	}
	return &Manager{
		js:     js,
		config: config,
	}
}

// BucketSpec defines the configuration for a bucket
type BucketSpec struct {
	Name        string
	Description string
	History     uint8         // Number of revisions to keep
	TTL         time.Duration // Default TTL for entries (0 = no expiry)
	MaxBytes    int64         // Max storage size (0 = unlimited)
	Replicas    int           // Number of replicas (1-5)
}

// DefaultCompanyBucketSpec returns the default spec for company buckets
func DefaultCompanyBucketSpec(name string) *BucketSpec {
	return &BucketSpec{
		Name:        name,
		Description: "OpenClaw company-wide memory",
		History:     25,  // Keep more history for company-wide data
		TTL:         0,   // No expiry - permanent storage
		MaxBytes:    0,   // Unlimited
		Replicas:    3,   // High availability
	}
}

// DefaultGroupBucketSpec returns the default spec for group buckets
func DefaultGroupBucketSpec(groupName string) *BucketSpec {
	return &BucketSpec{
		Name:        memory.DefaultBucketConfig().GroupPrefix + groupName,
		Description: fmt.Sprintf("OpenClaw group memory: %s", groupName),
		History:     15,
		TTL:         0,   // No expiry by default
		MaxBytes:    0,
		Replicas:    3,
	}
}

// DefaultSwarmBucketSpec returns the default spec for swarm buckets
func DefaultSwarmBucketSpec(swarmID string) *BucketSpec {
	return &BucketSpec{
		Name:        memory.DefaultBucketConfig().SwarmPrefix + swarmID,
		Description: fmt.Sprintf("OpenClaw swarm memory: %s", swarmID),
		History:     5,                    // Less history needed for ephemeral swarms
		TTL:         24 * time.Hour,       // Auto-cleanup after 24 hours
		MaxBytes:    100 * 1024 * 1024,    // 100MB limit per swarm
		Replicas:    1,                    // Single replica for ephemeral data
	}
}

// DefaultAgentBucketSpec returns the default spec for agent buckets
func DefaultAgentBucketSpec(agentID string) *BucketSpec {
	return &BucketSpec{
		Name:        memory.DefaultBucketConfig().AgentPrefix + agentID,
		Description: fmt.Sprintf("OpenClaw agent memory: %s", agentID),
		History:     10,
		TTL:         7 * 24 * time.Hour,  // Auto-cleanup after 7 days of inactivity
		MaxBytes:    50 * 1024 * 1024,    // 50MB limit per agent
		Replicas:    1,
	}
}

// EnsureBucket creates a bucket if it doesn't exist, or updates it if it does
func (m *Manager) EnsureBucket(ctx context.Context, spec *BucketSpec) (jetstream.KeyValue, error) {
	config := jetstream.KeyValueConfig{
		Bucket:      spec.Name,
		Description: spec.Description,
		History:     spec.History,
		TTL:         spec.TTL,
		MaxBytes:    spec.MaxBytes,
		Replicas:    spec.Replicas,
	}

	// Try to get existing bucket
	kv, err := m.js.KeyValue(ctx, spec.Name)
	if err == nil {
		// Bucket exists, try to update
		// Note: NATS KV doesn't support all updates, some require recreation
		return kv, nil
	}

	// Create new bucket
	kv, err = m.js.CreateKeyValue(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket %s: %w", spec.Name, err)
	}

	return kv, nil
}

// DeleteBucket deletes a bucket
func (m *Manager) DeleteBucket(ctx context.Context, name string) error {
	return m.js.DeleteKeyValue(ctx, name)
}

// ListBuckets lists all OpenClaw memory buckets
func (m *Manager) ListBuckets(ctx context.Context) ([]string, error) {
	names := m.js.KeyValueStoreNames(ctx)
	
	var buckets []string
	for name := range names.Name() {
		// Filter to only OpenClaw buckets
		if isOpenClawBucket(name, m.config) {
			buckets = append(buckets, name)
		}
	}

	return buckets, nil
}

func isOpenClawBucket(name string, config *memory.BucketConfig) bool {
	return name == config.CompanyKnowledge ||
		name == config.CompanyTools ||
		name == config.CompanyPolicies ||
		len(name) > len(config.GroupPrefix) && name[:len(config.GroupPrefix)] == config.GroupPrefix ||
		len(name) > len(config.SwarmPrefix) && name[:len(config.SwarmPrefix)] == config.SwarmPrefix ||
		len(name) > len(config.AgentPrefix) && name[:len(config.AgentPrefix)] == config.AgentPrefix
}

// InitializeCompanyBuckets creates all company-level buckets
func (m *Manager) InitializeCompanyBuckets(ctx context.Context) error {
	buckets := []string{
		m.config.CompanyKnowledge,
		m.config.CompanyTools,
		m.config.CompanyPolicies,
	}

	for _, name := range buckets {
		_, err := m.EnsureBucket(ctx, DefaultCompanyBucketSpec(name))
		if err != nil {
			return fmt.Errorf("failed to initialize company bucket %s: %w", name, err)
		}
	}

	return nil
}

// InitializeGroupBucket creates a group bucket
func (m *Manager) InitializeGroupBucket(ctx context.Context, groupName string) (jetstream.KeyValue, error) {
	return m.EnsureBucket(ctx, DefaultGroupBucketSpec(groupName))
}

// InitializeSwarmBucket creates a swarm bucket
func (m *Manager) InitializeSwarmBucket(ctx context.Context, swarmID string) (jetstream.KeyValue, error) {
	return m.EnsureBucket(ctx, DefaultSwarmBucketSpec(swarmID))
}

// InitializeAgentBucket creates an agent bucket
func (m *Manager) InitializeAgentBucket(ctx context.Context, agentID string) (jetstream.KeyValue, error) {
	return m.EnsureBucket(ctx, DefaultAgentBucketSpec(agentID))
}

// CleanupSwarmBucket deletes a swarm bucket (typically after task completion)
func (m *Manager) CleanupSwarmBucket(ctx context.Context, swarmID string) error {
	bucketName := m.config.SwarmPrefix + swarmID
	return m.DeleteBucket(ctx, bucketName)
}

// CleanupAgentBucket deletes an agent bucket
func (m *Manager) CleanupAgentBucket(ctx context.Context, agentID string) error {
	bucketName := m.config.AgentPrefix + agentID
	return m.DeleteBucket(ctx, bucketName)
}

// GetBucketStatus returns status information about a bucket
type BucketStatus struct {
	Name      string    `json:"name"`
	Entries   uint64    `json:"entries"`
	Bytes     uint64    `json:"bytes"`
	History   uint8     `json:"history"`
	TTL       time.Duration `json:"ttl"`
	Created   time.Time `json:"created"`
}

// GetBucketStatus returns status information about a bucket
func (m *Manager) GetBucketStatus(ctx context.Context, name string) (*BucketStatus, error) {
	kv, err := m.js.KeyValue(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %s", name)
	}

	status, err := kv.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket status: %w", err)
	}

	return &BucketStatus{
		Name:    status.Bucket(),
		Entries: status.Values(),
		Bytes:   status.Bytes(),
		History: status.History(),
		TTL:     status.TTL(),
	}, nil
}
