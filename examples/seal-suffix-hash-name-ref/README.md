# About

This example shows how to seal Secrets with KPT and then using those with Kustomize
to update any name references in other resources (like Deployments, Pods, etc).

# Setup

There are two `Secrets` in `secret-a.yaml` and `secret-b.yaml` in plain-text that are
referenced by a`Deployment` in `deploy.yaml`.

We'd like those reference names to change
as we change the content of these `Secrets`. This will let the `ReplicaSets` of the
`Deployments` to gracefully transition, making sure they're always using the correct
values of secret keys.

# Test

To seal the `Secrets` run:

```shell
kpt fn render
```

This will encrypt them to `SealedSecrets` with hashed names, then remove the plain-text `Secrets` and
finally generate "stub" `Secrets` that can be used by Kustomize to do name replacements.

Build the Kustomization to see the results:

```shell
kustomnize build
```

Now, let's say we want to change one of the values of the `Secret`.

Create a new `Secret` with the same name as the original:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-two
  namespace: test
stringData:
  a: "a"
  b: "new value"
```

One thing to note here is we have to supply all of the keys. There is no way to merge the values because each key's encrypted text is also dependent on the name of the Secret. And the
name of the Secret is dependent on the *whole*  of the data.

Now you can "update" the value by running:

```shell
kpt fn render
```

This will replace one of the `SealedSecrets` with a new one.

Run Kustomize again to se the results:

```shell
kustomize build
```
