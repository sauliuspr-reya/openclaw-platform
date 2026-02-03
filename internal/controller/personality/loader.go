// Package personality handles loading and injecting agent personalities (IDENTITY.md, SOUL.md)
package personality

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/openclaw/openclaw-platform/api/v1alpha1"
)

const (
	// MountPath is where personality files are mounted in the container
	MountPath = "/opt/openclaw/personality"

	// IdentityFile is the filename for identity configuration
	IdentityFile = "IDENTITY.md"

	// SoulFile is the filename for soul configuration
	SoulFile = "SOUL.md"

	// PersonalityVolumeName is the volume name for personality mounts
	PersonalityVolumeName = "personality"
)

// Loader handles loading personality configuration from various sources
type Loader struct {
	client    client.Client
	namespace string
}

// NewLoader creates a new personality loader
func NewLoader(c client.Client, namespace string) *Loader {
	return &Loader{
		client:    c,
		namespace: namespace,
	}
}

// LoadedPersonality contains the loaded personality content
type LoadedPersonality struct {
	Identity string
	Soul     string
}

// Load loads personality content from the specified sources
func (l *Loader) Load(ctx context.Context, identity, soul *platformv1alpha1.PersonalitySource) (*LoadedPersonality, error) {
	result := &LoadedPersonality{}

	if identity != nil {
		content, err := l.loadSource(ctx, identity)
		if err != nil {
			return nil, fmt.Errorf("failed to load identity: %w", err)
		}
		result.Identity = content
	}

	if soul != nil {
		content, err := l.loadSource(ctx, soul)
		if err != nil {
			return nil, fmt.Errorf("failed to load soul: %w", err)
		}
		result.Soul = content
	}

	return result, nil
}

func (l *Loader) loadSource(ctx context.Context, source *platformv1alpha1.PersonalitySource) (string, error) {
	if source.Inline != "" {
		return source.Inline, nil
	}

	if source.ConfigMap != "" {
		return l.loadFromConfigMap(ctx, source.ConfigMap)
	}

	if source.Secret != "" {
		return l.loadFromSecret(ctx, source.Secret)
	}

	if source.Git != "" {
		return l.loadFromGit(ctx, source.Git)
	}

	return "", nil
}

func (l *Loader) loadFromConfigMap(ctx context.Context, ref string) (string, error) {
	namespace, name, key := parseRef(ref, l.namespace)

	cm := &corev1.ConfigMap{}
	if err := l.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm); err != nil {
		return "", fmt.Errorf("failed to get configmap %s/%s: %w", namespace, name, err)
	}

	if key == "" {
		// If no key specified, try common keys
		for _, k := range []string{"content", "data", "IDENTITY.md", "SOUL.md"} {
			if v, ok := cm.Data[k]; ok {
				return v, nil
			}
		}
		// Return first value if only one key
		if len(cm.Data) == 1 {
			for _, v := range cm.Data {
				return v, nil
			}
		}
		return "", fmt.Errorf("configmap %s/%s has no recognizable content key", namespace, name)
	}

	if v, ok := cm.Data[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("key %s not found in configmap %s/%s", key, namespace, name)
}

func (l *Loader) loadFromSecret(ctx context.Context, ref string) (string, error) {
	namespace, name, key := parseRef(ref, l.namespace)

	secret := &corev1.Secret{}
	if err := l.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, name, err)
	}

	if key == "" {
		// If no key specified, try common keys
		for _, k := range []string{"content", "data", "IDENTITY.md", "SOUL.md"} {
			if v, ok := secret.Data[k]; ok {
				return string(v), nil
			}
		}
		// Return first value if only one key
		if len(secret.Data) == 1 {
			for _, v := range secret.Data {
				return string(v), nil
			}
		}
		return "", fmt.Errorf("secret %s/%s has no recognizable content key", namespace, name)
	}

	if v, ok := secret.Data[key]; ok {
		return string(v), nil
	}
	return "", fmt.Errorf("key %s not found in secret %s/%s", key, namespace, name)
}

