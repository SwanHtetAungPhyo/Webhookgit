package main

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"log"
)

type Repository struct {
	Name string `json:"name"`
}

type Author struct {
	Name string `json:"name"`
}

type Commit struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Author  Author `json:"author"`
}

type GitWebHookPayload struct {
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
	Commits    []Commit   `json:"commits"`
}

var (
	clients    = make(map[*websocket.Conn]bool)
	broadcast  = make(chan GitWebHookPayload)
	payloadLog []GitWebHookPayload
)

func websocketHandler(c *websocket.Conn) {
	clients[c] = true
	defer func() {
		delete(clients, c)
		err := c.Close()
		if err != nil {
			return
		}
	}()

	for {
		select {
		case payload := <-broadcast:
			data, err := json.Marshal(payload)
			if err != nil {
				log.Printf("Error serializing payload: %s", err)
				continue
			}
			for client := range clients {
				if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
					err := client.Close()
					if err != nil {
						return
					}
					delete(clients, client)
				}
			}
		}
	}
}

func main() {
	app := fiber.New()
	app.Post("/api/git-webhook", func(ctx *fiber.Ctx) error {
		var payload GitWebHookPayload
		if err := ctx.BodyParser(&payload); err != nil {
			log.Printf(err.Error())
			return ctx.Status(fiber.StatusBadRequest).SendString("Invalid payload")
		}

		log.Println(payload)
		payloadLog = append(payloadLog, payload)
		broadcast <- payload
		return ctx.SendString("Webhook received")
	})

	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		websocketHandler(conn)
	}))

	app.Get("/api/commits", func(ctx *fiber.Ctx) error {
		return ctx.JSON(payloadLog)
	})

	err := app.Listen(":3005")
	if err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
