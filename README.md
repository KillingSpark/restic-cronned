Restic-Cronned
==============
This tool is a cron-like daemon that performs periodic commands using restic. (backup/forget/...)  
Obviously depends on [restic](https://github.com/restic/restic)
# Features #
* multiple jobs
* jobs that can trigger follow-up jobs
* passwords from all keyrings that github.com/zalando/go-keyring supports
* timers in cron style format
* separate timer for retries from regular timers
* an http server where you can fetch the current state of your jobs

## Usage ##
```
usage: restic-cronned [<flags>]

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
  -p, --port=PORT              Which port the server should listen on (if any)
  -j, --jobpath=JOBPATH        Which directory contains the job descriptions
  -c, --configpath=CONFIGPATH  Which directory contains the config file

```
The port is optional, if not given the server wont be started
  
The config file resides in $HOME/.config/restic-cronned/ or /etc/restic-cronned/ and looks like this:

```
config.json

{
    "JobPath": "$HOME/.config/restic-cronned/jobs",
    "ServerPort": ":8080",
    "LogDir": "$HOME/.cache/restic-cronned",
    "LogMaxAge": 30,
    "LogMaxSize": 10
}
```
If any of the values are not present in your config they will default to these values.  
Note that the values for MaxAge are given in Days and MaxSize is in MB. They correspond with the values for https://github.com/rshmelev/lumberjack  
Note also that the path and port on the commandline take precedence over the config file.  


## Job definition ##
A Job is one restic action like backup or forget. It can be triggered periodically by itself or it can be triggered by another Job.  
A Job can for example backup a folder and then trigger a forget on the same repo. With this approach no lock races should occur.

Jobs are defined in json files with this structure (see ExampleBackup/Forget.json):  
These files need to be in a directory, that is specified by the first command line parameter
```
{
    "regularTimer":     string          //cron style definition of a time (non standard, the first entry is seconds not minutes)
    "retryTimer":       string          //cron style definition of a time (non standard, the first entry is seconds not minutes)          
    "maxFailedRetries": int,            //maximum retries before the job is killed entirely. Can be set to x < 0 for infinitly many  
    "JobName":          string,         //Identifies the Job. Recommended to be the same as the filename
    "NextJob"           string,         //Identifies the Job that should be triggered after this one
    "Username":         string,         //Username that was used to put the restic-repo password into the keyring
    "Service":          string,         //Service that was used to put the restic-repo password into the keyring
    "ResticPath":       string,          //Optional path to the executable of restic (maybe different versions for different repos, not in PATH...)
    "ResticArguments":  [string],        //all arguments for restic

    "CheckPrecondsEvery": int,           //If the check fails, retry x seconds later again
    "CheckPrecondsMaxTimes": int         //After y attempts the preconditions on this job are assumed to not be met any time in this period
    "Preconditions":
    {
        "PathesMust": [string],          //Pathes that must be present and not empty for the job to run (e.g. the mount point of an nfs)
        "HostsMustRoute": [string],      //Hosts that must be routable for the job to run (e.g. the host of an nfs or a sftp server)
        "HostsMustConnect":              //Hosts that must be connectable with tcp on the specified port
        [
            {"Host": string, "Port": int}
        ],
    },   
}
```

### Preconditions ###
Preconditions are checks that are performed before the actual restic command is executed. Especially network availability is an important precondition for users that suspend their system. It is very possible that a job would wake up and try to backup while the network isnt up yet. That would delay the 
job execution unecessarly until the next retry timer.

Retry timer exist for actual failures(maybe other processes lock the repo, the connection dropped in the middle,...)

### Example ###
This example backups /var/www/my-site at 02:00am to a nfs (served by the server mynfshost) mounted on /tmp/backup.  
If the backup failed (maybe for connectivity issues or whatever) it retries hourly, 3 times total.  
It also forgets about the old snapshots and keeps the last 30 (about a month) by triggering a ForgetJob from the BackupJob.  

ExmapleBackup.json
```
{
    "regularTimer": "0 0 2 * * *",
    "retryTimer": "0 0 * * * *",
    "maxFailedRetries": 2,
    "JobName": "ExampleBackup",
    "NextJob": "ExampleForget",
    "Username":"Apache",
    "Service": "restic-repo1",
    "ResticArguments": ["-r", "/tmp/backup", "backup", "/var/www/my-site"],

    "CheckPrecondsEvery": 20,
    "CheckPrecondsMaxTimes": 100,
    "Preconditions": {
        "HostsMustRoute": ["mynfshost"],
        "PathesMust": ["/tmp/backup"],
        "HostsMustConnect": [{"Host": "google.com", "Port": 80}]
    }
}
```
  
This job will not be triggered on its own, but only if the backup job succeeded
  
ExampleForget.json
```
{
    "retryTimer": "0 0 * * * *",
    "maxFailedRetries": 3,
    "JobName": "ExampleForget",
    "Username":"Apache",
    "Service": "restic-repo1",
    "ResticArguments": ["-r", "/tmp/backup", "forget", "--keep-last", "30"]
}
```

## Restarts/Suspends/Crashes ##
When a job gets scheduled it calculates the time when it should wake up. Then it sleeps for 10 seconds and checks against this time, until the limit is reached. This way restarts/suspends/chrashes should not bother the jobs too much. Jobs that should have been run when the system was suspended will be run (almost) immediatly when it becomes unsuspended.

## Passwords ##
For convenience (and to be sure the keys can be read correctly from the keyring) the rckeyutil should be used to set/get/delete the repo keys.  
Usage:
* `rckeyutil set Service Username key`
* `rckeyutil del Service Username`
* `rckeyutil get Service Username`

Example: Set a password '1234' with reference to the example jobs: `rckeyutil set restic-repo1 Apache 1234`

## Http server ##
Started only if a port is given as the second command line argument or in the config file  
Serves the queue in json format at "/queue". Note that times are represented in nano-seconds internally.  
Exposes commands as:  
* `/stop?name=JOBNAME`
* `/stopall`
* `/restart?name=JOBNAME`
* `/reload?name=JOBNAME` <-- requires the file to be named `JOBNAME.json`

You can use the rccommand tool to do these for you if you dont want to use curl
* rccommands COMMAND JOBNAME
Translates into ```/COMMAND?name=JOBNAME```. If no name is needed it is ignored if given. 

# Future plans #
2. Improve lock watching for repos. Right now there are still race conditions if two jobs are working on the same repo. But one can use the triggers to avoid these.
3. Better output for/from the command-wrapper tool

