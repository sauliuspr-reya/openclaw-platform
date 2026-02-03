// Package controller implements the OpenClaw operator controllers
package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/openclaw/openclaw-platform/api/v1alpha1"
	"github.com/openclaw/openclaw-platform/internal/controller/personality"
)

const (
	finalizerName = "platform.openclaw.io/finalizer"
)

// AgentReconciler reconciles an OpenClawAgent object
type AgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.openclaw.io,resources=openclawagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.openclaw.io,resources=openclawagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.openclaw.io,resources=openclawagents/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles the reconciliation loop for OpenClawAgent
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the OpenClawAgent instance
	agent := &platformv1alpha1.OpenClawAgent{}
	if err := r.Get(ctx, req.NamespacedName, agent); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("OpenClawAgent resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get OpenClawAgent")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !agent.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, agent)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(agent, finalizerName) {
		controllerutil.AddFinalizer(agent, finalizerName)
		if err := r.Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile the agent
	return r.reconcileAgent(ctx, agent)
}

func (r *AgentReconciler) handleDeletion(ctx context.Context, agent *platformv1alpha1.OpenClawAgent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(agent, finalizerName) {
		// Cleanup: delete the pod
		pod := &corev1.Pod{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: agent.Namespace,
			Name:      agent.Name,
		}, pod); err == nil {
			if err := r.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete pod")
				return ctrl.Result{}, err
			}
		}

		// Cleanup: delete personality ConfigMap if created
		cm := &corev1.ConfigMap{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: agent.Namespace,
			Name:      agent.Name + "-personality",
		}, cm); err == nil {
			if err := r.Delete(ctx, cm); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete personality configmap")
				return ctrl.Result{}, err
			}
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(agent, finalizerName)
		if err := r.Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AgentReconciler) reconcileAgent(ctx context.Context, agent *platformv1alpha1.OpenClawAgent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Update status to Initializing
	if agent.Status.Phase == "" {
		agent.Status.Phase = platformv1alpha1.AgentPhaseInitializing
		if err := r.Status().Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile personality ConfigMap (if using inline)
	if err := r.reconcilePersonalityConfigMap(ctx, agent); err != nil {
		logger.Error(err, "Failed to reconcile personality configmap")
		return ctrl.Result{}, err
	}

	// Reconcile Pod
	pod, err := r.reconcilePod(ctx, agent)
	if err != nil {
		logger.Error(err, "Failed to reconcile pod")
		r.updateStatus(ctx, agent, platformv1alpha1.AgentPhaseFailed, err.Error())
		return ctrl.Result{}, err
	}

	// Update status based on pod state
	return r.updateStatusFromPod(ctx, agent, pod)
}

func (r *AgentReconciler) reconcilePersonalityConfigMap(ctx context.Context, agent *platformv1alpha1.OpenClawAgent) error {
	// Check if we need to create a ConfigMap for inline personality
	needsConfigMap := false
	if agent.Spec.Identity != nil && agent.Spec.Identity.Inline != "" {
		needsConfigMap = true
	}
	if agent.Spec.Soul != nil && agent.Spec.Soul.Inline != "" {
		needsConfigMap = true
	}

	if !needsConfigMap {
		return nil
	}

	// Load personality content
	loader := personality.NewLoader(r.Client, agent.Namespace)
	loaded, err := loader.Load(ctx, agent.Spec.Identity, agent.Spec.Soul)
	if err != nil {
		return err
	}

	cm := personality.BuildInlineConfigMap(agent.Name, agent.Namespace, loaded)
	if cm == nil {
		return nil
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(agent, cm, r.Scheme); err != nil {
		return err
	}

	// Create or update
	existing := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cm.Namespace, Name: cm.Name}, existing); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, cm)
		}
		return err
	}

	existing.Data = cm.Data
	return r.Update(ctx, existing)
}

