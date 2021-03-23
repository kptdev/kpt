// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const markdownExtension = ".md"

func main() {
	source := "site/book"
	chapters := make([]chapter, 0)
	chapterDirs, err := os.ReadDir(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	for _, dir := range chapterDirs {
		chapterBuilder := chapter{}
		if dir.IsDir() {
			// Split into chapter number and hyphenated name
			splitDirName := strings.SplitN(dir.Name(), "-", 2)
			chapterBuilder.Number = splitDirName[0]
			chapterBuilder.Name = strings.Title(strings.ReplaceAll(splitDirName[1], "-", " "))

			chapterDir := filepath.Join(source, dir.Name())
			pageFiles, err := os.ReadDir(chapterDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			for _, pageFile := range pageFiles {
				if filepath.Ext(pageFile.Name()) == markdownExtension {
					// Split into page number and hyphenated name
					splitPageName := strings.SplitN(pageFile.Name(), "-", 2)

					pageTitle := regexp.MustCompile(`^\d\d-?`).ReplaceAll([]byte(pageFile.Name()), []byte(""))
					pageName := chapterBuilder.Name
					if pageFile.Name() != "00.md" {
						pageName = strings.Title(strings.ReplaceAll(strings.ReplaceAll(string(pageTitle), ".md", ""), "-", " "))
					}

					chapterBuilder.Pages = append(chapterBuilder.Pages,
						page{
							Number: splitPageName[0],
							Name:   pageName,
							Path:   filepath.Join(filepath.Join(source, dir.Name()), pageFile.Name()),
						})
				}
			}
		}
		chapters = append(chapters, chapterBuilder)
	}

	sort.Slice(chapters, func(i, j int) bool { return chapters[i].Number < chapters[j].Number })
	for _, chapterEntry := range chapters {
		for pageNumber, pageEntry := range chapterEntry.Pages {
			path := strings.Replace(pageEntry.Name, "site/", "", 1)
			if pageNumber == 0 {
				fmt.Printf("- [%s](%s)\n", pageEntry.Name, path)
			} else {
				fmt.Printf("\t- [%s](%s)\n", pageEntry.Name, path)
			}
		}
	}

}

type chapter struct {
	Name   string
	Pages  []page
	Number string
}

type page struct {
	Name   string
	Path   string
	Number string
}
