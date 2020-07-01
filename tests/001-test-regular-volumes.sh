#!/bin/bash

test_regular_volumes(){
  local test_status=1
  local testname=`basename "$0"`
  local errmsg="all snapshot destinations must be removed before removing the original volume"

  docker volume create -d lvm --opt size=0.2G --name foobar >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error creating regular volume foobar."
    return $test_status
  fi

  docker volume inspect foobar >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error inspecting volume foobar."
    return $test_status
  fi

  docker volume create -d lvm --opt snapshot=foobar --opt size=100M --name foobar_snapshot >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error creating snapshot of volume foobar."
    return $test_status
  fi

  docker volume inspect foobar_snapshot >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error inspecting volume foobar_snapshot."
    return $test_status
  fi

  docker volume rm foobar >> $tmplog 2>&1
  rc=$?
  if [ $rc -ne 0 ]; then
     if !(grep --no-messages -q "$errmsg" $tmplog); then
        echo "ERROR: $testname: Failed for a reason other than \"$errmsg\""
        return $test_status
     fi
  else
     echo "ERROR: $testname succeeded: Should have failed with reason \"$errmsg\""
     return $test_status
  fi

  docker volume rm foobar_snapshot >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error removing volume foobar_snapshot."
    return $test_status
  fi

  docker volume rm foobar >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "ERROR: $testname: Error removing volume foobar."
    return $test_status
  fi

  return 0
}

setup(){
  tmplog=$(mktemp --suffix=.log $TEMP_DIR/lvm.XXXXXX)
}

cleanup(){
  rm -f $tmplog
}

trap cleanup EXIT

setup
test_regular_volumes
