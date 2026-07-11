#!/bin/bash
while ! ./main -pumpx2-path pumpX2 $@
do
  sleep 1
  echo "Restarting program..."
done
