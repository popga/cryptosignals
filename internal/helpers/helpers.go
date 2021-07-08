package helpers

import (
	"github.com/spf13/viper"
	"os"
	"path"
)

func CreateDirIfNotExist(dir string) error {
	if err := os.Mkdir(dir, os.ModeDir); os.IsNotExist(err) {
		return err
	}
	return nil
}

func DeleteDir() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	fullpath := path.Join(cwd, viper.GetString("base.data.folder"))
	return os.RemoveAll(fullpath)
}
