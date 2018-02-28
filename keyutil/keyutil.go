package keyutil

import (
	"os"

	keyring "github.com/zalando/go-keyring"
)

func printKeyringUtilHelp() {
	println("need action, service, username, and possibly password")
	println("action: [get,set,del]")
	println("service, username: mandatory but any string")
	println("password: mandatory for set, ignored for del and get")
}

//KeyRingUtil starts the keyring util for setting/getting/deleting passwords
func KeyRingUtil() {
	if len(os.Args) < 4 {
		println(len(os.Args))
		printKeyringUtilHelp()
		return
	}

	var err error
	var key string
	switch os.Args[1] {
	case "set":
		err = keyring.Set(os.Args[2], os.Args[3], os.Args[4])
		break
	case "get":
		key, err = keyring.Get(os.Args[2], os.Args[3])
		if err == nil {
			println("Key: " + key)
		}
		break
	case "del":
		err = keyring.Delete(os.Args[2], os.Args[3])
		break
	default:
		printKeyringUtilHelp()
	}

	if err != nil {
		println(err.Error())
	}

}
