package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"transactions-service/ent"
	"transactions-service/ent/user"

	"github.com/nats-io/nats.go"
)

func SetupNATS(natsConn *nats.Conn, client *ent.Client) {
	natsConn.Subscribe("user-created", func(m *nats.Msg) {
		go func() {
			var userData map[string]interface{}
			if err := json.Unmarshal(m.Data, &userData); err != nil {
				log.Printf("error unmarshalling user-created message: %v", err)
				return
			}
			userID := int(userData["id"].(float64))
			email := userData["email"].(string)
			createdAt, err := time.Parse(time.RFC3339, userData["created_at"].(string))
			if err != nil {
				log.Printf("error parsing created_at: %v", err)
				return
			}
			_, err = client.User.Create().SetUserID(userID).SetEmail(email).SetCreatedAt(createdAt).SetBalance(0).Save(context.Background())
			if err != nil {
				log.Printf("error creating user: %v", err)
			}

			response := map[string]interface{}{
				"status":  "success",
				"message": "User created successfully in transaction-service",
			}
			responseData, err := json.Marshal(response)
			if err != nil {
				log.Printf("error marshalling response: %v", err)
				return
			}
			natsConn.Publish(m.Reply, responseData)
		}()
	})

	natsConn.Subscribe("get-balance", func(m *nats.Msg) {
		go func() {
			email := string(m.Data)
			u, err := client.User.Query().Where(user.EmailEQ(email)).Only(context.Background())
			if err != nil {
				natsConn.Publish(m.Reply, []byte("0"))
				return
			}
			natsConn.Publish(m.Reply, []byte(fmt.Sprintf("%d", u.Balance)))
		}()
	})
}
