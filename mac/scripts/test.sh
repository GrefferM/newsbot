#!/bin/sh
set -eu

cd "$(dirname "$0")/../Core"

swift test
