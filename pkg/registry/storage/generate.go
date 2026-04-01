package storage

//go:generate mockgen -destination=./mocks.go -package storage github.com/digitalocean/godo StorageService,StorageActionsService
