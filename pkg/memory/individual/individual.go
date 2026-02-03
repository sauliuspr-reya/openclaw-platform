// Package individual provides individual agent memory management
package individual

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openclaw/openclaw-platform/pkg/memory"
	"github.com/openclaw/openclaw-platform/pkg/memory/client"
)

// Manager handles individual agent memory operations
type Manager struct {
	client  *client.Client
	agentID string
	store   memory.Store
}

// NewManager creates a new individual memory manager
func NewManager(ctx context.Context, c *client.Client, agentID string) (*Manager, error) {
	store, err := c.Individual(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get individual store: %w", err)
	}

	return &Manager{
		client:  c,
		agentID: agentID,
		store:   store,
	}, nil
}

// Context keys for individual memory
const (
	KeyIdentity      = "identity"
	KeySoul          = "soul"
	KeyContext       = "context"
	KeyConversation  = "conversation"
	KeyWorkingFiles  = "working_files"
	KeyLocalCache    = "cache"
)

// Identity represents the agent's identity configuration
type Identity struct {
	Name        string            `json:"name"`
	Role        string            `json:"role"`
	Description string            `json:"description"`
	Expertise   []string          `json:"expertise,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	RawContent  string            `json:"raw_content,omitempty"` // Original IDENTITY.md content
	LoadedAt    time.Time         `json:"loaded_at"`
}

// Soul represents the agent's behavioral traits
type Soul struct {
	Traits          []string          `json:"traits,omitempty"`
	Values          []string          `json:"values,omitempty"`
	CommunicationStyle string         `json:"communication_style,omitempty"`
	Constraints     []string          `json:"constraints,omitempty"`
	Preferences     map[string]string `json:"preferences,omitempty"`
	RawContent      string            `json:"raw_content,omitempty"` // Original SOUL.md content
	LoadedAt        time.Time         `json:"loaded_at"`
}

// WorkingContext represents the current task context
type WorkingContext struct {
	TaskID          string            `json:"task_id,omitempty"`
	TaskDescription string            `json:"task_description,omitempty"`
	CurrentStep     string            `json:"current_step,omitempty"`
	Progress        float64           `json:"progress"` // 0.0 - 1.0
	StartedAt       time.Time         `json:"started_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	State           map[string]any    `json:"state,omitempty"`
}

