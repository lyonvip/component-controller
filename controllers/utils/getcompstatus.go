package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/IBM/sarama"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"time"
)

func GetCompStatus(compType string, genService *corev1.Service, secret *corev1.Secret) (bool, error) {
	switch compType {
	case "mysql":
		return getMysqlStatus(genService, secret)
	case "redis":
		return getRedisStatus(genService, secret)
	case "rabbitmq":
		return getRabbitmqStatus(genService, secret)
		//case "mongodb":
		//	return getMongodbStatus(svcUri, secret)
		//case "kafka":
		//	return getKafkaStatus(svcUri, secret)
	}
	return true, nil
}

func getMysqlStatus(genService *corev1.Service, secret *corev1.Secret) (bool, error) {
	user := string(secret.Data["mysql_prod_username"])
	pass := string(secret.Data["mysql_prod_password"])
	host := fmt.Sprintf("%s.%s", genService.Name, genService.Namespace)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/mysql?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return false, fmt.Errorf("[mysql-inspect] failed to connect: %s", err)
	}
	sqlDB, err := db.DB()
	defer sqlDB.Close()
	if err != nil {
		return false, err
	}
	err = sqlDB.Ping()
	if err != nil {
		return false, fmt.Errorf("[mysql-inspect] ping failed %s", err)
	}
	return true, nil
}

func getRedisStatus(genService *corev1.Service, secret *corev1.Secret) (bool, error) {
	user := string(secret.Data["redis_prod_username"])
	pass := string(secret.Data["redis_prod_password"])
	host := fmt.Sprintf("%s.%s", genService.Name, genService.Namespace)
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", host),
		Username: user,
		Password: pass,
		DB:       0,
	})

	timeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(timeCtx).Err(); err != nil {
		return false, fmt.Errorf("[redis-inspect] ping failed %s", err)
	}
	return true, nil
}

func getRabbitmqStatus(genService *corev1.Service, secret *corev1.Secret) (bool, error) {
	user := string(secret.Data["rabbitmq_username"])
	pass := string(secret.Data["rabbitmq_password"])
	host := fmt.Sprintf("%s.%s", genService.Name, genService.Namespace)
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:5672/", user, pass, host))
	if err != nil {
		return false, fmt.Errorf("[rabbitmq-inspect] failed to connect: %s", err)
	}
	defer conn.Close()
	return true, nil
}

func getKafkaStatus(svcUri string, secret *corev1.Secret) (bool, error) {
	uri := fmt.Sprintf("%s:9092", svcUri)
	config := sarama.NewConfig()
	config.Producer.Retry.Max = 5
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer([]string{uri}, config)
	if err != nil {
		return false, fmt.Errorf("[kafka-inspect] failed to connect: %s", err)
	}
	defer producer.Close()
	return true, nil
}

func getMongodbStatus(svcUri string, secret *corev1.Secret) (bool, error) {
	userBytes, err := base64.StdEncoding.DecodeString(string(secret.Data["mongodb_root_user"]))
	if err != nil {
		return false, fmt.Errorf("[mongodb-inspect] failed to decode username: %s", err)
	}
	passBytes, err := base64.StdEncoding.DecodeString(string(secret.Data["mongodb_root_pass"]))
	if err != nil {
		return false, fmt.Errorf("[mongodb-inspect] failed to decode password: %s", err)
	}
	user := string(userBytes)
	pass := string(passBytes)
	uri := fmt.Sprintf("mongodb://%s:%s@%s:27017", user, pass, svcUri)
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri).SetTimeout(3*time.Second))
	if err != nil {
		return false, fmt.Errorf("[mongodb-inspect] failed to connect: %s", err)
	}
	defer client.Disconnect(context.TODO())
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return false, fmt.Errorf("[mongodb-inspect] ping failed %s", err)
	}
	return true, nil
}
