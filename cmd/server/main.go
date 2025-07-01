package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/akave-ai/akavesdk/private/pb"
	"github.com/akave-ai/akavesdk/sdk"
	"google.golang.org/grpc"
)

var ipc *sdk.IPC

func initIPC() {
	conn, err := grpc.Dial("localhost:50051") // placeholder
	if err != nil {
		log.Fatalf("failed to connect to IPC node: %v", err)
	}

	// need to initialise the client the IPC struct here, need values???
	ipc = &sdk.IPC{
		Client: pb.NewIPCNodeAPIClient(conn),
		Conn:   conn,
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if ipc == nil {
		http.Error(w, "IPC client not initialized", http.StatusInternalServerError)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[1] != "files" || parts[3] != "download" {
		http.NotFound(w, r)
		return
	}

	bucketID := parts[0]
	fileID := parts[2]

	ctx := r.Context()
	fileDownload, err := ipc.CreateFileDownload(ctx, bucketID, fileID) // must take bucketName and FileName but URL provides only ID
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to download the file : %v", err), http.StatusInternalServerError)
		return
	}

	err = ipc.Download(ctx, fileDownload, w)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to download the file : %v", err), http.StatusInternalServerError)
		return
	}
}

func main() {
	initIPC()
	http.HandleFunc("/health", healthHandler)

	http.HandleFunc("/", downloadHandler) // must use router

	log.Println("Starting go-akavelink server on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
