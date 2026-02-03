// Package company provides company-wide memory management with promotion workflow
package company

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openclaw/openclaw-platform/pkg/memory"
	"github.com/openclaw/openclaw-platform/pkg/memory/client"
)

// Manager handles company-wide memory operations
type Manager struct {
	client   *client.Client
	agentID  string
	buckets  *memory.BucketConfig
}

// NewManager creates a new company memory manager
func NewManager(c *client.Client, agentID string) *Manager {
	return &Manager{
		client:  c,
		agentID: agentID,
		buckets: memory.DefaultBucketConfig(),
	}
}

// Well-known bucket types
const (
	BucketKnowledge = "knowledge"
	BucketTools     = "tools"
	BucketPolicies  = "policies"
)

// === Knowledge Management ===

// Knowledge represents company-wide knowledge
type Knowledge struct {
	ID            string            `json:"id"`
	Topic         string            `json:"topic"`
	Content       string            `json:"content"`
	Category      string            `json:"category,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	Source        KnowledgeSource   `json:"source"`
	Confidence    float64           `json:"confidence"` // 0.0 - 1.0
	Version       int               `json:"version"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	UpdatedBy     string            `json:"updated_by"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// KnowledgeSource tracks the origin of knowledge
type KnowledgeSource struct {
	Type       string `json:"type"` // human, agent, promotion, external
	OriginalID string `json:"original_id,omitempty"`
	AgentID    string `json:"agent_id,omitempty"`
	GroupName  string `json:"group_name,omitempty"`
	SwarmID    string `json:"swarm_id,omitempty"`
}

// AddKnowledge adds knowledge to the company-wide knowledge base (admin only)
func (m *Manager) AddKnowledge(ctx context.Context, knowledge *Knowledge) error {
	store, err := m.client.Company(ctx, m.buckets.CompanyKnowledge)
	if err != nil {
		return err
	}

	if knowledge.ID == "" {
		knowledge.ID = fmt.Sprintf("%s-%d", knowledge.Topic, time.Now().UnixNano())
	}
	knowledge.CreatedAt = time.Now()
	knowledge.UpdatedAt = time.Now()
	knowledge.UpdatedBy = m.agentID
	knowledge.Version = 1

	data, err := json.Marshal(knowledge)
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge: %w", err)
	}

	_, err = store.Put(ctx, knowledge.ID, data)
	return err
}

// GetKnowledge retrieves knowledge by ID
func (m *Manager) GetKnowledge(ctx context.Context, id string) (*Knowledge, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyKnowledge)
	if err != nil {
		return nil, err
	}

	entry, err := store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	var knowledge Knowledge
	if err := json.Unmarshal(entry.Value, &knowledge); err != nil {
		return nil, fmt.Errorf("failed to unmarshal knowledge: %w", err)
	}

	return &knowledge, nil
}

// SearchKnowledge searches knowledge by category and/or tags
func (m *Manager) SearchKnowledge(ctx context.Context, category string, tags []string) ([]*Knowledge, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyKnowledge)
	if err != nil {
		return nil, err
	}

	keys, err := store.List(ctx, "*")
	if err != nil {
		return nil, err
	}

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	var results []*Knowledge
	for _, key := range keys {
		entry, err := store.Get(ctx, key)
		if err != nil {
			continue
		}

		var knowledge Knowledge
		if err := json.Unmarshal(entry.Value, &knowledge); err != nil {
			continue
		}

		// Filter by category
		if category != "" && knowledge.Category != category {
			continue
		}

		// Filter by tags (if any specified)
		if len(tags) > 0 {
			matched := false
			for _, t := range knowledge.Tags {
				if tagSet[t] {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		results = append(results, &knowledge)
	}

	return results, nil
}

// ListAllKnowledge lists all knowledge entries
func (m *Manager) ListAllKnowledge(ctx context.Context) ([]*Knowledge, error) {
	return m.SearchKnowledge(ctx, "", nil)
}

// === Tool Catalog ===

// Tool represents a tool in the company catalog
type Tool struct {
	Name          string            `json:"name"`
	Version       string            `json:"version,omitempty"`
	Description   string            `json:"description"`
	Category      string            `json:"category"` // cli, library, service, etc.
	InstallMethod string            `json:"install_method"` // apt, binary, container, etc.
	InstallCmd    string            `json:"install_cmd,omitempty"`
	BinaryURL     string            `json:"binary_url,omitempty"`
	ContainerImage string           `json:"container_image,omitempty"`
	Profiles      []string          `json:"profiles,omitempty"` // Which immutable profiles include this
	Capabilities  []string          `json:"capabilities,omitempty"` // Required capabilities
	Dependencies  []string          `json:"dependencies,omitempty"`
	Verified      bool              `json:"verified"`
	AddedAt       time.Time         `json:"added_at"`
	AddedBy       string            `json:"added_by"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// AddTool adds a tool to the catalog (admin only)
func (m *Manager) AddTool(ctx context.Context, tool *Tool) error {
	store, err := m.client.Company(ctx, m.buckets.CompanyTools)
	if err != nil {
		return err
	}

	tool.AddedAt = time.Now()
	tool.AddedBy = m.agentID

	data, err := json.Marshal(tool)
	if err != nil {
		return fmt.Errorf("failed to marshal tool: %w", err)
	}

	_, err = store.Put(ctx, tool.Name, data)
	return err
}

// GetTool retrieves a tool by name
func (m *Manager) GetTool(ctx context.Context, name string) (*Tool, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyTools)
	if err != nil {
		return nil, err
	}

	entry, err := store.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	var tool Tool
	if err := json.Unmarshal(entry.Value, &tool); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool: %w", err)
	}

	return &tool, nil
}

