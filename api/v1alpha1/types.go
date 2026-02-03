// Package v1alpha1 contains API Schema definitions for the platform v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=platform.openclaw.io
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ContainerMode defines whether the agent runs in immutable or dynamic mode
// +kubebuilder:validation:Enum=immutable;dynamic
type ContainerMode string

const (
	// ContainerModeImmutable uses pre-baked images with fixed toolsets
	ContainerModeImmutable ContainerMode = "immutable"
	// ContainerModeDynamic allows runtime tool installation
	ContainerModeDynamic ContainerMode = "dynamic"
)

// ImmutableProfile defines pre-baked image profiles
// +kubebuilder:validation:Enum=minimal;python-ml;devops;web;data;security
type ImmutableProfile string

const (
	ProfileMinimal  ImmutableProfile = "minimal"
	ProfilePythonML ImmutableProfile = "python-ml"
	ProfileDevOps   ImmutableProfile = "devops"
	ProfileWeb      ImmutableProfile = "web"
	ProfileData     ImmutableProfile = "data"
	ProfileSecurity ImmutableProfile = "security"
)

// SwarmRole defines the agent's role in a swarm
// +kubebuilder:validation:Enum=worker;orchestrator
type SwarmRole string

const (
	SwarmRoleWorker       SwarmRole = "worker"
	SwarmRoleOrchestrator SwarmRole = "orchestrator"
)

// AccessMode defines read/write permissions for memory tiers
// +kubebuilder:validation:Enum=readonly;readwrite
type AccessMode string

const (
	AccessModeReadOnly  AccessMode = "readonly"
	AccessModeReadWrite AccessMode = "readwrite"
)

