# Overview 

This document should give an overview of all used concepts in restic cronned. This is going to look like a lot, but it is important to not that
most of it is not needed to _run_ restic cronned. For that you can probably just copy the Exmaples and do what feels intuitive.

For extending restic-cronned or advanced/complex flows you should read this though. 

Details to how the config for stuff looks like see the related document.

## Flows
FLows are a network of triggerers and targets that may themselves be triggerers again.
Currently the network is limitied to beeing a tree.

Think a tree where on the root a triggerer triggers all children which propagate the trigger further.


## Jobs 
Jobs represent the action of running restic (or any other executable really) _once_. They also can be defined with preconditions that they 
can wait for.

### Preconditions
Preconditions represent conditions that have to be met before starting a job. Currently there is support for:

1. Check for directories
2. Check for routing to a server
3. Check for ping to a server

There are plans to add support for udev checks.

## Timers
Timers represent a periodic trigger. They trigger all targets periodically defined by a cron-string. 

## Retrying
You may want to add a "retrier" into your config if you think your preconditions do not sufficiently ensure that your job will succeed.
They work similiarly to a timer, but they stop repeating as soon as the target(s) succeed (or the max number of tries is reached).

## Oneshot
If you want to fan out your flow into multiple, oneshots are meant to do that.
While timers/retriers do primitivly support multiple targets they really should only have one.

They can trigger all targets in sequence or parallel. They will in the future support sophisticated selection of 
results of the targets of the oneshot to pass back to the triggerer that triggered the oneshot.