# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# Shared set-up for the script tests. Each test runs a copy of one script
# in a throwaway git repository, with the commands that would reach the
# network or the clock (curl, sleep, go, gorelease) replaced by stubs on
# PATH. Nothing here touches the real repository, its tags or a network.
#
# SCRIPTS_DIR names the directory the script under test is copied from;
# it defaults to this checkout's scripts/, and pointing it at an older
# copy is how a regression test is shown to fail before its fix.

REPO_ROOT=$(cd "$(dirname "${BATS_TEST_FILENAME}")/../.." && pwd)
SCRIPTS_DIR=${SCRIPTS_DIR:-${REPO_ROOT}/scripts}

# sandbox makes a git repository at $SANDBOX holding scripts/<name>, with
# one commit, and puts $STUBS first on PATH. The git configuration is the
# sandbox's own, so a maintainer's signing settings do not apply to it.
sandbox() {
  local name=$1
  SANDBOX=${BATS_TEST_TMPDIR}/repo
  STUBS=${BATS_TEST_TMPDIR}/bin
  mkdir -p "${SANDBOX}/scripts" "${STUBS}"
  cp "${SCRIPTS_DIR}/${name}" "${SANDBOX}/scripts/${name}"
  chmod +x "${SANDBOX}/scripts/${name}"
  export GIT_CONFIG_GLOBAL=${BATS_TEST_TMPDIR}/gitconfig GIT_CONFIG_NOSYSTEM=1
  git config --global user.name "Script Test"
  git config --global user.email "script-test@example.invalid"
  git config --global init.defaultBranch main
  git -C "${SANDBOX}" init --quiet
  commit "initial"
  PATH="${STUBS}:${PATH}"
}

# commit records an empty commit in the sandbox.
commit() {
  git -C "${SANDBOX}" commit --quiet --allow-empty -m "$1"
}

# stub installs an executable named $1 whose body is read from stdin.
stub() {
  { echo '#!/usr/bin/env bash'; cat; } >"${STUBS}/$1"
  chmod +x "${STUBS}/$1"
}