func (r *AgentReconciler) reconcilePod(ctx context.Context, agent *platformv1alpha1.OpenClawAgent) (*corev1.Pod, error) {
	// Check if pod exists
	existing := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: agent.Namespace,
		Name:      agent.Name,
	}, existing)

	if err == nil {
		// Pod exists, return it
		return existing, nil
	}

	if !errors.IsNotFound(err) {
		return nil, err
	}

	// Create new pod
	pod := r.buildPod(agent)

	// Set owner reference
	if err := controllerutil.SetControllerReference(agent, pod, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, pod); err != nil {
		return nil, err
	}

	return pod, nil
}

func (r *AgentReconciler) buildPod(agent *platformv1alpha1.OpenClawAgent) *corev1.Pod {
	// Determine image based on mode and profile
	image := r.getImage(agent)

	// Build volumes
	volumes := r.buildVolumes(agent)

	// Build container
	container := r.buildContainer(agent, image, volumes)

	// Build pod spec
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "openclaw-agent",
				"app.kubernetes.io/instance":   agent.Name,
				"app.kubernetes.io/managed-by": "openclaw-operator",
				"platform.openclaw.io/mode":    string(agent.Spec.ContainerMode),
				"platform.openclaw.io/profile": string(agent.Spec.ImmutableProfile),
			},
		},
		Spec: corev1.PodSpec{
			Containers:         []corev1.Container{container},
			Volumes:            volumes,
			RestartPolicy:      corev1.RestartPolicyNever,
			ServiceAccountName: "openclaw-agent",
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: boolPtr(true),
				RunAsUser:    int64Ptr(1000),
				RunAsGroup:   int64Ptr(1000),
				FSGroup:      int64Ptr(1000),
			},
		},
	}

	// Add group label if specified
	if agent.Spec.Memory != nil && agent.Spec.Memory.Group != nil {
		pod.Labels["platform.openclaw.io/group"] = agent.Spec.Memory.Group.Name
	}

	// Add swarm labels if enabled
	if agent.Spec.Swarm != nil && agent.Spec.Swarm.Enabled {
		pod.Labels["platform.openclaw.io/swarm-enabled"] = "true"
		pod.Labels["platform.openclaw.io/swarm-role"] = string(agent.Spec.Swarm.Role)
	}

	return pod
}

func (r *AgentReconciler) getImage(agent *platformv1alpha1.OpenClawAgent) string {
	// If runtime image is specified, use it
	if agent.Spec.Runtime != nil && agent.Spec.Runtime.Image != "" {
		return agent.Spec.Runtime.Image
	}

	// Otherwise, use profile-based image
	registry := "ghcr.io/openclaw"
	profile := agent.Spec.ImmutableProfile
	if profile == "" {
		profile = platformv1alpha1.ProfileMinimal
	}

	return fmt.Sprintf("%s/agent:%s", registry, profile)
}

func (r *AgentReconciler) buildVolumes(agent *platformv1alpha1.OpenClawAgent) []corev1.Volume {
	var volumes []corev1.Volume

	// Workspace volume (always present)
	volumes = append(volumes, corev1.Volume{
		Name: "workspace",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	// Tool volumes for dynamic mode
	if agent.Spec.ContainerMode == platformv1alpha1.ContainerModeDynamic {
		if agent.Spec.Mutability != nil && agent.Spec.Mutability.PersistTools && agent.Spec.Mutability.ToolCachePVC != "" {
			// Use PVC for persistent tools
			volumes = append(volumes, corev1.Volume{
				Name: "tools",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: agent.Spec.Mutability.ToolCachePVC,
					},
				},
			})
		} else {
			// Use emptyDir for ephemeral tools
			volumes = append(volumes, corev1.Volume{
				Name: "tools",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
		}
	}

	// Personality volume
	personalityVolume := r.buildPersonalityVolume(agent)
	if personalityVolume != nil {
		volumes = append(volumes, *personalityVolume)
	}

	return volumes
}

func (r *AgentReconciler) buildPersonalityVolume(agent *platformv1alpha1.OpenClawAgent) *corev1.Volume {
	var sources []corev1.VolumeProjection

	// Check for inline personality (uses generated ConfigMap)
	hasInline := (agent.Spec.Identity != nil && agent.Spec.Identity.Inline != "") ||
		(agent.Spec.Soul != nil && agent.Spec.Soul.Inline != "")

	if hasInline {
		sources = append(sources, corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: agent.Name + "-personality",
				},
			},
		})
	}

	// Check for ConfigMap-based personality
	if agent.Spec.Identity != nil && agent.Spec.Identity.ConfigMap != "" && agent.Spec.Identity.Inline == "" {
		sources = append(sources, corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: agent.Spec.Identity.ConfigMap,
				},
				Items: []corev1.KeyToPath{
					{Key: "content", Path: personality.IdentityFile},
				},
			},
		})
	}

	if agent.Spec.Soul != nil && agent.Spec.Soul.ConfigMap != "" && agent.Spec.Soul.Inline == "" {
		sources = append(sources, corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: agent.Spec.Soul.ConfigMap,
				},
				Items: []corev1.KeyToPath{
					{Key: "content", Path: personality.SoulFile},
				},
			},
		})
	}

	if len(sources) == 0 {
		return nil
	}

	return &corev1.Volume{
		Name: personality.PersonalityVolumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: sources,
			},
		},
	}
}

