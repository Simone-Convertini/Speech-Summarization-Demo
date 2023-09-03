package services

import (
	"fmt"
	"os"
	"time"

	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/cli"
	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/models"
	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/repositories"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ScribeService struct {
	storeRepository *repositories.StoreRepository
	Minio           *cli.MinioClient
	Rabbit          *cli.RabbitClient
}

func GetScribeService(storeRepository *repositories.StoreRepository) *ScribeService {
	minioCli := cli.MinioClient{
		Enpoint:   os.Getenv("MINIO_ENDPOINT"),
		AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("MINIO_SECRET_KEY"),
		UseSSL:    false,
	}
	rabbit := cli.RabbitClient{Uri: os.Getenv("RABBIT_URI")}

	return &ScribeService{
		storeRepository: storeRepository,
		Minio:           &minioCli,
		Rabbit:          &rabbit,
	}
}

// Insert Transcription metadata, upload file on minio and emit event
func (scs *ScribeService) SaveScribe(file []byte, storeId string) (*minio.UploadInfo, error) {
	// Starting Session to rollback on minio failure
	session, err := cli.StartTransaction(scs.storeRepository.Context)
	if err != nil {
		return nil, err
	}

	// Insert on db
	scribe := models.Store{
		ID:        primitive.NewObjectID(),
		Name:      storeId,
		Type:      "SCRIBE",
		DateAdded: time.Now(),
	}
	res, err := scs.storeRepository.InsertStore(scribe)
	if err != nil {
		return nil, err
	}

	// Upload on Minio
	resId := res.InsertedID.(primitive.ObjectID).Hex()
	info, err := scs.Minio.PutObjectIn(scs.storeRepository.Context, "scribes", resId, file)
	if err != nil {
		session.AbortTransaction(scs.storeRepository.Context)
		return nil, err
	}

	// Emit Evet
	err = scs.Rabbit.EmitTrasciptionEvent(scs.storeRepository.Context, resId)
	if err != nil {
		return nil, err
	}

	err = cli.CommitTransaction(scs.storeRepository.Context, session)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

//	PERFORMANCE ON 3:10 Audio
//
// 1 Client  -> 4:39 -> 279s
// 2 Clients -> 2:37 -> 157s
// 3 Clients -> 1:54 -> 114s

// Perform business logic on rabbit message
func (scs *ScribeService) MessageListenerRoutine() {

	messagesRabbit, err := scs.Rabbit.GetSoundUploadChannel()
	if err != nil {
		fmt.Println(err)
		return
	}

	for msgRab := range messagesRabbit {
		fmt.Printf("Sound Recived: %v \n", time.Now())

		ch1 := make(chan models.QueueMessage)
		ch2 := make(chan models.QueueMessage)
		ch3 := make(chan models.QueueMessage)

		// Vosk clients runned in parallel
		vosk1 := cli.VoskWs{Host: "host.docker.internal", Port: "2700", WriteChannel: &ch1}
		vosk2 := cli.VoskWs{Host: "host.docker.internal", Port: "2701", WriteChannel: &ch2}
		vosk3 := cli.VoskWs{Host: "host.docker.internal", Port: "2702", WriteChannel: &ch3}

		fileId := string(msgRab.Body)

		// Retrive file form Minio
		f, err := scs.Minio.GetObjectFrom(scs.storeRepository.Context, "sounds", fileId)
		if err != nil {
			fmt.Println(err)
			continue
		}

		texts := make(chan string)
		go cli.ScribeParallel(f, &texts, &vosk1, &vosk2, &vosk3)

		final := ""
		for msg := range texts {
			if msg != "" {
				final += msg + " "
			}
		}

		_, err = scs.SaveScribe([]byte(final), fileId)
		if err != nil {
			fmt.Println(err)
			continue
		}

		err = msgRab.Ack(false)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("Transcription Saved: %v \n", time.Now())
	}
}
