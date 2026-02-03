// Package client provides a NATS-based implementation of the memory store.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/openclaw/openclaw-platform/pkg/memory"
)

// Client provides hierarchical memory access backed by NATS JetStream KV
type Client struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	config *memory.AgentMemoryConfig
	buckets map[string]jetstream.KeyValue
	mu     sync.RWMutex
}

// NewClient creates a new memory client
func NewClient(ctx context.Context, config *memory.AgentMemoryConfig) (*Client, error) {
	nc, err := nats.Connect(config.NATSUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	c := &Client{
		nc:      nc,
		js:      js,
		config:  config,
		buckets: make(map[string]jetstream.KeyValue),
	}

	return c, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	c.nc.Close()
	return nil
}

// getBucket gets or creates a KV bucket
func (c *Client) getBucket(ctx context.Context, name string) (jetstream.KeyValue, error) {
	c.mu.RLock()
	if kv, ok := c.buckets[name]; ok {
		c.mu.RUnlock()
		return kv, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if kv, ok := c.buckets[name]; ok {
		return kv, nil
	}

	// Try to get existing bucket first
	kv, err := c.js.KeyValue(ctx, name)
	if err == nil {
		c.buckets[name] = kv
		return kv, nil
	}

	// Create bucket if it doesn't exist
	kv, err = c.js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      name,
		Description: fmt.Sprintf("OpenClaw memory bucket: %s", name),
		History:     10, // Keep 10 revisions
		TTL:         0,  // No default TTL
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket %s: %w", name, err)
	}

	c.buckets[name] = kv
	return kv, nil
}

// Company returns a store for company-wide memory
func (c *Client) Company(ctx context.Context, bucket string) (memory.Store, error) {
	if c.config.Company == nil || !c.config.Company.Enabled {
		return nil, fmt.Errorf("company memory not enabled for this agent")
	}

	kv, err := c.getBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}

	return &kvStore{
		kv:       kv,
		bucket:   bucket,
		readOnly: c.config.Company.Mode == memory.AccessModeReadOnly,
	}, nil
}

// Group returns a store for group memory
func (c *Client) Group(ctx context.Context) (memory.Store, error) {
	if c.config.Group == nil || !c.config.Group.Enabled {
		return nil, fmt.Errorf("group memory not enabled for this agent")
	}

	bucketName := memory.DefaultBucketConfig().GroupPrefix + c.config.Group.Group
	kv, err := c.getBucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	return &kvStore{
		kv:       kv,
		bucket:   bucketName,
		readOnly: c.config.Group.Mode == memory.AccessModeReadOnly,
	}, nil
}

// Swarm returns a store for swarm memory
func (c *Client) Swarm(ctx context.Context, swarmID string) (memory.Store, error) {
	if c.config.Swarm == nil || !c.config.Swarm.Enabled {
		return nil, fmt.Errorf("swarm memory not enabled for this agent")
	}

	bucketName := memory.DefaultBucketConfig().SwarmPrefix + swarmID
	kv, err := c.getBucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	return &kvStore{
		kv:       kv,
		bucket:   bucketName,
		readOnly: c.config.Swarm.Mode == memory.AccessModeReadOnly,
	}, nil
}

// Individual returns a store for individual agent memory
func (c *Client) Individual(ctx context.Context) (memory.Store, error) {
	if c.config.Individual == nil || !c.config.Individual.Enabled {
		return nil, fmt.Errorf("individual memory not enabled for this agent")
	}

	bucketName := c.config.Individual.Bucket
	if bucketName == "" {
		bucketName = memory.DefaultBucketConfig().AgentPrefix + c.config.AgentID
	}

	kv, err := c.getBucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	return &kvStore{
		kv:       kv,
		bucket:   bucketName,
		readOnly: false, // Individual memory is always read-write
	}, nil
}

// kvStore implements memory.Store using NATS KV
type kvStore struct {
	kv       jetstream.KeyValue
	bucket   string
	readOnly bool
}

