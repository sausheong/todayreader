package main

import (
  "net/http"
  "io/ioutil"
  // "fmt"
  "log"  
  // "encoding/base64"
  "html/template"
  "strconv"
  "time"
  "github.com/julienschmidt/httprouter"
)

func failOnError(err error, msg string) {
  if err != nil {
    log.Fatalf("%s: %s", msg, err)
    // panic(fmt.Sprintf("%s: %s", msg, err))
  }
}

// struct for the Today paper
type Paper struct {
  NumOfPages    int
  DateRefreshed time.Time
  Pages         [][]byte
  Previews      [][]byte
}

func (paper *Paper) AddPage(pg []byte) {
  paper.Pages = append(paper.Pages, pg) 
}

func (paper *Paper) AddPreview(pre []byte) {
  paper.Previews = append(paper.Previews, pre) 
}

func (paper *Paper) ClearAll() {
  paper.Pages = nil
  paper.Previews = nil
}

func (paper *Paper) ShouldRefresh() (should bool) {
  if len(paper.Pages) == 0 {
    return true
  }
  now := time.Now().Format("20060201")
  if now == paper.DateRefreshed.Format("20060201") {
    return false
  } else {
    return true
  }
}

// stores the Today paper in memory
var paper Paper

func main() {
  r := httprouter.New()
  
  r.GET("/", index)
  r.GET("/page/:num", page)
  r.GET("/pdf/:num", pdf)
  r.GET("/image/:num", image)
  
  r.ServeFiles("/css/*filepath", http.Dir("public/css"))
  r.ServeFiles("/fonts/*filepath", http.Dir("public/fonts"))
  r.ServeFiles("/js/*filepath", http.Dir("public/js"))
  server := &http.Server{
  	Addr:           "0.0.0.0:40947",
  	Handler:        r,
  	ReadTimeout:    10 * time.Second,
  	WriteTimeout:   600 * time.Second,
  	MaxHeaderBytes: 1 << 20,
  }
  server.ListenAndServe()
  // http.ListenAndServe("0.0.0.0:40947", r)
}

// extract pages from the source
func refresh() {
  paper.ClearAll()
  paper.DateRefreshed = time.Now()
  paper.NumOfPages = 0
  
  now := time.Now()
  if now.Weekday().String() == "Sunday" {
    now = now.AddDate(0,0,-1)
  }  
  date := now.Format("20060201")
  for i := 1;; i++ { 
    
    url := "http://www.todayonline.com/sites/default/files/styles/large/public/" + 
           date + "_AP_page_"+ strconv.Itoa(i) + ".jpg"
    if isValidPage(url) {
      paper.NumOfPages = paper.NumOfPages + 1
      // extract pages for preview
      preview := extract(url)
      paper.AddPreview(preview)
      
      // extract pages
      pdf_url := "http://www.todayonline.com/sites/default/files/" + 
                 date + "_AP_page_" + strconv.Itoa(i) + ".pdf"
      pdf := extract(pdf_url)
      paper.AddPage(pdf) 

    } else {
      break
    }
  }
  println(paper.NumOfPages)
  println(len(paper.Previews))
}

// extract from the URL
func extract(url string) (body []byte){  
  resp, err := http.Get(url)
  failOnError(err, "Failed get " + url)  
  defer resp.Body.Close()
  body, err = ioutil.ReadAll(resp.Body)
  failOnError(err, "Failed read body")
  return
}

// check if this is a valid page; used in the iteration of the pages
func isValidPage(url string) (validity bool) {
  resp, err := http.Get(url)
  failOnError(err, "Failed get " + url)  
  if resp.StatusCode == 200 {
    validity = true
  } else {
    validity = false
  }
  return
}

// Handlers for the web app

// for route '/'
func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
  if paper.ShouldRefresh() {
    refresh()
  }
  
  t, err := template.ParseFiles("templates/index.html")
  failOnError(err, "Failed on getting template")
  err = t.Execute(w, paper)  
  failOnError(err, "Failed executing template")
}

// for route '/page/:num'
func page(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
  page_num := ps.ByName("num")
  num, _ := strconv.Atoi(page_num)
  
  var pre string; if num > 0 {
    pre = strconv.Itoa(num-1)
  } 
  
  var next string; if num < paper.NumOfPages - 1 {
    next = strconv.Itoa(num+1)
  } 
  
  nums := map[string]string {
    "Pre": pre,
    "Cur": page_num,
    "Next": next,
  }
  
  t, err := template.ParseFiles("templates/page.html")
  failOnError(err, "Failed on getting template")
  err = t.Execute(w, nums)  
  failOnError(err, "Failed executing template")
  
}

// for route '/pdf/:num'
func pdf(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
  page_num := ps.ByName("num")
  num, _ := strconv.Atoi(page_num)
  w.Header().Set("Content-Type", "application/pdf")
  w.Write(paper.Pages[num])  
}

// for route '/image/:num'
func image(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
  page_num := ps.ByName("num")
  num, _ := strconv.Atoi(page_num)
  w.Header().Set("Content-Type", "image/jpg")
  w.Write(paper.Previews[num])  
}

// Preview
// http://www.todayonline.com/sites/default/files/styles/large/public/20142607_AP_page_2.jpg

// PDF
// http://www.todayonline.com/sites/default/files/20142607_AP_page_51.pdf
