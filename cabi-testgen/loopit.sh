#!/bin/sh
#
# Simple test script for performing a series of test runs.
#
DOCLOBBER=no
DOSETGOGC=yes
DOMINIMIZE=no
#GEX=
#echo export GOEXPERIMENT=$GEX
#export GOEXPERIMENT=$GEX
HOWMANY=$1
if [ -z "$HOWMANY" ]; then
  HOWMANY=1
fi
DOCLEANCACHE=no
if [ $HOWMANY -gt 50 ]; then
  DOCLEANCACHE=yes
fi
GCFLAGS="-c=4"
GCFLAGS2="-c=1"
if [ $DOCLOBBER = "yes" ]; then
  GCFLAGS="-c=4 -clobberdead"
  GCFLAGS2="-c=1 -clobberdead"
fi
ITER=0
go build .
function cleanUnused() {
  local SAVE=$1
  local NUMP=$2
  local DIR=$3
  local CLEANED=""
  P=0
  while [ $P != $NUMP ]; do
    if [ $P != $SAVE ]; then
      CLEANED="$CLEANED $P"
      rm -rf $DIR/genChecker${P} ${DIR}/genCaller${P}
    fi
    P=`expr $P + 1`
  done
  echo "... cleaning unused: $CLEANED"
}
SEED=`seconds.py`
HERE=`pwd`
PRAG="-maxfail=9999"
NP=10
NF=10
NP=100
NF=20
while [ $ITER -lt ${HOWMANY} ]; do
  echo iter $ITER
  ITER=`expr $ITER + 1`
  echo "Iter $ITER"
  if [ "$DOCLEANCACHE" = "yes" ]; then
    MOD0=`expr $ITER % 100`
    if [ $MOD0 -eq 0 ]; then
      echo go clean -cache
      go clean -cache
    fi
  fi
  D=/tmp/cabiTest
  rm -rf $D ${D}.orig ${D}.pkg
  CMD="./cabi-testgen -q $NP -n $NF -s $SEED -o $D -p cabiTest $PRAG"
  echo $CMD
  $CMD
  if [ $? != 0 ]; then
    echo "*** gen failed: $CMD"
    exit 1
  fi
  BADP=unset
  BADF=unset
  cd $D
  rm -f cabiTest
  echo "... building"
  set -x
  go build -p=50 -gcflags=all="$GCFLAGS" . 1> ${HERE}/build.err.txt 2>&1
  RC=$?
  set +x
  if [ $RC != 0 ]; then
    echo "*** building generated code failed, SEED=$SEED see build.err.txt"
    if [ $DOMINIMIZE != "yes" ]; then
      exit 1
    fi
    echo "... serial build"
    go build -p=1 -gcflags=all="$GCFLAGS2" -p 1 . 1> ${HERE}/build.err.txt 2>&1
    echo "*** now trying to minimize by package"
    mv /tmp/cabiTest /tmp/cabiTest.orig
    # Minimize by package
    P=0
    while [ $P != $NP ]; do
      echo trying pkg $P
      cd $HERE
      rm -rf $D
      CMD="./cabi-testgen -q $NP -n $NF -s $SEED -o $D -p cabiTest $PRAG -P $P"
      echo $CMD
      $CMD
      if [ $? != 0 ]; then
        echo "*** gen failed: $CMD"
        exit 1
      fi
      cd $D
      rm -f cabiTest
      
      go build -p=1 -gcflags=all="$GCFLAGS2" . 1> build.err.${P}.txt 2>&1
      if [ $? != 0 ]; then
        echo "found offending package $P"
        BADP=$P
        break
      fi
      P=`expr $P + 1`
    done
    if [ $BADP = "unset" ]; then
      echo "*** could not find bad package"
      exit 1
    else
      echo "*** now trying to minimize by function"
      mv /tmp/cabiTest /tmp/cabiTest.pkg
      # Minimize by func
      F=0
      while [ $F != $NF ]; do
        echo trying fcn $F
        cd $HERE
        rm -rf $D
        CMD="./cabi-testgen -q $NP -n $NF -s $SEED -o $D -p cabiTest $PRAG -P $BADP -M $F"
        echo $CMD
        $CMD
        if [ $? != 0 ]; then
          echo "*** gen failed: $CMD"
          exit 1
        fi
        cd $D
        rm -f cabiTest
        go build -gcflags=all="$GCFLAGS2" . 1> build.err.${P}.${F}.txt 2>&1
        if [ $? != 0 ]; then
          echo "found offending function $F"
          BADF=$F
          break
        fi
        F=`expr $F + 1`
      done
      if [ $BADF = "unset" ]; then
        echo "*** could not find bad function"
      fi
      echo "... bad package $BADP bad function $BADF"
      
      # Clean unused packages
      cleanUnused $BADP $NP $D
    fi
    exit 1
  fi
  if [ $DOSETGOGC = "yes" ]; then
    echo export GOGC=1
    export GOGC=1
  fi
  echo "... running"
  ./cabiTest 1> ${HERE}/run.err.txt 2>&1
  RC=$?
  export GOGC=
  if [ $RC != 0 ]; then
    cd $HERE
    head run.err.txt
    PIPE="|"
    echo "*** running generated code failed, SEED=$SEED, see run.err.txt"
    if [ $DOMINIMIZE != "yes" ]; then
      exit 1
    fi
    # Minimize based on error in least-complex function.
    FAILURES=`cat run.err.txt | fgrep 'Error: fail' | cut -f2-4 -d${PIPE}`
    CM=99999999999
    PK="unset"
    FN="unset"
    for FR in $FAILURES
    do
      C=`echo $FR | cut -f1  -d${PIPE}`
      if [ $C -lt $CM ]; then
        PK=`echo $FR | cut -f2 -d${PIPE}`
        FN=`echo $FR | cut -f3 -d${PIPE}`
        CM=$C
      fi
    done
    echo PK=$PK FN=$FN CM=$CM
    if [ "$PK" != "" -a $PK -ge 0 -a $PK -lt $NP ]; then
      if [ "$FN" != "" -a $FN -ge 0 -a $FN -lt $NF ]; then
        CMD="./cabi-testgen -q $NP -n $NF -s $SEED -o $D -p cabiTest $PRAG -P $PK -M $FN"
        $CMD
        if [ $? != 0 ]; then
          echo "*** minimization run failed"
          exit 2
        fi
        # Clean unused packages
        cleanUnused $PK $NP $D
        echo "minimization complete"
      fi
    fi
    exit 1
  fi
  cd $HERE
  SEED=`expr $SEED + 17`
done
