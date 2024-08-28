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
	"github.com/yuin/goldmark/parser"
	"gopkg.in/yaml.v2"
)

func main() {
	mtx := gin.Default()

	mtx.LoadHTMLGlob("templates/*")

	mtx.GET("/posts/:slug", PostHandler(FileReader{}))
	mtx.GET("/", func(ctx *gin.Context) {
		posts, err := loadMarkdownPosts("./markdown")
		if err != nil {
			log.Fatal(err)
		}

		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"Posts": posts,
		})
	})

	mtx.Static("/static", "static")
	mtx.Run(":8080")
}

type SlugRender interface {
	Read(slug string) (string, error)
}

type FileReader struct{}

func (fRead FileReader) Read(slug string) (string, error) {
	fr, err := os.Open("markdown/" + slug + ".md")
	if err != nil {
		return "", err
	}
	defer fr.Close()
	b, err := io.ReadAll(fr)

	if err != nil {
		return "", err
	}

	return string(b), nil
}

type PostData struct {
	Title                   string `yaml:"Title"`
	Slug                    string `yaml:"Slug"`
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

func mdToHTML(md []byte) []byte {
	// Create a new Goldmark parser with extensions
	mdParser := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			extension.GFM,
		),
	)

	// Convert Markdown to HTML
	var buf bytes.Buffer
	if err := mdParser.Convert(md, &buf); err != nil {
		log.Fatal(err)
		// Handle error, e.g., log it or return an empty result
		return nil
	}

	return buf.Bytes()
}

func readAndParse(sl SlugRender, slug string) (PostData, string, error) {
	var post PostData

	postMarkdown, err := sl.Read(slug)
	if err != nil {
		return post, "", err
	}

	remainingMd, err := frontmatter.Parse(strings.NewReader(postMarkdown), &post)
	if err != nil {
		return post, "", err
	}

	return post, string(remainingMd), nil
}

func renderMarkdown(mdRenderer goldmark.Markdown, markdown string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func loadMarkdownPosts(dir string) ([]PostData, error) {
	md := goldmark.New()
	var posts []PostData

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a file and not a directory
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Assuming metadata is in YAML front matter
			var postData PostData
			var buf bytes.Buffer

			// Split content to extract YAML front matter and Markdown body
			split := strings.SplitN(string(content), "\n---\n", 2)
			if len(split) > 1 {
				// Parse YAML front matter
				err = yaml.Unmarshal([]byte(split[0]), &postData)
				if err != nil {
					return err
				}

				// Convert Markdown to HTML
				err = md.Convert([]byte(split[1]), &buf)
				if err != nil {
					return err
				}

				// Assign HTML content to PostData
				postData.Content = template.HTML(buf.String())
			} else {
				// Handle case where there is no front matter
				err = md.Convert(content, &buf)
				if err != nil {
					return err
				}

				postData.Content = template.HTML(buf.String())
			}

			// Append postData to the posts slice
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

	return func(c *gin.Context) {
		slug := c.Param("slug")

		post, remainingMd, err := readAndParse(sl, slug)
		if err != nil {
			c.String(http.StatusNotFound, "Post not found")
			return
		}

		post.Content, err = renderMarkdown(mdRenderer, remainingMd)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error rendering markdown")
			return
		}

		c.HTML(http.StatusOK, "post.html", post)
	}
}
