package main

import (
	"log"
	"sync"

	"github.com/voice0726/oauth-playground/client"
	"github.com/voice0726/oauth-playground/model"
	"github.com/voice0726/oauth-playground/server"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	err := migrate()
	if err != nil {
		log.Fatal(err)
	}
	lg, _ := zap.NewDevelopment()
	s, err := server.NewServer(lg)
	if err != nil {
		log.Fatal(err)
	}

	c, err := client.NewServer(lg)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Fatal(c.Start(":9090"))
	}()
	go func() {
		defer wg.Done()
		log.Fatal(s.Start(":9091"))
	}()

	wg.Wait()
}

func migrate() error {
	db, err := gorm.Open(sqlite.Open("dev.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	return db.AutoMigrate(&model.AuthCode{}, &model.Client{}, &model.AuthRequest{}, &model.Token{})
}