// OpenClawAgentSpec defines the desired state of OpenClawAgent
type OpenClawAgentSpec struct {
	// ContainerMode specifies whether to use immutable or dynamic containers
	// +kubebuilder:default=immutable
	// +optional
	ContainerMode ContainerMode `json:"containerMode,omitempty"`

	// ImmutableProfile specifies which pre-baked image to use (for immutable mode)
	// +kubebuilder:default=minimal
	// +optional
	ImmutableProfile ImmutableProfile `json:"immutableProfile,omitempty"`

	// Runtime configuration for the agent container
	// +optional
	Runtime *RuntimeSpec `json:"runtime,omitempty"`

	// Identity defines the agent's identity configuration (IDENTITY.md)
	// +optional
	Identity *PersonalitySource `json:"identity,omitempty"`

	// Soul defines the agent's behavioral traits (SOUL.md)
	// +optional
	Soul *PersonalitySource `json:"soul,omitempty"`

	// Memory defines the hierarchical memory configuration
	// +optional
	Memory *MemorySpec `json:"memory,omitempty"`

	// Swarm defines swarm participation configuration
	// +optional
	Swarm *SwarmSpec `json:"swarm,omitempty"`

	// Mutability defines which paths are writable (for dynamic mode)
	// +optional
	Mutability *MutabilitySpec `json:"mutability,omitempty"`

	// Capabilities lists the capabilities granted to this agent
	// +optional
	Capabilities []string `json:"capabilities,omitempty"`

	// Resources defines compute resources for the agent
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// RuntimeSpec defines the container runtime configuration
type RuntimeSpec struct {
	// Image is the container image to use
	// +optional
	Image string `json:"image,omitempty"`

	// ImagePullPolicy defines when to pull the image
	// +kubebuilder:default=IfNotPresent
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Command overrides the container entrypoint
	// +optional
	Command []string `json:"command,omitempty"`

	// Args provides arguments to the entrypoint
	// +optional
	Args []string `json:"args,omitempty"`

	// Env defines environment variables
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// PersonalitySource defines where to load personality configuration from
type PersonalitySource struct {
	// ConfigMap references a ConfigMap containing the personality
	// Format: configmap/name or configmap/namespace/name
	// +optional
	ConfigMap string `json:"configMap,omitempty"`

	// Secret references a Secret containing the personality
	// Format: secret/name or secret/namespace/name
	// +optional
	Secret string `json:"secret,omitempty"`

	// Git references a file in a Git repository
	// Format: git://repo/path/to/file.md
	// +optional
	Git string `json:"git,omitempty"`

	// Inline contains the personality content directly
	// +optional
	Inline string `json:"inline,omitempty"`
}

// MemorySpec defines the hierarchical memory configuration
type MemorySpec struct {
	// NATSUrl is the URL of the NATS server
	// +kubebuilder:default="nats://nats.openclaw.svc:4222"
	NATSUrl string `json:"natsUrl"`

	// Company defines company-wide memory access
	// +optional
	Company *CompanyMemorySpec `json:"company,omitempty"`

	// Group defines group memory access
	// +optional
	Group *GroupMemorySpec `json:"group,omitempty"`

	// Swarm defines swarm memory access
	// +optional
	Swarm *SwarmMemorySpec `json:"swarm,omitempty"`

	// Individual defines individual agent memory
	// +optional
	Individual *IndividualMemorySpec `json:"individual,omitempty"`
}

// CompanyMemorySpec defines company-wide memory configuration
type CompanyMemorySpec struct {
	// Enabled controls whether company memory is accessible
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Mode defines read/write access
	// +kubebuilder:default=readonly
	// +optional
	Mode AccessMode `json:"mode,omitempty"`

	// Buckets lists specific company buckets to access
	// Defaults to all company buckets if empty
	// +optional
	Buckets []string `json:"buckets,omitempty"`
}

// GroupMemorySpec defines group memory configuration
type GroupMemorySpec struct {
	// Enabled controls whether group memory is accessible
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Name is the group this agent belongs to
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Mode defines read/write access
	// +kubebuilder:default=readwrite
	// +optional
	Mode AccessMode `json:"mode,omitempty"`

	// CrossGroupRead lists other groups this agent can read from
	// +optional
	CrossGroupRead []string `json:"crossGroupRead,omitempty"`
}

// SwarmMemorySpec defines swarm memory configuration
type SwarmMemorySpec struct {
	// Enabled controls whether swarm memory is accessible
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// AutoJoin automatically joins swarm memory when assigned to a swarm
	// +kubebuilder:default=true
	// +optional
	AutoJoin bool `json:"autoJoin,omitempty"`

	// CurrentSwarmID is set by the operator when the agent joins a swarm
	// +optional
	CurrentSwarmID string `json:"currentSwarmID,omitempty"`
}

// IndividualMemorySpec defines individual agent memory configuration
type IndividualMemorySpec struct {
	// Enabled controls whether individual memory is accessible
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Bucket is the bucket name for this agent's individual memory
	// Defaults to memory.agent.{agent-name} if empty
	// +optional
	Bucket string `json:"bucket,omitempty"`

	// TTL defines how long individual memory persists after agent termination
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// SwarmSpec defines swarm participation configuration
type SwarmSpec struct {
	// Enabled controls whether this agent can participate in swarms
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Role defines whether this agent is a worker or orchestrator
	// +kubebuilder:default=worker
	// +optional
	Role SwarmRole `json:"role,omitempty"`

	// Groups lists which groups' swarms this agent can be recruited for
	// +optional
	Groups []string `json:"groups,omitempty"`

	// MaxShards is the maximum number of shards this agent can process (for workers)
	// +kubebuilder:default=1
	// +optional
	MaxShards int `json:"maxShards,omitempty"`

	// MaxWorkers is the maximum workers to coordinate (for orchestrators)
	// +kubebuilder:default=10
	// +optional
	MaxWorkers int `json:"maxWorkers,omitempty"`

	// ShardTimeout is the default timeout for shard processing
	// +optional
	ShardTimeout *metav1.Duration `json:"shardTimeout,omitempty"`
}

// MutabilitySpec defines writable paths for dynamic containers
type MutabilitySpec struct {
	// Enabled controls whether mutability is allowed
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// ToolPaths lists paths where tools can be installed
	// +kubebuilder:default={"/opt/openclaw/tools","/opt/openclaw/runtime"}
	// +optional
	ToolPaths []string `json:"toolPaths,omitempty"`

	// PersistTools controls whether installed tools persist across restarts
	// +kubebuilder:default=false
	// +optional
	PersistTools bool `json:"persistTools,omitempty"`

	// ToolCachePVC is the PVC name for persistent tool storage
	// +optional
	ToolCachePVC string `json:"toolCachePVC,omitempty"`
}

// OpenClawAgentStatus defines the observed state of OpenClawAgent
type OpenClawAgentStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Initializing;Ready;Running;Failed;Terminated
	Phase AgentPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// PodName is the name of the underlying Pod
	// +optional
	PodName string `json:"podName,omitempty"`

	// Memory status
	// +optional
	Memory *MemoryStatus `json:"memory,omitempty"`

	// Swarm status
	// +optional
	Swarm *SwarmStatus `json:"swarm,omitempty"`

	// StartTime is when the agent started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// LastHeartbeat is the last time the agent reported health
	// +optional
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`
}

// AgentPhase represents the lifecycle phase of an agent
// +kubebuilder:validation:Enum=Pending;Initializing;Ready;Running;Failed;Terminated
type AgentPhase string

const (
	AgentPhasePending      AgentPhase = "Pending"
	AgentPhaseInitializing AgentPhase = "Initializing"
	AgentPhaseReady        AgentPhase = "Ready"
	AgentPhaseRunning      AgentPhase = "Running"
	AgentPhaseFailed       AgentPhase = "Failed"
	AgentPhaseTerminated   AgentPhase = "Terminated"
)

// MemoryStatus represents the current memory connection status
type MemoryStatus struct {
	// Connected indicates whether the agent is connected to NATS
	Connected bool `json:"connected"`

	// CompanyAccess indicates company memory is accessible
	// +optional
	CompanyAccess bool `json:"companyAccess,omitempty"`

	// GroupAccess indicates group memory is accessible
	// +optional
	GroupAccess bool `json:"groupAccess,omitempty"`

	// SwarmAccess indicates swarm memory is accessible
	// +optional
	SwarmAccess bool `json:"swarmAccess,omitempty"`

	// IndividualBucket is the individual memory bucket name
	// +optional
	IndividualBucket string `json:"individualBucket,omitempty"`
}

// SwarmStatus represents the current swarm participation status
type SwarmStatus struct {
	// Active indicates whether the agent is currently in a swarm
	Active bool `json:"active"`

	// SwarmID is the current swarm ID (if active)
	// +optional
	SwarmID string `json:"swarmID,omitempty"`

	// Role is the current role in the swarm
	// +optional
	Role SwarmRole `json:"role,omitempty"`

	// ShardID is the current shard being processed (for workers)
	// +optional
	ShardID string `json:"shardID,omitempty"`

	// WorkerCount is the number of workers (for orchestrators)
	// +optional
	WorkerCount int `json:"workerCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.containerMode`
// +kubebuilder:printcolumn:name="Profile",type=string,JSONPath=`.spec.immutableProfile`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Swarm",type=string,JSONPath=`.status.swarm.swarmID`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenClawAgent is the Schema for the openclawagents API
type OpenClawAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenClawAgentSpec   `json:"spec,omitempty"`
	Status OpenClawAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenClawAgentList contains a list of OpenClawAgent
type OpenClawAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenClawAgent `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Orchestrator",type=string,JSONPath=`.spec.orchestratorRef`
// +kubebuilder:printcolumn:name="Workers",type=integer,JSONPath=`.status.workerCount`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenClawSwarm represents a coordinated group of agents working on a task
type OpenClawSwarm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenClawSwarmSpec   `json:"spec,omitempty"`
	Status OpenClawSwarmStatus `json:"status,omitempty"`
}

// OpenClawSwarmSpec defines the desired state of a swarm
type OpenClawSwarmSpec struct {
	// Goal describes what the swarm is trying to achieve
	Goal string `json:"goal"`

	// OrchestratorRef references the orchestrator agent
	// +optional
	OrchestratorRef string `json:"orchestratorRef,omitempty"`

	// WorkerSelector selects agents to participate as workers
	// +optional
	WorkerSelector *metav1.LabelSelector `json:"workerSelector,omitempty"`

	// MinWorkers is the minimum number of workers required
	// +kubebuilder:default=1
	// +optional
	MinWorkers int `json:"minWorkers,omitempty"`

	// MaxWorkers is the maximum number of workers allowed
	// +kubebuilder:default=10
	// +optional
	MaxWorkers int `json:"maxWorkers,omitempty"`

	// Timeout is the maximum duration for the swarm task
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Group restricts which group's agents can participate
	// +optional
	Group string `json:"group,omitempty"`

	// Constraints for task execution
	// +optional
	Constraints map[string]string `json:"constraints,omitempty"`
}

// OpenClawSwarmStatus defines the observed state of a swarm
type OpenClawSwarmStatus struct {
	// Phase represents the swarm lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Active;Completing;Completed;Failed
	Phase SwarmPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// WorkerCount is the current number of active workers
	WorkerCount int `json:"workerCount"`

	// Workers lists the worker agent names
	// +optional
	Workers []string `json:"workers,omitempty"`

	// ShardCount is the total number of shards
	ShardCount int `json:"shardCount"`

	// CompletedShards is the number of completed shards
	CompletedShards int `json:"completedShards"`

	// FailedShards is the number of failed shards
	FailedShards int `json:"failedShards"`

	// StartTime is when the swarm started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the swarm completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Result contains the merged result (if completed)
	// +optional
	Result string `json:"result,omitempty"`
}

// SwarmPhase represents the lifecycle phase of a swarm
// +kubebuilder:validation:Enum=Pending;Active;Completing;Completed;Failed
type SwarmPhase string

const (
	SwarmPhasePending    SwarmPhase = "Pending"
	SwarmPhaseActive     SwarmPhase = "Active"
	SwarmPhaseCompleting SwarmPhase = "Completing"
	SwarmPhaseCompleted  SwarmPhase = "Completed"
	SwarmPhaseFailed     SwarmPhase = "Failed"
)

// +kubebuilder:object:root=true

// OpenClawSwarmList contains a list of OpenClawSwarm
type OpenClawSwarmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenClawSwarm `json:"items"`
}
