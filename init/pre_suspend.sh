#!/bin/bash
BOARD_DIR=/etc/powerd/board
FUNC=pre_suspend
main() {
  for conf in $(ls ${BOARD_DIR}/*.conf 2>/dev/null); do
    if [ -r $conf ]; then
      source $conf
      if declare -F $FUNC  &>/dev/null; then
        $FUNC
        unset $FUNC
      fi
    fi 
  done
}

main
