#!/bin/bash

# LUKS = Linux unified key setup
test_luks_volumes(){
  local test_status=1
  local testname=`basename "$0"`

  docker volume create -d lvm --opt size=0.2G --opt keyfile=$TEMP_DIR/key.bin --name crypt_vol >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error creating luks volume crypt_vol."
    return $test_status
  fi

  docker volume inspect crypt_vol >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error inspecting volume crypt_vol."
    return $test_status
  fi

  docker volume create -d lvm --opt size=0.2G --opt snapshot=crypt_vol --name crypt_vol_snapshot >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error creating snapshot of volume crypt_vol."
    return $test_status
  fi

  docker volume inspect crypt_vol_snapshot >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error inspecting volume crypt_vol_snapshot."
    return $test_status
  fi

  docker volume rm crypt_vol_snapshot >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error removing volume crypt_vol_snapshot."
    return $test_status
  fi

  docker volume rm crypt_vol >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error removing volume crypt_vol."
    return $test_status
  fi

  return 0
}

setup(){
  echo "00110101" > $TEMP_DIR/key.bin
}

cleanup(){
  rm -f $TEMP_DIR/key.bin
}

trap cleanup EXIT

setup
test_luks_volumes
