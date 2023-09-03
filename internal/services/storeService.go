package services

import (
	"mime/multipart"
	"os"
	"time"

	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/cli"
	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/models"
	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/repositories"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StoreService struct {
	storeRepository *repositories.StoreRepository
	Minio           *cli.MinioClient
	Rabbit          *cli.RabbitClient
}

func GetStoreService(storeRepository *repositories.StoreRepository) *StoreService {
	minioCli := cli.MinioClient{
		Enpoint:   os.Getenv("MINIO_ENDPOINT"),
		AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("MINIO_SECRET_KEY"),
		UseSSL:    false,
	}
	rabbit := cli.RabbitClient{Uri: os.Getenv("RABBIT_URI")}

	return &StoreService{
		storeRepository: storeRepository,
		Minio:           &minioCli,
		Rabbit:          &rabbit,
	}
}

// Insert metadata record on db, upload file on Minio and emit event on Rabbit
func (sts *StoreService) UploadAudio(file *multipart.FileHeader) (*minio.UploadInfo, error) {

	// Starting Session to rollback on minio failure
	session, err := cli.StartTransaction(sts.storeRepository.Context)
	if err != nil {
		return nil, err
	}

	// Insert on db
	store := models.Store{
		ID:        primitive.NewObjectID(),
		Name:      file.Filename,
		Type:      "SOUND",
		DateAdded: time.Now(),
	}
	res, err := sts.storeRepository.InsertStore(store)
	if err != nil {
		return nil, err
	}

	// Upload on Minio
	resId := res.InsertedID.(primitive.ObjectID).Hex()
	info, err := sts.Minio.PutObjectInFile(sts.storeRepository.Context, "sounds", resId, file)
	if err != nil {
		session.AbortTransaction(sts.storeRepository.Context)
		return nil, err
	}

	// Emit Evet
	err = sts.Rabbit.EmitUploadEvent(sts.storeRepository.Context, resId)
	if err != nil {
		return nil, err
	}

	err = cli.CommitTransaction(sts.storeRepository.Context, session)
	if err != nil {
		return nil, err
	}

	return &info, nil
}