func (s *kvStore) Get(ctx context.Context, key string) (*memory.Entry, error) {
	entry, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return &memory.Entry{
		Key:      key,
		Value:    entry.Value(),
		Revision: entry.Revision(),
		Created:  entry.Created(),
		Modified: entry.Created(), // NATS KV doesn't track modified separately
	}, nil
}

func (s *kvStore) Put(ctx context.Context, key string, value []byte, opts ...memory.PutOption) (*memory.Entry, error) {
	if s.readOnly {
		return nil, fmt.Errorf("store is read-only")
	}

	options := &putOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var rev uint64
	var err error

	if options.revision > 0 {
		// CAS update
		rev, err = s.kv.Update(ctx, key, value, options.revision)
	} else {
		// Regular put
		rev, err = s.kv.Put(ctx, key, value)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to put key %s: %w", key, err)
	}

	return &memory.Entry{
		Key:      key,
		Value:    value,
		Revision: rev,
		Modified: time.Now(),
	}, nil
}

type putOptions struct {
	ttl      time.Duration
	revision uint64
	metadata map[string]string
}

func (s *kvStore) Delete(ctx context.Context, key string) error {
	if s.readOnly {
		return fmt.Errorf("store is read-only")
	}

	return s.kv.Delete(ctx, key)
}

func (s *kvStore) List(ctx context.Context, pattern string) ([]string, error) {
	keys, err := s.kv.Keys(ctx)
	if err != nil {
		if err == jetstream.ErrNoKeysFound {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	if pattern == "" || pattern == "*" {
		return keys, nil
	}

	// Simple pattern matching (supports prefix*)
	var matched []string
	prefix := strings.TrimSuffix(pattern, "*")
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			matched = append(matched, key)
		}
	}

	return matched, nil
}