// GetToolsByProfile returns tools included in a specific profile
func (m *Manager) GetToolsByProfile(ctx context.Context, profile string) ([]*Tool, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyTools)
	if err != nil {
		return nil, err
	}

	keys, err := store.List(ctx, "*")
	if err != nil {
		return nil, err
	}

	var tools []*Tool
	for _, key := range keys {
		entry, err := store.Get(ctx, key)
		if err != nil {
			continue
		}

		var tool Tool
		if err := json.Unmarshal(entry.Value, &tool); err != nil {
			continue
		}

		for _, p := range tool.Profiles {
			if p == profile {
				tools = append(tools, &tool)
				break
			}
		}
	}

	return tools, nil
}

// ListAllTools lists all tools
func (m *Manager) ListAllTools(ctx context.Context) ([]*Tool, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyTools)
	if err != nil {
		return nil, err
	}

	keys, err := store.List(ctx, "*")
	if err != nil {
		return nil, err
	}

	var tools []*Tool
	for _, key := range keys {
		entry, err := store.Get(ctx, key)
		if err != nil {
			continue
		}

		var tool Tool
		if err := json.Unmarshal(entry.Value, &tool); err != nil {
			continue
		}
		tools = append(tools, &tool)
	}

	return tools, nil
}

// === Policies ===

// Policy represents a company-wide policy
type Policy struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          PolicyType        `json:"type"`
	Description   string            `json:"description"`
	Rules         []PolicyRule      `json:"rules"`
	Scope         PolicyScope       `json:"scope"`
	Priority      int               `json:"priority"` // Higher = more important
	Enabled       bool              `json:"enabled"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	UpdatedBy     string            `json:"updated_by"`
}

// PolicyType categorizes policies
type PolicyType string

const (
	PolicyTypeSecurity    PolicyType = "security"
	PolicyTypeBehavior    PolicyType = "behavior"
	PolicyTypeCompliance  PolicyType = "compliance"
	PolicyTypeResource    PolicyType = "resource"
	PolicyTypeAccess      PolicyType = "access"
)

// PolicyScope defines where a policy applies
type PolicyScope struct {
	Global     bool     `json:"global"`
	Groups     []string `json:"groups,omitempty"`
	Profiles   []string `json:"profiles,omitempty"`
	Agents     []string `json:"agents,omitempty"`
}

// PolicyRule represents a single rule within a policy
type PolicyRule struct {
	ID          string         `json:"id"`
	Condition   string         `json:"condition"` // Expression to evaluate
	Action      string         `json:"action"`    // allow, deny, warn, log
	Message     string         `json:"message,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// AddPolicy adds a policy (admin only)
