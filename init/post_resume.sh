#!/bin/bash
# This script executes the `post_resume` function defined in configuration files
# located in the /etc/powerd/board directory. It is triggered after the system resumes.

BOARD_DIR=/etc/powerd/board
FUNC=post_resume

main() {
  # Iterate over all .conf files in the BOARD_DIR directory
  for conf in $(ls ${BOARD_DIR}/*.conf 2>/dev/null); do
    # Check if the configuration file is readable
    if [ -r $conf ]; then
      # Source the configuration file to load its functions
      source $conf
      # Check if the `post_resume` function is defined
      if declare -F $FUNC &>/dev/null; then
        # Execute the `post_resume` function
        $FUNC
        # Unset the function to avoid conflicts
        unset $FUNC
      fi
    fi 
  done
}

# Call the main function
main