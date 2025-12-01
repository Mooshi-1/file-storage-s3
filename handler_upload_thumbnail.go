package main

import (
	"fmt"
	"io"
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

	filepath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%v.%v", video.ID, split))

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
	dataURL := fmt.Sprintf("http://localhost:%v/assets/%v.%v", cfg.port, video.ID, split)
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
