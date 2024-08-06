#!/bin/sh

set -ex

rm -rf test/ && kpt fn source | kpt fn sink test/

kpt fn render test/
