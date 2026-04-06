package volumes

//go:generate mockgen -destination=./mocks.go -package volumes github.com/digitalocean/godo StorageService,StorageActionsService
