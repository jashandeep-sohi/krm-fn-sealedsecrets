[![Go Reference](https://pkg.go.dev/badge/github.com/jashandeep-sohi/krm-fn-sealedsecrets.svg)](https://pkg.go.dev/github.com/jashandeep-sohi/krm-fn-sealedsecrets)

# krm-fn-sealedsecrets

[KRM Functions](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md)
to manage [SealedSecrets](https://github.com/bitnami-labs/sealed-secrets).

# Usage

Two KRM functions are provided. One for sealing (encrypting) secrets and
another for unsealing (decrypting) them.

Unsealing secrets should be considered a debugging or disaster-recovery action that 
should rarely need to be performed. It relies on having access to the SealedSecret
controller private keys, and as such it should (and can) only be done by cluster-admins
who have full access to the K8s API server. You should also be very careful not
to accidently leak these keys.

Sealing secrets on the other hand can be done by anyone.

The API is a limited subset of the `kubeseal` CLI with some added features.

## Seal

To seal secrets, use the `ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/seal` function image.

It takes in a stream of `v1.Secrets` and generates `SealedSecrets` for each of them.
Configure it using a [ConfigMap](https://pkg.go.dev/github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/seal#ConfigMap).

### KPT

It can be used both imperatively and declaratively with [KPT](https://kpt.dev/).

For example, given you have a bunch of `Secrets`:

```shell
cat <<EOF > secret-a.yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-a
  namespace: test
stringData:
  SOMETHING: "VERY SECRET"
EOF

cat <<EOF > secret-b.yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-b
  namespace: test
stringData:
  SOMETHING_ELSE: "ALSO SECRET"
EOF
```

To seal it, run:

```shell
kpt fn eval --image ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/seal:latest -- "cert=$(kubeseal --fetch-cert)"
```

This will generate a `SealedSecret` for each of them and write it to the same file
as the original `Secret`.

The original `Secret` is also left in-place by default, but you should remove
it as soon as possible to avoid leaking it (e.g. checking it into Git).

To do that, run the function with `prune=true`:

```shell
kpt fn eval --image ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/seal:latest -- "cert=$(kubeseal --fetch-cert)" prune=true
```

Here, we're fetching the certificate using `kubeseal` via a shell command
substituion (`kubeseal --fetch-cert`), but it could come from anywhere (e.g. a local file).

This certificate is safe to check into Git, but keep in mind that the controller
generates new ones periodically, so you should always use the latest one.

You can also store the function config in a file:

```shell
cat <<EOF > fn-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: does-not-matter
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  prune: "true"
  cert: |-
    -----BEGIN CERTIFICATE-----
    MIIFazCCA1OgAwIBAgIUAZIki4fTKG4CPdFjcr+WOTPcZDwwDQYJKoZIhvcNAQEL
    BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
    GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNDA4MDIwMTM4NTNaFw0zNDA3
    MzEwMTM4NTNaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
    HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggIiMA0GCSqGSIb3DQEB
    AQUAA4ICDwAwggIKAoICAQDy4ju04H5cofL6Z3ZzG0NOaMJ149/oAlzmpdAY94HO
    Wg8SIuoXSpwNYAWutZbvvJ4RZH635gYRKmvb8QBG3fDpEg7cm+I5ys10rLIUoLU1
    PiID1cwywS9lLfpSKaB2LCdljK8KipruVVxuIsegPBcKd/iJuzZHEns8+SyVRrQv
    WhJwI/+69PKw9Yej2zaDonK4DPmwTvR17L3sxd4tkXtgMCPlybsNxkGrqWEqCBpY
    UJZSNL5r4TtzBEJaLdDkBUVIr30Q+l5OeFw6/WkkhDVnkiiMpttHRbUYgk2ZIz6d
    Gf1KPguAyYhvPL0T5JuGgjskS6s4LhYl68bvJOv92PCFz2NbtU0IZ1IOx6zfKqFN
    /Z68/amC4bGJ2EijHM8RQrY0HC/KfqDhf+yje3KRfjA8bVzCwTRjkyOmag//dfaZ
    4E78/hmJY7BAPbXBFX0CIauBd3OZiPAfrmgcYCQdWhjdM5QgkD/DiKENOA8Si+2g
    95AN5Yhr3S8cK6bAPtqS7SX85uynCw9hGdMvLe5f17GFU7wGuQ6K4QnbGxa62ta0
    BqzRXrvVgOCgA2rKuPgPnVwq4S0+PxcwGmDekSG/LZJehiRCTMlBruf1pXCqI5RN
    aKyJH7rIE9sAMGBCIt73/ybK5XCf+TfNLA342hWkYddQB5enBwPbOK/W+PCmBze2
    PwIDAQABo1MwUTAdBgNVHQ4EFgQUtTp3Abp5KlIuudFdnhcsQNA37PcwHwYDVR0j
    BBgwFoAUtTp3Abp5KlIuudFdnhcsQNA37PcwDwYDVR0TAQH/BAUwAwEB/zANBgkq
    hkiG9w0BAQsFAAOCAgEA5dwDIqQIKEJwQFKi/1oDp3LpmYIa7z5KuxffRZvtZy4L
    TwhNn5T8CRiDznrQhQ1aWzSFxW/aZO05TM65KzeOhgzrpVtd5EFGNoxAzEDtMwxy
    sZ40tMWXh+d9KINLpRbLhEPfk9JBm1RtMXViV12bJQqjJwRL74kA1wzFiXXHMNQ/
    9GyZDs4XRgOjYRqKbSofiPNjd7NsgkE1Z1HspdUKPMU3ptDeDoLeR+Ik6RovPidh
    0In0U+U6B0xzzhsnyDYSAkNfi52SRdA4XRVVRpp8di7FjX2o5FgJ4Qg2k5CPMV1p
    ty5yx92zZ85Bp/8eT/oq6yis0IZjbvcYOI9C5dn10THz/AZ4ROEvmz2Xpr/BNcEG
    itwoyXW+ENj8/acRjmLU32dsBOtfEUHOFXnkvoahEGQqvUrOt+s9SJUjwwt4v8ps
    Ilxwxso8pbD+wD1lDzJxHlb2xBoDfPZIuGfsZvIPQCa/rfspumJhTBwDU0H8/daF
    LCw/X7btQH45FmtxlVUUpdT97vW39y+t5BjlCcRkPsyE5oEm/KW5tNP3HseqyhL2
    OO34Ui8aqRA6W8oQAK1bR16+3kdTTs2ZgeZrexQPlsIiKAliVyBy+wlZjA832Bmt
    Nj7Wa3vwV8vMj9t6LUmQWquAjY1hKWiTjywSfDfNgZjqiVwIusyqfk2ceeqCWeQ=
    -----END CERTIFICATE-----
EOF
```

And pass it to the function with:

```shell
kpt fn eval --image ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/seal:latest --fn-config fn-config.yaml
```

### Kustomize

Kustomize also supports running KRM functions:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ./secret-a.yaml
  - ./secret-b.yaml

transformers:
  - |-
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: does-not-matter
      annotations:
        config.kubernetes.io/function: |
          container:
            image: ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/seal:latest
    data:
      prune: "true"
      cert: |-
        -----BEGIN CERTIFICATE-----
        MIIFazCCA1OgAwIBAgIUAZIki4fTKG4CPdFjcr+WOTPcZDwwDQYJKoZIhvcNAQEL
        BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
        GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNDA4MDIwMTM4NTNaFw0zNDA3
        MzEwMTM4NTNaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
        HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggIiMA0GCSqGSIb3DQEB
        AQUAA4ICDwAwggIKAoICAQDy4ju04H5cofL6Z3ZzG0NOaMJ149/oAlzmpdAY94HO
        Wg8SIuoXSpwNYAWutZbvvJ4RZH635gYRKmvb8QBG3fDpEg7cm+I5ys10rLIUoLU1
        PiID1cwywS9lLfpSKaB2LCdljK8KipruVVxuIsegPBcKd/iJuzZHEns8+SyVRrQv
        WhJwI/+69PKw9Yej2zaDonK4DPmwTvR17L3sxd4tkXtgMCPlybsNxkGrqWEqCBpY
        UJZSNL5r4TtzBEJaLdDkBUVIr30Q+l5OeFw6/WkkhDVnkiiMpttHRbUYgk2ZIz6d
        Gf1KPguAyYhvPL0T5JuGgjskS6s4LhYl68bvJOv92PCFz2NbtU0IZ1IOx6zfKqFN
        /Z68/amC4bGJ2EijHM8RQrY0HC/KfqDhf+yje3KRfjA8bVzCwTRjkyOmag//dfaZ
        4E78/hmJY7BAPbXBFX0CIauBd3OZiPAfrmgcYCQdWhjdM5QgkD/DiKENOA8Si+2g
        95AN5Yhr3S8cK6bAPtqS7SX85uynCw9hGdMvLe5f17GFU7wGuQ6K4QnbGxa62ta0
        BqzRXrvVgOCgA2rKuPgPnVwq4S0+PxcwGmDekSG/LZJehiRCTMlBruf1pXCqI5RN
        aKyJH7rIE9sAMGBCIt73/ybK5XCf+TfNLA342hWkYddQB5enBwPbOK/W+PCmBze2
        PwIDAQABo1MwUTAdBgNVHQ4EFgQUtTp3Abp5KlIuudFdnhcsQNA37PcwHwYDVR0j
        BBgwFoAUtTp3Abp5KlIuudFdnhcsQNA37PcwDwYDVR0TAQH/BAUwAwEB/zANBgkq
        hkiG9w0BAQsFAAOCAgEA5dwDIqQIKEJwQFKi/1oDp3LpmYIa7z5KuxffRZvtZy4L
        TwhNn5T8CRiDznrQhQ1aWzSFxW/aZO05TM65KzeOhgzrpVtd5EFGNoxAzEDtMwxy
        sZ40tMWXh+d9KINLpRbLhEPfk9JBm1RtMXViV12bJQqjJwRL74kA1wzFiXXHMNQ/
        9GyZDs4XRgOjYRqKbSofiPNjd7NsgkE1Z1HspdUKPMU3ptDeDoLeR+Ik6RovPidh
        0In0U+U6B0xzzhsnyDYSAkNfi52SRdA4XRVVRpp8di7FjX2o5FgJ4Qg2k5CPMV1p
        ty5yx92zZ85Bp/8eT/oq6yis0IZjbvcYOI9C5dn10THz/AZ4ROEvmz2Xpr/BNcEG
        itwoyXW+ENj8/acRjmLU32dsBOtfEUHOFXnkvoahEGQqvUrOt+s9SJUjwwt4v8ps
        Ilxwxso8pbD+wD1lDzJxHlb2xBoDfPZIuGfsZvIPQCa/rfspumJhTBwDU0H8/daF
        LCw/X7btQH45FmtxlVUUpdT97vW39y+t5BjlCcRkPsyE5oEm/KW5tNP3HseqyhL2
        OO34Ui8aqRA6W8oQAK1bR16+3kdTTs2ZgeZrexQPlsIiKAliVyBy+wlZjA832Bmt
        Nj7Wa3vwV8vMj9t6LUmQWquAjY1hKWiTjywSfDfNgZjqiVwIusyqfk2ceeqCWeQ=
        -----END CERTIFICATE-----
```

And then run:

```shell
kustomize build --enable-alpha-plugins
```

### Name Suffx Hash

You can suffix a hash to the name (that's dependent on the secret data) with `nameSuffixHash=true`:

```shell
kpt fn eval --image ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/seal:latest -- "cert=$(kubeseal --fetch-cert)" nameSuffixHash=t
```

### Kustomize Stub Secrets

When using Kustomize, you might have some references to `Secrets` in other
resources (e.g. `Pod.spec.containers.envFrom.secretsRef.name`) that'd you'd like to
be updated when the name changes because of `nameSuffixHash=true`.

You can do that by emmiting stub `v1.Secrets` along with the `SealedSecret`:

```shell
kpt fn eval --image ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/seal:latest -- "cert=$(kubeseal --fetch-cert)" nameSuffixHash=t emmitKustomizeStubSecrets=t
```

Checkout a more [complete example](./examples/seal-suffix-hash-name-ref).

**Note**: This is a hack that relies on annotations that are **internal** to Kustomize. There is no guarntee this will work with all version of Kustomize and may break at any time.


## Unseal

To unseal `SealedSecrets` use the `ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/unseal` function image.

It takes in a stream of `SealedSecrets` and generates `v1.SealedSecret` for each of them.
Configure it using a [ConfigMap](https://pkg.go.dev/github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/unseal#ConfigMap).


For example, with KPT you can run (subtitute `<controller-name>` with the namespace that sealed-secrets controller is running in):

```shell
kpt fn eval --image ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/unseal:latest -- "controllerPrivateKeys=$(kubectl -n <controller-namespace> get secret -l 'sealedsecrets.bitnami.com/sealed-secrets-key' -o yaml)"
```

Unsealed `v1.Secrets` do not clobber over any preexiting `v1.Secrets` with the same name/namespace,
so in some rare cases you might end up with duplicates, but Secrets generated by this function do have a
`krm-fn-sealedsecrets.io/unsealed=true` annotation set if you'd like to filter them out.

## Exec Nix

You can also run this function locally (i.e. without any of the restrictions & sandboxing that containers provide):

```shell
kpt fn eval --exec 'nix run github:jashandeep-sohi/krm-fn-sealedsecrets' ...
```

See ./examples/exec-nix-seal/ on how to do it declaratively.
