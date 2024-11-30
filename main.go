package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"log"
	"strings"
)

type GitWebHookPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"commits"`
}

var (
	clients       = make(map[*websocket.Conn]bool)
	broadcast     = make(chan string)
	recentCommits []string
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
		case messages := <-broadcast:
			for client := range clients {
				if err := client.WriteMessage(websocket.TextMessage, []byte(messages)); err != nil {
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
		repoName := payload.Repository.Name
		branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
		for _, commit := range payload.Commits {
			message := fmt.Sprintf("Repository: %s, Branch: %s, Author: %s, Message: %s",
				repoName, branch, commit.Author.Name, commit.Message)

			log.Println(message)
			recentCommits = append(recentCommits, message)
			broadcast <- message
		}
		return ctx.SendString("Webhook received")
	})

	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		websocketHandler(conn)
	}))
	app.Get("/api/commits", func(ctx *fiber.Ctx) error {
		return ctx.JSON(recentCommits)
	})
	err := app.Listen(":3005")
	if err != nil {
		return
	}

}
