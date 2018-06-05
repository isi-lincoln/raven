#!/bin/bash

set -e

BOLD="\e[1m"
BLUE="\e[34m"
CLEAR="\e[0m"

function phase() {
/bin/echo -e "$BOLD
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\r$1
$CLEAR"
}

# launch the system and wait till it is up

phase "Fetching walrus"
  if [[ ! -d walrustf ]]; then
    /usr/bin/git clone https://github.com/rcgoodfellow/walrustf
  fi

phase "Building"
  /bin/echo "clearing out any artifacts from previous runs"
  /usr/bin/sudo -E rvn destroy
  /bin/echo "building system"
  /usr/bin/sudo -E rvn build

phase "Deploying"
  /bin/echo "launching vms"
  /usr/bin/sudo -E rvn deploy
  /bin/echo "waiting for vms to come on network"
  /usr/bin/sudo -E rvn pingwait control walrus nimbus n0 n1

phase "Configuring"
  /usr/bin/sudo -E rvn configure

phase "Testing"
  /bin/echo "launching tests"
  /usr/bin/sudo -E rvn ansible walrus config/run_tests.yml
  wtf -collector=`rvn ip walrus` watch config/files/walrus/tests.json
