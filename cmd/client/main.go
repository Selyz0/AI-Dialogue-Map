package main

import (
	"AI-Dialogue-Map/internal/service" // Assuming mainApp is defined in model package
	"log"
)

func main() {
	log.Println("アプリケーションを開始します...")
	appInstance := service.NewMainApp()
	appInstance.Run()
	log.Println("アプリケーションを終了します。")
}
