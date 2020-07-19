package command

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ExecCmdDirBytes executes system command in given directory
// and return stdout, stderr in bytes type, along with possible error.
func ExecCmdDirBytes(dir, cmdName string, args ...string) ([]byte, []byte, error) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	cmd := exec.Command(cmdName, args...)
	cmd.Dir = dir
	cmd.Stdout = bufOut
	cmd.Stderr = bufErr

	err := cmd.Run()
	return bufOut.Bytes(), bufErr.Bytes(), err
}

// ExecCmdBytes executes system command
// and return stdout, stderr in bytes type, along with possible error.
func ExecCmdBytes(cmdName string, args ...string) ([]byte, []byte, error) {
	return ExecCmdDirBytes("", cmdName, args...)
}

// ExecCmdDir executes system command in given directory
// and return stdout, stderr in string type, along with possible error.
func ExecCmdDir(dir, cmdName string, args ...string) (string, string, error) {
	bufOut, bufErr, err := ExecCmdDirBytes(dir, cmdName, args...)
	return string(bufOut), string(bufErr), err
}

// ExecCmd executes system command
// and return stdout, stderr in string type, along with possible error.
func ExecCmd(cmdName string, args ...string) (string, string, error) {
	return ExecCmdDir("", cmdName, args...)
}

// 版本对比 v1比v2大返回1，小于返回-1，等于返回0
func VerCompare(ver1, ver2 string) int {
	ver1 = strings.TrimLeft(ver1, "ver") // 清除v,e,r
	ver2 = strings.TrimLeft(ver2, "ver") // 清除v,e,r
	p1 := strings.Split(ver1, ".")
	p2 := strings.Split(ver2, ".")

	ver1 = ""
	for _, v := range p1 {
		iv, _ := strconv.Atoi(v)
		ver1 = fmt.Sprintf("%s%04d", ver1, iv)
	}

	ver2 = ""
	for _, v := range p2 {
		iv, _ := strconv.Atoi(v)
		ver2 = fmt.Sprintf("%s%04d", ver2, iv)
	}

	return strings.Compare(ver1, ver2)
}
