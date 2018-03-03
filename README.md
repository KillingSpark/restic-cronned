Restic-Cronned
==============
This tool is a cron-like daemon that performs periodic commands using restic. (backup/forget/...)  
Obviously depends on [restic](https://github.com/restic/restic)
# Features #
* multiple jobs
* jobs that can trigger follow-up jobs
* passwords from all keyrings that github.com/zalando/go-keyring supports
* separate timer for retries from regular timers
* an http server where you can fetch the current state of your jobs

## Usage ##
`restic-cronned "/path/to/job/diretory" ":someport"`  
The port is optional, if not given the server wont be started
  
## Job definition ##
A Job is one restic action like backup or forget. It can be triggered periodically by itself or it can be triggered by another Job.  
A Job can for example backup a folder and then trigger a forget on the same repo. With this approach no lock races should occur.

Jobs are defined in json files with this structure (see ExampleBackup/Forget.json):  
These files need to be in a directory, that is specified by the first command line parameter
```
{
    "regularTimer":     int (seconds),  //interval for regular starting of a job. Can be set x < 0 for a job that only gets triggered by other jobs
    "retryTimer":       int (seconds),  //interval for retries if a job has failed.
    "maxFailedRetries": int,            //maximum retries before the job is killed entirely. Can be set to x < 0 for infinitly many  
    "JobName":          string,         //Identifies the Job. Recommended to be the same as the filename
    "NextJob"           string,         //Identifies the Job that should be triggered after this one
    "Username":         string,         //Username that was used to put the restic-repo password into the keyring
    "Service":          string,         //Service that was used to put the restic-repo password into the keyring
    "ResticPath":       string          //Optional path to the executable of restic (maybe different versions for different repos, not in PATH...)
    "ResticArguments":  [string]        //all arguments for restic   
}
```

This example backups /var/www/my-site daily.  
If the backup failed (maybe for connectivity issues or whatever) it retries 3 times hourly  
It also forgets about the old snapshots and keeps the last 30 (about a month) by triggering forget from the Backup 

ExmapleBackup.json
```
{
    "regularTimer": 86400,
    "retryTimer": 3600,
    "maxFailedRetries": 3,
    "JobName": "ExampleBackup",
    "NextJob": "ExampleForget",
    "Username":"Apache",
    "Service": "restic-repo1",
    "ResticArguments": ["-r", "/tmp/backup", "backup", "/var/www/my-site"]
}
```
  
This job will not be triggered on its own, but only if the backup job succeeded
  
ExampleForget.json
```
{
    "regularTimer": -1,
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
* `rckeyutil set Service Username key`
* `rckeyutil del Service Username`
* `rckeyutil get Service Username`

Example with reference to the example jobs: `rckeyutil set restic-repo1 Apache 1234`

## Http server ##
Started only if a port is given as the second command line argument  
Serves the queue in json format at "/queue". Note that times are represented in nano-seconds internally.  
Exposes commands as:  
* `/stop?name=JOBNAME`
* `/stopall`
* `/restart?name=JOBNAME`
* `/reload?name=JOBNAME` <-- requires the file to be named `JOBNAME.json`

You can use the rccommand tool to do these for you if you dont wnat to use curl

# Future plans #
1. Better configuration maybe using viper/cobra/...
1. Improve lock watching for repos. Right now there are still race conditions. Every job can still fail once for every race he looses.
1. better output for/from the command-wrapper tool

