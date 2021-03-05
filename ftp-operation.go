package main

import (
	"bufio"
	"context"
	"github.com/dutchcoders/goftp"
	"io"
	"k8s.io/client-go/rest"
	"strconv"
)

//func ftpUploader(c *Client, r io.Reader) {
func ftpUploader(c *Client, result2 *rest.Request) {

	dirPath := "user/" + strconv.Itoa(c.userIds.Uid) + "/" + strconv.Itoa(c.userIds.Tid) + "/log/"

	var err error
	var ftp *goftp.FTP

AGAIN:
	podLogs2, err := result2.Stream(context.TODO())
	if err != nil {
		Error.Printf("[%d, %d]: podLogs2 err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}
	defer podLogs2.Close()
	/*rr := bufio.NewReader(podLogs2)
	_, err = rr.ReadBytes('\n')
	if err == io.EOF {
		Trace.Println("FTP stream EOF get")
		goto AGAIN
	}*/

	if ftp, err = goftp.Connect(FTPSERVER); err != nil {
		Error.Printf("[%d, %d]: goftp connect err:%s\n", c.userIds.Uid, c.userIds.Tid, err)
	}
	defer ftp.Close()
	Trace.Printf("[%d, %d]: goftp connect successfully\n", c.userIds.Uid, c.userIds.Tid)

	// Username / password authentication
	if err = ftp.Login("ftper", "admin"); err != nil {
		Error.Printf("[%d, %d]: goftp login err:%s\n", c.userIds.Uid, c.userIds.Tid, err)
	}

	if err := ftp.Stor(dirPath+c.hub.clients[*c.userIds].Head.rm.FtpFileName, podLogs2); err != nil {
		Error.Printf("[%d, %d]: goftp stor err:%s, activate again\n", c.userIds.Uid, c.userIds.Tid, err)
		podLogs2.Close()
		ftp.Close()
		goto AGAIN
	} else {
		rr := bufio.NewReader(podLogs2)
		_, err = rr.ReadBytes('\n')
		if err == io.EOF {
			Trace.Printf("[%d, %d]: FTP stream EOF get, activate again...\n", c.userIds.Uid, c.userIds.Tid)
			podLogs2.Close()
			ftp.Close()
			goto AGAIN
		}
	}
}
