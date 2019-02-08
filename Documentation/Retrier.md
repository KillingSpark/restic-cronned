# Retrier
You may want to add a "retrier" into your config if you think your preconditions do not sufficiently ensure that your job will succeed.
They work similiarly to a timer, but they stop repeating as soon as the target(s) succeed (or the max number of tries is reached).


## Config

```
{
    "Kind": {
        "Name": "Retry"
    },
    "Spec": {
        "Name": "retry",
        "Timer": "@every 1s",
        "MaxFailedRetries": 2
    }
}
```