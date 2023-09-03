package services

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/cli"
	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/models"
	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/repositories"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SummaryService struct {
	storeRepository *repositories.StoreRepository
	Minio           *cli.MinioClient
	Rabbit          *cli.RabbitClient
	Gpt             *cli.GptClient
}

func GetSummaryService(storeRepository *repositories.StoreRepository) *SummaryService {
	minioCli := cli.MinioClient{
		Enpoint:   os.Getenv("MINIO_ENDPOINT"),
		AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("MINIO_SECRET_KEY"),
		UseSSL:    false,
	}
	rabbit := cli.RabbitClient{Uri: os.Getenv("RABBIT_URI")}
	gpt := cli.GptClient{
		ApiKey: os.Getenv("OPENAI_KEY"),
	}

	return &SummaryService{
		storeRepository: storeRepository,
		Minio:           &minioCli,
		Rabbit:          &rabbit,
		Gpt:             &gpt,
	}
}

// Insert Transcription metadata, upload file on minio
func (sum *SummaryService) SaveSummary(file []byte, storeId string) (*minio.UploadInfo, error) {
	// Starting Session to rollback on minio failure
	session, err := cli.StartTransaction(sum.storeRepository.Context)
	if err != nil {
		return nil, err
	}

	// Insert on db
	scribe := models.Store{
		ID:        primitive.NewObjectID(),
		Name:      storeId,
		Type:      "SUMMARY",
		DateAdded: time.Now(),
	}
	res, err := sum.storeRepository.InsertStore(scribe)
	if err != nil {
		return nil, err
	}

	// Upload on Minio
	resId := res.InsertedID.(primitive.ObjectID).Hex()
	info, err := sum.Minio.PutObjectIn(sum.storeRepository.Context, "summaries", resId, file)
	if err != nil {
		session.AbortTransaction(sum.storeRepository.Context)
		return nil, err
	}

	err = cli.CommitTransaction(sum.storeRepository.Context, session)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func (sum *SummaryService) MessageListenerRoutine() {
	messagesRabbit, err := sum.Rabbit.GetTranscriptionUploadChannel()
	if err != nil {
		fmt.Println(err)
		return
	}

	for msgRab := range messagesRabbit {
		fmt.Printf("Transcription Recived: %v \n", time.Now())
		fileId := string(msgRab.Body)

		// Retrive file form Minio
		f, err := sum.Minio.GetObjectFrom(sum.storeRepository.Context, "scribes", fileId)
		if err != nil {
			fmt.Println(err)
			continue
		}

		buffer := make([]byte, 0)
		for {
			currentBuff := make([]byte, 16000)
			dat, err := f.Read(currentBuff)
			if dat == 0 && err == io.EOF {
				break
			}

			buffer = append(buffer, currentBuff...)
		}

		texts, err := sum.Gpt.TextPerTokenSplit(string(buffer), 8000, "gpt-4")
		if err != nil {
			fmt.Println(err)
			continue
		}

		responses := make([]string, 0)
		for _, question := range texts {
			res, err := sum.Gpt.GetSummarization(question, sum.storeRepository.Context)
			if err != nil {
				fmt.Println(err)
				continue
			}

			responses = append(responses, *res)
		}

		summary := strings.Join(responses, " ")
		_, err = sum.SaveSummary([]byte(summary), fileId)
		if err != nil {
			fmt.Println(err)
			continue
		}

		err = msgRab.Ack(false)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("Summary Saved: %v \n", time.Now())
	}
}
