# Deluge

Simple library to do specific things with Deluge Web UI API.

Pretty much only gets a list of active transfers. Has a lot of boiler plate for
other features. I had no use for more than an active transfer list, and all
the meta data for each transfer.

Pull requests and feedback welcomed!


```golang
package main

import (
	"log"
	"time"
	"golift.io/deluge"
)

func main() {
	config := deluge.Config{
		URL:      "http://127.0.0.1:8112",
		Password: "superSe(re7",
	}
	server, err := deluge.New(context.TODO(), &config)
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
