# Deluge

Simple library to do specific things with Deluge Web UI API.

Pretty much only gets a list of active transfers. Has a lot of boiler plate for
other features, but I had no other se for them.

Pull requests and feedback welcomed!


```golang
package main

import (
	"log"
	"time"
	"github.com/golift/deluge"
)

func main() {
	config := deluge.Config{
		URL:      "http://127.0.0.1:8112",
		Password: "superSe(re7",
		Timeout:  time.Minute,
	}
	server, err := deluge.New(config)
	if err != nil {
		log.Fatal(err)
	}
	// This is the only method available for retrieving data.
	transfers, err := server.GetXfers()
	if err != nil {
		log.Fatal(err)
	}
	for _, xfer := range transfers {
		log.Println(xfer.Name)
	}
}
```
