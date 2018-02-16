package database

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	"fmt"
	"mime/multipart"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/roistat/go-clickhouse"
)

type FilesManagerClient struct {
	RootFolder string
}

func (client *FilesManagerClient) UploadFile(file multipart.File, header *multipart.FileHeader, uniqueID string) (string, error) {
	folderPath := fmt.Sprintf("%s/%s", client.RootFolder, uniqueID)
	fullPath := fmt.Sprintf("%s/%s", folderPath, header.Filename)

	filePathInUploadFolder := fmt.Sprintf("%s/%s", uniqueID, header.Filename)

	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		err = os.MkdirAll(folderPath, 0755)
		if err != nil {
			return "", err
		}
	}

	f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()
	io.Copy(f, file)
	return filePathInUploadFolder, nil
}

func (client *FilesManagerClient) ReadFile(filePath string) (*bytes.Buffer, error) {
	pdf, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", client.RootFolder, filePath))
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(pdf), nil
}

var (
	// TODO: use di or context in future
	Postgres     *gorm.DB
	Redis        *redis.Client
	Redis2       *redis.Client
	ClickHouse   *clickhouse.Conn
	FilesManager *FilesManagerClient
)
