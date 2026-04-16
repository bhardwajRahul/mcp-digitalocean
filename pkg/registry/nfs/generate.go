package nfs

//go:generate mockgen -destination=./mocks.go -package nfs github.com/digitalocean/godo NfsService,NfsActionsService
