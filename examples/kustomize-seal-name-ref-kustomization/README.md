This example shows how to use `nameRefKustomizationDir` & `nameSuffixHash` to correct name references to
sealed secrets using Kustomize.

Create clone of this package to test it out (KPT is only used to clone the example):

```shell
rm -rf test/ && kpt fn source | kpt fn sink test/ && pushd test/
```

Notice that the `Deployment` refernces the two `Secrets` using their pre-hashed names:

```yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy
  namespace: test
spec:
  template:
    spec:
      containers:
        - name: container
          envFrom:
            - secretRef:
                name: secret-one
            - secretRef:
                name: secret-two
```

Seal the `Secrets`:

```shell
kustomize fn run .
```

Building the Kustomization at this point will not correctly update name refernces:

```shell
kustomize build
```

But if you add the generated Kustomization to the root Kustomization:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ./deploy.yaml
  - ./secret-one.yaml
  - ./secret-two.yaml
  - ./fix-name-refs # <---- Add this
```

And build again:

```shell
kustomize build
```

The name references should use the hashed names.

If you want to reseal the `Secret` with new values, create a `Secret` resource
with the same name as the original (i.e. the pre hashed name):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-one
  namespace: test
stringData:
  c: "c"
```

Render again:

```shell
kustomize fn run .
```

Kustomization build should update the reference to the new secret:

```shell
kustomize build
```
