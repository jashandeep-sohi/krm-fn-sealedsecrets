package seal

import (
	"github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

type ConfigMap struct {
	corev1.ConfigMap
	Data ConfigMapData `json:"data"`
}

type ConfigMapData struct {

	// Cert is the PEM encoded certificate used to encrypt secrets.
	// You can get it using `kubeseal --fetch-cert`.
	Cert *string `json:"cert,omitempty"`

	// Prune will remove all v1.Secrets that were sealed from the output. By default they are left in.
	Prune types.Bool `json:"prune,omitempty"`

	// nameSuffixHash will append a short hash to the names of input v1.Secrets
	// based on their data before sealing them. Disabled by default.
	NameSuffixHash types.Bool `json:"nameSuffixHash,omitempty"`

	// EmmitKustomizeStubSecrets will emmit a stub v1.Secret for every SealedSecret that's generated.
	// This Secret does not actually contain any data, except that it can be used
	// by Kustomize for name reference substiutions.
	//
	// It will have the same name as the SealedSecret and a `config.kubernetes.io/local-config=true` annotation added
	// so that a `kustomize build` does not emmit it in the final output.
	//
	// In addition, Kustomize uses a few annotations internally to keep track of resource names. Those
	// will also be added to this secret.
	//
	// Note: This is a hack that relies on annotations that are **internal** to Kustomize. There is no
	// guarntee this will work with all version of Kustomize and may break at any time.
	EmmitKustomizeStubSecrets types.Bool `json:"emmitKustomizeStubSecrets,omitempty"`
}
