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

	const maxMemory int = (10 << 20)
	r.ParseMultipartForm(int64(maxMemory))

	file, header, err := r.FormFile("thumbnail")
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
	if mediatype != "image/jpeg" && mediatype != "image/png" {
		respondWithError(w, http.StatusBadRequest, "contenttype is not png or jpeg", err)
		return
	}

	split := strings.TrimPrefix(contenttype, "image/")
	// bytes, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "unable to parse file", err)
	// 	return
	// }

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "cannot find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "id's do not match", err)
		return
	}

	// videoThumbnails[video.ID] = thumbnail{
	// 	data:      bytes,
	// 	mediaType: contenttype,
	// }

	//encodedData := base64.StdEncoding.EncodeToString(bytes)
	// dataURL := fmt.Sprintf("data:%v;base64,%v", contenttype, encodedData)
	//video.ThumbnailURL = &dataURL

	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "rand read err", err)
		return
	}

	url := base64.RawURLEncoding.EncodeToString(b)
	filename := fmt.Sprintf("%v.%v", url, split)

	filepath := filepath.Join(cfg.assetsRoot, filename)

	f, err := os.Create(
		//fmt.Sprintf("./assets/%v.%v", video.ID, split)
		filepath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to create file", err)
		return
	}
	_, err = io.Copy(f, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "cannot copy file", err)
		return
	}
	dataURL := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, filename)
	video.ThumbnailURL = &dataURL

	// newURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", cfg.port, video.ID)
	// video.ThumbnailURL = &newURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "cannot update video", err)
		return
	}

	w.Header().Add("thumbnail", dataURL)

	respondWithJSON(w, http.StatusOK, video)
}
