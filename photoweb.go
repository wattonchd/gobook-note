package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strings"
)

var templates = make(map[string]*template.Template)

const (
	TEMPLATE_DIR = "./views"
	UPLOAD_DIR   = "./uploads"
	ListDir      = 0x0001
)

func init() {
	// 初始化数据 载入静态数据和模板
	// 这样模板只需要读取一次
	fileInfos, err := ioutil.ReadDir(TEMPLATE_DIR)
	checkErr(err)
	for _, fileInfo := range fileInfos {
		templateName := fileInfo.Name()
		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}
		tmpl := strings.Split(templateName, ".")[0]
		templatePath := TEMPLATE_DIR + "/" + templateName
		log.Printf("Loadting template: %v", templateName)
		t := template.Must(template.ParseFiles(templatePath)) // 通过模板路径获取模板
		templates[tmpl] = t
	}
	fmt.Println(templates)
}

func main() {
	mux := http.NewServeMux()
	staticHandler(mux, "/assets/", "./public", 0)
	mux.HandleFunc("/", safeHandler(listHandler))
	mux.HandleFunc("/upload", safeHandler(uploadHandler))
	mux.HandleFunc("/view", safeHandler(viewHandler))
	err := http.ListenAndServe(":8088", mux)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId
	exist, _ := isExist(imagePath)
	if !exist {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image")
	http.ServeFile(w, r, imagePath)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderHtml(w, "upload", nil)
	}
	if r.Method == "POST" {
		file, m, err := r.FormFile("image")
		checkErr(err)
		defer file.Close()
		filename := m.Filename
		t, err := os.Create(UPLOAD_DIR + "/" + filename)
		checkErr(err)
		_, err = io.Copy(t, file)
		checkErr(err)
		http.Redirect(w, r, "/view?id="+filename, http.StatusFound)
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	fileInfoArr, err := ioutil.ReadDir(UPLOAD_DIR)
	checkErr(err)
	locals := make(map[string]interface{})
	var images []string
	for _, fileInfo := range fileInfoArr {
		images = append(images, fileInfo.Name())
	}
	locals["images"] = images
	renderHtml(w, "list", locals)
}

func renderHtml(w http.ResponseWriter, tmpl string, locals map[string]interface{}) {
	err := templates[tmpl].Execute(w, locals)
	checkErr(err)
}

func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				log.Printf("WARN: panic in:%v. - %v", fn, e)
				log.Println(string(debug.Stack()))
			}
		}()
		fn(w, r)
	}
}

func staticHandler(mux *http.ServeMux, prefix string, staticDir string, flags int) {
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		file := staticDir + r.URL.Path[len(prefix)-1:]
		if (flags & ListDir) == 0 {
			if exist, _ := isExist(file); !exist {
				http.NotFound(w, r)
				return
			}
		}
		http.ServeFile(w, r, file)
	})
}

func isExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if e, ok := err.(*os.PathError); ok && e.Err == os.ErrNotExist {
		return false, nil
	}
	return true, err
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
