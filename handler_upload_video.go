package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory int = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxMemory))

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not video owner", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	// TODO: implement the upload here

	formfile, ffheader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusUnprocessableEntity, "Failed to upload video", err)
		return
	}
	defer formfile.Close()

	content_type := ffheader.Header.Get("Content-Type")
	media_type, _, _ := mime.ParseMediaType(content_type)
	if media_type != "video/mp4" {
		respondWithError(w, http.StatusUnprocessableEntity, "Video must be mp4", fmt.Errorf("video must be mp4"))
		return
	}

	tempfile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Write to temp failed", err)
		return
	}
	defer os.Remove(tempfile.Name())
	defer tempfile.Close()

	writ, err := io.Copy(tempfile, formfile)
	if err != nil || writ == 0 {
		respondWithError(w, http.StatusInternalServerError, "Write failed", err)
		return
	}

	var prefix string
	aspect_ratio, err := getVideoAspectRatio(tempfile.Name())
	if err != nil {
		log.Printf("getVideoAspectRatio return: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to parse video", err)
		return
	}

	switch aspect_ratio {
	case "16:9":
		prefix = "landscape"
	case "9:16":
		prefix = "portrait"
	default:
		prefix = "other"
	}

	tempfile.Seek(0, io.SeekStart)

	processedfilename, err := processVideoForFastStart(tempfile.Name())
	if err != nil {
		log.Printf("processVideoForFastStart return: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}

	procfile, err := os.Open(processedfilename)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}

	randByte := make([]byte, 32)
	_, _ = rand.Read(randByte)
	randName := fmt.Sprintf("%s.mp4", base64.RawStdEncoding.EncodeToString(randByte))
	randName = strings.ReplaceAll(randName, "/", "0")
	videoName := fmt.Sprintf("%s/%s", prefix, randName)

	s3PutParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &videoName,
		Body:        procfile,
		ContentType: &media_type,
	}
	_, err = cfg.s3Client.PutObject(r.Context(), &s3PutParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Write failed", err)
		return
	}

	videoUrl := fmt.Sprintf("https://%s/%s/%s", cfg.s3CfDistribution, prefix, url.PathEscape(randName))

	video.VideoURL = &videoUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
