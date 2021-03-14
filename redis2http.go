package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/skroczek/redis2http/cache"
	"github.com/skroczek/redis2http/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

var c *cache.Cache

var rdb *redis.Client

type key int

const (
	requestIDKey key = 0
)

type StatusRecorder struct {
	http.ResponseWriter
	Status int
	Size   int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *StatusRecorder) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b) // write response using original http.ResponseWriter
	r.Size += size                         // capture size
	return size, err
}

func buildHandler(config config.Config, logger *log.Logger) http.HandlerFunc {
	keys := config.Redis.Keys
	sort.Strings(keys)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		listName := strings.TrimLeft(r.RequestURI, "/")
		i := sort.SearchStrings(keys, listName)
		if !(i < len(keys) && keys[i] == listName) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 File not found.")
			return
		}

		// Get the map. The same approach works for HmGet().
		res := rdb.HGetAll(context.Background(), listName)
		if res.Err() != nil {
			logger.Printf("Error: %s", res.Err().Error())
			http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
			return
			//panic(res.Err())
		}

		m, err := res.Result()
		if err != nil {
			panic(err)
		}
		content := ""

		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			content += k + "\n"

		}

		hash := sha256.Sum256([]byte(content))
		loc, err := time.LoadLocation("GMT")
		if err != nil {
			logger.Printf("Error: %s", err)
			http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
			return
		}
		dateTime := time.Now().In(loc).Truncate(time.Second)

		if c.Has(listName) && c.Key(listName).Hash == hash {
			dateTime = c.Key(listName).DateTime
		} else {
			c.Set(listName, cache.Entry{
				Hash:     hash,
				DateTime: dateTime,
			})
		}

		if modified := r.Header.Get("If-Modified-Since"); modified != "" {
			t, err := time.Parse(time.RFC1123, modified)
			if err != nil {
				panic(err)
			}
			//logger.Printf("Header:  %s", t)
			//logger.Printf("Current: %s", dateTime)
			if dateTime.Equal(t) || dateTime.Before(t) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		if r.Method != "HEAD" {
			w.Header().Set("Content-Type", "application/txt")
			fmt.Fprint(w, content)
		}
	})
}

// NewRouter generates the router used in the HTTP Server
func NewRouter(config config.Config, logger *log.Logger) *http.ServeMux {
	// Create router and define routes and return that router
	router := http.NewServeMux()
	router.Handle("/", buildHandler(config, logger))

	return router
}

// Run will run the HTTP Server
func Run(config *config.Config, logger *log.Logger) {
	// Set up a channel to listen to for interrupt signals
	var runChan = make(chan os.Signal, 1)

	rdb = redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password, // no password set
		DB:       config.Redis.DB,       // use default DB
	})

	// Set up a context to allow for graceful server shutdowns in the event
	// of an OS interrupt (defers the cancel just in case)
	ctx, cancel := context.WithTimeout(
		context.Background(),
		config.Server.Timeout.Server,
	)
	defer cancel()

	// Define server options
	server := &http.Server{
		Addr:         config.Server.Host + ":" + config.Server.Port,
		Handler:      logging(logger)(NewRouter(*config, logger)),
		ReadTimeout:  config.Server.Timeout.Read,
		WriteTimeout: config.Server.Timeout.Write,
		IdleTimeout:  config.Server.Timeout.Idle,
	}

	// Handle ctrl+c/ctrl+x interrupt
	signal.Notify(runChan, os.Interrupt, syscall.SIGTSTP)

	// Alert the user that the server is starting
	logger.Printf("Server is starting on %s with PID %d\n", server.Addr, os.Getpid())

	// Run the server on a new goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				// Normal interrupt operation, ignore
			} else {
				logger.Fatalf("Server failed to start due to err: %v", err)
			}
		}
	}()

	// Block on this channel listeninf for those previously defined syscalls assign
	// to variable so we can let the user know why the server is shutting down
	interrupt := <-runChan

	// If we get one of the pre-prescribed syscalls, gracefully terminate the server
	// while alerting the user
	log.Printf("Server is shutting down due to %+v\n", interrupt)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server was unable to gracefully shutdown due to err: %+v", err)
	}
}

// Func main should be as small as possible and do as little as possible by convention
func main() {
	c = cache.NewCache()
	// Generate our config based on the config supplied
	// by the user in the flags
	cfgPath, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.New(os.Stdout, cfg.Logging.Prefix, cfg.Logging.Flags)

	// Run the server
	Run(cfg, logger)
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &StatusRecorder{
				ResponseWriter: w,
				Status:         200,
			}
			start := time.Now()
			defer func() {
				logger.Println(r.Proto, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent(), recorder.Status, recorder.Size, time.Since(start))
			}()
			next.ServeHTTP(recorder, r)
		})
	}
}
