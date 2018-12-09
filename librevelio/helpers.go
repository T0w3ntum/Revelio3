package librevelio

import (
	"bytes"
	"io"
	"os/exec"
)

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)
		switch {
		case err == io.EOF:
			return count, nil
		case err != nil:
			return count, err
		}
	}
}

func commandWorker(command string, args []string) (error){
	cmd := exec.Command(command, args...)
	err := cmd.Run()
	if err != nil{
		return err
	} else {
		return nil
	}
}
