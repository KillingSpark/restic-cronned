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
If it is missing for you: its just a symlink to the main executable (it determines behaviour from the executables name)
* rckeyutil set Service Username key
* rckeyutil del Service Username
* rckeyutil get Service Username

## Http server ##
Started onyl if a port is given as the second commandline argument
Serves the queue in json format at "/". Note that times are represented in nano-seconds internally.  
The status will maybe in the future be mapped to a string for better readability. Until then refer to the job.go file for semantics.  

# Future plans #
1. Better configuration maybe using viper/cobra/...

