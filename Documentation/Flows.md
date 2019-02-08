# Flows
FLows are a network of triggerers and targets that may themselves be triggerers again.
Currently the network is limitied to beeing a tree.

Think a tree where on the root a triggerer triggers all children which propagate the trigger further.

## Config
The Toplevel "Flows" can hold any amount of different flows. You can also have multiple .flow files, they will be merged into one 
big flow collection internally (as long as the names stay unique).
{
    "Flows":{
        "BackupFlow": {
            "Name": "BaclupFlow",
            "Root": {
                "Name": "timer",
                "Targets": [
                    {
                        "Name": "retry",
                        "Targets": [
                            {
                                "Name": "oneshot",
                                "Targets": [
                                    {
                                        "Name": "backup"
                                    }
                                ]
                            },
                            {
                                "Name": "backup"
                            },
                            {
                                "Name": "oneshot",
                                "Targets": [
                                    {
                                        "Name": "backup"
                                    },
                                    {
                                        "Name": "backup"
                                    }
                                ]
                            }
                        ] 
                    }
                ]
            }
        }
    }
}