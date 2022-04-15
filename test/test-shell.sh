#!/bin/bash

mosn=$1
config_file=${mosn%/*}/'mosn_config.json'
default_config_dir=${mosn%/*}/configs
default_config_file_path=${mosn%/*}/configs/'mosn_config.json'
admin_port=34901

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

function check_mosn_state {
  state=$(curl 127.0.0.1:$admin_port/api/v1/states)
  state=${state:(-1)}
  if [ $state = '6' ]; then
    echo "mosn is running"
    return 0
  fi
  echo "mosn is not running, state($state)"
  return 1
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
echo "TEST Case 3: start with -c then stop"
exec_bg $mosn start -c $config_file
if [ $PID = 0 ]; then
  echo "mosn start with $@ failed"
  exit 1
fi
echo "mosn start successfully, pid($PID)"
# sleep 10s to mosn init
sleep 10

echo "checking mosn state"
check_mosn_state
if [ $? != 0 ]; then
  exit 1
fi

$mosn stop -c $config_file
sleep 1
ps -p $PID
if [ $? = 0 ]; then
  echo "mosn stop with $@ failed, pid($PID)"
  exit 1
fi

sleep 1

# TEST 4. start with default then stop
echo "TEST Case 4: start with default then stop"
mkdir $default_config_dir
cp $config_file $default_config_file_path
exec_bg $mosn start
if [ $PID = 0 ]; then
  echo "mosn start with default failed"
  exit 1
fi
echo "mosn start successfully, pid($PID)"
# sleep 10s to mosn init
sleep 10

echo "checking mosn state"
check_mosn_state
if [ $? != 0 ]; then
  exit 1
fi

$mosn stop
sleep 1
ps -p $PID
if [ $? = 0 ]; then
  echo "mosn stop with default failed, pid($PID)"
  exit 1
fi