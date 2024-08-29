package main

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"gopkg.in/yaml.v2"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()

	route.LoadHTMLGlob("templates/*")

	route.GET("/posts/:slug", PostHandler(FileReader{}))
	route.GET("/", func(ctx *gin.Context) {
		posts, err := loadMarkdownPosts("./markdown")
		if err != nil {
			log.Fatal(err)
		}

		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"Posts": posts,
		})
	})

	route.Static("/static", "static")
	route.Run(":8080")
}

type PostData struct {
	Title                   string `yaml:"Title"`
	Slug                    string `yaml:"Slug"`
	Date                    string `yaml:"Date"`
	Description             string `yaml:"Description"`
	Order                   int    `yaml:"Order"`
	MetaDescription         string `yaml:"MetaDescription"`
	MetaPropertyTitle       string `yaml:"MetaPropertyTitle"`
	MetaPropertyDescription string `yaml:"MetaPropertyDescription"`
	MetaOgURL               string `yaml:"MetaOgURL"`
	Author                  Author `yaml:"author"`
	Content                 template.HTML
}

type Author struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type SlugRender interface {
	Read(slug string) (string, error)
}

type FileReader struct{}

func (fRead FileReader) Read(slug string) (string, error) {
	fileRead, err := os.Open("markdown/" + slug + ".md")
	if err != nil {
		return "", err
	}
	defer fileRead.Close()
	b, err := io.ReadAll(fileRead)

	if err != nil {
		return "", err
	}

	return string(b), nil
}

func loadMarkdownPosts(dir string) ([]PostData, error) {
	md := goldmark.New()
	var posts []PostData

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check file .md
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var postData PostData
			var buf bytes.Buffer

			// Split content to extract YAML front matter and Markdown body
			split := strings.SplitN(string(content), "\n---\n", 2)
			if len(split) > 1 {
				// Parse YAML front matter -> Convert Markdown to HTML -> Assign HTML content to PostData
				err = yaml.Unmarshal([]byte(split[0]), &postData)
				if err != nil {
					return err
				}

				err = md.Convert([]byte(split[1]), &buf)
				if err != nil {
					return err
				}

				postData.Content = template.HTML(buf.String())
			} else {
				// Handle case where there is no front matter
				err = md.Convert(content, &buf)
				if err != nil {
					return err
				}

				postData.Content = template.HTML(buf.String())
			}

			posts = append(posts, postData)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return posts, nil
}

func PostHandler(sl SlugRender) gin.HandlerFunc {
	mdRenderer := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
		),
	)

	return func(ctx *gin.Context) {
		slug := ctx.Param("slug")
		postMarkdown, err := sl.Read(slug)

		if err != nil {
			ctx.String(http.StatusNotFound, "Post not found", err)
			return
		}

		var post PostData
		remainingMd, err := frontmatter.Parse(strings.NewReader(postMarkdown), &post)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Error parsing frontmatter", err)
			return
		}

		var buf bytes.Buffer
		err = mdRenderer.Convert([]byte(remainingMd), &buf)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Error rendering markdown")
			return
		}

		post.Content = template.HTML(buf.String())

		ctx.HTML(http.StatusOK, "post.html", post)
	}
}
