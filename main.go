package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
)

func main() {
	log.Println("Mysql Backup service running...")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file: %s", err)
	}

	c := cron.New()

	scheduleDefault := "0 1 * * *"
	cronSchedule := viper.GetString("schedule.cron")
	if cronSchedule == "" {
		cronSchedule = scheduleDefault
	}
	// Schedule a daily backup at a specific time (e.g., 2:00 AM)
	err := c.AddFunc(cronSchedule, func() {
		backupAndUpload()
	})
	if err != nil {
		return
	}

	c.Start()

	select {}
}

func backupAndUpload() {
	mysqlUsername := viper.GetString("database.username")
	mysqlPassword := viper.GetString("database.password")
	mysqlHost := viper.GetString("database.host")
	mysqlDatabase := viper.GetString("database.dbname")
	mysqlPort := viper.GetString("database.port")
	awsAccessKey := viper.GetString("aws.accessKey")
	awsSecretKey := viper.GetString("aws.secretKey")
	awsRegion := viper.GetString("aws.region")
	s3Bucket := viper.GetString("aws.bucket")
	s3BackupFolder := viper.GetString("aws.backupFolder")
	// Create a timestamp for the backup file
	timestamp := time.Now().Format("20060102T150405")
	backupFolder := "backups/"
	backupFileName := fmt.Sprintf("%s%s_%s_backup.sql", backupFolder, mysqlDatabase, timestamp)

	// Backup the MySQL database
	backupMySQL(mysqlUsername, mysqlPassword, mysqlDatabase, mysqlHost, mysqlPort, backupFileName)

	if !viper.GetBool("aws.enable") {
		// Initialize the AWS S3 session
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(awsRegion),
			Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
		})

		handleErr(err)

		uploadToS3(sess, s3Bucket, s3BackupFolder, backupFileName)

		log.Printf("Backup saved to S3://%s/%s%s\n", s3Bucket, s3BackupFolder, backupFileName)
	}
}

func backupMySQL(username string, password string, database string, host string, port string, backupFileName string) {
	log.Printf("Starting backup\n")
	go func() {
		cmdArgs := []string{
			"-u" + username,
			"-p" + password,
			"-h" + host,
			"--port=" + port,
			"--databases", database,
			"--result-file=" + backupFileName,
		}

		// Execute mysqldump
		cmd := exec.Command("mariadb-dump", cmdArgs...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		err := cmd.Run()
		if err != nil {
			log.Println(err)
		} else {
			log.Printf("Backup " + backupFileName + " saved\n")
		}
	}()
}

func uploadToS3(sess *session.Session, bucket, folder, fileName string) {
	svc := s3.New(sess)

	file, err := os.Open(fileName)
	handleErr(err)

	defer file.Close()

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(folder + fileName),
		Body:   file,
	})
	handleErr(err)
}

func handleErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
