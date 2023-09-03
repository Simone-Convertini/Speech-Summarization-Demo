# Speech Summarization Api in Go using Vosk and Gpt-4
Hey, I've written an article about the implementation and the underlying idea, [let's take a look](https://medium.com/@simone-convertini). I truly appreciate this kind of usage of LLMs, aiming to facilitate communication and knowledge transfer.

## Api Endpoint:
POST /stores to upload the wav audio file as a  multipart/form-data.

## What the App needs to Run:
I suggest to use Docker to install all the dipendences. Dockerfiles are provided. To ensure the API functions correctly, set all the environment variables in the .env file.

### MinIO
MinIO is an open-source Object Storage system, API compatible with Amazon S3. The app utilizes MinIO to manage uploaded and generated files.

### MongoDB
Mongo is a non-relational database used in this app to store metadata.

### RabbitMQ
RabbitMQ is an open-source message-broker commonly employed in distributed systems to enable services to respond to events. This demo utilizes RabbitMQ to generate upload and transcription events and execute the corresponding business logic.

### Vosk
Vosk is a speech recognition toolkit.