package jxscout

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	assetfetcher "github.com/francisconeves97/jxscout/internal/core/asset-fetcher"
	assetservice "github.com/francisconeves97/jxscout/internal/core/asset-service"
	"github.com/francisconeves97/jxscout/internal/core/common"
	"github.com/francisconeves97/jxscout/internal/core/database"
	dbeventbus "github.com/francisconeves97/jxscout/internal/core/dbeventbus"
	"github.com/francisconeves97/jxscout/internal/core/errutil"
	"github.com/francisconeves97/jxscout/internal/core/eventbus"
	"github.com/francisconeves97/jxscout/internal/core/websocket"
	astanalyzer "github.com/francisconeves97/jxscout/internal/modules/ast-analyzer"
	"github.com/francisconeves97/jxscout/internal/modules/beautifier"
	chunkdiscoverer "github.com/francisconeves97/jxscout/internal/modules/chunk-discoverer"
	gitcommiter "github.com/francisconeves97/jxscout/internal/modules/git-committer"
	htmlingestion "github.com/francisconeves97/jxscout/internal/modules/html-ingestion"
	"github.com/francisconeves97/jxscout/internal/modules/ingestion"
	jsingestion "github.com/francisconeves97/jxscout/internal/modules/js-ingestion"
	"github.com/francisconeves97/jxscout/internal/modules/overrides"
	"github.com/francisconeves97/jxscout/internal/modules/sourcemaps"
	jxscouttypes "github.com/francisconeves97/jxscout/pkg/types"
	"github.com/jmoiron/sqlx"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type jxscout struct {
	ctx             context.Context
	cancel          context.CancelFunc
	logBuffer       *logBuffer
	log             *slog.Logger
	dbEventBus      jxscouttypes.DBEventBus
	eventBus        jxscouttypes.EventBus
	options         jxscouttypes.Options
	assetService    jxscouttypes.AssetService
	assetFetcher    jxscouttypes.AssetFetcher
	httpServer      jxscouttypes.HTTPServer
	websocketServer *websocket.WebsocketServer
	scopeChecker    jxscouttypes.Scope
	fileService     jxscouttypes.FileService
	db              *sqlx.DB
	overridesModule overrides.OverridesModule

	modules      []jxscouttypes.Module
	modulesMutex sync.Mutex
	modulesSDK   *jxscouttypes.ModuleSDK

	started bool
	server  *http.Server
}

func CheckAndMigrateOldVersion() error {
	privateDirRoot := common.GetPrivateDirectoryRoot()
	workingDirRoot := filepath.Join(common.GetHome(), "jxscout")
	currentProjectPath := filepath.Join(privateDirRoot, "current_project")

	// Check if .jxscout exists but current_project doesn't
	info, err := os.Stat(privateDirRoot)
	if err == nil && info.IsDir() {
		// .jxscout exists and is a directory
		_, err = os.Stat(currentProjectPath)
		if os.IsNotExist(err) {
			// current_project doesn't exist, this is an old version
			fmt.Println("This version of jxscout contains a breaking change in how projects are managed.")
			fmt.Println("Projects will now have their own database and configuration.")
			fmt.Println("Your existing data will be backed up to:")
			fmt.Printf("- %s\n", privateDirRoot+"bak")
			fmt.Printf("- %s\n", workingDirRoot+"bak")
			fmt.Println("\nDo you want to proceed with the migration? (y/n)")

			var response string
			fmt.Scanln(&response)

			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				return fmt.Errorf("migration aborted by user")
			}

			// Backup .jxscout
			if err := os.Rename(privateDirRoot, privateDirRoot+"bak"); err != nil {
				return errutil.Wrap(err, "failed to backup .jxscout directory")
			}

			// Backup jxscout
			if _, err := os.Stat(workingDirRoot); err == nil {
				if err := os.Rename(workingDirRoot, workingDirRoot+"bak"); err != nil {
					return errutil.Wrap(err, "failed to backup jxscout directory")
				}
			}

			fmt.Println("\nBackup completed successfully.")
			fmt.Println("You can find your old data in the .bak directories.")
		}
	}

	return nil
}

