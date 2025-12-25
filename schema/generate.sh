#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Type generation is done manually for better type safety."
echo "The types are defined based on schema/events.schema.json"
echo ""
echo "Generated files:"
echo "  - backend/events_gen.go (Go types)"
echo "  - frontend/src/lib/types.ts (TypeScript types)"
echo ""
echo "If you modify the schema, please update both files manually."
