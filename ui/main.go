package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/authorization"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/internal/handlers"
	"github.com/churrodata/churro/pkg/config"
	"github.com/gorilla/mux"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	zerolog.TimeFieldFormat = time.RFC822

	log.Info().Msg("mock web app")

	// make sure the db cert path exists
	dbCertPath := "/dbcerts"
	_, err := os.Stat(dbCertPath)
	if err != nil {
		log.Error().Stack().Err(err).Msg("dbcerts path problem")
		os.Exit(1)
	}

	dbCreds := config.DBCredentials{
		CAKeyPath:       dbCertPath + "/ca.key",
		CACertPath:      dbCertPath + "/ca.crt",
		SSLRootKeyPath:  dbCertPath + "/client." + "root" + ".key",
		SSLRootCertPath: dbCertPath + "/client." + "root" + ".crt",
	}
	src := v1alpha1.Source{}

	dbType := os.Getenv("DATABASE_TYPE")
	switch dbType {
	case domain.DatabaseCockroach:
		src.Username = "root"
		src.Host = "cockroachdb-public"
		src.Port = 26257
		src.Database = "defaultdb"
	case domain.DatabaseMysql:
		src.Username = "root"
		src.Password = os.Getenv("MYSQL_ROOT_PASSWORD")
		src.Host = "churro-ui-mysql-mysql"
		src.Port = 3306
		src.Database = "mysql"
	case domain.DatabaseSinglestore:
		src.Username = "admin"
		src.Password = os.Getenv("SINGLESTORE_ROOT_PASSWORD")
		src.Host = "svc-memsql-cluster-ddl"
		src.Port = 3306
		src.Database = "memsql"
	default:
		fmt.Printf("error:  unsupported DATABASE_TYPE env var  value\n")
		os.Exit(1)
	}
	u := handlers.HandlerWrapper{}
	u.DatabaseType = dbType

	churroDB, err := db.NewChurroDB(dbType)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting db connection")
		os.Exit(1)
	}
	err = churroDB.GetConnection(dbCreds, src)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting db connection")
		os.Exit(1)
	}

	if dbType == domain.DatabaseSinglestore {
		databaseTries := 7
		for i := 0; i < databaseTries; i++ {
			_, err := churroDB.GetVersion()
			if err != nil {
				log.Error().Stack().Err(err).Msg("can not connect to database")
				if i >= databaseTries {
					os.Exit(1)
				}
				time.Sleep(time.Second * 10)
				continue
			}
			log.Info().Msg("connected to database")
			break
		}
	}

	authorization.AdminDB.DBCreds = dbCreds
	authorization.AdminDB.Source = src

	src.Database = "churro"
	err = churroDB.CreateChurroDatabase(src.Database)
	if err != nil {
		log.Error().Stack().Err(err).Msg("can not create churro db")
		os.Exit(1)
	}

	//switch the connection to the churro database
	src.Database = "churro"
	authorization.AdminDB.Source = src
	err = churroDB.GetConnection(dbCreds, src)
	if err != nil {
		log.Error().Stack().Err(err).Msg("erro getting db connection")
		os.Exit(1)
	}

	err = churroDB.CreateAuthObjects()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error creating auth objects")
		os.Exit(1)
	}
	//authorization.AdminDB.db = db
	authorization.AdminDB.Created = true

	//err = authorization.AdminDB.Bootstrap()
	err = churroDB.Bootstrap()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error during bootstrap")
		os.Exit(1)
	}

	//authorization.AdminDB.ConnectionString = connString

	debug := flag.Bool("debug", false, "debug flag")
	serviceCertPath := flag.String("servicecert", "", "path to service cert files e.g. service.crt")
	log.Info().Msg(fmt.Sprintf("%t %s \n", *debug, *serviceCertPath))

	flag.Parse()

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", u.Pipelines).Methods("GET")
	r.HandleFunc("/users", u.Users).Methods("GET")
	r.HandleFunc("/useradmin", u.UserAdmin).Methods("POST")
	r.HandleFunc("/useradmin/show-create", u.ShowCreateUser).Methods("GET")
	r.HandleFunc("/useradmin/{id}", u.UserAdminDetail).Methods("GET")
	r.HandleFunc("/useradmin/{id}/update", u.UserAdminUpdate).Methods("POST")
	r.HandleFunc("/useradmin/{id}/delete", u.UserAdminDelete).Methods("GET")
	r.HandleFunc("/login", u.Login).Methods("GET")
	r.HandleFunc("/logintry", u.LoginTry).Methods("POST")
	r.HandleFunc("/logout", u.Logout).Methods("GET")
	r.HandleFunc("/profile", u.Profile).Methods("GET")
	r.HandleFunc("/profileupdate", u.ProfileUpdate).Methods("POST")
	r.HandleFunc("/pipeline/show-create", u.ShowCreatePipeline).Methods("GET")
	r.HandleFunc("/pipeline", u.CreatePipelineDetail).Methods("POST")
	r.HandleFunc("/pipelines/{id}/status", u.PipelineStatusHandler).Methods("GET")
	r.HandleFunc("/pipelines/{id}/status/{jobname}/log", u.PipelineStatusJobLogHandler).Methods("GET")
	r.HandleFunc("/pipelines/{id}", u.PipelineDetailHandler).Methods("GET")
	r.HandleFunc("/pipelines/{id}/pipeline", u.UpdatePipelineDetail).Methods("POST")
	r.HandleFunc("/pipelines/{id}/deletejobs", u.DeleteJobs).Methods("POST")
	r.HandleFunc("/deletepipelines", u.DeletePipelines).Methods("POST")
	r.HandleFunc("/pipelines/{id}/deletepipeline", u.DeletePipelineDetail).Methods("GET")
	r.HandleFunc("/pipelines/{id}/downloadcreds", u.PipelineDownloadFile).Methods("GET")
	r.HandleFunc("/pipelines/{id}/downloadextract", u.DownloadExtract).Methods("POST")
	r.Path("/pipelines/{id}/tfunctions/show-create").HandlerFunc(u.ShowCreateTransformFunction).Methods("GET")
	r.Path("/pipelines/{id}/tfunctions/{tfid}").HandlerFunc(u.TransformFunction).Methods("GET")
	r.HandleFunc("/pipelines/{id}/tfunctions/{tfid}/tfunction", u.UpdateTransformFunction).Methods("POST")
	r.HandleFunc("/pipelines/{id}/deletetfunctions/{tfid}", u.DeleteTransformFunction).Methods("GET")
	r.HandleFunc("/pipelines/{id}/tfunction", u.CreateTransformFunction).Methods("POST")

	r.HandleFunc("/pipelines/{id}/extractsources/show-create", u.ShowCreateExtractSource).Methods("GET")
	r.HandleFunc("/pipelines/{id}/extractsource/{extractsourceid}/uploadfile", u.UploadToExtractSource).Methods("POST")
	r.HandleFunc("/pipelines/{id}/extractsource/{extractsourceid}/uploadurl", u.UploadURLToExtractSource).Methods("POST")
	r.HandleFunc("/pipelines/{id}/extractsource/{extractsourceid}/uploadschema", u.UploadSchema).Methods("POST")
	r.Path("/pipelines/{id}/startextractsource/{extractsourceid}").HandlerFunc(u.StartExtractSource).Methods("GET")
	r.Path("/pipelines/{id}/stopextractsource/{extractsourceid}").HandlerFunc(u.StopExtractSource).Methods("GET")
	r.Path("/pipelines/{id}/extractsources/{extractsourceid}").HandlerFunc(u.PipelineExtractSource).Methods("GET")
	r.HandleFunc("/pipelines/{id}/updateextractsource/{extractsourceid}", u.UpdateExtractSource).Methods("POST")
	r.HandleFunc("/pipelines/{id}/deleteextractsources/{extractsourceid}", u.DeleteExtractSource).Methods("GET")
	r.HandleFunc("/pipelines/{id}/extractsource", u.CreateExtractSource).Methods("POST")
	r.HandleFunc("/pipelines/{id}/extractsources/{extractsourceid}/rule", u.CreateExtractRule).Methods("POST")

	r.Path("/pipelines/{id}/extractsources/{extractsourceid}/rule/show-create").HandlerFunc(u.ShowCreateExtractRule).Methods("GET")
	r.Path("/pipelines/{id}/extractsources/{extractsourceid}/rule/{rid}").HandlerFunc(u.ExtractRule).Methods("GET")
	r.HandleFunc("/pipelines/{id}/extractsources/{extractsourceid}/updaterule/{rid}", u.UpdateExtractRule).Methods("POST")
	r.Path("/pipelines/{id}/extractsources/{extractsourceid}/deleterule/{rid}").HandlerFunc(u.DeleteExtractRule).Methods("GET")

	r.HandleFunc("/pipelines/{id}/extractsources/{extractsourceid}/extension", u.CreateExtension).Methods("POST")
	r.Path("/pipelines/{id}/extractsources/{extractsourceid}/extension/show-create").HandlerFunc(u.ShowCreateExtension).Methods("GET")
	r.Path("/pipelines/{id}/extractsources/{extractsourceid}/extension/{eid}").HandlerFunc(u.Extension).Methods("GET")
	r.HandleFunc("/pipelines/{id}/extractsources/{extractsourceid}/updateextension/{eid}", u.UpdateExtension).Methods("POST")
	r.Path("/pipelines/{id}/extractsources/{extractsourceid}/deleteextension/{eid}").HandlerFunc(u.DeleteExtension).Methods("GET")

	r.Path("/pipelines/{id}/users/show").HandlerFunc(u.ShowPipelineUsers).Methods("GET")
	r.HandleFunc("/pipelines/{id}/users/add", u.UpdatePipelineUsers).Methods("POST")
	r.Path("/pipelines/{id}/users/{userid}").HandlerFunc(u.PipelineUser).Methods("GET")
	r.HandleFunc("/pipelines/{id}/deleteuser/{uid}", u.DeletePipelineUser).Methods("GET")
	r.HandleFunc("/pipelines/{id}/users/{uid}", u.UpdatePipelineUser).Methods("POST")

	amw := handlers.AuthenticationTokens{}
	r.Use(amw.Middleware)

	staticDir := "/static/"
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir("."+staticDir))))

	// Bind to a port and pass our router in
	port := os.Getenv("PORT")
	log.Info().Msg("listening on port " + port)
	log.Fatal().Err(http.ListenAndServeTLS(":"+port, "https-server.crt", "https-server.key", r))
}
