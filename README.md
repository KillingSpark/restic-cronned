Restic-Cronned
==============
This tool is designed to perform periodic backups using restic (and it got a bit out of hand by means of complexity).

## Why would i not just use _CRON_?
You could honestly. But if you write a cron job for this you would probably have to implement many of the building blocks of this project in bash.

If you were to define another cron for another restic repo you would have to copy your code, or build a library for your multiple cron jobs.
With growing number of jobs/complexity this gets out of hand easily.

With this tool you dont need to bother about this issue. You can write simple timers or complex flows with retrying and parallel/sequential fanning out.

The "Jobs" can be defined to use any executable and parameters
* so you can also use borg or whatever tool you like to backup
* or run any other tool you want to run periodically

Plans exist to add an alternative to timers which listen on udev events or on a unix-socket.

I will rename this project as soon as I find a suitable name for all the features this tool captures right now.

## Documentation
The wiki is out of date , sorry about that. If I find the time I will copy the files in "Documentation" into the wiki. Until such time please just refer to the markdown in the "Documentation" for up to date information.
