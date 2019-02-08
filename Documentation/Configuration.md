# Configuartion
Configuration is done in the file config.json.

{
    "Dir": "$HOME/.config/restic-cronned/",
    "LogDir": "$HOME/.cache/restic-cronned",
    "LogMaxAge": 30,
    "LogMaxSize": 10
}

* "Dir" tells restic-cronned where it should look for files containing jobs/timers/...
* "LogDir" tells restic-cronned where to write logs to
* "Log*" are parameters that are passed to lumberjack which makes to logs

## Objects
Jobs/Timers/... are objects which are generally configured like this:

{
    "Kind": {
        "Name": "TYPENAME"
    },
    "Spec": {
       "TYPE":"SPECIFIC"
    }
}

See related files for the concrete configs.