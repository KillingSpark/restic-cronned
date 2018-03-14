#!/bin/sh

mkdir -p $HOME/.local/share/applications
ln -s desktopfiles/restic-cronned.desktop $HOME/.local/share/applications/restic-cronned.desktop
go build
mkdir -p $HOME/.local/bin
ln -s restic-cronned $HOME/.local/bin/restic-cronned
mkdir -p $HOME/.config/restic-cronned/jobs
