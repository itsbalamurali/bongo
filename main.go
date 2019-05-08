package bongo

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type Config struct {
	ConnectionString string
	Database         string
	ClientOptions         *options.ClientOptions
}

// var EncryptionKey [32]byte
// var EnableEncryption bool

type Connection struct {
	Config  *Config
	Session *mongo.Client
	// collection []Collection
	Context *Context
}

// Create a new connection and run Connect()
func Connect(config *Config) (*Connection, error) {
	conn := &Connection{
		Config:  config,
		Context: &Context{},
	}

	err := conn.Connect()

	return conn, err
}

// Connect to the database using the provided config
func (m *Connection) Connect() (err error) {
	defer func() {
		if r := recover(); r != nil {
			// panic(r)
			// return
			if e, ok := r.(error); ok {
				err = e
			} else if e, ok := r.(string); ok {
				err = errors.New(e)
			} else {
				err = errors.New(fmt.Sprint(r))
			}

		}
	}()

	client, err := mongo.NewClient(options.Client().ApplyURI(m.Config.ConnectionString))
	if err != nil { return err }
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil { return err }

	m.Session = client

	return nil
}

// CollectionFromDatabase ...
func (m *Connection) CollectionFromDatabase(name string, database string) *Collection {
	// Just create a new instance - it's cheap and only has name and a database name
	return &Collection{
		Connection: m,
		Context:    m.Context,
		Database:   database,
		Name:       name,
	}
}

// Collection ...
func (m *Connection) Collection(name string) *Collection {
	return m.CollectionFromDatabase(name, m.Config.Database)
}
