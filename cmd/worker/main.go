package main

import (
	"os"

	"github.com/KevinNaserwan/fleet-tracking/internal/lifecycle"
)

func main() {
	os.Exit(lifecycle.Run("worker"))
}
