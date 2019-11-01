package main

import (
	"strconv"

	"fmt"

	"encoding/xml"
	//	"github.com/hornbill/goApiLib"
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
			if team.TeamName == recordName {
				boolReturn = true
				strReturn = team.TeamID
			}
		}
		mutexTeams.Unlock()
	case "Analyst":
		//-- Check if record in Analyst Cache
		mutexAnalysts.Lock()
		for _, analyst := range analysts {
			if analyst.AnalystID == recordName {
				boolReturn = true
				strReturn = analyst.AnalystName
			}
		}
		mutexAnalysts.Unlock()
	case "Customer":
		//-- Check if record in Customer Cache
		mutexCustomers.Lock()
		for _, customer := range customers {
			if customer.CustomerID == recordName {
				boolReturn = true
				strReturn = customer.CustomerName + "{||||||}" + customer.CustomerHornbillID + "{||||||}" + customer.CustomerOrgID
				//strReturn = customer.CustomerHornbillID
			}
		}
		mutexCustomers.Unlock()
	case "Company":
		//-- Check if record in Customer Cache
		mutexCompanies.Lock()
		for _, company := range companies {
			if company.OrgID == recordName {
				boolReturn = true
				strReturn = company.ContainerID
				//strReturn = customer.CustomerHornbillID
			}
		}
		mutexCompanies.Unlock()
	}
	return boolReturn, strReturn
}

func loadOrgs() {

	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		fmt.Println("AAA")
		return
	}

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("queryName", "getOrganizationContainers")
	XMLOrgSearch, xmlmcErr := espXmlmc.Invoke("data", "queryExec")
	if xmlmcErr != nil {
		return
	}

	var xmlRespon xmlmcOrgListResponse
	//fmt.Println(XMLOrgSearch)
	err2 := xml.Unmarshal([]byte(XMLOrgSearch), &xmlRespon)
	if err2 != nil {
		return
	}

	mutexCompanies.Lock()

	for index := range xmlRespon.RowResult {
		var newOrgForCache companyListStruct
		newOrgForCache.OrgID = xmlRespon.RowResult[index].OrgID
		newOrgForCache.ContainerID = xmlRespon.RowResult[index].ContainerID
		companyNamedMap := []companyListStruct{newOrgForCache}
		companies = append(companies, companyNamedMap...)
	}

	mutexCompanies.Unlock()

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