func (s *kvStore) Watch(ctx context.Context, pattern string) (<-chan memory.WatchEvent, error) {
	watcher, err := s.kv.WatchAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	events := make(chan memory.WatchEvent, 100)

	go func() {
		defer close(events)
		defer watcher.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-watcher.Updates():
				if entry == nil {
					continue
				}

				// Filter by pattern
				if pattern != "" && pattern != "*" {
					prefix := strings.TrimSuffix(pattern, "*")
					if !strings.HasPrefix(entry.Key(), prefix) {
						continue
					}
				}

				event := memory.WatchEvent{
					Key: entry.Key(),
				}

				if entry.Operation() == jetstream.KeyValueDelete {
					event.Type = memory.WatchEventDelete
				} else {
					event.Type = memory.WatchEventPut
					event.Entry = &memory.Entry{
						Key:      entry.Key(),
						Value:    entry.Value(),
						Revision: entry.Revision(),
						Created:  entry.Created(),
						Modified: entry.Created(),
					}
				}

				select {
				case events <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return events, nil
}

func (s *kvStore) History(ctx context.Context, key string, limit int) ([]*memory.Entry, error) {
	history, err := s.kv.History(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for key %s: %w", key, err)
	}

	var entries []*memory.Entry
	count := 0
	for entry := range history {
		if limit > 0 && count >= limit {
			break
		}
		entries = append(entries, &memory.Entry{
			Key:      entry.Key(),
			Value:    entry.Value(),
			Revision: entry.Revision(),
			Created:  entry.Created(),
			Modified: entry.Created(),
		})
		count++
	}

	return entries, nil
}

// Coordination helpers for swarm memory

// PublishSignal publishes a coordination signal to the swarm
func (c *Client) PublishSignal(ctx context.Context, signal *memory.CoordinationSignal) error {
	data, err := json.Marshal(signal)
	if err != nil {
		return fmt.Errorf("failed to marshal signal: %w", err)
	}

	subject := fmt.Sprintf("openclaw.swarm.%s.signals", signal.SwarmID)
	return c.nc.Publish(subject, data)
}

// SubscribeSignals subscribes to coordination signals for a swarm
func (c *Client) SubscribeSignals(ctx context.Context, swarmID string) (<-chan *memory.CoordinationSignal, error) {
	subject := fmt.Sprintf("openclaw.swarm.%s.signals", swarmID)
	signals := make(chan *memory.CoordinationSignal, 100)

	sub, err := c.nc.Subscribe(subject, func(msg *nats.Msg) {
		var signal memory.CoordinationSignal
		if err := json.Unmarshal(msg.Data, &signal); err != nil {
			return
		}
		select {
		case signals <- &signal:
		default:
			// Drop if channel is full
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to signals: %w", err)
	}

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(signals)
	}()

	return signals, nil
}

// WriteGoal writes the swarm goal to swarm memory
func (c *Client) WriteGoal(ctx context.Context, goal *memory.SwarmGoal) error {
	store, err := c.Swarm(ctx, goal.SwarmID)
	if err != nil {
		return err
	}

	data, err := json.Marshal(goal)
	if err != nil {
		return fmt.Errorf("failed to marshal goal: %w", err)
	}

	_, err = store.Put(ctx, "goal", data)
	return err
}

// ReadGoal reads the swarm goal from swarm memory
func (c *Client) ReadGoal(ctx context.Context, swarmID string) (*memory.SwarmGoal, error) {
	store, err := c.Swarm(ctx, swarmID)
	if err != nil {
		return nil, err
	}

	entry, err := store.Get(ctx, "goal")
	if err != nil {
		return nil, err
	}

	var goal memory.SwarmGoal
	if err := json.Unmarshal(entry.Value, &goal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal goal: %w", err)
	}

	return &goal, nil
}

// AssignShard assigns a shard to an agent
func (c *Client) AssignShard(ctx context.Context, assignment *memory.ShardAssignment) error {
	store, err := c.Swarm(ctx, assignment.SwarmID)
	if err != nil {
		return err
	}

	data, err := json.Marshal(assignment)
	if err != nil {
		return fmt.Errorf("failed to marshal assignment: %w", err)
	}

	_, err = store.Put(ctx, fmt.Sprintf("shards.%s", assignment.ShardID), data)
	return err
}

// GetShardAssignment gets a shard assignment
func (c *Client) GetShardAssignment(ctx context.Context, swarmID, shardID string) (*memory.ShardAssignment, error) {
	store, err := c.Swarm(ctx, swarmID)
	if err != nil {
		return nil, err
	}

	entry, err := store.Get(ctx, fmt.Sprintf("shards.%s", shardID))
	if err != nil {
		return nil, err
	}

	var assignment memory.ShardAssignment
	if err := json.Unmarshal(entry.Value, &assignment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal assignment: %w", err)
	}

	return &assignment, nil
}

// WriteShardResult writes a shard result
func (c *Client) WriteShardResult(ctx context.Context, result *memory.ShardResult) error {
	store, err := c.Swarm(ctx, result.SwarmID)
	if err != nil {
		return err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	_, err = store.Put(ctx, fmt.Sprintf("results.%s", result.ShardID), data)
	return err
}

// GetShardResult gets a shard result
func (c *Client) GetShardResult(ctx context.Context, swarmID, shardID string) (*memory.ShardResult, error) {
	store, err := c.Swarm(ctx, swarmID)
	if err != nil {
		return nil, err
	}

	entry, err := store.Get(ctx, fmt.Sprintf("results.%s", shardID))
	if err != nil {
		return nil, err
	}

	var result memory.ShardResult
	if err := json.Unmarshal(entry.Value, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// ListShardResults lists all shard results for a swarm
func (c *Client) ListShardResults(ctx context.Context, swarmID string) ([]*memory.ShardResult, error) {
	store, err := c.Swarm(ctx, swarmID)
	if err != nil {
		return nil, err
	}

	keys, err := store.List(ctx, "results.*")
	if err != nil {
		return nil, err
	}

	var results []*memory.ShardResult
	for _, key := range keys {
		entry, err := store.Get(ctx, key)
		if err != nil {
			continue
		}

		var result memory.ShardResult
		if err := json.Unmarshal(entry.Value, &result); err != nil {
			continue
		}
		results = append(results, &result)
	}

	return results, nil
}
