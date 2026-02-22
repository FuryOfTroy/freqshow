//go:build gui

package main

import (
	"context" // Added import
	"embed"
	"fmt" // Added import for Sprintf
	"io/fs"
	"log"
	"net"    // Added import
	"net/http"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/webview/webview_go"
)

//go:embed webview_assets/*
var content embed.FS

func init() {
	var guiCmd = &cobra.Command{
		Use:   "gui",
		Short: "Launch the graphical user interface for freqshow",
		Run: func(cmd *cobra.Command, args []string) {
			runGUI()
		},
	}
	rootCmd.AddCommand(guiCmd)
}

func runGUI() {
	debug := true // Set to false for production
	if os.Getenv("WEBVIEW_DEBUG") == "0" {
		debug = false
	}

	runtime.LockOSThread()

	w := webview.New(debug)
	defer w.Destroy()

	w.SetTitle("Freqshow EQ GUI (Webview)")
	w.SetSize(800, 600, webview.HintNone)

	app := &App{}
	if err := w.Bind("RunEQCommand", app.RunEQCommand); err != nil {
		log.Fatalf("Failed to bind RunEQCommand: %v", err)
	}

	assets, err := fs.Sub(content, "webview_assets")
	if err != nil {
		log.Fatalf("could not get subfolder: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(assets)))

	listener, err := net.Listen("tcp", "127.0.0.1:0") // Listen on a random available port
	if err != nil {
		log.Fatalf("could not create listener: %v", err)
	}
	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("could not start server: %v", err)
		}
	}()

	// Construct the URL using the listener's assigned port
	appURL := fmt.Sprintf("http://%s", listener.Addr().String())
	log.Printf("Serving webview assets from: %s", appURL)

	w.Navigate(appURL)

	w.Run()
	log.Println("Webview closed.")
	// When webview closes, shut down the HTTP server
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
}
