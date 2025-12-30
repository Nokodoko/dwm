#!/bin/sh

# This script is used by lf to preview files.
# It uses chafa for images and bat for text files.

case "$1" in
    *.jpg|*.jpeg|*.png|*.gif|*.bmp|*.webp)
        # Use chafa to display an image preview in the terminal.
        # Explicitly setting format to kitty for better compatibility.
        chafa --format kitty -s "$2x$3" "$1"
        ;;
    *)
        # For other files, try to display them as text using bat.
        # bat provides syntax highlighting and is a good default.
        bat --color=always --paging=never --style=plain "$1"
        ;;
esac