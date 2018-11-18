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
  
The config file resides in $HOME/.config/restic-cronned/config and looks like this:
```
{
    "JobPath": "$HOME/.config/restic-cronned/jobs",
    "SrvConf": {"Port": ":8080"},
    "LogConf": {
        "LogDir": "$HOME/.cache/restic-cronned",
        "MaxAge": 30,
        "MaxSize": 10
    }
}
```
If any of the values are not present in your config they will default to these values.  
Note that the values for MaxAge are given in Days and MAxSize is in MB. They correspond with the values for https://github.com/rshmelev/lumberjack  
Note also that the path and port on the commandline take precedence over the config file.  


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
    "ResticPath":       string,          //Optional path to the executable of restic (maybe different versions for different repos, not in PATH...)
    "ResticArguments":  [string],        //all arguments for restic

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

This example backups /var/www/my-site daily to a nfs (served by the server mynfshost) mounted on /tmp/backup
If the backup failed (maybe for connectivity issues or whatever) it retries 3 times hourly  
It also forgets about the old snapshots and keeps the last 30 (about a month) by triggering forget from the Backup 

ExmapleBackup.json
```
{
    "regularTimer": 86400,
    "retryTimer": 3600,
    "maxFailedRetries": 2,
    "JobName": "ExampleBackup",
    "NextJob": "ExampleForget",
    "Username":"Apache",
    "Service": "restic-repo1",
    "ResticArguments": ["-r", "/tmp/backup", "backup", "/var/www/my-site"],
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
    "regularTimer": -1,
    "retryTimer": 3600,
    "maxFailedRetries": 3,
    "JobName": "ExampleForget",
    "Username":"Apache",
    "Service": "restic-repo1",
    "ResticArguments": ["-r", "/tmp/backup", "forget", "--keep-last", "30"]
}
```

Jobs save their last successful run time in a file in $HOME/.local/share/restic-cronned/<Jobname>. If this file exists when the Job gets started it 
calculates his initial trigger time accordingly so the jobs run somewhat regularly if the system/the queue is restarted

## Restarts/Suspends/Crashes ##
Whenever a job schedules a trigger a file is written: "$HOME/.local/shar/restic-cronned/JOBNAME" this contains a timestamp when the job should be triggered. When a job is started it looks for this file.  
* If it does not exist the job gets triggered right away (new jobs for example!).  
* If it does exist but is not readable/decodable the job will wait its regular time for the next trigger (which will be written into this file)  
* If it does exist and the timestamp is still in the future it will wait the time left until this timestamp.
* If it does exist and the timestamo is in the past the job will be triggered right away. Until now the job will then wait its regular timer. That can shift your cycle. Plans to fix this exist.   


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

You can use the rccommand tool to do these for you if you dont want to use curl
* rccommands COMMAND JOBNAME
Translates into ```/COMMAND?name=JOBNAME```. If no name is needed it is ignored if given. 

# Future plans #
1. Better configuration maybe using viper/cobra/... (right now just loading a config json file. Works well enough though)
1. Improve lock watching for repos. Right now there are still race conditions if two jobs are working on the same repo. But one can use the triggers to avoid these.
1. Better output for/from the command-wrapper tool

