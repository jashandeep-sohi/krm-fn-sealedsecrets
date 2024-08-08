package seal

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/common"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/bitnami-labs/sealed-secrets/pkg/kubeseal"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/kustomize/api/hasher"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	ssv1alpha1 "github.com/bitnami-labs/sealed-secrets/pkg/apis/sealedsecrets/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OriginalNameAnnotation = common.AnnotationPrefix + "originalName"
)

func Process(rl *fn.ResourceList) (bool, error) {
	if rl.FunctionConfig == nil {
		return false, fn.ErrMissingFnConfig{}
	}

	cm := &ConfigMap{}
	if err := rl.FunctionConfig.As(cm); err != nil {
		return false, fmt.Errorf("failed to parse function config: %w", err)
	}

	if cm.Data.Cert == nil || *cm.Data.Cert == "" {
		return false, fmt.Errorf("cert is needed encrypt Secrets")
	}

	pubKey, err := kubeseal.ParseKey(strings.NewReader(*cm.Data.Cert))
	if err != nil {
		return false, fmt.Errorf("failed to parse certificate: %w", err)
	}

	isCandidateSecret := func(warn bool) func(o *fn.KubeObject) bool {
		return func(o *fn.KubeObject) bool {
			if !o.IsGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret")) {
				return false
			}

			if o.IsLocalConfig() {
				if warn {
					rl.Results.Warningf("skipped %s (%s is set)", o.ShortString(), fn.KptLocalConfig)
				}
				return false
			}

			if o.GetName() == "" {
				if warn {
					rl.Results.Warningf("skipped %s (it does not have a name)", o.ShortString())
				}
				return false
			}

			s := &corev1.Secret{}
			err := o.As(s)
			if err != nil {
				if warn {
					rl.Results.Warningf("skipped %s (failed to parse)", o.ShortString())
				}
				return false
			}

			if scope := ssv1alpha1.SecretScope(s); scope != ssv1alpha1.ClusterWideScope && s.GetNamespace() == "" {
				if warn {
					rl.Results.Warningf("skipped %s (namespace not set but scope is '%s')", o.ShortString(), scope.String())
				}
				return false
			}

			return true
		}
	}

	secrets := rl.Items.Where(isCandidateSecret(true))

	if cm.Data.Prune {
		rl.Items = rl.Items.WhereNot(isCandidateSecret(false))
	}

	h := &hasher.Hasher{}

	for _, secret := range secrets {
		err := sealOne(rl, secret, pubKey, cm, h)
		if err != nil {
			return false, err
		}
	}

	if cm.Data.NameRefKustomizationDir != nil && *cm.Data.NameRefKustomizationDir != "" {
		err = generateNameRefKustomization(rl, *cm.Data.NameRefKustomizationDir)
		if err != nil {
			return false, err
		}
	}

	err = common.NormalizeIndexAnnotation(rl)
	if err != nil {
		return false, err
	}

	return true, nil
}

func generateNameRefKustomization(rl *fn.ResourceList, dir string) error {
	// Clean out all old files at the desired Kustomization dir
	rl.Items = rl.Items.Where(func(o *fn.KubeObject) bool {
		return !strings.HasPrefix(o.GetAnnotation(fn.PathAnnotation), dir)
	})

	ssecrets := rl.Items.Where(func(o *fn.KubeObject) bool {
		if !o.IsGroupVersionKind(ssv1alpha1.SchemeGroupVersion.WithKind("SealedSecret")) {
			return false
		}

		_, ok := o.GetAnnotations()[OriginalNameAnnotation]

		return ok
	})

	k := fn.NewEmptyKubeObject()

	err := errors.Join(
		k.SetAPIVersion("kustomize.config.k8s.io/v1beta1"),
		k.SetKind("Kustomization"),
		k.SetAnnotation(fn.PathAnnotation, path.Join(dir, "kustomization.yaml")),
	)
	if err != nil {
		return err
	}

	for _, ss := range ssecrets {
		s := fn.NewEmptyKubeObject()
		t := fn.NewEmptyKubeObject()

		originalName := ss.GetAnnotation(OriginalNameAnnotation)

		err := errors.Join(
			s.SetAPIVersion("v1"),
			s.SetKind("Secret"),
			s.SetName(originalName),
			s.SetNamespace(ss.GetNamespace()),
			s.SetAnnotation(fn.PathAnnotation, path.Join(dir, "resources.yaml")),
			s.SetAnnotation(fn.KptLocalConfig, "true"),
		)
		if err != nil {
			return err
		}

		err = rl.UpsertObjectToItems(s, neverEqual, true)
		if err != nil {
			return err
		}

		err = errors.Join(
			t.SetAPIVersion("builtin"),
			t.SetKind("PatchTransformer"),
			t.SetName(ss.GetName()),
			t.SetNamespace(ss.GetNamespace()),
			t.SetAnnotation(fn.PathAnnotation, path.Join(dir, "transformers.yaml")),
			t.SetNestedString(fmt.Sprintf(`[{"op": "replace", "path": "/metadata/name", "value": "%s"}]`, ss.GetName()), "patch"),
			t.SetNestedString(s.GetId().Group, "target", "group"),
			t.SetNestedString(s.GetId().Version, "target", "version"),
			t.SetNestedString(s.GetKind(), "target", "kind"),
			t.SetNestedString(s.GetName(), "target", "name"),
			t.SetNestedString(s.GetNamespace(), "target", "namespace"),
		)
		if err != nil {
			return err
		}

		err = rl.UpsertObjectToItems(t, neverEqual, true)
		if err != nil {
			return err
		}
	}

	if ssecrets.Len() > 0 {
		err = errors.Join(
			k.SetNestedStringSlice([]string{"resources.yaml"}, "resources"),
			k.SetNestedStringSlice([]string{"transformers.yaml"}, "transformers"),
		)

		if err != nil {
			return err
		}
	}

	err = rl.UpsertObjectToItems(k, neverEqual, true)
	if err != nil {
		return err
	}

	rl.Results.Infof("Created name reference Kustomization '%s'. Add this to your Kustomization to update Secret name references.", dir)

	return nil
}

