package main

import (
	"encoding/xml"
	"strconv"
)

// recordInCache -- Function to check if passed-thorugh record name has been cached
// if so, pass back the Record ID
func recordInCache(recordName, recordType string) (bool, string) {
	boolReturn := false
	strReturn := ""
	switch recordType {
	case "Service":
		//-- Check if record in Service Cache
		mutexServices.Lock()
		for _, service := range services {
			if service.ServiceName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(service.ServiceID)
			}
		}
		mutexServices.Unlock()
	case "Priority":
		//-- Check if record in Priority Cache
		mutexPriorities.Lock()
		for _, priority := range priorities {
			if priority.PriorityName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(priority.PriorityID)
			}
		}
		mutexPriorities.Unlock()
	case "Site":
		//-- Check if record in Site Cache
		mutexSites.Lock()
		for _, site := range sites {
			if site.SiteName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(site.SiteID)
			}
		}
		mutexSites.Unlock()
	case "Team":
		//-- Check if record in Team Cache
		mutexTeams.Lock()
		for _, team := range teams {
			if team.Name == recordName {
				boolReturn = true
				strReturn = team.ID
			}
		}
		mutexTeams.Unlock()
	case "Company":
		mutexCompanies.Lock()
		for _, company := range companies {
			if company.ID == recordName {
				boolReturn = true
				strReturn = company.Name
			}
		}
		mutexCompanies.Unlock()
	case "Organisation":
		//-- Check if record in Org Cache
		mutexOrgs.Lock()
		for _, org := range organisations {
			if org.OrgID == recordName {
				boolReturn = true
				strReturn = org.ContainerID
			}
		}
		mutexOrgs.Unlock()
	}
	return boolReturn, strReturn
}

func userInCache(userID string) (inCache bool, userName, homeOrg string) {
	inCache = false
	userName = ""
	homeOrg = ""
	mutexAnalysts.Lock()
	for _, user := range users {
		if user.UserID == userID {
			inCache = true
			userName = user.Name
			homeOrg = user.HomeOrg
		}
	}
	mutexAnalysts.Unlock()
	return
}

func contactInCache(contactID string) (inCache bool, contactName, contactPK, contactOrgID string) {
	contactName = ""
	contactPK = ""
	contactOrgID = ""
	mutexCustomers.Lock()
	for _, customer := range customers {
		if customer.CustomerID == contactID {
			inCache = true
			contactName = customer.CustomerName
			contactPK = customer.CustomerHornbillID
			contactOrgID = customer.CustomerOrgID
		}
	}
	mutexCustomers.Unlock()
	return
}

func loadOrgs() error {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return err
	}
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("queryName", "getOrganizationContainers")
	XMLOrgSearch, xmlmcErr := espXmlmc.Invoke("data", "queryExec")
	if xmlmcErr != nil {
		return xmlmcErr
	}
	var xmlRespon xmlmcOrgListResponse
	err2 := xml.Unmarshal([]byte(XMLOrgSearch), &xmlRespon)
	if err2 != nil {
		return err2
	}
	mutexOrgs.Lock()
	for index := range xmlRespon.RowResult {
		var newOrgForCache orgListStruct
		newOrgForCache.OrgID = xmlRespon.RowResult[index].OrgID
		newOrgForCache.ContainerID = xmlRespon.RowResult[index].ContainerID
		orgNamedMap := []orgListStruct{newOrgForCache}
		organisations = append(organisations, orgNamedMap...)
	}
	mutexOrgs.Unlock()
	return nil
}

// categoryInCache -- Function to check if passed-thorugh category been cached
// if so, pass back the Category ID and Full Name
func categoryInCache(recordName, recordType string) (bool, string, string) {
	boolReturn := false
	idReturn := ""
	strReturn := ""
	switch recordType {
	case "RequestCategory":
		//-- Check if record in Category Cache
		mutexCategories.Lock()
		for _, category := range categories {
			if category.CategoryCode == recordName {
				boolReturn = true
				idReturn = category.CategoryID
				strReturn = category.CategoryName
			}
		}
		mutexCategories.Unlock()
	case "ClosureCategory":
		//-- Check if record in Category Cache
		mutexCloseCategories.Lock()
		for _, category := range closeCategories {
			if category.CategoryCode == recordName {
				boolReturn = true
				idReturn = category.CategoryID
				strReturn = category.CategoryName
			}
		}
		mutexCloseCategories.Unlock()
	}
	return boolReturn, idReturn, strReturn
}
