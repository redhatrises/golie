package rolie_source

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/rolieup/golie/pkg/models"
)

const (
	feedRootElement        = "feed"
	entryRootElement       = "entry"
	serviceRootElement     = "service"
	atom2005HttpsUri       = "https://www.w3.org/2005/Atom"
	atom2005HttpUri        = "http://www.w3.org/2005/Atom"
	atomPublishingHttpsUri = "https://www.w3.org/2007/app"
	atomPublishingHttpUri  = "http://www.w3.org/2007/app"
)

// Rolie Document. Either Feed, Entry or Service
type Document struct {
	XMLName xml.Name `json:"-"`
	*models.Feed
	*models.Entry
	*models.Service
}

func ReadDocument(r io.Reader) (*Document, error) {
	rawBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	d := xml.NewDecoder(bytes.NewReader(rawBytes))
	for {
		token, err := d.Token()
		if err != nil || token == nil {
			break
		}
		switch startElement := token.(type) {
		case xml.StartElement:
			switch startElement.Name.Local {
			case feedRootElement:
				var feed models.Feed
				if err := d.DecodeElement(&feed, &startElement); err != nil {
					return nil, err
				}
				return &Document{Feed: &feed}, assertAtomNamespace(feed.XMLName.Space)
			case entryRootElement:
				var entry models.Entry
				if err := d.DecodeElement(&entry, &startElement); err != nil {
					return nil, err
				}
				return &Document{Entry: &entry}, assertAtomNamespace(entry.XMLName.Space)
			case serviceRootElement:
				var service models.Service
				if err := d.DecodeElement(&service, &startElement); err != nil {
					return nil, err
				}
				return &Document{Service: &service}, assertAtomPublishingNamespace(service.XMLName.Space)
			}
		}
	}

	var jsonTemp map[string]json.RawMessage
	if err := json.Unmarshal(rawBytes, &jsonTemp); err == nil {
		for k, v := range jsonTemp {
			switch k {
			case feedRootElement:
				var feed models.Feed
				if err := json.Unmarshal(v, &feed); err != nil {
					return nil, err
				}
				return &Document{Feed: &feed}, nil
			case entryRootElement:
				var entry models.Entry
				if err := json.Unmarshal(v, &entry); err != nil {
					return nil, err
				}
				return &Document{Entry: &entry}, nil
			case serviceRootElement:
				var service models.Service
				if err := json.Unmarshal(v, &service); err != nil {
					return nil, err
				}
				return &Document{Service: &service}, nil
			}
		}
	}

	return nil, errors.New("Malformed rolie document. Must be XML or JSON.")
}

func ReadDocumentFromFile(path string) (*Document, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return ReadDocument(reader)
}

func assertAtomNamespace(namespace string) error {
	switch namespace {
	case atom2005HttpsUri:
		fallthrough
	case atom2005HttpUri:
		return nil
	default:
		return fmt.Errorf("Unknown xml namespace '%s' expected %s", namespace, atom2005HttpsUri)
	}
}

func assertAtomPublishingNamespace(namespace string) error {
	switch namespace {
	case atomPublishingHttpsUri:
		fallthrough
	case atomPublishingHttpUri:
		return nil
	default:
		return fmt.Errorf("Unknown xml namespace '%s' expected %s", namespace, atomPublishingHttpsUri)
	}
}