func (m *Manager) AddPolicy(ctx context.Context, policy *Policy) error {
	store, err := m.client.Company(ctx, m.buckets.CompanyPolicies)
	if err != nil {
		return err
	}

	if policy.ID == "" {
		policy.ID = fmt.Sprintf("%s-%d", policy.Name, time.Now().UnixNano())
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	policy.UpdatedBy = m.agentID

	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	_, err = store.Put(ctx, policy.ID, data)
	return err
}

// GetPolicy retrieves a policy by ID
func (m *Manager) GetPolicy(ctx context.Context, id string) (*Policy, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyPolicies)
	if err != nil {
		return nil, err
	}

	entry, err := store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	var policy Policy
	if err := json.Unmarshal(entry.Value, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	return &policy, nil
}

// GetApplicablePolicies returns policies applicable to an agent
func (m *Manager) GetApplicablePolicies(ctx context.Context, agentID, groupName, profile string) ([]*Policy, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyPolicies)
	if err != nil {
		return nil, err
	}

	keys, err := store.List(ctx, "*")
	if err != nil {
		return nil, err
	}

	var applicable []*Policy
	for _, key := range keys {
		entry, err := store.Get(ctx, key)
		if err != nil {
			continue
		}

		var policy Policy
		if err := json.Unmarshal(entry.Value, &policy); err != nil {
			continue
		}

		if !policy.Enabled {
			continue
		}

		if m.policyApplies(&policy, agentID, groupName, profile) {
			applicable = append(applicable, &policy)
		}
	}

	return applicable, nil
}

func (m *Manager) policyApplies(policy *Policy, agentID, groupName, profile string) bool {
	if policy.Scope.Global {
		return true
	}

	for _, a := range policy.Scope.Agents {
		if a == agentID {
			return true
		}
	}

	for _, g := range policy.Scope.Groups {
		if g == groupName {
			return true
		}
	}

	for _, p := range policy.Scope.Profiles {
		if p == profile {
			return true
		}
	}

	return false
}

// === Promotion Workflow ===

