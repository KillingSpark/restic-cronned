# Oneshot
If you want to fan out your flow into multiple, oneshots are meant to do that.
While timers/retriers do primitivly support multiple targets they really should only have one.

They can trigger all targets in sequence or parallel. They will in the future support sophisticated selection of 
results of the targets of the oneshot to pass back to the triggerer that triggered the oneshot.

## Config

```
{
    "Kind": {
        "Name": "Oneshot"
    },
    "Spec": {
        "Name": "parshot",
        "Parallel": true
    }
}
```