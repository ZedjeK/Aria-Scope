// Aria Scope : l'œil qui fouille ton disque. Analyse l'espace occupé par
// chaque dossier, en parallèle, dans un seul exe Windows portable.
package main

import (
	"embed"
	"log"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"aria-scope/internal/reglages"
	"aria-scope/internal/scan"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/icon.ico
var appIcon []byte

// version est fixée à la compilation (-X main.version=…).
var version = "dev"

func main() {
	svc := &ScopeService{}
	if m, err := reglages.Ouvrir(); err == nil {
		svc.reglages = m
	}
	var window *application.WebviewWindow

	app := application.New(application.Options{
		Name:        "Aria Scope",
		Description: "Analyse de l'espace disque",
		Icon:        appIcon,
		Services:    []application.Service{application.NewService(svc)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "io.github.zedjek.ariascope",
			// Relancer l'exe ramène la fenêtre déjà ouverte au premier plan.
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				application.InvokeAsync(func() {
					if window != nil {
						window.Restore()
						window.Focus()
					}
				})
			},
		},
	})
	svc.app = app

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "Aria Scope",
		Width:            1180,
		Height:           760,
		MinWidth:         480,
		MinHeight:        520,
		BackgroundColour: application.NewRGB(3, 8, 18),
		URL:              "/",
		Windows:          application.WindowsWindow{Theme: application.Dark},
	})

	// Analyse de tout le PC dès l'ouverture. On attend que l'appli soit
	// démarrée : une seconde instance, aussitôt refermée, ne lance rien.
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		if svc.Reglages().AnalyseAuto {
			if err := svc.LancerPC(scan.Filtres{}); err != nil {
				log.Printf("analyse automatique : %v", err)
			}
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// joindre ajoute un nom de fichier à un dossier, en refusant tout ce qui
// ressemble à un chemin (séparateurs, « .. »).
func joindre(dossier, nom string) string {
	if nom == "" || nom == "." || nom == ".." || strings.ContainsAny(nom, `/\`) {
		return dossier
	}
	return filepath.Join(dossier, nom)
}
