#!/bin/sh

set -ex

rm -rf test/ && kpt fn source | kpt fn sink test/ && pushd test/

kustomize fn run .

kustomize edit add resource fix-name-refs

kustomize build

popd
