package main

import (
	"github.com/joho/godotenv"
	"go.uber.org/fx"

	"audiotranscrib/internal/app"
)

func main() {

	_ = godotenv.Load()

	fx.New(
		app.Module,
	).Run()
}
