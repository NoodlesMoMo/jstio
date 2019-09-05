package util

import (
	"net"
	"regexp"
	"strings"
	"time"
)

var (
	multiBlankReg = regexp.MustCompile("[\n]{2,}")
)

func GetLocalIPV4Addr() string {
	conn, err := net.DialTimeout("udp", "sogou.com:80", 3*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()

	ip, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	return ip
}

func RemoveMultiBlankLine(src string) string {
	return multiBlankReg.ReplaceAllString(src, "\n\n")
}

func RemoveMultiBlankLineEx(src string) string {
	return strings.TrimSpace(RemoveMultiBlankLine(src)) + "\n"
}
