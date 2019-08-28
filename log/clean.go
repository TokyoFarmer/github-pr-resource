package log

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func cleanStaleLogs(dir string) error {
	tmpfiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range tmpfiles {
		if strings.HasPrefix(file.Name(), "resource-") && file.Mode().IsRegular() {
			if time.Now().Sub(file.ModTime()) > 72*time.Hour {
				err = os.Remove(fmt.Sprintf("%s%v%s", dir, os.PathSeparator, file.Name()))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
