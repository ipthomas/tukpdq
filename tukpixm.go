package tukpixm

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/ipthomas/tukhttp"

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
	Count      int          `json:"count"`
	PID        string       `json:"pid"`
	PIDOID     string       `json:"pidoid"`
	PIX_URL    string       `json:"pixurl"`
	NHS_OID    string       `json:"nhsoid"`
	Region_OID string       `json:"regionoid"`
	Response   []PIXPatient `json:"response"`
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
	pixmreq := tukhttp.PIXmRequest{
		URL:     i.PIX_URL,
		PID_OID: i.PIDOID,
		PID:     i.PID,
	}
	if err := tukhttp.NewRequest(&pixmreq); err != nil {
		return err
	}
	log.Println("Received PIXm Response")
	if strings.Contains(string(pixmreq.Response), "Error") {
		return errors.New(string(pixmreq.Response))
	}
	pixmrsp := PIXmResponse{}
	if err := json.Unmarshal(pixmreq.Response, &pixmrsp); err != nil {
		return err
	}
	log.Printf("%v Patient Entries in Response", pixmrsp.Total)
	i.Count = pixmrsp.Total
	if i.Count > 0 {
		for cnt := 0; cnt < len(pixmrsp.Entry); cnt++ {
			rsppat := pixmrsp.Entry[cnt]
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
			i.Response = append(i.Response, tukpat)
			log.Printf("Added Patient %s to response", tukpat.NHSID)
		}
	} else {
		log.Println("patient is not registered")
	}
	return nil
}
