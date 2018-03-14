#!/bin/sh

ln -s desktopfiles/restic-cronned.desktop $HOME/.local/share/applications/restic-cronned.desktop
go build
ln -s restic-cronned $HOME/.local/bin/restic-cronned
mkdir -p $HOME/.config/restic-cronned/jobs