func (r *AgentReconciler) buildContainer(agent *platformv1alpha1.OpenClawAgent, image string, volumes []corev1.Volume) corev1.Container {
	// Volume mounts
	var mounts []corev1.VolumeMount

	// Workspace mount
	mounts = append(mounts, corev1.VolumeMount{
		Name:      "workspace",
		MountPath: "/workspace",
	})

	// Tool mounts for dynamic mode
	if agent.Spec.ContainerMode == platformv1alpha1.ContainerModeDynamic {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "tools",
			MountPath: "/opt/openclaw/tools",
		})
	}

	// Personality mount
	for _, v := range volumes {
		if v.Name == personality.PersonalityVolumeName {
			mounts = append(mounts, corev1.VolumeMount{
				Name:      personality.PersonalityVolumeName,
				MountPath: personality.MountPath,
				ReadOnly:  true,
			})
			break
		}
	}

	// Environment variables
	env := r.buildEnv(agent)

	// Resources
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}
	if agent.Spec.Resources != nil {
		if agent.Spec.Resources.Requests != nil {
			resources.Requests = agent.Spec.Resources.Requests
		}
		if agent.Spec.Resources.Limits != nil {
			resources.Limits = agent.Spec.Resources.Limits
		}
	}

	container := corev1.Container{
		Name:         "agent",
		Image:        image,
		VolumeMounts: mounts,
		Env:          env,
		Resources:    resources,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: boolPtr(false),
			ReadOnlyRootFilesystem:   boolPtr(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	}

	// Override command/args if specified
	if agent.Spec.Runtime != nil {
		if len(agent.Spec.Runtime.Command) > 0 {
			container.Command = agent.Spec.Runtime.Command
		}
		if len(agent.Spec.Runtime.Args) > 0 {
			container.Args = agent.Spec.Runtime.Args
		}
		if len(agent.Spec.Runtime.Env) > 0 {
			container.Env = append(container.Env, agent.Spec.Runtime.Env...)
		}
	}

	return container
}

