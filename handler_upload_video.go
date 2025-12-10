package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	log.Printf("start upload")

	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

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

	fmt.Println("uploading video", videoID, "by user", userID)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "cannot find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "id's do not match", err)
		return
	}

	// const maxMemory int = (10 << 20)
	// r.ParseMultipartForm(int64(maxMemory))

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to formfile", err)
		return
	}
	defer file.Close()

	contenttype := header.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(contenttype)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to parse contenttype", err)
		return
	}
	if mediatype != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "contenttype is wrong", err)
		return
	}

	tf, err := os.CreateTemp("", "upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "temp file fail", err)
		return
	}
	defer os.Remove(tf.Name())
	defer tf.Close()

	_, err = io.Copy(tf, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "cannot copy file", err)
		return
	}

	_, err = tf.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "seek err", err)
		return
	}

	log.Print("before bytes")

	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "rand read err", err)
		return
	}
	url := base64.RawURLEncoding.EncodeToString(b)
	key := url + ".mp4"

	details := &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(key),
		Body:        tf,
		ContentType: aws.String("video/mp4"),
	}

	log.Printf("put object")

	_, err = cfg.s3Client.PutObject(r.Context(), details)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "putobj fail", err)
		return
	}

	// filepath := filepath.Join(cfg.assetsRoot, key)

	// f, err := os.Create(
	// 	//fmt.Sprintf("./assets/%v.%v", video.ID, split)
	// 	filepath)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "unable to create file", err)
	// 	return
	// }

	dataURL := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, key)
	fmt.Print(dataURL)
	video.VideoURL = &dataURL

	// newURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", cfg.port, video.ID)
	// video.ThumbnailURL = &newURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "cannot update video", err)
		return
	}

	w.Header().Add("video", dataURL)

	respondWithJSON(w, http.StatusOK, video)

}
