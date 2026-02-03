// Package group provides group memory management for functional teams
package group

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openclaw/openclaw-platform/pkg/memory"
	"github.com/openclaw/openclaw-platform/pkg/memory/client"
)

// Manager handles group memory operations
type Manager struct {
	client    *client.Client
	groupName string
	agentID   string
	store     memory.Store
}

// NewManager creates a new group memory manager
func NewManager(ctx context.Context, c *client.Client, groupName, agentID string) (*Manager, error) {
	store, err := c.Group(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get group store: %w", err)
	}

	return &Manager{
		client:    c,
		groupName: groupName,
		agentID:   agentID,
		store:     store,
	}, nil
}

// Well-known group memory keys
const (
	KeyKnowledge    = "knowledge"
	KeyPreferences  = "preferences"
	KeyLearnings    = "learnings"
	KeyResources    = "resources"
	KeyMembers      = "members"
	KeyAnnouncements = "announcements"
)

// Knowledge represents specialized domain knowledge for the group
type Knowledge struct {
	Topic       string            `json:"topic"`
	Content     string            `json:"content"`
	Tags        []string          `json:"tags,omitempty"`
	Source      string            `json:"source,omitempty"`
	ContributedBy string          `json:"contributed_by"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// GroupPreferences represents group-level preferences and standards
type GroupPreferences struct {
	CodingStandards   map[string]string `json:"coding_standards,omitempty"`
	ReviewCriteria    []string          `json:"review_criteria,omitempty"`
	CommunicationStyle string           `json:"communication_style,omitempty"`
	ToolPreferences   map[string]string `json:"tool_preferences,omitempty"`
	CustomRules       map[string]any    `json:"custom_rules,omitempty"`
	UpdatedAt         time.Time         `json:"updated_at"`
	UpdatedBy         string            `json:"updated_by"`
}

// Learning represents a cross-agent learning
type Learning struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Pattern     string            `json:"pattern,omitempty"`
	Context     string            `json:"context,omitempty"`
	Outcome     string            `json:"outcome,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	DiscoveredBy string           `json:"discovered_by"`
	ValidatedBy []string          `json:"validated_by,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	Confidence  float64           `json:"confidence"` // 0.0 - 1.0
}

// Resource represents a shared group resource
type Resource struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // api_key, endpoint, credential, etc.
	Value       string            `json:"value,omitempty"`
	SecretRef   string            `json:"secret_ref,omitempty"` // Reference to K8s secret
	Description string            `json:"description,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Member represents a group member
type Member struct {
	AgentID    string    `json:"agent_id"`
	Role       string    `json:"role,omitempty"` // lead, member, observer
	JoinedAt   time.Time `json:"joined_at"`
	LastActive time.Time `json:"last_active"`
	Expertise  []string  `json:"expertise,omitempty"`
}

// Announcement represents a group-wide announcement
type Announcement struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Priority    string    `json:"priority"` // low, normal, high, urgent
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	AcknowledgedBy []string `json:"acknowledged_by,omitempty"`
}

// AddKnowledge adds knowledge to the group
func (m *Manager) AddKnowledge(ctx context.Context, knowledge *Knowledge) error {
	knowledge.ContributedBy = m.agentID
	knowledge.CreatedAt = time.Now()
	knowledge.UpdatedAt = time.Now()

	data, err := json.Marshal(knowledge)
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge: %w", err)
	}

	key := fmt.Sprintf("%s.%s", KeyKnowledge, knowledge.Topic)
	_, err = m.store.Put(ctx, key, data)
	return err
}

// GetKnowledge retrieves knowledge by topic
func (m *Manager) GetKnowledge(ctx context.Context, topic string) (*Knowledge, error) {
	key := fmt.Sprintf("%s.%s", KeyKnowledge, topic)
	entry, err := m.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var knowledge Knowledge
	if err := json.Unmarshal(entry.Value, &knowledge); err != nil {
		return nil, fmt.Errorf("failed to unmarshal knowledge: %w", err)
	}

	return &knowledge, nil
}

// ListKnowledge lists all knowledge topics
func (m *Manager) ListKnowledge(ctx context.Context) ([]*Knowledge, error) {
	keys, err := m.store.List(ctx, KeyKnowledge+".*")
	if err != nil {
		return nil, err
	}

	var knowledgeList []*Knowledge
	for _, key := range keys {
		entry, err := m.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var knowledge Knowledge
		if err := json.Unmarshal(entry.Value, &knowledge); err != nil {
			continue
		}
		knowledgeList = append(knowledgeList, &knowledge)
	}

	return knowledgeList, nil
}

