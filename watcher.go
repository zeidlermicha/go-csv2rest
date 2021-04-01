package main

import (
	"encoding/csv"
	"encoding/xml"
	"errors"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const MimeCsv = "text/csv; charset=utf-8"

type CSVDataSet struct {
	Data *[][]string
	Meta *[]string
}
type Watcher struct {
	watcher  *fsnotify.Watcher
	dataSets map[string]*CSVDataSet
	mu       sync.Mutex
	filterFn func(path string) bool
}

type Record map[string]string

func NewWatcher(path string, filter func(path string) bool) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dataSets := make(map[string]*CSVDataSet)
	w := &Watcher{watcher: watcher, dataSets: dataSets, filterFn: filter}
	if err := w.init(path, filter); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Watcher) init(path string, filter func(path string) bool) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if filter(path) {
				if err := w.reload(path); err != nil {
					log.Println(err.Error())
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if err := w.watcher.Add(path); err != nil {
			return err
		}
	} else {
		return errors.New("not a folder")
	}
	return nil
}

func (w *Watcher) GetData(ctx *gin.Context) {
	format := ctx.NegotiateFormat(gin.MIMEJSON, gin.MIMEXML, gin.MIMEXML2, gin.MIMEYAML, gin.MIMEHTML, "text/csv")
	path := ctx.Param("any")
	path = path[1:]
	_, ok := w.dataSets[path]
	if !ok {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}
	dataSet := w.dataSets[path]
	switch format {
	case gin.MIMEJSON:
		data := ConvertToObject(dataSet)
		ctx.JSON(http.StatusOK, data)
		break
	case gin.MIMEYAML:
		data := ConvertToObject(dataSet)
		ctx.YAML(http.StatusOK, data)
		break
	case gin.MIMEXML:
	case gin.MIMEXML2:
		data := ConvertToObject(dataSet)
		ctx.XML(http.StatusOK, data)
		break
	case gin.MIMEHTML:
		ctx.HTML(http.StatusOK, "info.tmpl", gin.H{"Info": w.dataSets, "Data": dataSet})
		break
	case "text/csv":
		ctx.Render(http.StatusOK, dataSet)
	default:
		ctx.AbortWithStatus(http.StatusBadRequest)
	}

}

func (w *Watcher) Watch() {
	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					if w.filterFn(event.Name) {
						w.mu.Lock()
						if err := w.reload(event.Name); err != nil {
							log.Println(err)
						}
						w.mu.Unlock()
					}
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
}

func (w *Watcher) reload(path string) error {
	start := time.Now()
	r, err := ReadCSV(path)
	if err != nil {
		return err
	}
	records, err := r.ReadAll()
	if err != nil {
		return err
	}
	if len(records) > 0 {
		dataSet := CSVDataSet{Meta: &records[0]}
		if len(records) > 1 {
			r := records[1:]
			dataSet.Data = &r
		} else {
			data := make([][]string, 0)
			dataSet.Data = &data
		}
		w.dataSets[path] = &dataSet
		log.Printf("Reloaded %s in %s\n", path, time.Since(start))
	} else {
		log.Printf("File %s has no records", path)
	}
	return nil
}

func ReadCSV(path string) (*csv.Reader, error) {
	csvFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(csvFile)
	return r, nil
}

func (w *Watcher) GetInfo(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "info.tmpl", gin.H{"Info": w.dataSets})
}

func (w *Watcher) Close() error {
	return w.watcher.Close()
}

func ConvertToObject(dataSet *CSVDataSet) []*Record {
	result := make([]*Record, 0)
	meta := *dataSet.Meta
	for _, row := range *dataSet.Data {
		m := make(Record)
		for i, entry := range row {
			m[meta[i]] = entry
		}
		result = append(result, &m)
	}
	return result
}

func (o *Record) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	m := *o
	for k, v := range m {
		if err := e.EncodeElement(v, xml.StartElement{Name: xml.Name{Local: k}}); err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}
func (r *CSVDataSet) Render(w http.ResponseWriter) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	w.Header().Set("Content-Type", MimeCsv)
	return csvWriter.WriteAll(*r.Data)
}

func (r *CSVDataSet) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", MimeCsv)
}
