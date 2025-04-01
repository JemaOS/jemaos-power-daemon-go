#!/bin/bash
# This script executes the `pre_suspend` function defined in configuration files
# located in the /etc/powerd/board directory. It is triggered before the system suspends.

BOARD_DIR=/etc/powerd/board
FUNC=pre_suspend

main() {
  # Iterate over all .conf files in the BOARD_DIR directory
  for conf in $(ls ${BOARD_DIR}/*.conf 2>/dev/null); do
    # Check if the configuration file is readable
    if [ -r $conf ]; then
      # Source the configuration file to load its functions
      source $conf
      # Check if the `pre_suspend` function is defined
      if declare -F $FUNC &>/dev/null; then
        # Execute the `pre_suspend` function
        $FUNC
        # Unset the function to avoid conflicts
        unset $FUNC
      fi
    fi 
  done
}

# Call the main function
main