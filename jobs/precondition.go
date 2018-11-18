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
	res := make(chan bool)
	for _, pm := range jp.PathesMust {
		go func() { res <- pm.CheckCondition() }()
	}
	for _, hmr := range jp.HostsMustRoute {
		go func() { res <- hmr.CheckCondition() }()
	}
	for _, hmc := range jp.HostsMustConnect {
		go func() { res <- hmc.CheckCondition() }()
	}

	allGood := true

	for i := 0; i < len(jp.HostsMustConnect)+len(jp.HostsMustRoute)+len(jp.PathesMust); i++ {
		allGood = allGood && <-res
	}
	return allGood
}

type PathPrecond struct {
	Path string `json:"Path"`
}

func (pp *PathPrecond) CheckCondition() bool {
	stat, err := os.Stat(pp.Path)
	if err != nil {
		return false
	}
	if stat.IsDir() {
		list, err := ioutil.ReadDir(path.Join(pp.Path))
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

type HostRoutePrecond struct {
	Host string `json:"Host"`
}

func (hrp *HostRoutePrecond) CheckCondition() bool {
	_, err := net.LookupIP(hrp.Host)
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
