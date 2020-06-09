package rolie

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocomply/scap/pkg/scap/constants"
	"github.com/gocomply/scap/pkg/scap/scap_document"
	"github.com/rolieup/golie/pkg/models"
	"github.com/rolieup/golie/pkg/rolie_source"
)

func New(directoryPath string) error {
	scapFiles, err := traverseScapFiles(directoryPath)
	if err != nil {
		return err
	}
	for f := range scapFiles {
		fmt.Println("Processing SCAP file: ", f.Path)
		entry, err := f.RolieEntry()
		if err != nil {
			return err
		}
		doc := rolie_source.Document{
			Entry: entry,
		}
		var testJson strings.Builder
		err = doc.JSON(&testJson, true)
		if err != nil {
			return err
		}
		fmt.Printf("\n-------------------\n%s\n", testJson.String())
	}

	return nil
}

type scapFile struct {
	Path string
	*scap_document.Document
	Size         int64
	ModifiedTime time.Time
}

func (scap *scapFile) RolieEntry() (*models.Entry, error) {
	var entry models.Entry
	var err error

	entry.ID = scap.Document.Type.ShortName() + ":" + scap.Path
	entry.Title, err = scap.Title()
	if err != nil {
		return nil, err
	}
	entry.Link = []models.Link{
		models.Link{
			Href:   "www.todo.acme.org/" + scap.Path,
			Length: uint64(scap.Size),
		},
	}
	entry.Updated = models.Time(scap.ModifiedTime)
	entry.Published = models.Time(time.Now())

	return &entry, nil
}

func (scap *scapFile) Title() (string, error) {
	switch scap.Document.Type {
	case constants.DocumentTypeXccdfBenchmark:
		if len(scap.Document.Benchmark.Title) > 0 {
			return scap.Document.Benchmark.Title[0].Text, nil
		}
	case constants.DocumentTypeCpeDict:
		return "CPE Dictionary", nil
	case constants.DocumentTypeOcil:
		return "OCIL Questionaire", nil
	case constants.DocumentTypeOvalDefinitions:
		classes := scap.Document.OvalDefinitions.DefinitionClasses()
		switch len(classes) {
		case 0:
			return "Empty OVAL Definitions", nil
		case 1:
			return "OVAL " + strings.Title(string(classes[0])) + " Definitions", nil
		default:
			return "OVAL Definitions", nil
		}
	case constants.DocumentTypeSourceDataStream:
		datastreams := scap.Document.DataStreamCollection.DataStream
		switch len(datastreams) {
		case 0:
			return "Empty SCAP DataStream Collection", nil
		case 1:
			if datastreams[0].Checklists != nil {
				checklists := datastreams[0].Checklists.ComponentRef
				switch len(checklists) {
				case 0:
					return "SCAP DataStream without checklists", nil
				case 1:
					component := scap.Document.DataStreamCollection.GetComponentByRef(&checklists[0])
					if component != nil && component.DocumentType() == constants.DocumentTypeXccdfBenchmark {
						if len(component.Benchmark.Title) > 0 {
							return "SCAP DataStream for " + component.Benchmark.Title[0].Text, nil
						}
					}
				default:
					return "SCAP DataStream with multiple checklists", nil
				}
			}
		default:
			return "SCAP DataStream Collection", nil
		}

	}
	return "", fmt.Errorf("Not implemented: could not determine sensible atom:title for document '%s'", scap.Path)
}

func traverseScapFiles(directoryPath string) (<-chan scapFile, error) {
	out := make(chan scapFile)

	var err error
	go func() {
		err = filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("Internal error: could not walk the filesystem: %v", err)
			}
			if info.IsDir() || !strings.HasSuffix(path, ".xml") {
				return nil
			}

			doc, err := scap_document.ReadDocumentFromFile(path)
			if err != nil {
				return nil
			}
			out <- scapFile{
				Path:         strings.TrimPrefix(path, directoryPath),
				Document:     doc,
				Size:         info.Size(),
				ModifiedTime: info.ModTime(),
			}
			return nil
		})
		close(out)
	}()
	return out, err
}