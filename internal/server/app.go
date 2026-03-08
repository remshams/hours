package server

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	pers "github.com/dhth/hours/internal/persistence"
	"github.com/spf13/cobra"
)

const (
	defaultDBName     = "hours.db"
	defaultListenAddr = "127.0.0.1:8787"
)

var (
	lookupUserHomeDir           = os.UserHomeDir
	errCouldntGetHomeDir        = errors.New("couldn't get home directory")
	ErrDBFileExtIncorrect       = errors.New("db file needs to end with .db")
	errCouldntCreateDBDirectory = errors.New("couldn't create directory for database")
	errCouldntCreateDB          = errors.New("couldn't create database")
	errCouldntInitializeDB      = errors.New("couldn't initialize database")
	errCouldntOpenDB            = errors.New("couldn't open database")
)

type serveOptions struct {
	userHomeDir string
	dbPath      string
	listenAddr  string
}

func Execute() error {
	rootCmd, err := NewRootCommand()
	if err != nil {
		return err
	}

	return rootCmd.Execute()
}

func NewRootCommand() (*cobra.Command, error) {
	userHomeDir, err := lookupUserHomeDir()
	defaultDBPath := defaultDBName
	if err == nil {
		defaultDBPath = filepath.Join(userHomeDir, defaultDBName)
	}

	rootCmd := newServeCommand(
		"hours-server",
		"Run the hours HTTP sync server",
		userHomeDir,
		defaultDBPath,
	)
	rootCmd.Long = `Run the hours HTTP sync server.

This dedicated binary only serves the sync API and does not start the TUI client.
`
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	return rootCmd, nil
}

func newServeCommand(use string, short string, userHomeDir string, defaultDBPath string) *cobra.Command {
	options := serveOptions{
		userHomeDir: userHomeDir,
		dbPath:      defaultDBPath,
		listenAddr:  defaultListenAddr,
	}

	cmd := &cobra.Command{
		Use:          use,
		Short:        short,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return options.run()
		},
	}

	cmd.Flags().StringVarP(&options.dbPath, "dbpath", "d", defaultDBPath, "location of hours' database file")
	cmd.Flags().StringVar(&options.listenAddr, "listen", defaultListenAddr, "address for the sync server to listen on")

	return cmd
}

func (o serveOptions) run() error {
	dbPathFull, err := expandTilde(o.dbPath, o.userHomeDir)
	if err != nil {
		return err
	}
	if filepath.Ext(dbPathFull) != ".db" {
		return ErrDBFileExtIncorrect
	}

	db, err := setupDB(dbPathFull)
	if err != nil {
		return err
	}
	defer db.Close()

	return ListenAndServe(o.listenAddr, db)
}

func expandTilde(path string, homeDir string) (string, error) {
	pathWithoutTilde, found := strings.CutPrefix(path, "~/")
	if !found {
		return path, nil
	}
	if homeDir == "" {
		return "", errCouldntGetHomeDir
	}

	return filepath.Join(homeDir, pathWithoutTilde), nil
}

func setupDB(dbPathFull string) (*sql.DB, error) {
	_, err := os.Stat(dbPathFull)
	if errors.Is(err, fs.ErrNotExist) {
		dir := filepath.Dir(dbPathFull)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("%w: %s", errCouldntCreateDBDirectory, err.Error())
		}

		db, err := pers.GetDB(dbPathFull)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errCouldntCreateDB, err.Error())
		}

		if err := pers.InitDB(db); err != nil {
			return nil, fmt.Errorf("%w: %s", errCouldntInitializeDB, err.Error())
		}
		if err := pers.UpgradeDB(db, 1); err != nil {
			return nil, err
		}

		return db, nil
	}

	db, err := pers.GetDB(dbPathFull)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errCouldntOpenDB, err.Error())
	}
	if err := pers.UpgradeDBIfNeeded(db); err != nil {
		return nil, err
	}

	return db, nil
}
