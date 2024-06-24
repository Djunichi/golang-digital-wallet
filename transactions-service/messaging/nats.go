package messaging

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"
	"transactions-service/ent"
	"transactions-service/ent/user"

	"github.com/nats-io/nats.go"
)

func SetupNATS(natsConn *nats.Conn, client *ent.Client) {
	subscribeUserCreated(natsConn, client)
	subscribeGetBalance(natsConn, client)
}

func subscribeUserCreated(natsConn *nats.Conn, client *ent.Client) {
	natsConn.Subscribe("user-created", func(m *nats.Msg) {
		go handleUserCreated(natsConn, client, m)
	})
}

func subscribeGetBalance(natsConn *nats.Conn, client *ent.Client) {
	natsConn.Subscribe("get-balance", func(m *nats.Msg) {
		go handleGetBalance(natsConn, client, m)
	})
}

func handleUserCreated(natsConn *nats.Conn, client *ent.Client, m *nats.Msg) {
	var userData map[string]interface{}
	if err := json.Unmarshal(m.Data, &userData); err != nil {
		sendErrorResponse(natsConn, m.Reply, "error unmarshalling user-created message: "+err.Error())
		return
	}

	email, ok := userData["email"].(string)
	if !ok {
		sendErrorResponse(natsConn, m.Reply, "error: email is not a string")
		return
	}

	id, ok := userData["id"].(float64)
	if !ok {
		sendErrorResponse(natsConn, m.Reply, "error: id is not a float64")
		return
	}

	createdAtStr, ok := userData["created_at"].(string)
	if !ok {
		sendErrorResponse(natsConn, m.Reply, "error: created_at is not a string")
		return
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		sendErrorResponse(natsConn, m.Reply, "error parsing created_at: "+err.Error())
		return
	}

	_, err = client.User.Create().
		SetID(int(id)).
		SetEmail(email).
		SetCreatedAt(createdAt).
		SetBalance(0).
		Save(context.Background())
	if err != nil {
		sendErrorResponse(natsConn, m.Reply, "error creating user: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "User created successfully in transaction-service",
	}
	responseData, err := json.Marshal(response)
	if err != nil {
		sendErrorResponse(natsConn, m.Reply, "error marshalling response: "+err.Error())
		return
	}
	natsConn.Publish(m.Reply, responseData)
}

func handleGetBalance(natsConn *nats.Conn, client *ent.Client, m *nats.Msg) {
	email := string(m.Data)
	u, err := client.User.Query().Where(user.EmailEQ(email)).Only(context.Background())
	if err != nil {
		sendErrorResponse(natsConn, m.Reply, "error querying user: "+err.Error())
		return
	}

	balanceStr := strconv.FormatFloat(float64(u.Balance), 'f', -1, 32)
	natsConn.Publish(m.Reply, []byte(balanceStr))
}

func sendErrorResponse(natsConn *nats.Conn, reply string, errMsg string) {
	log.Println(errMsg)
	response := map[string]interface{}{
		"status":  "error",
		"message": errMsg,
	}
	responseData, err := json.Marshal(response)
	if err != nil {
		log.Printf("error marshalling error response: %v", err)
		return
	}
	natsConn.Publish(reply, responseData)
}
