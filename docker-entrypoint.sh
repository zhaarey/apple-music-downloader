#!/bin/sh
# Container entrypoint for amdl.
#
# The default `lite-server` (127.0.0.1) points at the container itself, so the
# wrapper-lite address is usually supplied from outside. Setting
# AMDL_LITE_SERVER does that without editing config.yaml; an explicit
# --lite-server argument still wins.
set -e

if [ -n "${AMDL_LITE_SERVER:-}" ]; then
  case " $* " in
    *" --lite-server"*) ;;
    *) set -- --lite-server "$AMDL_LITE_SERVER" "$@" ;;
  esac
fi

exec amdl "$@"
