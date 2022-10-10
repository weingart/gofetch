package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/blang/semver"
)

const goURL = `https://go.dev/dl/?mode=json` // List of latest production versions

var verbose = flag.Bool("v", false, "Print verbose instructions.")

type GoPackage struct {
	Version string
	Stable  bool
	v       semver.Version
}

type GoPackages []*GoPackage

func (g GoPackages) Len() int           { return len(g) }
func (g GoPackages) Less(i, j int) bool { return g[i].v.LT(g[j].v) }
func (g GoPackages) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }

// GoVersionToSemVer tries to convert a Go version to something that can be
// parsed into something resembling a SemVer.
//
// Dear $DEITY, why was this not done before!?!
//
// $AUTHOR proposes that future versions of Go follow a more parseable and
// simple format: $THING-$SEMVER.EXTENSIONS
//
// For example:
//
//	go-1.8-beta.1+linux-s390x.tar.gz
//	go-tools-1.8.1+src.tar.gz
func GoVersionToSemVer(name string) (semver.Version, error) {
	// Because we have something even more... ugh, don't even.
	return semver.ParseTolerant(strings.TrimPrefix(name, "go"))
}

func main() {
	flag.Parse()

	resp, err := http.Get(goURL)
	if err != nil {
		log.Fatalf("Error retrieving Go versions: %v", err)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	var versions GoPackages
	if err := json.Unmarshal(buf, &versions); err != nil {
		log.Fatalf("Error parsing Go versions: %v", err)
	}

	for _, pkg := range versions {
		pkg.v, err = GoVersionToSemVer(pkg.Version)
		if err != nil {
			log.Fatalf("Error decoding Go version %q: %v", pkg.Version, err)
		}
	}

	sort.Sort(sort.Reverse(versions))
	if len(versions) <= 0 {
		log.Fatal("No Go versions detected!")
	}
	if *verbose {
		fmt.Println("# Run the following commands:")
		fmt.Printf("go install golang.org/dl/%s@latest\n", versions[0].Version)
		fmt.Printf("%s download\n", versions[0].Version)
		fmt.Printf("ln -sf $(which %s) $(which go)\n", versions[0].Version)
	} else {
		fmt.Println(versions[0].Version)
	}
}
