#!/bin/bash

mosn=$1
config_file=${mosn%/*}/'mosn_config.json'
echo $config_file
verbose=0

if [ "$2" = "-v" ]; then
  verbose=1
fi

if [ -z $mosn ]; then
  echo "not found mosn binary path"
  echo "Usage: sh test/test-shell.sh /path/to/mosn/binary or make test-shell"
  exit 1
fi

if [ ! -f $mosn ]; then
  echo "$mosn is not existing, please build it firstly"
  exit 1
fi

# === global variables === #

PID=0

# ============  util functions  ============ #

function exec_bg {
  if [ $verbose = 1 ]; then
    echo "exec shell in background: $@"
  fi

  nohup $@ &
  PID=$!

  # sleep 1s to wait the background process
  sleep 1

  ps -p $PID
  if [ $? = 0 ]; then
    if [ $verbose = 1 ]; then
      echo "exec shell success(pid $PID): $@"
    fi
    return
  fi

  echo "exec shell failed(pid not existing): $@"
  PID=0
  return
}

function start_mosn_with_quit {
  exec_bg $mosn start $@
  if [ $PID = 0 ]; then
    echo "mosn start with $@ failed"
    return 1
  fi

  for ((i=0; i<100; i++)); do
    if [ $verbose = 1 ]; then
      echo "killing mosn (pid $PID)"
    fi
    kill $PID
    sleep 1

    ps -p $PID
    if [ $? != 0 ]; then
      return 0
    fi
  done

  echo "kill mosn (pid $PID) failed"
  exit 1
}

# run_shell ls -lh

# ============  test cases ============ #

# TEST 1. start with no arguments
start_mosn_with_quit
if [ $? != 0 ]; then
  exit 1
fi

# TEST 2. start with unknown arguments
start_mosn_with_quit -unknown
if [ $? = 0 ]; then
  exit 1
fi

# TEST 3. start with -c then stop
#exec_bg $mosn start -c $config_file
#  if [ $PID = 0 ]; then
#    echo "mosn start with $@ failed"
#    return 1
#  fi
#sleep 5
#
#nohup $mosn stop -c $config_file &
#  ps -p $PID
#  if [ $? != 0 ]; then
#    exit 1
#  fi
#sleep 2

# TEST 4. start with default params then stop
exec_bg $mosn start
  if [ $PID = 0 ]; then
    echo "mosn start with default failed"
    return 1
  fi
sleep 5

nohup $mosn stop &
  ps -p $PID
  if [ $? != 0 ]; then
    exit 1
  fi
sleep 2
