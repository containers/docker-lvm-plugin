#!/bin/bash

set -euo pipefail

export TEMP_DIR=$(pwd)/temp
CONFIG_FILE=/etc/docker/docker-lvm-plugin
CONFIG_FILE_BACKUP=/etc/docker/docker-lvm-plugin.bkp
DEFAULT_VOLUME_GROUP=lvm-tests-vg0

# Keeps track of overall pass/failure status of tests. Even if single test
# fails, PASS_STATUS will be set to 1 and returned to caller when all
# tests have run.
PASS_STATUS=0

setup(){
  echo "INFO: Starting setup..."
  mkdir -p $TEMP_DIR
  echo "INFO: Creating a sparse file for loopback device."
  dd if=/dev/zero of=$TEMP_DIR/loopfile bs=1024k count=2000 status=none 2>&1 >/dev/null
  echo "INFO: Checking available loopback devices."
  loopdevice=$(losetup --find)
  echo "INFO: Using available loopback device $loopdevice for running tests."
  losetup $loopdevice $TEMP_DIR/loopfile 2>&1 >/dev/null
  echo "INFO: Creating physical volume (PV) on $loopdevice."
  pvcreate $loopdevice 2>&1 >/dev/null
  echo "INFO: Creating default volume group (VG) $DEFAULT_VOLUME_GROUP."
  vgcreate $DEFAULT_VOLUME_GROUP $loopdevice 2>&1 >/dev/null
  echo "INFO: Creating thinpool $DEFAULT_VOLUME_GROUP/mythinpool"
  lvcreate -L 1G -T $DEFAULT_VOLUME_GROUP/mythinpool 2>&1 >/dev/null
  echo "INFO: Backup $CONFIG_FILE config file."
  mv $CONFIG_FILE $CONFIG_FILE_BACKUP
  echo "INFO: Creating docker-lvm-plugin config file for running tests."
  echo "VOLUME_GROUP=$DEFAULT_VOLUME_GROUP" > $CONFIG_FILE
  echo "INFO: Setup complete."
}

cleanup(){
  echo "INFO: Starting cleanup..."
  echo "INFO: Removing $DEFAULT_VOLUME_GROUP/mythinpool"
  lvremove -y $DEFAULT_VOLUME_GROUP/mythinpool 2>&1 >/dev/null
  echo "INFO: Removing default volume group (VG) $DEFAULT_VOLUME_GROUP."
  vgremove $DEFAULT_VOLUME_GROUP 2>&1 >/dev/null
  echo "INFO: Removing physical volume (PV)."
  pvremove $loopdevice 2>&1 >/dev/null
  echo "INFO: Detaching loopback device $loopdevice."
  losetup --detach $loopdevice 2>&1 >/dev/null
  echo "INFO: Removing $TEMP_DIR."
  rm -rf $TEMP_DIR 2>&1 >/dev/null
  echo "INFO: Restoring $CONFIG_FILE."
  mv $CONFIG_FILE_BACKUP $CONFIG_FILE
  echo "INFO: Cleanup complete."
}

check_root(){
  if [ $(id -u) != 0 ]; then
    echo "ERROR: Run tests as root user."
    exit 1
  fi
}

check_docker_active() {
  if !(systemctl -q is-active "docker.service"); then
    echo "ERROR: docker daemon is not running. Please start docker daemon before running tests."
    exit 1
  fi
}

check_docker_lvm_plugin_active(){
  if !(systemctl -q is-active "docker-lvm-plugin.service"); then
    echo "ERROR: docker-lvm-plugin is not running. Please start docker-lvm-plugin before running tests."
    exit 1
  fi
}

run_test () {
  testfile=$1

  echo "INFO: Running test `basename $testfile`"
  bash -c $testfile

  if [ $? -eq 0 ];then
    echo "PASS: $(basename $testfile)"
  else
    echo "FAIL: $(basename $testfile)"
    PASS_STATUS=1
  fi
}

run_tests() {
  local srcdir=`dirname $0`
  if [ $# -gt 0 ]; then
    local files=$@
  else
    local files="$srcdir/[0-9][0-9][0-9]-test-*"
  fi
  for t in $files;do
    run_test ./$t
  done
}

trap cleanup EXIT

check_root
check_docker_active
check_docker_lvm_plugin_active
setup
run_tests $@
exit $PASS_STATUS

