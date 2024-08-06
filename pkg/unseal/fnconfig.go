package unseal

import (
	"github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/types"

	corev1 "k8s.io/api/core/v1"
)

type ConfigMap struct {
	corev1.ConfigMap
	Data ConfigMapData `json:"data"`
}

type ConfigMapData struct {

	// ControllerPrivateKeys is a PEM encoded string or JSON/YAML encoded v1.Secret/v1.SecretList of the controller private key used to decrypt SealedSecrets.
	// You can get this using `kubectl -n <controller-namespace> get secret -l 'sealedsecrets.bitnami.com/sealed-secrets-key' -o yaml` and paste it as a string.
	ControllerPrivateKeys string `json:"controllerPrivateKeys,omitempty"`

	// Prune will remove all SealedSecrets that were unsealed from the output. By default they are left in.
	Prune types.Bool `json:"prune,omitempty"`

	// Base64 will output Secret values in base64 encoded format (i.e. `Secret.data`). By default they are output in plain-text (i.e. `Secret.stringData`).
	Base64 types.Bool `json:"base64,omitempty"`
}
