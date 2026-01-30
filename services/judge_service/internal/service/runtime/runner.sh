#!/bin/sh
set -eu

LANG=""
WORKDIR=""
PHASE=""
OUTBIN=""
TIMEOUT=""

while [ $# -gt 0 ]; do
  case "$1" in
    --lang)
      LANG="$2"; shift 2;;
    --workdir)
      WORKDIR="$2"; shift 2;;
    --phase)
      PHASE="$2"; shift 2;;
    --outbin)
      OUTBIN="$2"; shift 2;;
    --timeout)
      TIMEOUT="$2"; shift 2;;
    *)
      echo "unknown arg: $1" >&2; exit 2;;
  esac
done

if [ -z "$LANG" ] || [ -z "$WORKDIR" ] || [ -z "$PHASE" ] || [ -z "$TIMEOUT" ]; then
  echo "missing args" >&2
  exit 2
fi

cd "$WORKDIR"

case "$LANG" in
  go)
    case "$PHASE" in
      compile)
        if [ ! -f go.mod ]; then
          go mod init sandbox >/dev/null 2>&1 || true
        fi
        timeout "${TIMEOUT}s" go mod tidy >/dev/null 2>&1 || true
        timeout "${TIMEOUT}s" go build -o "$OUTBIN" .
        ;;
      run)
        timeout "${TIMEOUT}s" "$OUTBIN"
        ;;
      *)
        echo "unknown phase: $PHASE" >&2; exit 2;;
    esac
    ;;
  python)
    case "$PHASE" in
      compile)
        exit 0
        ;;
      run)
        timeout "${TIMEOUT}s" python3 main.py
        ;;
      *)
        echo "unknown phase: $PHASE" >&2; exit 2;;
    esac
    ;;
  *)
    echo "unsupported language: $LANG" >&2
    exit 2
    ;;
esac