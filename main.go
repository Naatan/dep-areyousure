package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/KyleBanks/depth"
)

type GoSearchPackage struct {
	Package    string
	StarCount  int
	Imported   []string
	StaticRank int
}

func main() {
	forwardGet()

	flag.Parse()
	pkg := flag.Arg(0)

	if pkg == "" {
		log.Fatal("No package specified")
	}

	direct, indirect := getDependencies(pkg)
	total := len(direct) + len(indirect)

	if total > 2 {
		fmt.Printf("\033[1mDirect dependencies (%d):\033[0m", len(direct))
		fmt.Println()
		fmt.Println(" " + strings.Join(direct, ", "))

		fmt.Println()
		fmt.Printf("\033[1mIndirect dependencies (%d):\033[0m", len(indirect))
		fmt.Println()
		fmt.Println(" " + strings.Join(indirect, ", "))

		fmt.Println()
		fmt.Printf(
			"Package \033[1m%s\033[0m has a total of \033[1m%d\033[0m dependencies, of which \033[1m%d\033[0m are direct "+
				"dependencies and \033[1m%d\033[0m indirect dependencies",
			pkg, total, len(direct), len(indirect))

		fmt.Println()
		stats := getStats(pkg)
		fmt.Printf(
			" \\- used in \033[1m%d\033[0m other packages, has \033[1m%d\033[0m stars and a ranking of \033[1m%d\033[0m",
			len(stats.Imported), stats.StarCount, stats.StaticRank)

		fmt.Println()
		if !askForConfirmation(pkg) {
			os.Exit(0)
		}
	}

	fmt.Println("Forwarding your request to `dep ensure` ..")
	fmt.Println()
	forwardDep()
}

func getDependencies(rootPkgName string) ([]string, []string) {
	var t depth.Tree
	err := t.Resolve(rootPkgName)
	if err != nil {
		log.Fatal(err)
	}

	direct := []string{}
	var indirect []string

	for _, pkg := range t.Root.Deps {
		if pkg.Internal {
			continue
		}
		direct = append(direct, pkg.Name)
		indirect = walkDependencies(pkg.Deps)
	}

	direct = unique(direct, []string{})
	indirect = unique(indirect, direct)

	return direct, indirect
}

func unique(slice []string, secondarySlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range secondarySlice {
		keys[entry] = true
	}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func getStats(rootPkgName string) *GoSearchPackage {
	re := regexp.MustCompile(`(.*?\/.*?\/.*?)(?:\/|$)`)
	res := re.FindStringSubmatch(rootPkgName)

	if len(res) > 0 {
		rootPkgName = res[1]
	}

	v := url.Values{}
	v.Set("id", rootPkgName)

	resp, err := http.Get("http://go-search.org/api?action=package&" + v.Encode())
	if err != nil {
		log.Fatal(err.Error())
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err.Error())
	}

	data := &GoSearchPackage{}
	bdy := string(body)
	_ = bdy
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		log.Fatal(err.Error())
	}

	return data
}

func walkDependencies(pkgs []depth.Pkg) []string {
	pkgNames := []string{}

	for _, pkg := range pkgs {
		if pkg.Internal {
			continue
		}
		pkgNames = append(pkgNames, pkg.Name)
		pkgNames = append(pkgNames, walkDependencies(pkg.Deps)...)
	}

	return pkgNames
}

// askForConfirmation uses Scanln to parse user input. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user. Typically, you should use fmt to print out a question
// before calling askForConfirmation. E.g. fmt.Println("WARNING: Are you sure? (yes/no)")
// Based on https://gist.github.com/albrow/5882501
func askForConfirmation(pkgName string) bool {
	fmt.Println()
	fmt.Printf("Are you sure you wish to install \033[1m%s\033[0m and all above dependencies? [Y/N]", pkgName)
	fmt.Println()

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation(pkgName)
	}
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

func forwardDep() {
	args := os.Args[1:]
	args = append([]string{"ensure"}, args...)
	cmd := exec.Command("dep", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func forwardGet() {
	fmt.Println("Ensuring dependencies are on our GOPATH..")
	args := os.Args[1:]
	args = append([]string{"get"}, args...)
	cmd := exec.Command("go", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	fmt.Println()

	if err != nil {
		log.Fatal(err)
	}
}