func neverEqual(o, another *fn.KubeObject) bool {
	return false
}

func sealOne(rl *fn.ResourceList, secret *fn.KubeObject, pubKey *rsa.PublicKey, cm *ConfigMap, h *hasher.Hasher) error {
	s := &corev1.Secret{}
	err := secret.As(s)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}
	sanitizeMetadata(s)
	removeInternalAnnotations(s)

	if cm.Data.NameSuffixHash {
		hash, err := computeHash(s, h)
		if err != nil {
			return err
		}
		name := fmt.Sprintf("%s-%s", secret.GetName(), hash)
		s.SetName(name)
	}

	sealedsecret, err := ssv1alpha1.NewSealedSecret(scheme.Codecs, pubKey, s)
	if err != nil {
		return fmt.Errorf("failed to create SealedSecret: %w", err)
	}

	ss, err := fn.NewFromTypedObject(sealedsecret)
	if err != nil {
		return fmt.Errorf("failed to convert SealedSecret to KubeObject: %w", err)
	}
	err = errors.Join(
		ss.SetAPIVersion(ssv1alpha1.SchemeGroupVersion.String()),
		ss.SetKind("SealedSecret"),
		ss.SetAnnotation(OriginalNameAnnotation, secret.GetName()),
	)
	if err != nil {
		return err
	}

	if secret.PathAnnotation() != "" {
		err = ss.SetAnnotation(fn.PathAnnotation, secret.PathAnnotation())
		if err != nil {
			return err
		}
	}

	err = rl.UpsertObjectToItems(ss, isSameObj, true)
	if err != nil {
		return fmt.Errorf("failed to add SealedSecret to output: %w", err)
	}

	rl.Results.Infof("sealed %s (path=%s)", ss.ShortString(), ss.PathAnnotation())

	if cm.Data.EmmitKustomizeStubSecrets {
		r, err := createStubSecret(s, secret.GetName(), ss.GetName(), ss.PathAnnotation())
		if err != nil {
			return err
		}

		err = rl.UpsertObjectToItems(r, isSameObj, true)
		if err != nil {
			return fmt.Errorf("failed to add stub Secret to output: %w", err)
		}

		rl.Results.Infof("added stub Secret for %s (path=%s)", ss.ShortString(), ss.PathAnnotation())
	}

	return nil
}

func createStubSecret(s *corev1.Secret, originalName string, newName string, pathAnnotationValue string) (*resource.Resource, error) {
	s.Data = nil
	s.StringData = nil

	r, err := toResource(s)
	if err != nil {
		return nil, err
	}

	// This will set interal kustomize annotations
	err = r.SetName(originalName)
	if err != nil {
		return nil, err
	}

	r.StorePreviousId()

	err = r.SetName(newName)
	if err != nil {
		return nil, err
	}

	ra := r.GetAnnotations()
	ra[OriginalNameAnnotation] = originalName
	ra[fn.KptLocalConfig] = "true"
	if pathAnnotationValue != "" {
		ra[fn.PathAnnotation] = pathAnnotationValue
	}

	err = r.SetAnnotations(ra)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func toRNode(s *corev1.Secret) (*yaml.RNode, error) {
	sObj, err := fn.NewFromTypedObject(s)
	if err != nil {
		return nil, err
	}

	return yaml.Parse(sObj.String())
}

func toResource(s *corev1.Secret) (*resource.Resource, error) {
	rn, err := toRNode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to RNode: %s", err)
	}

	return &resource.Resource{RNode: *rn}, nil
}

func computeHash(s *corev1.Secret, h *hasher.Hasher) (string, error) {
	rn, err := toRNode(s)
	if err != nil {
		return "", err
	}

	return h.Hash(rn)
}

func isSameObj(obj, another *fn.KubeObject) bool {
	if obj.GroupVersionKind().String() != another.GroupVersionKind().String() {
		return false
	}

	if obj.GetId().Namespace != another.GetId().Namespace {
		return false
	}

	if obj.GetAnnotation(OriginalNameAnnotation) != another.GetAnnotation(OriginalNameAnnotation) {
		return false
	}

	return true
}

func isInternalAnnotation(k string) bool {
	switch {
	case
		strings.HasPrefix(k, fn.ConfigPrefix),
		strings.HasPrefix(k, fn.KptUseOnlyPrefix),
		strings.HasPrefix(k, "config.k8s.io/"),
		strings.HasPrefix(k, "kustomize.config.k8s.io/"),
		strings.HasPrefix(k, "internal.config.kubernetes.io/"),
		strings.HasPrefix(k, "internal.config.k8s.io/"):
		return true
	default:
		return false
	}
}

func removeInternalAnnotations(s *corev1.Secret) {
	for k := range s.GetAnnotations() {
		if isInternalAnnotation(k) {
			delete(s.Annotations, k)
		}
	}
}

func sanitizeMetadata(s *corev1.Secret) {
	s.SetSelfLink("")
	s.SetUID("")
	s.SetResourceVersion("")
	s.Generation = 0
	s.SetCreationTimestamp(metav1.Time{})
	s.SetDeletionTimestamp(nil)
	s.DeletionGracePeriodSeconds = nil
}
