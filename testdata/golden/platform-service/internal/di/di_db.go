package di

import (
	"database/sql"

	"github.com/vukyn/testproj/internal/constants"

	pkgBunHooks "github.com/vukyn/kuery/bun/hooks"

	"github.com/sarulabs/di/v2"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/vukyn/kuery/log"
)

func defineDB() *di.Def {
	return &di.Def{
		Name:  constants.CONTAINER_NAME_DB,
		Scope: di.App,
		Build: func(ctn di.Container) (any, error) {
			dbPath := "db/app.db"
			sqldb, err := sql.Open(sqliteshim.ShimName, dbPath)
			if err != nil {
				return nil, err
			}
			db := bun.NewDB(sqldb, sqlitedialect.New())
			log.New().Infof("Database initialized with file-based SQLite at %s", dbPath)
			db.AddQueryHook(pkgBunHooks.NewQueryHook(log.New()))
			return db, nil
		},
		Close: func(obj any) error {
			db := obj.(*bun.DB)
			log.New().Debug("Database closed")
			return db.Close()
		},
	}
}

func GetDB(ctn di.Container) *bun.DB {
	return ctn.Get(constants.CONTAINER_NAME_DB).(*bun.DB)
}
