#!/bin/bash

HOSTS=("monty" "base")
SRC="/home/n0ko/bling/dwm/"
DEST="bling/dwm/"

for host in "${HOSTS[@]}"; do
    echo "Syncing to $host..."
    ssh "$host" "mkdir -p $DEST"
    rsync -avz --exclude='.git' --exclude='*.o' --exclude='dwm' "$SRC" "$host:$DEST"
done