// ConversationEntry represents a single conversation turn
type ConversationEntry struct {
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ConversationHistory stores recent conversation turns
type ConversationHistory struct {
	Entries   []ConversationEntry `json:"entries"`
	MaxLength int                 `json:"max_length"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// WorkingFile represents a file the agent is working with
type WorkingFile struct {
	Path       string    `json:"path"`
	Action     string    `json:"action"` // read, write, create, delete
	ModifiedAt time.Time `json:"modified_at"`
}

// CacheEntry represents a cached value from higher tiers
type CacheEntry struct {
	Key       string    `json:"key"`
	Value     []byte    `json:"value"`
	Source    string    `json:"source"` // company, group
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// StoreIdentity stores the identity configuration
func (m *Manager) StoreIdentity(ctx context.Context, identity *Identity) error {
	identity.LoadedAt = time.Now()
	data, err := json.Marshal(identity)
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %w", err)
	}
	_, err = m.store.Put(ctx, KeyIdentity, data)
	return err
}

// LoadIdentity loads the identity configuration
func (m *Manager) LoadIdentity(ctx context.Context) (*Identity, error) {
	entry, err := m.store.Get(ctx, KeyIdentity)
	if err != nil {
		return nil, err
	}

	var identity Identity
	if err := json.Unmarshal(entry.Value, &identity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal identity: %w", err)
	}

	return &identity, nil
}

// StoreSoul stores the soul configuration
func (m *Manager) StoreSoul(ctx context.Context, soul *Soul) error {
	soul.LoadedAt = time.Now()
	data, err := json.Marshal(soul)
	if err != nil {
		return fmt.Errorf("failed to marshal soul: %w", err)
	}
	_, err = m.store.Put(ctx, KeySoul, data)
	return err
}

// LoadSoul loads the soul configuration
func (m *Manager) LoadSoul(ctx context.Context) (*Soul, error) {
	entry, err := m.store.Get(ctx, KeySoul)
	if err != nil {
		return nil, err
	}

	var soul Soul
	if err := json.Unmarshal(entry.Value, &soul); err != nil {
		return nil, fmt.Errorf("failed to unmarshal soul: %w", err)
	}

	return &soul, nil
}

// UpdateContext updates the working context
func (m *Manager) UpdateContext(ctx context.Context, workCtx *WorkingContext) error {
	workCtx.UpdatedAt = time.Now()
	data, err := json.Marshal(workCtx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}
	_, err = m.store.Put(ctx, KeyContext, data)
	return err
}

// GetContext retrieves the working context
func (m *Manager) GetContext(ctx context.Context) (*WorkingContext, error) {
	entry, err := m.store.Get(ctx, KeyContext)
	if err != nil {
		return nil, err
	}

	var workCtx WorkingContext
	if err := json.Unmarshal(entry.Value, &workCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	return &workCtx, nil
}

// AddConversationEntry adds a conversation entry
func (m *Manager) AddConversationEntry(ctx context.Context, entry ConversationEntry) error {
	history, err := m.GetConversationHistory(ctx)
	if err != nil {
		// Initialize new history
		history = &ConversationHistory{
			Entries:   []ConversationEntry{},
			MaxLength: 100,
		}
	}

	entry.Timestamp = time.Now()
	history.Entries = append(history.Entries, entry)

	// Trim if over max length
	if len(history.Entries) > history.MaxLength {
		history.Entries = history.Entries[len(history.Entries)-history.MaxLength:]
	}

	history.UpdatedAt = time.Now()

	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}
	_, err = m.store.Put(ctx, KeyConversation, data)
	return err
}

// GetConversationHistory retrieves the conversation history
func (m *Manager) GetConversationHistory(ctx context.Context) (*ConversationHistory, error) {
	entry, err := m.store.Get(ctx, KeyConversation)
	if err != nil {
		return nil, err
	}

	var history ConversationHistory
	if err := json.Unmarshal(entry.Value, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}

	return &history, nil
}

// TrackWorkingFile tracks a file being worked on
func (m *Manager) TrackWorkingFile(ctx context.Context, file WorkingFile) error {
	files, err := m.GetWorkingFiles(ctx)
	if err != nil {
		files = []WorkingFile{}
	}

	// Update or add file
	found := false
	for i, f := range files {
		if f.Path == file.Path {
			files[i] = file
			found = true
			break
		}
	}
	if !found {
		files = append(files, file)
	}

	data, err := json.Marshal(files)
	if err != nil {
		return fmt.Errorf("failed to marshal working files: %w", err)
	}
	_, err = m.store.Put(ctx, KeyWorkingFiles, data)
	return err
}

// GetWorkingFiles retrieves the list of working files
func (m *Manager) GetWorkingFiles(ctx context.Context) ([]WorkingFile, error) {
	entry, err := m.store.Get(ctx, KeyWorkingFiles)
	if err != nil {
		return nil, err
	}

	var files []WorkingFile
	if err := json.Unmarshal(entry.Value, &files); err != nil {
		return nil, fmt.Errorf("failed to unmarshal working files: %w", err)
	}

	return files, nil
}

// CacheFromHigherTier caches a value from a higher tier
func (m *Manager) CacheFromHigherTier(ctx context.Context, key, source string, value []byte, ttl time.Duration) error {
	cacheEntry := CacheEntry{
		Key:      key,
		Value:    value,
		Source:   source,
		CachedAt: time.Now(),
	}
	if ttl > 0 {
		cacheEntry.ExpiresAt = time.Now().Add(ttl)
	}

	data, err := json.Marshal(cacheEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	cacheKey := fmt.Sprintf("%s.%s.%s", KeyLocalCache, source, key)
	_, err = m.store.Put(ctx, cacheKey, data)
	return err
}

// GetCached retrieves a cached value
func (m *Manager) GetCached(ctx context.Context, key, source string) (*CacheEntry, error) {
	cacheKey := fmt.Sprintf("%s.%s.%s", KeyLocalCache, source, key)
	entry, err := m.store.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	var cacheEntry CacheEntry
	if err := json.Unmarshal(entry.Value, &cacheEntry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	// Check expiry
	if !cacheEntry.ExpiresAt.IsZero() && time.Now().After(cacheEntry.ExpiresAt) {
		// Delete expired entry
		m.store.Delete(ctx, cacheKey)
		return nil, fmt.Errorf("cache entry expired")
	}

	return &cacheEntry, nil
}

// Clear clears all individual memory
func (m *Manager) Clear(ctx context.Context) error {
	keys, err := m.store.List(ctx, "*")
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	for _, key := range keys {
		if err := m.store.Delete(ctx, key); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", key, err)
		}
	}

	return nil
}
