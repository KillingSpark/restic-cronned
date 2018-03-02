Restic-Cronned
==============
This tool is a cron-like daemon that performs periodic commands using restic. (backup/forget/...)  
Obviously depends on [restic](https://github.com/restic/restic)
# Features #
* multiple jobs
* passwords from all keyrings that github.com/zalando/go-keyring supports
* separate timer for retries from regular timers
* an http server where you can fetch the current state of your jobs

## Usage ##
`restic-cronned "/path/to/job/diretory" ":someport"`  
The port is optional, if not given the server wont be started
  
## Job definition ##
Jobs are defined in json files with this structure (see ExampleBackup/Forget.json):  
These files need to be in a directory, that is specified by as the first commandline parameter
```
{
    "regularTimer":     int (seconds),  //intervall for regular starting of a job
    "retryTimer":       int (seconds),  //intervall for retries if a job has failed.
    "maxFailedRetries": int,            //maximum retries before the job is killed entirely. Can be set to x < 0 for infinitly many  
    "JobName":          string,         //Identifies the Job. Recommended to be the same as the filename
    "Username":         string,         //Username that was used to put the restic-repo password into the keyring
    "Service":          string,         //Service that was used to put the restic-repo password into the keyring
    "ResticPath":       string          //Optional path to the executable of restic (maybe different versions for different repos, not in PATH...)
    "ResticArguments":  [string]        //all arguments for restic   
}
```

This example backups /var/www/my-site dayly.  
If the backup failed (maybe for connectivity issues or whatever) it retries 3 times hourly  
It also forgets about the old snapshots and keeps the last 30 (about a month)

ExmapleBackup.json
```
{
    "regularTimer": 86400,
    "retryTimer": 3600,
    "maxFailedRetries": 3,
    "JobName": "ExampleBackup",
    "Username":"Apache",
    "Service": "restic-repo1",
    "ResticArguments": ["-r", "/tmp/backup", "backup", "/var/www/my-site"]
}
```

ExampleForget.json
```
{
    "regularTimer": 86400,
    "retryTimer": 3600,
    "maxFailedRetries": 3,
    "JobName": "ExampleForget",
    "Username":"Apache",
    "Service": "restic-repo1",
    "ResticArguments": ["-r", "/tmp/backup", "forget", "--keep-last", "30"]
}
```

## Passwords ##
For convenience (and to be sure the keys can be read correctly from the keyring) the rckeyutil can be used to set/get/delete the repo keys.  
Usage:
* rckeyutil set Service Username key
* rckeyutil del Service Username
* rckeyutil get Service Username
Example with reference to the example jobs: `rckeyutil set restic-repo1 Apache 1234`

## Http server ##
Started only if a port is given as the second commandline argument  
Serves the queue in json format at "/queue". Note that times are represented in nano-seconds internally.  
Exposes commands as:  
* /stop?name=JOBNAME
* /restart?name=JOBNAME
* /reload?name=JOBNAME <-- requires the file to be named JOBNAME.json

# Future plans #
1. Better configuration maybe using viper/cobra/...
1. Improve lock watching for repos. Right now there are still raceconditions. Every job can still fail once for every race he looses.
1. Tool to wrap the commands to the http server. Users could use curl but... meh

