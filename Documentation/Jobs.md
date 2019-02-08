# Jobs 
Jobs represent the action of running restic (or any other executable really) _once_. They also can be defined with preconditions that they 
can wait for.

## Preconditions
Preconditions represent conditions that have to be met before starting a job. Currently there is support for:

1. Check for directories
2. Check for routing to a server
3. Check for ping to a server

There are plans to add support for udev checks.

## Config
{
    "Kind": {
        "Name": "Job"
    },
    "Spec": {
        "Name": "backup",
        "ResticPath": "/bin/false",

        "CheckPrecondsEvery": 20000,
        "CheckPrecondsMaxTimes": 100,
        "Preconditions": {
            "HostsMustRoute": ["localhost"],
            "PathesMust": ["./testrepo"],
            "HostsMustConnect": [{"Host": "google.com", "Port": 80}]
    }
}