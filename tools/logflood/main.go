package main

import (
	"log"
	"runtime"
)

func main() {
	// count := flag.Int("c", 1000, "Number of log entries to generate")
	// category := flag.String("cat", "info", "Category of log")
	// body := flag.String("body", "This is a test log entry", "Text body to use for generated log entries")
	// url := flag.String("url", "nats://localhost:4222", "NATS server URL")
	// flag.Parse()

	// _, err := ddnats.Connect(*url)
	// if err != nil {
	// 	log.Printf("Exiting application due to NATS connection failure, err: %s", err.Error())
	// 	return
	// }

	// for i := 0; i < *count; i++ {
	// 	ddsvc.Log(*category, "Test log entyry", fmt.Sprintf("%s: %x", *body, i))
	// }
	log.Printf("Not yet implemented")
	runtime.Goexit()
}
