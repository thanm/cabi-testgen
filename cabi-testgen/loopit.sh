#!/bin/sh
#
# Simple test script for performing a series of test runs.
#
HOWMANY=$1
if [ -z "$HOWMANY" ]; then
  HOWMANY=1
fi
ITER=${HOWMANY}
go build .
SEED=`seconds.py`
HERE=`pwd`
PRAG="-pragma registerparams"
PRAG=""
while [ $ITER !=  0 ]; do
  echo iter $ITER
  ITER=`expr $ITER - 1`
  echo "Iter $ITER"
  D=/tmp/cabiTest
  rm -rf $D
  CMD="./cabi-testgen -q 20 -n 20 -s $SEED -o $D -p cabiTest"
  echo $CMD
  $CMD
  if [ $? != 0 ]; then
    echo "*** gen failed: $CMD"
    exit 1
  fi
  cd $D
  rm -f cabiTest
  go build -p 1 . 1> build.err.txt 2>&1
  if [ $? != 0 ]; then
     echo "*** building generated code failed, SEED=$SEED see build.err.txt"
     exit 1
  fi
  ./cabiTest
  if [ $? != 0 ]; then
     echo "*** running generated code failed, SEED=$SEED"
     exit 1
  fi
  cd $HERE
  SEED=`expr $SEED + 17`
done