func (r *AgentReconciler) buildEnv(agent *platformv1alpha1.OpenClawAgent) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: "OPENCLAW_AGENT_NAME", Value: agent.Name},
		{Name: "OPENCLAW_AGENT_NAMESPACE", Value: agent.Namespace},
		{Name: "OPENCLAW_CONTAINER_MODE", Value: string(agent.Spec.ContainerMode)},
		{Name: "OPENCLAW_PROFILE", Value: string(agent.Spec.ImmutableProfile)},
	}

	// Memory configuration
	if agent.Spec.Memory != nil {
		env = append(env, corev1.EnvVar{
			Name:  "OPENCLAW_NATS_URL",
			Value: agent.Spec.Memory.NATSUrl,
		})

		if agent.Spec.Memory.Group != nil {
			env = append(env, corev1.EnvVar{
				Name:  "OPENCLAW_GROUP",
				Value: agent.Spec.Memory.Group.Name,
			})
		}

		if agent.Spec.Memory.Individual != nil && agent.Spec.Memory.Individual.Bucket != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OPENCLAW_INDIVIDUAL_BUCKET",
				Value: agent.Spec.Memory.Individual.Bucket,
			})
		}
	}

	// Swarm configuration
	if agent.Spec.Swarm != nil && agent.Spec.Swarm.Enabled {
		env = append(env, corev1.EnvVar{
			Name:  "OPENCLAW_SWARM_ENABLED",
			Value: "true",
		})
		env = append(env, corev1.EnvVar{
			Name:  "OPENCLAW_SWARM_ROLE",
			Value: string(agent.Spec.Swarm.Role),
		})
	}

	// Capabilities
	if len(agent.Spec.Capabilities) > 0 {
		caps := ""
		for i, c := range agent.Spec.Capabilities {
			if i > 0 {
				caps += ","
			}
			caps += c
		}
		env = append(env, corev1.EnvVar{
			Name:  "OPENCLAW_CAPABILITIES",
			Value: caps,
		})
	}

	return env
}

func (r *AgentReconciler) updateStatusFromPod(ctx context.Context, agent *platformv1alpha1.OpenClawAgent, pod *corev1.Pod) (ctrl.Result, error) {
	// Map pod phase to agent phase
	var phase platformv1alpha1.AgentPhase
	switch pod.Status.Phase {
	case corev1.PodPending:
		phase = platformv1alpha1.AgentPhasePending
	case corev1.PodRunning:
		phase = platformv1alpha1.AgentPhaseRunning
	case corev1.PodSucceeded:
		phase = platformv1alpha1.AgentPhaseTerminated
	case corev1.PodFailed:
		phase = platformv1alpha1.AgentPhaseFailed
	default:
		phase = platformv1alpha1.AgentPhasePending
	}

	// Update status
	agent.Status.Phase = phase
	agent.Status.PodName = pod.Name

	if pod.Status.StartTime != nil {
		agent.Status.StartTime = pod.Status.StartTime
	}

	// Update memory status
	if agent.Spec.Memory != nil {
		agent.Status.Memory = &platformv1alpha1.MemoryStatus{
			Connected: phase == platformv1alpha1.AgentPhaseRunning,
		}
		if agent.Spec.Memory.Company != nil && agent.Spec.Memory.Company.Enabled {
			agent.Status.Memory.CompanyAccess = true
		}
		if agent.Spec.Memory.Group != nil && agent.Spec.Memory.Group.Enabled {
			agent.Status.Memory.GroupAccess = true
		}
		if agent.Spec.Memory.Swarm != nil && agent.Spec.Memory.Swarm.Enabled {
			agent.Status.Memory.SwarmAccess = true
		}
		if agent.Spec.Memory.Individual != nil {
			bucket := agent.Spec.Memory.Individual.Bucket
			if bucket == "" {
				bucket = fmt.Sprintf("memory.agent.%s", agent.Name)
			}
			agent.Status.Memory.IndividualBucket = bucket
		}
	}

	// Update swarm status
	if agent.Spec.Swarm != nil && agent.Spec.Swarm.Enabled {
		agent.Status.Swarm = &platformv1alpha1.SwarmStatus{
			Active: false,
			Role:   agent.Spec.Swarm.Role,
		}
	}

	if err := r.Status().Update(ctx, agent); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue if pod is not in terminal state
	if phase == platformv1alpha1.AgentPhasePending || phase == platformv1alpha1.AgentPhaseRunning {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *AgentReconciler) updateStatus(ctx context.Context, agent *platformv1alpha1.OpenClawAgent, phase platformv1alpha1.AgentPhase, message string) {
	agent.Status.Phase = phase
	now := metav1.Now()
	agent.Status.Conditions = append(agent.Status.Conditions, metav1.Condition{
		Type:               string(phase),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             string(phase),
		Message:            message,
	})
	r.Status().Update(ctx, agent)
}

// SetupWithManager sets up the controller with the Manager
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.OpenClawAgent{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}