// SearchKnowledge searches knowledge by tags
func (m *Manager) SearchKnowledge(ctx context.Context, tags []string) ([]*Knowledge, error) {
	allKnowledge, err := m.ListKnowledge(ctx)
	if err != nil {
		return nil, err
	}

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	var matches []*Knowledge
	for _, k := range allKnowledge {
		for _, t := range k.Tags {
			if tagSet[t] {
				matches = append(matches, k)
				break
			}
		}
	}

	return matches, nil
}

// UpdatePreferences updates group preferences
func (m *Manager) UpdatePreferences(ctx context.Context, prefs *GroupPreferences) error {
	prefs.UpdatedAt = time.Now()
	prefs.UpdatedBy = m.agentID

	data, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	_, err = m.store.Put(ctx, KeyPreferences, data)
	return err
}

// GetPreferences retrieves group preferences
func (m *Manager) GetPreferences(ctx context.Context) (*GroupPreferences, error) {
	entry, err := m.store.Get(ctx, KeyPreferences)
	if err != nil {
		return nil, err
	}

	var prefs GroupPreferences
	if err := json.Unmarshal(entry.Value, &prefs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	return &prefs, nil
}

// AddLearning adds a learning to the group
func (m *Manager) AddLearning(ctx context.Context, learning *Learning) error {
	learning.DiscoveredBy = m.agentID
	learning.CreatedAt = time.Now()
	if learning.ID == "" {
		learning.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	data, err := json.Marshal(learning)
	if err != nil {
		return fmt.Errorf("failed to marshal learning: %w", err)
	}

	key := fmt.Sprintf("%s.%s", KeyLearnings, learning.ID)
	_, err = m.store.Put(ctx, key, data)
	return err
}

// ValidateLearning validates a learning by another agent
func (m *Manager) ValidateLearning(ctx context.Context, learningID string) error {
	learning, err := m.GetLearning(ctx, learningID)
	if err != nil {
		return err
	}

	// Check if already validated by this agent
	for _, v := range learning.ValidatedBy {
		if v == m.agentID {
			return nil // Already validated
		}
	}

	learning.ValidatedBy = append(learning.ValidatedBy, m.agentID)
	// Increase confidence based on validations
	learning.Confidence = float64(len(learning.ValidatedBy)) / 10.0
	if learning.Confidence > 1.0 {
		learning.Confidence = 1.0
	}

	data, err := json.Marshal(learning)
	if err != nil {
		return fmt.Errorf("failed to marshal learning: %w", err)
	}

	key := fmt.Sprintf("%s.%s", KeyLearnings, learningID)
	_, err = m.store.Put(ctx, key, data)
	return err
}

// GetLearning retrieves a learning by ID
func (m *Manager) GetLearning(ctx context.Context, id string) (*Learning, error) {
	key := fmt.Sprintf("%s.%s", KeyLearnings, id)
	entry, err := m.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var learning Learning
	if err := json.Unmarshal(entry.Value, &learning); err != nil {
		return nil, fmt.Errorf("failed to unmarshal learning: %w", err)
	}

	return &learning, nil
}

// ListLearnings lists all learnings
func (m *Manager) ListLearnings(ctx context.Context) ([]*Learning, error) {
	keys, err := m.store.List(ctx, KeyLearnings+".*")
	if err != nil {
		return nil, err
	}

	var learnings []*Learning
	for _, key := range keys {
		entry, err := m.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var learning Learning
		if err := json.Unmarshal(entry.Value, &learning); err != nil {
			continue
		}
		learnings = append(learnings, &learning)
	}

	return learnings, nil
}

// GetHighConfidenceLearnings returns learnings above a confidence threshold
func (m *Manager) GetHighConfidenceLearnings(ctx context.Context, minConfidence float64) ([]*Learning, error) {
	allLearnings, err := m.ListLearnings(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*Learning
	for _, l := range allLearnings {
		if l.Confidence >= minConfidence {
			filtered = append(filtered, l)
		}
	}

	return filtered, nil
}

// SetResource sets a shared resource
func (m *Manager) SetResource(ctx context.Context, resource *Resource) error {
	resource.UpdatedAt = time.Now()
	if resource.CreatedAt.IsZero() {
		resource.CreatedAt = time.Now()
	}

	data, err := json.Marshal(resource)
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	key := fmt.Sprintf("%s.%s", KeyResources, resource.Name)
	_, err = m.store.Put(ctx, key, data)
	return err
}

// GetResource retrieves a shared resource
func (m *Manager) GetResource(ctx context.Context, name string) (*Resource, error) {
	key := fmt.Sprintf("%s.%s", KeyResources, name)
	entry, err := m.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var resource Resource
	if err := json.Unmarshal(entry.Value, &resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}

	return &resource, nil
}

// RegisterMember registers this agent as a group member
func (m *Manager) RegisterMember(ctx context.Context, expertise []string) error {
	member := &Member{
		AgentID:    m.agentID,
		Role:       "member",
		JoinedAt:   time.Now(),
		LastActive: time.Now(),
		Expertise:  expertise,
	}

	data, err := json.Marshal(member)
	if err != nil {
		return fmt.Errorf("failed to marshal member: %w", err)
	}

	key := fmt.Sprintf("%s.%s", KeyMembers, m.agentID)
	_, err = m.store.Put(ctx, key, data)
	return err
}

// UpdateActivity updates the last active time for this agent
func (m *Manager) UpdateActivity(ctx context.Context) error {
	key := fmt.Sprintf("%s.%s", KeyMembers, m.agentID)
	entry, err := m.store.Get(ctx, key)
	if err != nil {
		// Not registered, register now
		return m.RegisterMember(ctx, nil)
	}

	var member Member
	if err := json.Unmarshal(entry.Value, &member); err != nil {
		return fmt.Errorf("failed to unmarshal member: %w", err)
	}

	member.LastActive = time.Now()

	data, err := json.Marshal(member)
	if err != nil {
		return fmt.Errorf("failed to marshal member: %w", err)
	}

	_, err = m.store.Put(ctx, key, data)
	return err
}

// ListMembers lists all group members
func (m *Manager) ListMembers(ctx context.Context) ([]*Member, error) {
	keys, err := m.store.List(ctx, KeyMembers+".*")
	if err != nil {
		return nil, err
	}

	var members []*Member
	for _, key := range keys {
		entry, err := m.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var member Member
		if err := json.Unmarshal(entry.Value, &member); err != nil {
			continue
		}
		members = append(members, &member)
	}

	return members, nil
}

// GetActiveMembers returns members active within the given duration
func (m *Manager) GetActiveMembers(ctx context.Context, within time.Duration) ([]*Member, error) {
	allMembers, err := m.ListMembers(ctx)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-within)
	var active []*Member
	for _, member := range allMembers {
		if member.LastActive.After(cutoff) {
			active = append(active, member)
		}
	}

	return active, nil
}

// PostAnnouncement posts an announcement to the group
func (m *Manager) PostAnnouncement(ctx context.Context, announcement *Announcement) error {
	announcement.CreatedBy = m.agentID
	announcement.CreatedAt = time.Now()
	if announcement.ID == "" {
		announcement.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	data, err := json.Marshal(announcement)
	if err != nil {
		return fmt.Errorf("failed to marshal announcement: %w", err)
	}

	key := fmt.Sprintf("%s.%s", KeyAnnouncements, announcement.ID)
	_, err = m.store.Put(ctx, key, data)
	return err
}

// AcknowledgeAnnouncement acknowledges an announcement
func (m *Manager) AcknowledgeAnnouncement(ctx context.Context, announcementID string) error {
	key := fmt.Sprintf("%s.%s", KeyAnnouncements, announcementID)
	entry, err := m.store.Get(ctx, key)
	if err != nil {
		return err
	}

	var announcement Announcement
	if err := json.Unmarshal(entry.Value, &announcement); err != nil {
		return fmt.Errorf("failed to unmarshal announcement: %w", err)
	}

	// Check if already acknowledged
	for _, a := range announcement.AcknowledgedBy {
		if a == m.agentID {
			return nil
		}
	}

	announcement.AcknowledgedBy = append(announcement.AcknowledgedBy, m.agentID)

	data, err := json.Marshal(announcement)
	if err != nil {
		return fmt.Errorf("failed to marshal announcement: %w", err)
	}

	_, err = m.store.Put(ctx, key, data)
	return err
}

// GetActiveAnnouncements returns non-expired announcements
func (m *Manager) GetActiveAnnouncements(ctx context.Context) ([]*Announcement, error) {
	keys, err := m.store.List(ctx, KeyAnnouncements+".*")
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var active []*Announcement
	for _, key := range keys {
		entry, err := m.store.Get(ctx, key)
		if err != nil {
			continue
		}

		var announcement Announcement
		if err := json.Unmarshal(entry.Value, &announcement); err != nil {
			continue
		}

		// Skip expired
		if announcement.ExpiresAt != nil && announcement.ExpiresAt.Before(now) {
			continue
		}

		active = append(active, &announcement)
	}

	return active, nil
}
