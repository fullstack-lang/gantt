package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"

	"github.com/fullstack-lang/gantt"
	gantt_fullstack "github.com/fullstack-lang/gantt/go/fullstack"
	"github.com/fullstack-lang/gantt/go/gantt2svg"
	gantt_models "github.com/fullstack-lang/gantt/go/models"

	gongdoc_load "github.com/fullstack-lang/gongdoc/go/load"

	// import this package in order to have the scheduler start a thread that will
	// generate a new svg diagram each time the repo has been modified
	_ "github.com/fullstack-lang/gantt/go/gantt2svg"

	gongsvg_fullstack "github.com/fullstack-lang/gongsvg/go/fullstack"
	gongsvg_models "github.com/fullstack-lang/gongsvg/go/models"
)

var (
	logDBFlag  = flag.Bool("logDB", false, "log mode for db")
	logGINFlag = flag.Bool("logGIN", false, "log mode for gin")

	backupFlag  = flag.Bool("backup", false, "read database file, generate backup and exits")
	restoreFlag = flag.Bool("restore", false, "generate restore and exits")

	marshallOnStartup  = flag.String("marshallOnStartup", "", "at startup, marshall staged data to a go file with the marshall name and '.go' (must be lowercased without spaces). If marshall arg is '', no marshalling")
	unmarshall         = flag.String("unmarshall", "", "unmarshall data from marshall name and '.go' (must be lowercased without spaces), If unmarshall arg is '', no unmarshalling")
	marshallOnCommit   = flag.String("marshallOnCommit", "", "on all commits, marshall staged data to a go file with the marshall name and '.go' (must be lowercased without spaces). If marshall arg is '', no marshalling")
	unmarshallFromCode = flag.String("unmarshallFromCode", "", "unmarshall data from go file and '.go' (must be lowercased without spaces), If unmarshallFromCode arg is '', no unmarshalling")

	diagrams = flag.Bool("diagrams", true, "parse diagrams (takes a few seconds)")

	embeddedDiagrams = flag.Bool("embeddedDiagrams", false, "parse/analysis go/models and go/embeddedDiagrams")
)

// InjectionGateway is the singloton that stores all functions
// that can set the objects the stage
// InjectionGateway stores function as a map of names
var InjectionGateway = make(map[string](func()))

// hook marhalling to stage
type BeforeCommitImplementation struct {
}

func (impl *BeforeCommitImplementation) BeforeCommit(stage *gantt_models.StageStruct) {
	file, err := os.Create(fmt.Sprintf("./%s.go", *marshallOnCommit))
	if err != nil {
		log.Fatal(err.Error())
	}
	defer file.Close()

	gantt_models.Stage.Checkout()
	gantt_models.Stage.Marshall(file, "github.com/fullstack-lang/gantt/go/models", "main")
	gantt2svg.GanttToSVGTranformerSingloton.GenerateSvg(stage)
}

func main() {

	log.SetPrefix("gantt: ")
	log.SetFlags(0)

	// parse program arguments
	flag.Parse()

	// setup controlers
	if !*logGINFlag {
		myfile, _ := os.Create("/tmp/server.log")
		gin.DefaultWriter = myfile
	}
	r := gin.Default()
	r.Use(cors.Default())

	gantt_fullstack.Init(r)
	gongsvg_fullstack.Init(r)

	// generate injection code from the stage
	if *marshallOnStartup != "" {

		if strings.Contains(*marshallOnStartup, " ") {
			log.Fatalln(*marshallOnStartup + " must not contains blank spaces")
		}
		if strings.ToLower(*marshallOnStartup) != *marshallOnStartup {
			log.Fatalln(*marshallOnStartup + " must be lowercases")
		}

		file, err := os.Create(fmt.Sprintf("./%s.go", *marshallOnStartup))
		if err != nil {
			log.Fatal(err.Error())
		}
		defer file.Close()

		gantt_models.Stage.Checkout()
		gantt_models.Stage.Marshall(file, "github.com/fullstack-lang/gantt/go/models", "main")
		os.Exit(0)
	}

	// setup the stage by injecting the code from code database
	stage := &gantt_models.Stage
	_ = stage
	if *unmarshall != "" {
		gantt_models.Stage.Checkout()
		gantt_models.Stage.Reset()
		gantt_models.Stage.Commit()
		if InjectionGateway[*unmarshall] != nil {
			InjectionGateway[*unmarshall]()
		}

		gantt_models.Stage.Commit()
	}
	if *unmarshallFromCode != "" {
		gantt_models.Stage.Checkout()
		gantt_models.Stage.Reset()
		gantt_models.Stage.Commit()
		err := gantt_models.ParseAstFile(*unmarshallFromCode)

		// if the application is run with -unmarshallFromCode=xxx.go -marshallOnCommit
		// xxx.go might be absent the first time. However, this shall not be a show stopper.
		if err != nil {
			log.Println("no file to read " + err.Error())
		}

		gantt_models.Stage.Commit()
	} else {
		// in case the database is used, checkout the content to the stage
		gantt_models.Stage.Checkout()
	}

	// hook automatic marshall to go code at every commit
	if *marshallOnCommit != "" {
		hook := new(BeforeCommitImplementation)
		gantt_models.Stage.OnInitCommitFromFrontCallback = hook
	}

	// put all to database
	gantt_models.Stage.Commit()
	gongsvg_models.Stage.Commit()

	gongdoc_load.Load(
		"gantt",
		"github.com/fullstack-lang/gantt/go/models",
		gantt.GoDir,
		r,
		*embeddedDiagrams,
		&gantt_models.Stage.Map_GongStructName_InstancesNb)

	// provide the static route for the angular pages
	r.Use(static.Serve("/", EmbedFolder(gantt.NgDistNg, "ng/dist/ng")))
	r.NoRoute(func(c *gin.Context) {
		fmt.Println(c.Request.URL.Path, "doesn't exists, redirect on /")
		c.Redirect(http.StatusMovedPermanently, "/")
		c.Abort()
	})

	gantt2svg.GanttToSVGTranformerSingloton.GenerateSvg(stage)

	log.Printf("Server ready serve on localhost:8080")
	r.Run()
}

type embedFileSystem struct {
	http.FileSystem
}

func (e embedFileSystem) Exists(prefix string, path string) bool {
	_, err := e.Open(path)
	return err == nil
}

func EmbedFolder(fsEmbed embed.FS, targetPath string) static.ServeFileSystem {
	fsys, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		panic(err)
	}
	return embedFileSystem{
		FileSystem: http.FS(fsys),
	}
}
