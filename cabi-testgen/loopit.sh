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
PRAG=""
PRAG="-pragma registerparams -method=0"
NP=20
NF=20
while [ $ITER !=  0 ]; do
  echo iter $ITER
  ITER=`expr $ITER - 1`
  echo "Iter $ITER"
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
  go build -gcflags="-c=1" -p 1 . 1> ${HERE}/build.err.txt 2>&1
  if [ $? != 0 ]; then
     echo "*** building generated code failed, SEED=$SEED see build.err.txt"
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
       go build -gcflags="-c=1" -p 1 . 1> build.err.${P}.txt 2>&1
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
         go build -gcflags="-c=1" -p 1 . 1> build.err.${P}.${F}.txt 2>&1
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
       P=0
       while [ $P != $NP ]; do
         if [ $P != $BADP ]; then
           echo "... cleaning unused $P"
           rm -rf $D/genChecker$P $D/genCaller$P
         fi
         P=`expr $P + 1`
       done
     fi
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
