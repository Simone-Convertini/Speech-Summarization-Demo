package cli

import (
	"bytes"
	"context"
	"mime/multipart"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Struct to hanlde configs
type MinioClient struct {
	Enpoint   string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

var minioClient *minio.Client
var connLockMinio sync.Mutex

// Return minio client from configs
func getMinioClient(mc *MinioClient) (*minio.Client, error) {
	connLockMinio.Lock()
	defer connLockMinio.Unlock()

	if minioClient != nil {
		return minioClient, nil
	}

	minioClient, err := minio.New(mc.Enpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(mc.AccessKey, mc.SecretKey, ""),
		Secure: mc.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return minioClient, nil
}

// Put a *multipart.File inside the specified bucket
func (mc *MinioClient) PutObjectInFile(ctx context.Context, bucket string, objectName string, file *multipart.FileHeader) (minio.UploadInfo, error) {
	client, err := getMinioClient(mc)
	var info minio.UploadInfo

	if err != nil {
		return info, err
	}

	fileData, err := file.Open()
	if err != nil {
		return info, err
	}
	defer fileData.Close()

	options := minio.PutObjectOptions{ContentType: file.Header.Get("Content-Type")}
	info, err = client.PutObject(ctx, bucket, objectName, fileData, file.Size, options)
	if err != nil {
		return info, err
	}

	return info, nil
}

// Put an object inside the specified bucket
func (mc *MinioClient) PutObjectIn(ctx context.Context, bucket string, objectName string, file []byte) (minio.UploadInfo, error) {
	client, err := getMinioClient(mc)
	var info minio.UploadInfo

	if err != nil {
		return info, err
	}

	info, err = client.PutObject(ctx, bucket, objectName, bytes.NewReader(file), int64(len(file)), minio.PutObjectOptions{})
	if err != nil {
		return info, err
	}

	return info, nil
}

func (mc *MinioClient) GetObjectFrom(ctx context.Context, bucket string, name string) (*minio.Object, error) {
	client, err := getMinioClient(mc)
	if err != nil {
		return nil, err
	}

	file, err := client.GetObject(ctx, bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return file, nil
}
