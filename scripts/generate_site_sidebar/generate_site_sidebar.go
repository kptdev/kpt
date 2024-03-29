// Copyright 2019 The kpt Authors
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
	"html/template"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/igorsobreira/titlecase"
)

const markdownExtension = ".md"
const introPage = "00.md"
const templatePath = "scripts/generate_site_sidebar/sidebar_template.md.tmpl"

var pagePrefix = regexp.MustCompile(`^\d\d-?`)

func main() {
	fmt.Println("<!--This sidebar was automatically generated by 'generate_site_sidebar'-->")
	t := template.Must(
		template.New(path.Base(templatePath)).
			Funcs(template.FuncMap{"bookLayout": getBookOutline}).
			ParseFiles(templatePath))

	err := t.Execute(os.Stdout, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func getBookOutline() string {
	sourcePath := "site/book"
	chapters := collectChapters(sourcePath)

	return getChapterBlock(chapters)
}

func collectChapters(source string) []chapter {
	chapters := make([]chapter, 0)
	chapterDirs, err := os.ReadDir(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	for _, dir := range chapterDirs {
		if dir.IsDir() {
			chapters = append(chapters, getChapter(dir.Name(), filepath.Join(source, dir.Name())))
		}
	}

	return chapters
}

func getChapter(chapterDirName string, chapterDirPath string) chapter {
	chapterBuilder := chapter{}

	// Split into chapter number and hyphenated name
	splitDirName := strings.SplitN(chapterDirName, "-", 2)
	chapterBuilder.Number = splitDirName[0]
	chapterBuilder.Name = titlecase.Title(strings.ReplaceAll(splitDirName[1], "-", " "))

	pageFiles, err := os.ReadDir(chapterDirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	for _, pageFile := range pageFiles {
		if filepath.Ext(pageFile.Name()) == markdownExtension && pagePrefix.MatchString(pageFile.Name()) {
			chapterBuilder.Pages = append(chapterBuilder.Pages,
				getPage(pageFile.Name(), chapterBuilder.Name, chapterDirPath))
		}
	}

	return chapterBuilder
}

func getPage(pageFileName string, defaultName string, parentPath string) page {
	// Split into page number and hyphenated name.
	splitPageName := strings.SplitN(pageFileName, "-", 2)

	pageName := defaultName
	if pageFileName != introPage {
		// Strip page number and extension from file name.
		pageTitle := pagePrefix.ReplaceAll([]byte(pageFileName), []byte(""))
		pageName = titlecase.Title(strings.ReplaceAll(strings.ReplaceAll(string(pageTitle), ".md", ""), "-", " "))
	}

	return page{
		Number: splitPageName[0],
		Name:   pageName,
		Path:   filepath.Join(parentPath, pageFileName),
	}
}

func getChapterBlock(chapters []chapter) string {
	// Sort chapters in ascending order by chapter number.
	sort.Slice(chapters, func(i, j int) bool { return chapters[i].Number < chapters[j].Number })
	var sb strings.Builder
	for chapterIndex, chapterEntry := range chapters {
		chapterNumber := chapterIndex + 1
		for pageIndex, pageEntry := range chapterEntry.Pages {
			// Make path relative to site directory.
			path := strings.Replace(pageEntry.Path, "site/", "", 1)

			// Print non-chapter intro pages as children of chapter intro page.
			if pageIndex == 0 {
				sb.WriteString(fmt.Sprintf("\t- [%d %s](%s)\n", chapterNumber, pageEntry.Name, path))
			} else {
				sb.WriteString(fmt.Sprintf("\t\t- [%d.%d %s](%s)\n", chapterNumber, pageIndex, pageEntry.Name, path))
			}
		}
	}
	return strings.TrimRight(sb.String(), "\n")
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
