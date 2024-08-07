package unseal

import (
	"crypto/rsa"
	_ "encoding/base64"
	"errors"
	"fmt"
	"reflect"

	"github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/common"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/bitnami-labs/sealed-secrets/pkg/crypto"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/keyutil"

	ssv1alpha1 "github.com/bitnami-labs/sealed-secrets/pkg/apis/sealedsecrets/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UnsealedAnnotation = common.AnnotationPrefix + "unsealed"
)

func Process(rl *fn.ResourceList) (bool, error) {
	if rl.FunctionConfig == nil {
		return false, fn.ErrMissingFnConfig{}
	}

	cm := &ConfigMap{}
	if err := rl.FunctionConfig.As(cm); err != nil {
		return false, fmt.Errorf("failed to parse function config: %w", err)
	}

	if cm.Data.ControllerPrivateKeys == "" {
		return false, fmt.Errorf("controllerPrivateKeys is needed to decrypt SealedSecrets")
	}

	privateKeys, err := parseControllerPrivateKeys(cm.Data.ControllerPrivateKeys)
	if err != nil {
		return false, fmt.Errorf("failed to parse controllerPrivateKeys: %w", err)
	}

	privateKeysByFingerprint, err := fingerprintKeys(privateKeys)
	if err != nil {
		return false, fmt.Errorf("failed to fingerprint private keys: %w", err)
	}

	isSealedSecret := func(o *fn.KubeObject) bool {
		return o.IsGroupVersionKind(ssv1alpha1.SchemeGroupVersion.WithKind("SealedSecret"))
	}

	ssecrets := rl.Items.Where(isSealedSecret)

	if cm.Data.Prune {
		rl.Items = rl.Items.WhereNot(isSealedSecret)
	}

	for _, ssecret := range ssecrets {
		ss := ssv1alpha1.SealedSecret{}

		err := ssecret.As(&ss)
		if err != nil {
			return false, fmt.Errorf("failed to parse: %w", err)
		}

		secret, err := ss.Unseal(scheme.Codecs, privateKeysByFingerprint)
		if err != nil {
			return false, fmt.Errorf("failed to unseal %s: %w", ssecret.ShortString(), err)
		}

		secret.OwnerReferences = []metav1.OwnerReference{}

		if !cm.Data.Base64 {
			secret.StringData = make(map[string]string)
			for k, v := range secret.Data {
				secret.StringData[k] = string(v[:])
			}
			secret.Data = nil
		}

		s, err := fn.NewFromTypedObject(secret)
		if err != nil {
			return false, fmt.Errorf("failed to convert Secret to KubeObject: %w", err)
		}

		err = errors.Join(
			s.SetAPIVersion(corev1.SchemeGroupVersion.String()),
			s.SetKind("Secret"),
			s.SetAnnotation(UnsealedAnnotation, "true"),
		)
		if err != nil {
			return false, err
		}

		if ssecret.PathAnnotation() != "" {
			err = s.SetAnnotation(fn.PathAnnotation, ssecret.PathAnnotation())
			if err != nil {
				return false, err
			}
		}

		err = rl.UpsertObjectToItems(
			s,
			func(obj, another *fn.KubeObject) bool {
				if _, ok := obj.GetAnnotations()[UnsealedAnnotation]; !ok {
					return false
				}

				if _, ok := another.GetAnnotations()[UnsealedAnnotation]; !ok {
					return false
				}

				return reflect.DeepEqual(obj.GetId(), another.GetId())
			},
			true,
		)
		if err != nil {
			return false, fmt.Errorf("failed to add Secret %s to output: %w", s.ShortString(), err)
		}

		rl.Results.Infof("unsealed %s (path=%s)", s.ShortString(), s.PathAnnotation())
	}

	err = common.NormalizeIndexAnnotation(rl)
	if err != nil {
		return false, err
	}

	return true, nil
}

func parsePrivateKey(b []byte) (*rsa.PrivateKey, error) {
	key, err := keyutil.ParsePrivateKeyPEM(b)
	if err != nil {
		return nil, err
	}
	switch rsaKey := key.(type) {
	case *rsa.PrivateKey:
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unexpected private key type %T", key)
	}
}

func parseControllerPrivateKeys(j string) ([]*rsa.PrivateKey, error) {
	// PEM encoded
	key, err := parsePrivateKey([]byte(j))

	if err == nil {
		return []*rsa.PrivateKey{key}, nil
	}

	// Otherwise interpert it as v1.Secret or a v1.SecretList
	obj, err := fn.ParseKubeObject([]byte(j))
	if err != nil {
		return nil, err
	}

	sl := corev1.SecretList{}

	err = obj.As(&sl)
	if err != nil {
		err = fmt.Errorf("failed to parse as a v1.SecetList: %w", err)

		s := corev1.Secret{}
		err2 := obj.As(&s)
		if err2 != nil {
			err2 = fmt.Errorf("failed to parse as a v1.Secret: %w", err2)
			return nil, errors.Join(err, err2)
		}

		sl.Items = append(sl.Items, s)
	}

	var keys []*rsa.PrivateKey
	for _, s := range sl.Items {
		tlsKey, ok := s.Data["tls.key"]
		if !ok {
			return nil, fmt.Errorf("secret %s/%s must contain a 'tls.key' key", s.GetNamespace(), s.GetName())
		}
		pk, err := parsePrivateKey(tlsKey)
		if err != nil {
			return nil, err
		}
		keys = append(keys, pk)
	}

	return keys, nil
}

func fingerprintKeys(keys []*rsa.PrivateKey) (map[string]*rsa.PrivateKey, error) {
	res := map[string]*rsa.PrivateKey{}

	for _, key := range keys {
		fingerprint, err := crypto.PublicKeyFingerprint(&key.PublicKey)
		if err != nil {
			return nil, err
		}

		res[fingerprint] = key
	}

	return res, nil
}
