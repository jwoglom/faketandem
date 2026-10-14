#!/bin/bash
while ! ./main $@
do
  sleep 1
  echo "Restarting program..."
done
