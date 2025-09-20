package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	var psUrl string

	psClient := s3.NewPresignClient(s3Client)

	psReq, err := psClient.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		log.Print("Failed to get presigned URL")
		return "", err
	}
	psUrl = psReq.URL

	return psUrl, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		log.Print("video URL is null pointer")
		return video, nil
	}
	strs := strings.Split(*video.VideoURL, ",")
	if len(strs) <= 1 {
		log.Print("dbVideoToSignedVideo: old video url passed")
		return video, fmt.Errorf("old video url")
	}
	bucket := strs[0]
	key := strs[1]

	newUrl, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*time.Duration(5))
	if err != nil {
		log.Print("Failed to generate presigned URL")
		return database.Video{}, err
	}

	newVideo := database.Video{
		ID:                video.ID,
		CreatedAt:         video.CreatedAt,
		UpdatedAt:         video.UpdatedAt,
		ThumbnailURL:      video.ThumbnailURL,
		VideoURL:          &newUrl,
		CreateVideoParams: video.CreateVideoParams,
	}
	return newVideo, nil
}