func NewJXScout(options jxscouttypes.Options) (jxscouttypes.JXScout, error) {
	jxscout, err := initJxscout(options)
	if err != nil {
		return nil, errutil.Wrap(err, "failed to initialize jxscout")
	}

	jxscout.registerCoreModules()

	return jxscout, nil
}

func initJxscout(options jxscouttypes.Options) (*jxscout, error) {
	err := validateOptions(options)
	if err != nil {
		return nil, errutil.Wrap(err, "provided options are not valid")
	}

	err = common.UpdateProjectName(options.ProjectName)
	if err != nil {
		return nil, errutil.Wrap(err, "failed to update project name")
	}

	// buffer that stores logs to show in the UI
	logBuffer := newLogBuffer(options.LogBufferSize)

	logger := initializeLogger(logBuffer, options)

	scopeRegex := initializeScope(options.ScopePatterns)

	scopeChecker := newScopeChecker(scopeRegex, logger)

	fileService := assetservice.NewFileService(common.GetWorkingDirectory(options.ProjectName), logger)

	eventBus := eventbus.NewInMemoryEventBus()

	db, err := database.GetDatabase(options.ProjectName)
	if err != nil {
		return nil, errutil.Wrap(err, "failed to initialize database")
	}

	dbEventBus, err := dbeventbus.NewEventBus(db, logger)
	if err != nil {
		return nil, errutil.Wrap(err, "failed to initialize db event bus")
	}

	assetService, err := assetservice.NewAssetService(assetservice.AssetServiceConfig{
		EventBus:         dbEventBus,
		SaveConcurrency:  options.AssetSaveConcurrency,
		FetchConcurrency: options.AssetFetchConcurrency,
		Logger:           logger,
		FileService:      fileService,
		Database:         db,
		ProjectName:      options.ProjectName,
		HTMLCacheTTL:     options.HTMLRequestsCacheTTL,
		JSAssetCacheTTL:  options.JavascriptRequestsCacheTTL,
	})
	if err != nil {
		return nil, errutil.Wrap(err, "failed to initialize asset service")
	}

	httpServer := newHttpServer(logger)

	assetFetcher := assetfetcher.NewAssetFetcher(assetfetcher.AssetFetcherOptions{
		RateLimitingMaxRequestsPerMinute: options.RateLimitingMaxRequestsPerMinute,
		RateLimitingMaxRequestsPerSecond: options.RateLimitingMaxRequestsPerSecond,
		ProxyURL:                        options.ProxyURL,
	})

	ctx, cancel := context.WithCancel(context.Background())

	jxscout := &jxscout{
		options:      options,
		logBuffer:    logBuffer,
		log:          logger,
		dbEventBus:   dbEventBus,
		eventBus:     eventBus,
		assetService: assetService,
		modules:      []jxscouttypes.Module{},
		httpServer:   httpServer,
		scopeChecker: scopeChecker,
		assetFetcher: assetFetcher,
		fileService:  fileService,
		db:           db,
		ctx:          ctx,
		cancel:       cancel,
	}

	return jxscout, nil
}

func (s *jxscout) registerCoreModules() {
	overridesModule := overrides.NewOverridesModule(s.options.CaidoHostname, s.options.CaidoPort)
	s.overridesModule = overridesModule

	coreModules := []jxscouttypes.Module{
		ingestion.NewIngestionModule(),
		jsingestion.NewJSIngestionModule(s.options.DownloadReferedJS),
		htmlingestion.NewHTMLIngestionModule(),
		beautifier.NewBeautifier(s.options.BeautifierConcurrency),
		chunkdiscoverer.NewChunkDiscovererModule(
			s.options.ChunkDiscovererConcurrency,
			s.options.ChunkDiscovererBruteForceLimit,
		),
		gitcommiter.NewGitCommiter(time.Minute * 5),
		sourcemaps.NewSourceMaps(s.options.AssetSaveConcurrency),
		overridesModule,
		astanalyzer.NewAstAnalyzerModule(s.options.ASTAnalyzerConcurrency),
	}

	for _, module := range coreModules {
		s.RegisterModule(module)
	}
}

func (s *jxscout) RegisterModule(module jxscouttypes.Module) error {
	s.modulesMutex.Lock()
	defer s.modulesMutex.Unlock()

	if s.started {
		return errors.New("cant register module after server is started")
	}

	s.modules = append(s.modules, module)

	return nil
}

