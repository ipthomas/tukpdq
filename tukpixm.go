package tukpixm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	cnst "github.com/ipthomas/tukcnst"
)

type PIXmResponse struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id"`
	Type         string `json:"type"`
	Total        int    `json:"total"`
	Link         []struct {
		Relation string `json:"relation"`
		URL      string `json:"url"`
	} `json:"link"`
	Entry []struct {
		FullURL  string `json:"fullUrl"`
		Resource struct {
			ResourceType string `json:"resourceType"`
			ID           string `json:"id"`
			Identifier   []struct {
				Use    string `json:"use,omitempty"`
				System string `json:"system"`
				Value  string `json:"value"`
			} `json:"identifier"`
			Active bool `json:"active"`
			Name   []struct {
				Use    string   `json:"use"`
				Family string   `json:"family"`
				Given  []string `json:"given"`
			} `json:"name"`
			Gender    string `json:"gender"`
			BirthDate string `json:"birthDate"`
			Address   []struct {
				Use        string   `json:"use"`
				Line       []string `json:"line"`
				City       string   `json:"city"`
				PostalCode string   `json:"postalCode"`
				Country    string   `json:"country"`
			} `json:"address"`
		} `json:"resource"`
	} `json:"entry"`
}
type PIXmQuery struct {
	Count        int          `json:"count"`
	PID          string       `json:"pid"`
	PIDOID       string       `json:"pidoid"`
	PIX_URL      string       `json:"pixurl"`
	NHS_OID      string       `json:"nhsoid"`
	Region_OID   string       `json:"regionoid"`
	Timeout      int64        `json:"timeout"`
	StatusCode   int          `json:"statuscode"`
	Response     []byte       `json:"response"`
	PIXmResponse PIXmResponse `json:"pixmresponse"`
	Patients     []PIXPatient `json:"patients"`
}
type PIXPatient struct {
	PIDOID     string `json:"pidoid"`
	PID        string `json:"pid"`
	REGOID     string `json:"regoid"`
	REGID      string `json:"regid"`
	NHSOID     string `json:"nhsoid"`
	NHSID      string `json:"nhsid"`
	GivenName  string `json:"givenname"`
	FamilyName string `json:"familyname"`
	Gender     string `json:"gender"`
	BirthDate  string `json:"birthdate"`
	Street     string `json:"street"`
	Town       string `json:"town"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	Zip        string `json:"zip"`
}
type PIXmInterface interface {
	pdq() error
}

func PDQ(i PIXmInterface) error {
	return i.pdq()
}
func (i *PIXmQuery) pdq() error {
	if err := i.newRequest(); err != nil {
		return err
	}
	if strings.Contains(string(i.Response), "Error") {
		return errors.New(string(i.Response))
	}
	i.PIXmResponse = PIXmResponse{}
	if err := json.Unmarshal(i.Response, &i.PIXmResponse); err != nil {
		return err
	}
	log.Printf("%v Patient Entries in Response", i.PIXmResponse.Total)
	i.Count = i.PIXmResponse.Total
	if i.Count > 0 {
		for cnt := 0; cnt < len(i.PIXmResponse.Entry); cnt++ {
			rsppat := i.PIXmResponse.Entry[cnt]
			tukpat := PIXPatient{}
			for _, id := range rsppat.Resource.Identifier {
				if id.System == cnst.URN_OID_PREFIX+i.Region_OID {
					tukpat.REGID = id.Value
					tukpat.REGOID = i.Region_OID
					log.Printf("Set Reg ID %s %s", tukpat.REGID, tukpat.REGOID)
				}
				if id.Use == "usual" {
					tukpat.PID = id.Value
					tukpat.PIDOID = strings.Split(id.System, ":")[2]
					log.Printf("Set PID %s %s", tukpat.PID, tukpat.PIDOID)
				}
				if id.System == cnst.URN_OID_PREFIX+i.NHS_OID {
					tukpat.NHSID = id.Value
					tukpat.NHSOID = i.NHS_OID
					log.Printf("Set NHS ID %s %s", tukpat.NHSID, tukpat.NHSOID)
				}
			}
			gn := ""
			for _, name := range rsppat.Resource.Name {
				for _, n := range name.Given {
					gn = gn + n + " "
				}
			}

			tukpat.GivenName = strings.TrimSuffix(gn, " ")
			tukpat.FamilyName = rsppat.Resource.Name[0].Family
			tukpat.BirthDate = strings.ReplaceAll(rsppat.Resource.BirthDate, "-", "")
			tukpat.Gender = rsppat.Resource.Gender

			if len(rsppat.Resource.Address) > 0 {
				tukpat.Zip = rsppat.Resource.Address[0].PostalCode
				if len(rsppat.Resource.Address[0].Line) > 0 {
					tukpat.Street = rsppat.Resource.Address[0].Line[0]
					if len(rsppat.Resource.Address[0].Line) > 1 {
						tukpat.Town = rsppat.Resource.Address[0].Line[1]
					}
				}
				tukpat.City = rsppat.Resource.Address[0].City
				tukpat.Country = rsppat.Resource.Address[0].Country
			}
			i.Patients = append(i.Patients, tukpat)
			log.Printf("Added Patient %s to response", tukpat.NHSID)
		}
	} else {
		log.Println("patient is not registered")
	}
	return nil
}
func (i *PIXmQuery) newRequest() error {
	if i.PID == "" || i.Region_OID == "" || i.PIX_URL == "" {
		return errors.New("invalid request, not all mandated values provided in pixmquery (pid, region_oid and pix_url)")
	}
	if i.Timeout == 0 {
		i.Timeout = 5
	}
	if i.NHS_OID == "" {
		i.NHS_OID = "2.16.840.1.113883.2.1.4.1"
	}
	if i.PIDOID == "" && len(i.PID) == 10 {
		i.PIDOID = i.NHS_OID
	}
	i.PIX_URL = i.PIX_URL + "?identifier=" + i.PIDOID + "%7C" + i.PID + cnst.FORMAT_JSON_PRETTY
	req, _ := http.NewRequest(cnst.HTTP_GET, i.PIX_URL, nil)
	req.Header.Set(cnst.CONTENT_TYPE, cnst.APPLICATION_JSON)
	req.Header.Set(cnst.ACCEPT, cnst.ALL)
	req.Header.Set(cnst.CONNECTION, cnst.KEEP_ALIVE)
	i.logRequest(req.Header)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(i.Timeout)*time.Second)
	defer cancel()
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	i.StatusCode = resp.StatusCode
	i.Response, err = io.ReadAll(resp.Body)
	defer resp.Body.Close()
	i.logResponse()
	return err
}
func (i *PIXmQuery) logRequest(headers http.Header) {
	log.Println("HTTP GET Request Headers")
	b, _ := json.MarshalIndent(headers, "", "  ")
	log.Println(string(b))
	log.Printf("HTTP Request\nURL = %s", i.PIX_URL)
}
func (i *PIXmQuery) logResponse() {
	log.Printf("HTML Response - Status Code = %v\n%s", i.StatusCode, string(i.Response))
}
