package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/rakyll/statik/fs"
)

// now would this be shitposting if there were _tests_?

func main() {

	listenAddress := flag.String("listen", ":8080", "address to liste")
	imageDir := flag.String("images", "./", "location of images to host")
	thumbDir := flag.String("thumbs", "", "if set, location to hold thumbnails")
	staticDir := flag.String("static", "", "if set, alternate location to serve as /static/")
	templateFile := flag.String("template", "", "if set, alternate template to use")
	thumbWidth := flag.Int("thumbWidth", 310, "width of thumbnails to create")
	thumbHeight := flag.Int("thumbHeight", 200, "width of thumbnails to create")

	flag.Parse()

	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Llongfile)

	fs, err := fs.New()
	if err != nil {
		logger.Fatal(err)
	}

	var templ *template.Template
	if *templateFile == "" {
		// have to do it the hard way because it comes from fs
		templFile, err := fs.Open("/page.template")
		if err != nil {
			logger.Fatal(err)
		}
		templData, err := ioutil.ReadAll(templFile)
		if err != nil {
			logger.Fatal(err)
		}
		templ, err = template.New("page.template").Parse(string(templData))
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		// if an alternate template was provided, i can use that instead
		templ, err = template.ParseFiles(*templateFile)
		if err != nil {
			logger.Fatal(err)
		}
	}

	var thumbnailPath string
	if *thumbDir != "" {
		if *imageDir != "" {
			thumbnailPath = fmt.Sprintf("%s-%s", *imageDir, *thumbDir)
			err = os.MkdirAll(thumbnailPath, 0750)
			if err != nil {
				logger.Fatalf("Could not create tempoary thumbnail directory - %s", err)
			}
		} else {
			thumbnailPath, err = ioutil.TempDir("", "thumbnailcache-")
			if err != nil {
				logger.Fatalf("Could not create tempoary thumbnail directory - %s", err)
			}
		}
	}

	mux := http.NewServeMux()
	done := make(chan struct{})
	defer close(done)

	var staticHandler http.Handler
	if *staticDir == "" {
		staticHandler = InternalHandler(logger, fs)
	} else {
		staticHandler = http.FileServer(http.Dir(*staticDir))
	}

	mux.Handle(
		"/", DirSplitHandler(logger, *imageDir, done,
			IndexHandler(logger, *imageDir, done, templ),
			ContentTypeHandler(logger, *imageDir),
		),
	)

	mux.Handle("/static/",
		http.StripPrefix("/static/",
			SplitHandler(
				http.RedirectHandler("/", 302),
				staticHandler,
			),
		),
	)

	mux.Handle("/thumb/",
		http.StripPrefix("/thumb/",
			SplitHandler(
				http.RedirectHandler("/", 302),
				ThumbnailHandler(logger, *thumbWidth, *thumbHeight, *imageDir, thumbnailPath, "jpg"),
			),
		),
	)

	logger.Fatal(http.ListenAndServe(*listenAddress, mux))
}
