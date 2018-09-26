package main

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

//getCallServiceID takes the Call Record and returns a correct Service ID if one exists on the Instance
func getCallServiceID(swService string) string {
	serviceID := ""
	serviceName := ""
	if swImportConf.ServiceMapping[swService] != nil {
		serviceName = fmt.Sprintf("%s", swImportConf.ServiceMapping[swService])

		if serviceName != "" {
			serviceID = getServiceID(serviceName)
		}
	}
	return serviceID
}

//getServiceID takes a Service Name string and returns a correct Service ID if one exists in the cache or on the Instance
func getServiceID(serviceName string) string {
	serviceID := ""
	if serviceName != "" {
		serviceIsInCache, ServiceIDCache := recordInCache(serviceName, "Service")
		//-- Check if we have cached the Service already
		if serviceIsInCache {
			serviceID = ServiceIDCache
		} else {
			serviceIsOnInstance, ServiceIDInstance := searchService(serviceName)
			//-- If Returned set output
			if serviceIsOnInstance {
				serviceID = strconv.Itoa(ServiceIDInstance)
			}
		}
	}
	return serviceID
}

// seachService -- Function to check if passed-through service name is on the instance
func searchService(serviceName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}
	//-- ESP Query for service
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Services")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	//espXmlmc.SetParam("h_servicename", serviceName)
	espXmlmc.SetParam("column", "h_servicename")
	espXmlmc.SetParam("value", serviceName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLServiceSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Service: "+fmt.Sprintf("%v", xmlmcErr), false)
		//log.Fatal(xmlmcErr)
		return boolReturn, intReturn
	}
	var xmlRespon xmlmcServiceListResponse

	err = xml.Unmarshal([]byte(XMLServiceSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Service: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to Search for Service: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.ServiceName != "" {
				if strings.ToLower(xmlRespon.ServiceName) == strings.ToLower(serviceName) {
					intReturn = xmlRespon.ServiceID
					boolReturn = true
					//-- Add Service to Cache
					var newServiceForCache serviceListStruct
					newServiceForCache.ServiceID = intReturn
					newServiceForCache.ServiceName = serviceName
					newServiceForCache.ServiceBPMIncident = xmlRespon.BPMIncident
					newServiceForCache.ServiceBPMService = xmlRespon.BPMService
					newServiceForCache.ServiceBPMChange = xmlRespon.BPMChange
					newServiceForCache.ServiceBPMProblem = xmlRespon.BPMProblem
					newServiceForCache.ServiceBPMKnownError = xmlRespon.BPMKnownError
					serviceNamedMap := []serviceListStruct{newServiceForCache}
					mutexServices.Lock()
					services = append(services, serviceNamedMap...)
					mutexServices.Unlock()
				}
			}
		}
	}
	//Return Service ID once cached - we can now use this in the calling function to get all details from cache
	return boolReturn, intReturn
}
