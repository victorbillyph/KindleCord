#!/bin/sh
EXT_DIR="$(dirname "$0")/.."
echo "test ok $(date)" > "$EXT_DIR/test_output.txt"
echo "PWD: $(pwd)" >> "$EXT_DIR/test_output.txt"
echo "0: $0" >> "$EXT_DIR/test_output.txt"