// PromotionRequest represents a request to promote memory to company level
type PromotionRequest struct {
	ID            string            `json:"id"`
	SourceTier    string            `json:"source_tier"` // group, swarm
	SourceID      string            `json:"source_id"`   // group name or swarm ID
	TargetBucket  string            `json:"target_bucket"` // knowledge, tools, policies
	Content       json.RawMessage   `json:"content"`
	Reason        string            `json:"reason"`
	Status        PromotionStatus   `json:"status"`
	RequestedBy   string            `json:"requested_by"`
	RequestedAt   time.Time         `json:"requested_at"`
	ReviewedBy    string            `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time        `json:"reviewed_at,omitempty"`
	ReviewComment string            `json:"review_comment,omitempty"`
	AutoPromoted  bool              `json:"auto_promoted"`
	Votes         []PromotionVote   `json:"votes,omitempty"`
}

// PromotionStatus represents the status of a promotion request
type PromotionStatus string

const (
	PromotionStatusPending  PromotionStatus = "pending"
	PromotionStatusApproved PromotionStatus = "approved"
	PromotionStatusRejected PromotionStatus = "rejected"
	PromotionStatusApplied  PromotionStatus = "applied"
)

// PromotionVote represents a vote on a promotion request
type PromotionVote struct {
	AgentID   string    `json:"agent_id"`
	GroupName string    `json:"group_name,omitempty"`
	Vote      bool      `json:"vote"` // true = approve, false = reject
	Comment   string    `json:"comment,omitempty"`
	VotedAt   time.Time `json:"voted_at"`
}

// PromotionConfig defines promotion rules
type PromotionConfig struct {
	AutoPromoteThreshold int  `json:"auto_promote_threshold"` // Number of votes for auto-promotion
	RequireHumanApproval bool `json:"require_human_approval"`
	AllowedBuckets       []string `json:"allowed_buckets"`
}

// DefaultPromotionConfig returns default promotion settings
func DefaultPromotionConfig() *PromotionConfig {
	return &PromotionConfig{
		AutoPromoteThreshold: 3,
		RequireHumanApproval: false,
		AllowedBuckets:       []string{BucketKnowledge},
	}
}

// RequestPromotion submits a promotion request
func (m *Manager) RequestPromotion(ctx context.Context, request *PromotionRequest) error {
	store, err := m.client.Company(ctx, m.buckets.CompanyPolicies)
	if err != nil {
		return err
	}

	if request.ID == "" {
		request.ID = fmt.Sprintf("promo-%d", time.Now().UnixNano())
	}
	request.RequestedBy = m.agentID
	request.RequestedAt = time.Now()
	request.Status = PromotionStatusPending

	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	key := fmt.Sprintf("promotions.%s", request.ID)
	_, err = store.Put(ctx, key, data)
	return err
}

// GetPromotionRequest retrieves a promotion request
func (m *Manager) GetPromotionRequest(ctx context.Context, id string) (*PromotionRequest, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyPolicies)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("promotions.%s", id)
	entry, err := store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var request PromotionRequest
	if err := json.Unmarshal(entry.Value, &request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &request, nil
}

// ListPendingPromotions lists pending promotion requests
func (m *Manager) ListPendingPromotions(ctx context.Context) ([]*PromotionRequest, error) {
	store, err := m.client.Company(ctx, m.buckets.CompanyPolicies)
	if err != nil {
		return nil, err
	}

	keys, err := store.List(ctx, "promotions.*")
	if err != nil {
		return nil, err
	}

	var pending []*PromotionRequest
	for _, key := range keys {
		entry, err := store.Get(ctx, key)
		if err != nil {
			continue
		}

		var request PromotionRequest
		if err := json.Unmarshal(entry.Value, &request); err != nil {
			continue
		}

		if request.Status == PromotionStatusPending {
			pending = append(pending, &request)
		}
	}

	return pending, nil
}

// VoteOnPromotion votes on a promotion request
func (m *Manager) VoteOnPromotion(ctx context.Context, requestID string, approve bool, comment string) error {
	request, err := m.GetPromotionRequest(ctx, requestID)
	if err != nil {
		return err
	}

	if request.Status != PromotionStatusPending {
		return fmt.Errorf("request is not pending")
	}

	// Check if already voted
	for _, v := range request.Votes {
		if v.AgentID == m.agentID {
			return fmt.Errorf("already voted on this request")
		}
	}

	request.Votes = append(request.Votes, PromotionVote{
		AgentID: m.agentID,
		Vote:    approve,
		Comment: comment,
		VotedAt: time.Now(),
	})

	// Check for auto-promotion
	config := DefaultPromotionConfig()
	approveCount := 0
	for _, v := range request.Votes {
		if v.Vote {
			approveCount++
		}
	}

	if approveCount >= config.AutoPromoteThreshold && !config.RequireHumanApproval {
		request.Status = PromotionStatusApproved
		request.AutoPromoted = true
		now := time.Now()
		request.ReviewedAt = &now
	}

	store, err := m.client.Company(ctx, m.buckets.CompanyPolicies)
	if err != nil {
		return err
	}

	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	key := fmt.Sprintf("promotions.%s", requestID)
	_, err = store.Put(ctx, key, data)
	return err
}

// ApplyPromotion applies an approved promotion (admin only)
func (m *Manager) ApplyPromotion(ctx context.Context, requestID string) error {
	request, err := m.GetPromotionRequest(ctx, requestID)
	if err != nil {
		return err
	}

	if request.Status != PromotionStatusApproved {
		return fmt.Errorf("request is not approved")
	}

	// Apply based on target bucket
	var targetBucket string
	switch request.TargetBucket {
	case BucketKnowledge:
		targetBucket = m.buckets.CompanyKnowledge
		var knowledge Knowledge
		if err := json.Unmarshal(request.Content, &knowledge); err != nil {
			return fmt.Errorf("failed to unmarshal knowledge: %w", err)
		}
		knowledge.Source = KnowledgeSource{
			Type:       "promotion",
			OriginalID: request.SourceID,
			AgentID:    request.RequestedBy,
		}
		if err := m.AddKnowledge(ctx, &knowledge); err != nil {
			return err
		}
	case BucketTools:
		targetBucket = m.buckets.CompanyTools
		var tool Tool
		if err := json.Unmarshal(request.Content, &tool); err != nil {
			return fmt.Errorf("failed to unmarshal tool: %w", err)
		}
		if err := m.AddTool(ctx, &tool); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported target bucket: %s", request.TargetBucket)
	}

	// Update request status
	request.Status = PromotionStatusApplied
	now := time.Now()
	request.ReviewedAt = &now
	request.ReviewedBy = m.agentID

	store, err := m.client.Company(ctx, m.buckets.CompanyPolicies)
	if err != nil {
		return err
	}

	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	key := fmt.Sprintf("promotions.%s", requestID)
	_, err = store.Put(ctx, key, data)
	if err != nil {
		return err
	}

	_ = targetBucket // Used for logging/metrics in production
	return nil
}
