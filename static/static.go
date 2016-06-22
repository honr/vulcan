package static

import (
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/honr/vulcan/htl"
)

type Resource struct {
	ContentType string
	Content []byte
}

func htlToHTML(r *Resource) error {
	n, err := htl.Parse(string(r.Content))
	if err != nil {
		return err
	}
	r.ContentType = mime.TypeByExtension(".html")
	r.Content = []byte(n.String())
	return nil
}

var transformers = map[string]func(*Resource)error {
	".htl": htlToHTML,
}

func ResourceFromFile(filename string) (*Resource, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(filename)
	resource := &Resource{
		ContentType: mime.TypeByExtension(ext),
		Content: content,
	}

	if f, has := transformers[ext]; has {
		if err = f(resource); err != nil {
			return nil, err
		}
	}
	return resource, nil
}

func HandlerFuncFromFile(filename string, dev bool) (http.HandlerFunc, error) {
	if dev {
		return func(w http.ResponseWriter, r *http.Request) {
			resource, err := ResourceFromFile(filename)
			if err != nil {
				fmt.Println(err)
				// Log?
				return
			}
			w.Header().Add("Content-Type", resource.ContentType)
			w.Write(resource.Content)
		}, nil
	}
	resource, err := ResourceFromFile(filename)
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", resource.ContentType)
		w.Write(resource.Content)
	}, nil
}

func HandlersFromDirs(dirs []string, dev bool) (map[string]http.HandlerFunc, error) {
	m := map[string]http.HandlerFunc{}
	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, errIn error) error {
			if errIn != nil {
				return errIn
			}
			subpath := strings.TrimPrefix(path, dir)
			if subpath == "" {
				return nil // skip the root.
			}
			h, err := HandlerFuncFromFile(path, dev)
			if err != nil {
				return err
			}
			m[subpath] = h
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}
