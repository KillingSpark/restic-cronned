# Timers
Timers represent a periodic trigger. They trigger all targets periodically defined by a cron-string. 
They are meant to be at the root of a flow and let the targets handle fanning out and retrying if necessary.

## Config

```
{
    "Kind": {
        "Name": "Timer"
    },
    "Spec": {
        "Name": "backuptimer",
        "Timer": "@every 3s"
    }
}
```