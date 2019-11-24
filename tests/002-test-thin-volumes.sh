#!/bin/bash

test_thin_volumes(){
  local test_status=1
  local testname=`basename "$0"`

  docker volume create -d lvm --opt size=0.2G --opt thinpool=mythinpool --name thin_vol 2>&1 >/dev/null
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error creating thin volume thin_vol."
    return $test_status
  fi

  docker volume inspect thin_vol 2>&1 >/dev/null
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error inspecting volume thin_vol."
    return $test_status
  fi

  docker volume create -d lvm --opt snapshot=thin_vol --name thin_vol_snapshot 2>&1 >/dev/null
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error creating snapshot of thin volume thin_vol."
    return $test_status
  fi

  docker volume inspect thin_vol_snapshot 2>&1 >/dev/null
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error inspecting thin volume thin_vol_snapshot."
    return $test_status
  fi

  docker volume rm thin_vol_snapshot 2>&1 >/dev/null
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error removing thin volume thin_vol_snapshot."
    return $test_status
  fi

  docker volume rm thin_vol 2>&1 >/dev/null
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error removing volume thin_vol."
    return $test_status
  fi

  return 0
}

test_thin_volumes
