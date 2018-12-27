package main

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"cloud.google.com/go/storage"
	"github.com/blang/semver"
	"github.com/kr/pretty"
	"golang.org/x/net/context"
)

type GoPackage struct {
	Filename string
	Version  semver.Version
}

type GoPackages []*GoPackage

func (g GoPackages) Len() int           { return len(g) }
func (g GoPackages) Less(i, j int) bool { return g[i].Version.LT(g[j].Version) }
func (g GoPackages) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }

func findMetadata(parts []string) int {

	i := 1
	for ; i < len(parts); i++ {
		r, _ := utf8.DecodeRuneInString(parts[i])
		if !unicode.IsNumber(r) {
			return i
		}
	}

	return i
}

func parseMetadata(b, m string) string {

	return "+" + b + "." + strings.Replace(m, "osx10.", "osx10-", -1)
}

func parseExtra(s string) string {
	re1 := regexp.MustCompile(`([[:digit:]]+)`)
	re2 := regexp.MustCompile(`\.\.+`)

	s = strings.Replace(s, "-", ".", -1)
	s = re1.ReplaceAllString(s, ".$1.")
	s = re2.ReplaceAllString(s, ".")
	s = strings.TrimLeft(s, ".")
	s = strings.TrimRight(s, ".")
	if s != "" {
		s = "-" + s
	}

	return s
}

func parseVersion(s string) (string, string) {
	v := regexp.MustCompile(`^[[:digit:]]+(\.[[:digit:]]+)+`).FindString(s)

	return v, strings.TrimPrefix(s, v)
}

func parseVersionAndExtra(s string) (string, string) {
	v, e := parseVersion(s)
	for len(strings.Split(v, ".")) < 3 {
		v = v + ".0"
	}

	return v, parseExtra(e)
}

func TrimBase(s string) (string, string) {
	base := regexp.MustCompile(`^[[:alpha:]-]+`).FindString(s)

	return strings.TrimPrefix(s, base), strings.TrimRight(base, "-")
}

func TrimMetadata(s string) (string, string) {
	part := strings.Split(s, ".")
	metaidx := findMetadata(part)
	metadata := strings.Join(part[metaidx:], ".")

	return strings.TrimSuffix(s, "."+metadata), metadata
}

// FilenameToSemVer tries to convert a Go package name to something that
// can be parsed into something resembling a SemVer along with some extra
// information to identify the full filename.
//
// Dear $DEITY, why was this not done before!?!
//
// $AUTHOR proposes that future filenames of published Go packages follow
// a much more parseable and simple format:
//
// $FILENAME-$SEMVER
//
// For example:
//
// go-1.8-beta.1+linux-s390x.tar.gz
// go-tools-1.8.1+src.tar.gz
//
// In this way, splitting the first part off is simple, as long as the
// first portion does not include the regex "-[0-9]".
//
func FilenameToSemVer(name string) (*GoPackage, error) {
	// Because we have something even more... ugh, don't even.
	//n := strings.Replace(name, "-bootstrap-", ".bootstrap-", -1)
	n := name
	fmt.Println("Filename:", name)

	// Grab the base
	n, base := TrimBase(n)

	// Grab suffix/build info
	n, metadata := TrimMetadata(n)
	metadata = parseMetadata(base, metadata)

	// Find the first non-extension part
	version, rest := parseVersionAndExtra(n)

	// Should look like a semver now (maybe)
	fmt.Println("Parsing:", version+rest+metadata)
	v, err := semver.ParseTolerant(version + rest + metadata)
	if err != nil {
		return nil, err
	}
	return &GoPackage{
		Filename: name,
		Version:  v,
	}, nil
}

func GetFilenames(c *storage.Client, bucket string) (names []string) {

	b := c.Bucket(bucket)
	q := &storage.Query{
		Delimiter: "/",
		// Prefix: "go1.7.4",
		// Versions: false,
	}

	it := b.Objects(context.Background(), q)
	for obj, err := it.Next(); obj != nil; obj, err = it.Next() {
		if err != nil {
			log.Fatal(err)
		}
		if len(obj.Name) > 0 {
			names = append(names, obj.Name)
		}
	}

	return
}

func main() {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	files := GetFilenames(client, "golang")

	versions := make(GoPackages, 0)
	for _, name := range files {
		gp, err := FilenameToSemVer(name)
		if err != nil {
			log.Printf("Bad parse of filename '%v': %v\n", name, err)
			continue
		}
		versions = append(versions, gp)
	}

	sort.Sort(versions)
	for _, v := range versions {
		pretty.Println(v)
		fmt.Println("")
	}
}
