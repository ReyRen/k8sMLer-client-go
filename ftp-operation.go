package main

import (
	"github.com/dutchcoders/goftp"
	"io"
	"strconv"
	"time"
)

func ftpUploader(c *Client, r io.Reader) {

	dirPath := "user/" + strconv.Itoa(c.userIds.Uid) + "/" + strconv.Itoa(c.userIds.Tid) + "/log/"

	timestamp := time.Now().Unix()
	tm := time.Unix(timestamp, 0)
	timeStamp := tm.Format("20060102030405")

	fileName := strconv.Itoa(c.userIds.Uid) +
		"_" + strconv.Itoa(c.userIds.Tid) +
		"_" + timeStamp +
		"_log.txt"

	var err error
	var ftp *goftp.FTP

	if ftp, err = goftp.Connect(FTPSERVER); err != nil {
		Error.Printf("[%d, %d]: goftp connect err:%s\n", c.userIds.Uid, c.userIds.Tid, err)
	}

	defer ftp.Close()
	Trace.Printf("[%d, %d]: goftp connect successfully\n", c.userIds.Uid, c.userIds.Tid)

	// Username / password authentication
	if err = ftp.Login("ftper", "admin"); err != nil {
		Error.Printf("[%d, %d]: goftp login err:%s\n", c.userIds.Uid, c.userIds.Tid, err)
	}

	if err := ftp.Stor(dirPath+fileName, r); err != nil {
		Error.Printf("[%d, %d]: goftp stor err:%s\n", c.userIds.Uid, c.userIds.Tid, err)
	}
}
