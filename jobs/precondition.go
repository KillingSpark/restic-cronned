package jobs

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
)

type JobPreconditions struct {
	PathesMust       []PathPrecond      `json:"PathesMust"`
	HostsMustRoute   []HostRoutePrecond `json:"HostsMustRoute"`
	HostsMustConnect []HostTCPPrecond   `json:"HostsMustConnect"`
}

func (jp *JobPreconditions) CheckAll() bool {
	allGood := true
	for _, pm := range jp.PathesMust {
		allGood = allGood && pm.CheckCondition()
	}
	for _, hmr := range jp.HostsMustRoute {
		allGood = allGood && hmr.CheckCondition()
	}
	for _, hmc := range jp.HostsMustConnect {
		allGood = allGood && hmc.CheckCondition()
	}
	return allGood
}

type PathPrecond string

func (pp *PathPrecond) CheckCondition() bool {
	stat, err := os.Stat(string(*pp))
	if err != nil {
		return false
	}
	if stat.IsDir() {
		list, err := ioutil.ReadDir(path.Join(string(*pp)))
		if err != nil {
			return false
		}
		if len(list) <= 0 {
			return false
		}
		return true
	}
	return false
}

type HostRoutePrecond string

func (hrp *HostRoutePrecond) CheckCondition() bool {
	_, err := net.LookupIP(string(*hrp))
	if err != nil {
		return false
	}
	return true
}

type HostTCPPrecond struct {
	Host string `json:"Host"`
	Port int    `json:"Port"`
}

func (htp *HostTCPPrecond) CheckCondition() bool {

	tcpAddr, err := net.ResolveTCPAddr("tcp", htp.Host+":"+strconv.Itoa(htp.Port))
	if err != nil {
		return false
	}
	con, err := net.DialTCP("tcp", nil, tcpAddr)
	defer con.Close()
	if err != nil {
		return false
	}
	return true
}