// Starts the jxscout server with the prompt
func (s *jxscout) Start() error {
	err := s.start()
	if err != nil {
		return errutil.Wrap(err, "failed to start server")
	}

	s.runPrompt()

	return nil
}

func (s *jxscout) start() error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	websocketServer := websocket.NewWebsocketServer(r, s.log)
	s.websocketServer = websocketServer

	err := s.initializeModules(r)
	if err != nil {
		return errutil.Wrap(err, "failed to initialize modules")
	}

	s.log.Info("starting server", "port", s.options.Port)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.options.Hostname, s.options.Port),
		Handler: r,
	}

	if os.Getenv("JXSCOUT_DEBUG") == "true" {
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.log.Error("failed to start server", "port", s.options.Port, "error", err)
		}
	} else {
		go func() {
			err := s.server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				s.log.Error("failed to start server", "port", s.options.Port, "error", err)
				return
			}
		}()
	}

	return nil
}

func (s *jxscout) initializeModules(r jxscouttypes.Router) error {
	s.modulesMutex.Lock()
	defer s.modulesMutex.Unlock()

	s.started = true

	s.modulesSDK = &jxscouttypes.ModuleSDK{
		DBEventBus:       s.dbEventBus,
		InMemoryEventBus: s.eventBus,
		Router:           r,
		AssetService:     s.assetService,
		Options:          s.options,
		HTTPServer:       s.httpServer,
		WebsocketServer:  s.websocketServer,
		Logger:           s.log,
		Scope:            s.scopeChecker,
		AssetFetcher:     s.assetFetcher,
		FileService:      s.fileService,
		Database:         s.db,
		Ctx:              s.ctx,
	}

	for _, module := range s.modules {
		err := module.Initialize(s.modulesSDK)
		if err != nil {
			return errutil.Wrap(err, "failed to initialize module")
		}
	}

	return nil
}

// Stop gracefully shuts down the JXScout server
func (s *jxscout) Stop() error {
	s.cancel()

	if s.server == nil {
		return nil
	}

	s.log.Info("shutting down server")

	if err := s.websocketServer.Shutdown(time.Second * 10); err != nil {
		s.log.Error("failed to shutdown websocket server gracefully", "error", err)
		return errutil.Wrap(err, "failed to shutdown websocket server gracefully")
	}

	if err := s.server.Shutdown(s.ctx); err != nil {
		s.log.Error("failed to shutdown server gracefully", "error", err)
		return errutil.Wrap(err, "failed to shutdown server gracefully")
	}

	s.log.Info("server stopped successfully")

	return nil
}

func (s *jxscout) Restart(options jxscouttypes.Options) (*jxscout, error) {
	jxscout, err := initJxscout(options)
	if err != nil {
		s.log.Error("failed to restart jxscout", "error", err)
		return nil, errutil.Wrap(err, "failed to restart jxscout")
	}

	s.log.Info("restarting server")

	err = s.Stop()
	if err != nil {
		s.log.Error("failed to stop server", "error", err)
		return nil, errutil.Wrap(err, "failed to stop server")
	}

	s.log.Info("server stopped successfully")

	s.log.Info("starting new server")

	s = jxscout

	s.registerCoreModules()

	go func() {
		// use private method so we don't restart the prompt
		err := s.start()
		if err != nil {
			s.log.Error("failed to restart server", "error", err)
		}
	}()

	return s, nil
}

func (s *jxscout) GetOptions() jxscouttypes.Options {
	return s.options
}

func (s *jxscout) GetOverridesModule() overrides.OverridesModule {
	return s.overridesModule
}

func (s *jxscout) Ctx() context.Context {
	return s.ctx
}

func (s *jxscout) GetAssetService() jxscouttypes.AssetService {
	return s.assetService
}

func (s *jxscout) TruncateTables() error {
	_, err := s.db.Exec(`
		DELETE FROM asset_relationships;
		DELETE FROM assets;
		DELETE FROM ast_analysis_results;
		DELETE FROM event_processing;
		DELETE FROM events;
		DELETE FROM overrides;
		DELETE FROM reversed_sourcemaps;
		DELETE FROM sourcemaps;
	`)
	if err != nil {
		return errutil.Wrap(err, "failed to truncate tables")
	}

	return nil
}
