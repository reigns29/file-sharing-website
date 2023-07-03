package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/skip2/go-qrcode"
	"github.com/joho/godotenv"
)

const (
	uploadDirectory = "./uploads"
)

func main() {
	router := mux.NewRouter()
	godotenv.Load()

	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/upload", uploadHandler).Methods("POST")

	fs := http.FileServer(http.Dir(uploadDirectory))
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", fs))

	log.Println("Server is running on " + os.Getenv("SERVER_URL"))
	log.Fatal(http.ListenAndServe(":8080", router))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Println(err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create the uploads directory if it doesn't exist
	if _, err := os.Stat(uploadDirectory); os.IsNotExist(err) {
		err := os.Mkdir(uploadDirectory, 0755)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// Save the uploaded file
	filePath := filepath.Join(uploadDirectory, handler.Filename)
	out, err := os.Create(filePath)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Generate QR code
	qrCodeFilePath := filepath.Join(uploadDirectory, handler.Filename+".png")
	err = qrcode.WriteFile(fmt.Sprintf(os.Getenv("SERVER_URL") + "/uploads/%s", handler.Filename), qrcode.Medium, 256, qrCodeFilePath)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/success.html")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Filename        string
		DownloadLink    string
		QRCodeImagePath string
	}{
		Filename:        handler.Filename,
		DownloadLink:    fmt.Sprintf(os.Getenv("SERVER_URL") + "/uploads/%s", handler.Filename),
		QRCodeImagePath: fmt.Sprintf(os.Getenv("SERVER_URL") + "/uploads/%s.png", handler.Filename),
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
