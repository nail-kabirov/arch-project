#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "Expects exactly 1 parameter - logged user session_id"
    exit 1
fi

while true; do
  ab -n 1000 -c 50  -C "session_id=$1" http://arch.homework/lot/api/v1/lots
done
