package gojenkins

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
)

type Credentials struct {
	Raw     *CredentialsResponse
	Jenkins *Jenkins
	Base    string
}

type UserCredential struct {
	Description string `xml:"description"`
	DisplayName string `xml:"displayName"`
	Fingerprint string `xml:"fingerprint"`
	FullName    string `xml:"fullName"`
	ID          string `xml:"id"`
	TypeName    string `xml:"typeName"`
}

type DomainWrapper struct {
	XMLName         xml.Name         `xml:"domainWrapper"`
	Class           string           `xml:"_class,attr"`
	Description     string           `xml:"description"`
	DisplayName     string           `xml:"displayName"`
	FullDisplayName string           `xml:"fullDisplayName"`
	FullName        string           `xml:"fullName"`
	Global          string           `xml:"global"`
	URLName         string           `xml:"urlName"`
	UserCredentials []UserCredential `xml:"credential"`
}

type CredentialsResponse struct {
}

func (c Credentials) Create(credentialsData string) error {
	resp, err := c.Jenkins.Requester.Post(c.Base+"createCredentials", bytes.NewBufferString(credentialsData), c.Raw, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Error with response code %d", resp.StatusCode))
	}
	return nil
}

func (c Credentials) GetAll() ([]UserCredential, error) {
	var data string
	endpoint := c.Base + "api/xml"
	qeuryString := map[string]string{
		"depth": "1",
	}
	_, err := c.Jenkins.Requester.GetXML(endpoint, &data, qeuryString)
	if err != nil {
		return nil, err
	}
	var result DomainWrapper
	err = xml.Unmarshal(bytes.NewBufferString(data).Bytes(), &result)
	if err != nil {
		return nil, err
	}
	return result.UserCredentials, nil
}

// Remove /credentials/store/system/domain/_/credential/auto-test-888/doDelete
func (c Credentials) Remove(credentialsID string) error {
	endpoint := fmt.Sprintf("%scredential/%s/doDelete", c.Base, credentialsID)
	resp, err := c.Jenkins.Requester.Post(endpoint, nil, nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 302 {
		return fmt.Errorf("Remove credentials %s with status code %d", credentialsID, resp.StatusCode)
	}
	return nil
}
