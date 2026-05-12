#!/usr/bin/env bash
#
# GitHub‑friendly batch committing script
# ---------------------------------------
# Safe for thousands of binary WebP files. It:
#   • Creates batches (batch_000, batch_001, …)
#   • Moves N files per batch (default 250)
#   • Commits each batch separately
#   • Waits 2 seconds between commits
#
# Usage:
#   ./batch_commit.sh <source_dir> <batch_dir> [batch_size]
#
# Example:
#   ./batch_commit.sh ./output_images ./uploads 250

set -euo pipefail

SRC="${1:-}"
DEST="${2:-uploads}"
BATCH_SIZE="${3:-250}"

if [[ -z "$SRC" ]]; then
    echo "Error: missing <source_dir>"
    exit 1
fi

if [[ ! -d "$SRC" ]]; then
    echo "Error: source directory '$SRC' does not exist."
    exit 1
fi

# Create destination root if missing
mkdir -p "$DEST"

echo "----------------------------------------------"
echo "GitHub‑Friendly Batch Committer"
echo "Source directory:     $SRC"
echo "Destination root:     $DEST"
echo "Files per batch:      $BATCH_SIZE"
echo "----------------------------------------------"

# Collect all WebP files
mapfile -t FILES < <(find "$SRC" -maxdepth 1 -type f -name "*.webp" | sort)

TOTAL=${#FILES[@]}
if [[ "$TOTAL" -eq 0 ]]; then
    echo "No .webp files found in $SRC"
    exit 0
fi

echo "Found $TOTAL WebP files."

# Calculate total batches
BATCHES=$(( (TOTAL + BATCH_SIZE - 1) / BATCH_SIZE ))

echo "Will create $BATCHES batches."

for (( i=0; i<BATCHES; i++ )); do
    BATCH_NAME=$(printf "batch_%03d" "$i")
    BATCH_PATH="$DEST/$BATCH_NAME"

    # Skip if folder already exists (resume support)
    if [[ -d "$BATCH_PATH" ]]; then
        echo "Batch $BATCH_NAME already exists — skipping."
        continue
    fi

    mkdir -p "$BATCH_PATH"

    start=$(( i * BATCH_SIZE ))
    end=$(( start + BATCH_SIZE ))
    if (( end > TOTAL )); then
        end=$TOTAL
    fi

    echo "Creating $BATCH_NAME   ($start → $end)"

    # Move files into batch folder
    for (( j=start; j<end; j++ )); do
        mv "${FILES[j]}" "$BATCH_PATH/"
    done

    # Commit the batch
    git add "$BATCH_PATH"
    git commit -m "Add $BATCH_NAME ($((end-start)) files)"
    # git push

    echo "Pushed $BATCH_NAME. Waiting 2 seconds…"
    sleep 2
done

echo "----------------------------------------------"
echo "All batches committed and pushed successfully."
echo "----------------------------------------------"
