package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory int = 10 << 20
	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusUnprocessableEntity, "Failed to upload thumbnail", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusUnprocessableEntity, "Failed to upload thumbnail", err)
		return
	}
	content_type := header.Header.Get("Content-Type")
	media_type, _, _ := mime.ParseMediaType(content_type)
	if media_type != "image/png" && media_type != "image/jpeg" {
		respondWithError(w, http.StatusUnprocessableEntity, "Thumbprint must be an image", fmt.Errorf("thumprint must be an image"))
	}

	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Failed to read file", err)
	// 	return
	// }

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not video owner", err)
		return
	}

	// tn := thumbnail{
	// 	data:      data,
	// 	mediaType: media_type,
	// }

	// videoThumbnails[videoID] = tn
	// tn_url := fmt.Sprintf("http://localhost:%s/api/thumbnails/%v", cfg.port, videoID)
	// video.ThumbnailURL = &tn_url
	// err = cfg.db.UpdateVideo(video)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
	// 	return
	// }

	// b64tn := base64.StdEncoding.EncodeToString(data)
	// data_url := fmt.Sprintf("data:%s;base64,%s", media_type, b64tn)

	var extension string
	splits := strings.Split(header.Filename, ".")
	if len(splits) <= 1 {
		extension = strings.TrimPrefix(media_type, "image/")
		switch extension {
		case "jpeg":
			extension = "jpg"
		case "svg+xml":
			extension = "svg"
		}
	} else {
		extension = splits[len(splits)-1]
	}

	randByte := make([]byte, 32)
	_, _ = rand.Read(randByte)
	randName := base64.RawStdEncoding.EncodeToString(randByte)

	fn := fmt.Sprintf("%s.%s", randName, extension)
	fpath := filepath.Join(cfg.assetsRoot, fn)
	osfile, err := os.Create(fpath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save thumbnail", err)
		return
	}

	_, err = io.Copy(osfile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save thumbnail", err)
		return
	}

	data_url := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fn)

	video.ThumbnailURL = &data_url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
