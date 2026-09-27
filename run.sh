#!/bin/bash
while ! ./main -pumpx2-jar-path third_party/pumpx2-cliparser-1.9.2.jar $@
do
  sleep 1
  echo "Restarting program..."
done
