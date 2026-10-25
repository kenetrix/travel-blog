package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"travel-blog/backend/configmigrations"
	"travel-blog/backend/internal/adapter/http/auth"
	"travel-blog/backend/internal/adapter/http/handler"
	"travel-blog/backend/internal/adapter/http/oidc"
	"travel-blog/backend/internal/adapter/http/router"
	"travel-blog/backend/internal/adapter/repository/sqlite"
	"travel-blog/backend/internal/adapter/storage"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
	"travel-blog/backend/migrations"
)

func main() {
	dev := flag.Bool("dev", false,
		"development only: requests without a session run as the passwordless administrator")
	flag.Parse()

	db, err := sqlite.NewDB("./data/travel-blog.db", migrations.FS)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	// Settings of the installation live apart from the travel data
	configDB, err := sqlite.NewDB("./data/config.db", configmigrations.FS)
	if err != nil {
		log.Fatalf("config db init failed: %v", err)
	}
	defer configDB.Close()
	settings := sqlite.NewSettingsRepo(configDB)

	files, err := storage.NewLocalStorage("./data/photos")
	if err != nil {
		log.Fatalf("photo storage: %v", err)
	}
	defer files.Close()

	users := sqlite.NewUserRepo(db)
	accounts := service.NewAccountService(users, sqlite.NewSessionRepo(db), files, settings)
	travel := service.NewTravelService(sqlite.NewCountryEntryRepo(db), sqlite.NewFlightRepo(db), users)

	// On first start (or after an update without administrator) the installation
	// receives an administrator with a generated password
	var asUser []domain.UserID
	if *dev {
		id, err := accounts.EnsureDevAdmin()
		if err != nil {
			log.Fatalf("dev admin: %v", err)
		}
		asUser = append(asUser, id)
		log.Println("Development mode: requests without a session run as the administrator \"admin\"")
	} else {
		created, err := accounts.EnsureAdmin("./data/initial-admin.txt")
		if err != nil {
			log.Fatalf("initial admin: %v", err)
		}
		if created {
			log.Println("Administrator created, credentials in data/initial-admin.txt")
		}
	}

	// OpenID Connect login: OIDC_* variables win over the admin-area configuration.
	// The provider always exists and stays off until something is configured.
	envCfg, haveEnv := oidc.ConfigFromEnv(os.Getenv)
	var fromEnv *domain.OIDCConfig
	if haveEnv {
		fromEnv = &envCfg
	}
	provider := oidc.New(oidc.Config{}, accounts)
	oidcConfig := service.NewOIDCConfigService(accounts, settings, provider, fromEnv)
	if cfg, ok, err := oidcConfig.Effective(); err != nil {
		log.Printf("oidc: saved configuration unusable, login disabled: %v", err)
	} else if ok {
		provider.Set(cfg)
	}

	r := router.New(handler.New(travel, accounts, handler.WithOIDC(provider), handler.WithOIDCConfig(oidcConfig)),
		auth.Sessions(accounts, asUser...), provider.Routes)

	// Listen on localhost only - avoids the Windows firewall dialog on every go run
	log.Println("Server listening on localhost:8080")
	srv := &http.Server{
		Addr:              "localhost:8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second, // photo uploads up to 20 MB
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
	log.Fatal(srv.ListenAndServe())
}