func (l *Loader) loadFromGit(ctx context.Context, gitRef string) (string, error) {
	// Git loading would require git operations
	// For now, return an error indicating it's not implemented
	// In production, this would clone/fetch the repo and read the file
	return "", fmt.Errorf("git source not yet implemented: %s", gitRef)
}

// parseRef parses a reference string like "configmap/name" or "configmap/namespace/name" or "configmap/namespace/name/key"
func parseRef(ref, defaultNamespace string) (namespace, name, key string) {
	parts := strings.Split(ref, "/")

	switch len(parts) {
	case 1:
		// Just name
		return defaultNamespace, parts[0], ""
	case 2:
		// type/name or namespace/name
		if parts[0] == "configmap" || parts[0] == "secret" {
			return defaultNamespace, parts[1], ""
		}
		return parts[0], parts[1], ""
	case 3:
		// type/namespace/name or namespace/name/key
		if parts[0] == "configmap" || parts[0] == "secret" {
			return parts[1], parts[2], ""
		}
		return parts[0], parts[1], parts[2]
	case 4:
		// type/namespace/name/key
		return parts[1], parts[2], parts[3]
	default:
		return defaultNamespace, ref, ""
	}
}

// VolumeConfig returns the volume and volumeMount configuration for personality injection
type VolumeConfig struct {
	Volume      corev1.Volume
	VolumeMount corev1.VolumeMount
}

// BuildVolumeConfig creates volume configuration for mounting personality from ConfigMaps
func BuildVolumeConfig(identity, soul *platformv1alpha1.PersonalitySource, namespace string) (*VolumeConfig, error) {
	// Determine sources
	var sources []corev1.VolumeProjection

	if identity != nil && identity.ConfigMap != "" {
		_, name, key := parseRef(identity.ConfigMap, namespace)
		projection := corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
			},
		}
		if key != "" {
			projection.ConfigMap.Items = []corev1.KeyToPath{
				{Key: key, Path: IdentityFile},
			}
		}
		sources = append(sources, projection)
	}

	if soul != nil && soul.ConfigMap != "" {
		_, name, key := parseRef(soul.ConfigMap, namespace)
		projection := corev1.VolumeProjection{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
			},
		}
		if key != "" {
			projection.ConfigMap.Items = []corev1.KeyToPath{
				{Key: key, Path: SoulFile},
			}
		}
		sources = append(sources, projection)
	}

	if identity != nil && identity.Secret != "" {
		_, name, key := parseRef(identity.Secret, namespace)
		projection := corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
			},
		}
		if key != "" {
			projection.Secret.Items = []corev1.KeyToPath{
				{Key: key, Path: IdentityFile},
			}
		}
		sources = append(sources, projection)
	}

	if soul != nil && soul.Secret != "" {
		_, name, key := parseRef(soul.Secret, namespace)
		projection := corev1.VolumeProjection{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
			},
		}
		if key != "" {
			projection.Secret.Items = []corev1.KeyToPath{
				{Key: key, Path: SoulFile},
			}
		}
		sources = append(sources, projection)
	}

	if len(sources) == 0 {
		return nil, nil
	}

	return &VolumeConfig{
		Volume: corev1.Volume{
			Name: PersonalityVolumeName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: sources,
				},
			},
		},
		VolumeMount: corev1.VolumeMount{
			Name:      PersonalityVolumeName,
			MountPath: MountPath,
			ReadOnly:  true,
		},
	}, nil
}

// BuildInlineConfigMap creates a ConfigMap for inline personality content
func BuildInlineConfigMap(name, namespace string, personality *LoadedPersonality) *corev1.ConfigMap {
	data := make(map[string]string)

	if personality.Identity != "" {
		data[IdentityFile] = personality.Identity
	}
	if personality.Soul != "" {
		data[SoulFile] = personality.Soul
	}

	if len(data) == 0 {
		return nil
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-personality",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "openclaw-agent",
				"app.kubernetes.io/instance":   name,
				"app.kubernetes.io/component":  "personality",
				"app.kubernetes.io/managed-by": "openclaw-operator",
			},
		},
		Data: data,
	}
}
