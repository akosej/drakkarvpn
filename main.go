package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wintun.dll
var wintunDLL []byte

func main() {
	// Setup logging for production in home directory to avoid permission issues
	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".akosvpn", "app.log")
	os.MkdirAll(filepath.Join(homeDir, ".akosvpn"), 0755)

	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if logFile != nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	log.Println("Aplicación iniciada...")
	if _, err := os.Stat("wintun.dll"); os.IsNotExist(err) {
		err = os.WriteFile("wintun.dll", wintunDLL, 0644)
		if err != nil {
			log.Printf("Error extrayendo wintun.dll: %v", err)
		}
	}
	// Create an instance of the app structure
	app := NewApp()

	// Subpath for assets
	assetsFS, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatalf("Error al obtener subdirectorio de activos: %v", err)
	}

	// Verificar si index.html existe en el FS embebido
	_, err = assetsFS.Open("index.html")
	if err != nil {
		log.Printf("ERROR CRITICO: No se encuentra index.html en assetsFS: %v", err)
	} else {
		log.Println("index.html validado correctamente en assetsFS")
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "OWL VPN",
		Width:  1280,
		Height: 960,
		AssetServer: &assetserver.Options{
			Assets: assetsFS,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Printf("Error al iniciar Wails: %v", err)
	}
}
