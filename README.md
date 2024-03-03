## Usage
  Jemaos power daemon is wroted to receive the signals from cros power daemon, and run some script.

## Signals 
#### Suspend/Resume
config dirctory: /etc/powerd/board
config file: ${board-name}/${target-name}.conf

format: bash script

functions:
  pre_suspend: 
     to run some commands before system suspend
     limitation: 0.2s timeout
  post_resume:
     to run some commands after system resume

test the script:
  run /etc/powerd/pre_suspend.sh to test pre suspend config
  run /etc/powerd/post_resume.sh to test post resume config

#### SetScreenBrightness
Store the screen brightness which set by users, and restore it at system starting.
